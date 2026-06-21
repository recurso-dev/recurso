# Spec: Phase 4 - Text-to-SQL GenAI Analytics 🤖

## Objective
To provide a natural language interface for financial and billing analytics. Users should be able to ask questions like "What was my churn rate last month?" or "Show me the top 5 customers by revenue in INR" and receive a data-driven answer generated via SQL.

### User Stories
- As a CFO, I want to query billing data without knowing SQL or complex BI tools.
- As a developer, I want to provide conversational analytics to my end-users.

## Tech Stack
- **Language**: Go 1.20+
- **LLM**: OpenAI GPT-4o or Gemini 1.5 Pro (Configurable via Adapter)
- **Database**: PostgreSQL (ReadOnly access for AI-generated queries)

## Commands
- Build: `make build`
- Test: `go test ./internal/service/genai_test.go`

## Project Structure
- `internal/core/port/ai.go`         → Interface for LLM providers
- `internal/adapter/ai/openai.go`    → implementation of OpenAI adapter
- `internal/service/genai.go`        → Business logic for schema-to-prompt and query execution
- `internal/adapter/handler/analytics.go` → Updated to include `POST /v1/analytics/ask`

## Architecture: RAG-lite for SQL
1. **Context Extraction**: The service will maintain a "Schema Map" of public tables (Customers, Invoices, Subscriptions, Plans).
2. **Prompt Engineering**: 
   - System Prompt: "You are a senior PostgreSQL expert. Given the schema below, generate a SQL query to answer the user's question. Return ONLY the SQL code."
   - Schema: DDL for relevant tables.
3. **Execution**: The generated SQL is executed against a Read-Only database connection.
4. **Result Formatting**: Returns the raw data (JSON) and optionally a text summary.

## Security & Boundaries
- **Always**: Use a restricted Read-Only DB user for AI queries. 
- **Always**: Validate the generated SQL starts with `SELECT` (No mutations).
- **Always**: Sanitize user input to prevent prompt injection.
- **Never**: Include sensitive data like API keys or passwords in the schema context sent to the LLM.
- **Never**: Allow the AI to see the `tenants` or `api_keys` tables directly.

## Success Criteria
- [ ] Pluggable LLM interface implemented.
- [ ] Schema-to-SQL prompt successfully generates valid PostgreSQL queries.
- [ ] `/v1/analytics/ask` endpoint returns correct data for "Total revenue per currency".
- [ ] System rejects non-SELECT queries.
- [ ] Support for tenant isolation (AI queries must be scoped to the current `tenant_id`).

## Open Questions
1. Should we use a separate "Analytics" DB or a Read-Only replica of the main DB? (Initial: Read-Only user on main DB for simplicity).
2. How to handle large result sets? (Initial: Limit to top 100 rows).
