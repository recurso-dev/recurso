# Spec: Phase 4 - Smart Dunning (RL-Powered) 🧠

## Objective
To improve payment recovery rates by using Reinforcement Learning (Contextual Bandits) to determine the optimal timing for retrying failed payments. Instead of static heuristics (e.g., "retry every 24 hours"), the system will learn from past successes and failures based on customer attributes, payment methods, and failure reasons.

### User Stories
- As a merchant, I want my failed invoices to be recovered automatically using the most effective schedule.
- As a developer, I want to see the performance of the AI dunning system vs. static rules.
- As a system, I need to collect retry outcomes to improve future predictions.

## Tech Stack
- **Language**: Go 1.20+
- **Database**: PostgreSQL (for outcome tracking and state)
- **AI Approach**: Contextual Multi-Armed Bandits (Thompson Sampling or Epsilon-Greedy)
- **Scheduler**: Existing `internal/scheduler` and `internal/adapter/worker`

## Commands
- Build: `make build`
- Test: `go test ./internal/service/smart_retry_test.go`
- Seed Data: `go run cmd/seed/main.go` (extended for retry history)

## Project Structure
- `internal/core/domain/dunning.go`      → RL state and outcome models
- `internal/core/service/smart_retry.go` → Logic for bandit selection and reward processing
- `internal/adapter/db/dunning_repo.go`  → Persistence for retry history and bandit weights
- `internal/adapter/worker/retry_worker.go` → Updated to use RL service

## Code Style
```go
// Example of Bandit selection
func (s *SmartRetryService) GetNextRetryInterval(ctx context.Context, invoice *domain.Invoice) time.Duration {
    context := s.extractContext(invoice)
    action := s.bandit.SelectAction(context)
    return action.Interval
}
```

## Testing Strategy
- **Unit Tests**: Test the bandit logic (ensure it shifts toward successful actions over time).
- **Integration Tests**: Verify that `RetryWorker` correctly updates bandit state after a successful/failed payment.
- **Simulation**: Create a test suite that simulates 1000s of payments with different "optimal" windows to see if the RL agent converges.

## Boundaries
- **Always**: Log all retry decisions and their outcomes, handle database failures gracefully (fallback to static rules).
- **Ask first**: Adding external ML libraries (e.g., TensorFlow, PyTorch), changing the `Invoice` domain model significantly.
- **Never**: Block the main payment flow for AI inference, use customer PII (email/name) as features for the RL model without anonymization.

## Success Criteria
- [ ] RL framework implemented with at least 3 retry "arms" (e.g., 1h, 24h, 3d).
- [ ] Data collection captures: `invoice_id`, `retry_interval`, `outcome` (success/fail), `failure_reason`.
- [ ] Bandit weights are persisted and updated after each outcome.
- [ ] Recovery rate improvement can be tracked (Log rewards).
- [ ] System falls back to static 24h retry if AI service is unavailable.

## Open Questions
1. Should we use a pure Go implementation of Bandits or integrate with an external service? (Initial recommendation: Pure Go for low latency and zero dependencies).
2. What are the initial "Context" features? (Proposed: `currency`, `gateway_error_code`, `attempt_number`, `amount_bin`).
