# Tasks: Text-to-SQL Analytics

## Phase 1: AI Abstraction
- [ ] **Task 1.1**: Create `internal/core/port/ai.go`.
  - Acceptance: Interface `LLMProvider` exists.
  - Verify: Code compiles.
- [ ] **Task 1.2**: Create `internal/adapter/ai/openai.go`.
  - Acceptance: Implementation for `GenerateCompletion`.
  - Verify: Mock test or build.

## Phase 2: Analytics Service
- [ ] **Task 2.1**: Implement `internal/service/genai.go`.
  - Acceptance: Logic for `GenerateSQL` and `ExecuteQuery`.
  - Verify: Unit test with mock LLM provider.
- [ ] **Task 2.2**: Implement SQL sanitization and tenant scoping.
  - Acceptance: All queries automatically include `tenant_id` check.
  - Verify: Test with query missing the tenant clause.

## Phase 3: API Implementation
- [ ] **Task 3.1**: Update `internal/adapter/handler/analytics.go`.
  - Acceptance: New route `/v1/analytics/ask`.
  - Verify: Curl request returns JSON data.
- [ ] **Task 3.2**: Wire in `main.go`.
  - Acceptance: Service initialized with `OPENAI_API_KEY`.
  - Verify: Build check.

## Phase 4: Documentation
- [ ] **Task 4.1**: Update `docs/api_contract.md`.
  - Acceptance: New endpoint documented.
