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

// ErrGenAICannotAnswer marks the case where the model declines a question because
// it can't be satisfied from the limited genai schema (e.g. "MRR growth over the
// last 3 months" — there is no MRR time-series in the exposed views). This is an
// EXPECTED, user-facing outcome, not a server fault, so handlers map it to 422
// rather than a 500 "internal error".
var ErrGenAICannotAnswer = errors.New("question not answerable from the available data")

// GenAICannotAnswerMessage is the user-facing text shown verbatim in the Ask AI
// panel when the model declines (paired with ErrGenAICannotAnswer).
const GenAICannotAnswerMessage = "I couldn't answer that from the data available here. Try rephrasing, or ask about customers, invoices, subscriptions, plans, or prices — e.g. \"which plan has the most active subscriptions?\""

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
6. MRR / recurring-revenue questions use mrr_snapshots (a daily time series, one
   row per active subscription per day). Total MRR on a date = SUM(mrr_amount)
   over rows with that snapshot_date. Treat MAX(snapshot_date) as "now"/current
   MRR. For "growth over N months", compare current-day SUM(mrr_amount) to the
   snapshot_date on/nearest to N months earlier. For churn/trend questions, group
   by snapshot_date (or plan_id) across the series.
7. For PER-ENTITY questions (e.g. "what is <Entity>'s MRR growth?"), join
   mrr_snapshots.entity_id to entities.id and filter by entities.name (case-
   insensitive: ILIKE). mrr_snapshots.entity_id is always the concrete entity
   (the primary entity for primary subscriptions), so a plain equality/join works.
   Grouping MRR by entities.name gives a per-entity breakdown.

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

-- mrr_snapshots — daily Monthly Recurring Revenue history. One row per active
-- subscription per day; use it for MRR totals, growth, churn, and trends over
-- time (see rule 6). Total MRR on a day = SUM(mrr_amount) for that snapshot_date.
CREATE TABLE mrr_snapshots (
    subscription_id UUID,
    customer_id UUID,
    plan_id UUID,
    entity_id UUID,       -- the legal entity this MRR belongs to (see rule 7)
    snapshot_date DATE,   -- the day this MRR figure was recorded
    mrr_amount BIGINT,    -- this subscription's MRR that day, in minor units
    created_at TIMESTAMP
);

-- entities — the tenant's legal entities (Multi-Entity Books). Join
-- mrr_snapshots.entity_id = entities.id to scope/group MRR by entity (rule 7).
CREATE TABLE entities (
    id UUID PRIMARY KEY,
    name VARCHAR(255),
    legal_name VARCHAR(255),
    is_primary BOOLEAN,
    country_code VARCHAR(2),
    created_at TIMESTAMP
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
		// The model followed its instruction to decline (question not answerable
		// with the exposed schema). Surface a friendly, actionable message — not a
		// 500. The model's own reason is logged for debugging but not shown raw.
		return nil, "", fmt.Errorf("%w (model said: %s)", ErrGenAICannotAnswer, strings.TrimSpace(strings.TrimPrefix(sqlQuery, "ERROR:")))
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
