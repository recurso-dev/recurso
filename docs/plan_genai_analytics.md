# Implementation Plan: Text-to-SQL Analytics 🤖

## Overview
This plan outlines the steps to build a natural language to SQL analytics engine. The engine will bridge the gap between user questions and the PostgreSQL database using an LLM.

## Phase 1: Port & Provider
1.  **Define Port**: Create `internal/core/port/ai.go` with `LLMProvider` interface.
2.  **OpenAI Adapter**: Implement a basic adapter that calls the OpenAI chat completion API.

## Phase 2: Schema Context & Prompting
1.  **Schema Definition**: Hardcode the DDL for `customers`, `invoices`, `subscriptions`, and `plans` to be used in the prompt.
2.  **Prompt Builder**: Logic to combine system instructions, schema, and user query.

## Phase 3: SQL Service
1.  **GenAIService**:
    - `GenerateSQL(question, tenantID)`: Calls LLM and ensures `WHERE tenant_id = ...` is included.
    - `ExecuteQuery(sql)`: Runs the SQL against a Read-Only DB handle and returns a map of results.

## Phase 4: API & Integration
1.  **Endpoint**: Implement `POST /v1/analytics/ask` in `internal/adapter/handler/analytics.go`.
2.  **Mock Mode**: If no API key is provided, return a mock SQL result for demo purposes.

## Phase 5: Verification
1.  **Unit Tests**: Test prompt generation and SQL validation logic.
2.  **Manual Test**: Ask "Show me my total revenue" and verify it maps to a `SELECT SUM(total)` query.

## Risks & Mitigations
- **Prompt Injection**: LLM might generate a `DROP TABLE` query. 
  - *Mitigation*: Regex check for `^SELECT` and use a Read-Only DB user with restricted permissions.
- **Accuracy**: LLM might hallucinate column names.
  - *Mitigation*: Provide explicit column list in the system prompt.
