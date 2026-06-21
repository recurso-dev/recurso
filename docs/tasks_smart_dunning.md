# Tasks: Smart Dunning (RL-Powered)

## Phase 1: Database & Domain
- [ ] **Task 1.1**: Create `internal/core/domain/dunning.go` with bandit types.
  - Acceptance: Structs for `DunningContext`, `DunningAction`, and `DunningWeight` exist.
  - Verify: Code compiles.
- [ ] **Task 1.2**: Create migration `000029_create_dunning_tables.up.sql`.
  - Acceptance: Tables `dunning_history` and `dunning_weights` defined.
  - Verify: Run `make run` and check schema in DB.

## Phase 2: Bandit Service
- [ ] **Task 2.1**: Implement `SmartRetryService.SelectAction` in `internal/core/service/smart_retry.go`.
  - Acceptance: Returns an interval based on epsilon-greedy logic.
  - Verify: Unit test with mock weights.
- [ ] **Task 2.2**: Implement `SmartRetryService.UpdateWeights`.
  - Acceptance: Correctly updates average rewards in memory/DB.
  - Verify: Unit test showing weight shift after 100 successful "rewards".

## Phase 3: Integration
- [ ] **Task 3.1**: Connect `RetryWorker` to `SmartRetryService`.
  - Acceptance: Worker no longer uses static 24h interval.
  - Verify: E2E test shows non-static `next_retry_at`.
- [ ] **Task 3.2**: Implement reward callback in payment success/fail handlers.
  - Acceptance: Outcome is recorded in `dunning_history`.
  - Verify: Database check after successful payment.

## Phase 4: Final Verification
- [ ] **Task 4.1**: Create simulation test `internal/service/smart_dunning_sim_test.go`.
  - Acceptance: Agent converges on optimal arm.
  - Verify: `go test -v internal/service/smart_dunning_sim_test.go`.
