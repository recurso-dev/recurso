package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/port"
)

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

const systemPromptTemplate = `You are a senior PostgreSQL expert for Recurso, a billing engine.
Given the schema below, generate a SQL query to answer the user's question.

RULES:
1. Return ONLY the SQL code. No explanation, no Markdown formatting like ` + "```sql" + `.
2. All queries MUST be scoped to the tenant using "tenant_id = '%s'".
3. Use only SELECT statements.
4. If the question cannot be answered with the schema, return "ERROR: I cannot answer that question with the current data."

SCHEMA:
-- Customers
CREATE TABLE customers (
    id UUID PRIMARY KEY,
    tenant_id UUID,
    email VARCHAR(255),
    name VARCHAR(255),
    tax_type VARCHAR(20), -- 'business', 'consumer'
    created_at TIMESTAMP
);

-- Invoices
CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    tenant_id UUID,
    customer_id UUID,
    invoice_number VARCHAR(50),
    status VARCHAR(20), -- 'paid', 'open', 'void', 'past_due'
    currency VARCHAR(3),
    subtotal BIGINT, -- in cents
    tax_amount BIGINT,
    total BIGINT,
    created_at TIMESTAMP
);

-- Subscriptions
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    tenant_id UUID,
    customer_id UUID,
    plan_id UUID,
    status VARCHAR(20), -- 'active', 'canceled', 'paused', 'past_due'
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    created_at TIMESTAMP
);

-- Plans
CREATE TABLE plans (
    id UUID PRIMARY KEY,
    tenant_id UUID,
    name VARCHAR(255),
    code VARCHAR(50),
    active BOOLEAN
);
`

func (s *GenAIService) Ask(ctx context.Context, tenantID uuid.UUID, question string) (interface{}, string, error) {
	// 1. Generate SQL
	systemPrompt := fmt.Sprintf(systemPromptTemplate, tenantID.String())
	sqlQuery, err := s.llm.GenerateCompletion(ctx, systemPrompt, question)
	if err != nil {
		return nil, "", err
	}

	// Basic safety checks
	sqlQuery = strings.TrimSpace(sqlQuery)
	if strings.HasPrefix(sqlQuery, "ERROR:") {
		return nil, "", fmt.Errorf("%s", sqlQuery)
	}

	// Force SELECT check
	if !strings.HasPrefix(strings.ToUpper(sqlQuery), "SELECT") {
		return nil, "", fmt.Errorf("AI generated a non-SELECT query: %s", sqlQuery)
	}

	// 2. Execute Query
	rows, err := s.db.QueryContext(ctx, sqlQuery)
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
