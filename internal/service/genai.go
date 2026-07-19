package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// ErrGenAINotConfigured is returned when no LLM provider is wired (e.g.
// OPENAI_API_KEY unset). Callers map it to 503 rather than a 500.
var ErrGenAINotConfigured = errors.New("genai analytics is not configured on this server")

type GenAIService struct {
	llm port.LLMProvider
	db  *sql.DB
}

func NewGenAIService(llm port.LLMProvider, db *sql.DB) *GenAIService {
	return &GenAIService{
		llm: llm,
		db:  db,
	}
}

const systemPrompt = `You are a senior PostgreSQL expert for Recurso, a billing engine.
Given the schema below, generate a SQL query to answer the user's question.

RULES:
1. Return ONLY the SQL code. No explanation, no Markdown formatting like ` + "```sql" + `.
2. Use only SELECT statements — a single statement, no semicolons.
3. Only the tables below exist. They are already scoped to the current
   account; there is no tenant column and no filtering is needed.
4. All monetary amounts are integers in the currency's smallest unit
   (cents/paise) — divide by 100.0 for display values.
5. If the question cannot be answered with the schema, return "ERROR: I cannot answer that question with the current data."

SCHEMA:
-- customers
CREATE TABLE customers (
    id UUID PRIMARY KEY,
    email VARCHAR(255),
    name VARCHAR(255),
    tax_type VARCHAR(20), -- 'business', 'consumer'
    created_at TIMESTAMP
);

-- invoices
CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    customer_id UUID,
    invoice_number VARCHAR(50),
    status VARCHAR(20), -- 'paid', 'open', 'void', 'past_due', 'draft', 'uncollectible'
    currency VARCHAR(3),
    subtotal BIGINT,
    tax_amount BIGINT,
    total BIGINT,
    amount_paid BIGINT,
    due_date TIMESTAMP,
    paid_at TIMESTAMP,
    created_at TIMESTAMP
);

-- subscriptions
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    customer_id UUID,
    plan_id UUID,
    status VARCHAR(20), -- 'active', 'trialing', 'canceled', 'paused', 'past_due'
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    trial_end TIMESTAMP,
    created_at TIMESTAMP
);

-- plans
CREATE TABLE plans (
    id UUID PRIMARY KEY,
    name VARCHAR(255),
    code VARCHAR(50),
    interval_unit VARCHAR(10),
    interval_count INT,
    active BOOLEAN,
    created_at TIMESTAMP
);

-- prices
CREATE TABLE prices (
    id UUID PRIMARY KEY,
    plan_id UUID,
    currency VARCHAR(3),
    amount BIGINT,
    type VARCHAR(20)
);
`

func (s *GenAIService) Ask(ctx context.Context, tenantID uuid.UUID, question string) (interface{}, string, error) {
	// No provider (OPENAI_API_KEY unset): fail explicitly instead of
	// panicking on the nil interface below.
	if s.llm == nil {
		return nil, "", ErrGenAINotConfigured
	}

	// 1. Generate SQL. The prompt never sees the tenant id — isolation is
	// enforced by the database below, not by the model.
	sqlQuery, err := s.llm.GenerateCompletion(ctx, systemPrompt, question)
	if err != nil {
		return nil, "", err
	}

	sqlQuery = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sqlQuery), ";"))
	if strings.HasPrefix(sqlQuery, "ERROR:") {
		return nil, "", fmt.Errorf("%s", sqlQuery)
	}
	if err := guardGeneratedSQL(sqlQuery); err != nil {
		return nil, sqlQuery, err
	}

	// 2. Execute under the genai_readonly role (ENG-137). The role can see
	// ONLY the genai schema of tenant-scoped views, so even a fully
	// adversarial query cannot read other tenants' rows or sensitive tables —
	// it gets "permission denied" from Postgres, not a prompt suggestion.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, sqlQuery, err
	}
	defer func() { _ = tx.Rollback() }()

	for _, setup := range []string{
		"SET LOCAL ROLE genai_readonly",
		"SET LOCAL search_path = genai",
		"SET LOCAL statement_timeout = '5s'",
	} {
		if _, err := tx.ExecContext(ctx, setup); err != nil {
			return nil, sqlQuery, fmt.Errorf("failed to prepare sandboxed query session: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
		return nil, sqlQuery, fmt.Errorf("failed to scope query session: %w", err)
	}

	// Cap the result set regardless of what the model generated.
	wrapped := "SELECT * FROM (" + sqlQuery + ") AS genai_result LIMIT 500"

	rows, err := tx.QueryContext(ctx, wrapped)
	if err != nil {
		return nil, sqlQuery, fmt.Errorf("failed to execute AI-generated query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// 3. Parse Results into dynamic slice of maps
	columns, err := rows.Columns()
	if err != nil {
		return nil, sqlQuery, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, sqlQuery, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	return results, sqlQuery, nil
}

// guardGeneratedSQL is belt-and-braces on top of the genai_readonly role:
// single SELECT statement, no schema escapes. The role remains the real
// security boundary — these checks just fail obvious abuse faster.
func guardGeneratedSQL(q string) error {
	upper := strings.ToUpper(q)
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return fmt.Errorf("AI generated a non-SELECT query")
	}
	if strings.Contains(q, ";") {
		return fmt.Errorf("AI generated a multi-statement query")
	}
	lower := strings.ToLower(q)
	for _, denied := range []string{"pg_", "information_schema", "current_setting", "set_config", "public."} {
		if strings.Contains(lower, denied) {
			return fmt.Errorf("AI-generated query references a restricted identifier (%s)", denied)
		}
	}
	return nil
}
