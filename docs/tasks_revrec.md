# Tasks: Revenue Recognition

## Phase 1: Database & Domain
- [ ] **Task 1.1**: Create `internal/core/domain/revrec.go`.
  - Acceptance: Structs for `RevenueSchedule` and `RecognitionEvent` exist.
  - Verify: Code compiles.
- [ ] **Task 1.2**: Create migration `000031_create_revrec_tables.up.sql`.
  - Acceptance: Schema for revrec persistence is live.
  - Verify: Tables exist in PG.

## Phase 2: RevRec Logic
- [ ] **Task 2.1**: Implement `internal/service/revrec.go` with allocation logic.
  - Acceptance: Successfully splits a $120 annual sub into 12 x $10 monthly events.
  - Verify: Unit test `TestCalculateMonthlyAllocation`.
- [ ] **Task 2.2**: Implement `RevRecRepository`.
  - Acceptance: CRUD for schedules and events.
  - Verify: Integration test with DB.

## Phase 3: Integration & Worker
- [ ] **Task 3.1**: Trigger schedule creation on payment success.
  - Acceptance: Paid invoices automatically get a RevRec schedule.
  - Verify: Create a paid invoice in E2E and check `revenue_schedules` table.
- [ ] **Task 3.2**: Implement `RevRecWorker`.
  - Acceptance: Background job processes due recognition events.
  - Verify: Log showing "Recognized $X for Tenant Y".
- [ ] **Task 3.3**: TigerBeetle Integration.
  - Acceptance: Recognition events trigger real TB transfers.
  - Verify: Check TB account balances after worker run.

## Phase 4: Reporting
- [ ] **Task 4.1**: Implement `GET /v1/finance/revrec/report`.
  - Acceptance: Returns monthly recognized vs deferred totals.
  - Verify: Curl request matches expected accounting values.
