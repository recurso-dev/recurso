# Architecture Decision Records

Decisions that would be expensive to reverse or re-litigate. Don't delete old
ADRs — supersede them with a new one that references the old.

| ADR | Decision | Status |
|---|---|---|
| [ADR-001](ADR-001-scoped-rate-limiting.md) | Scope-namespaced fixed-window rate limiting | Accepted |
| [ADR-002](ADR-002-ledger-posting-semantics.md) | Best-effort ledger legs + reconciliation detection + invariant-harness prevention | Accepted |
| [ADR-003](ADR-003-claim-based-worker-concurrency.md) | Claim-based concurrency for money-path workers | Accepted |
| [ADR-004](ADR-004-one-off-revenue-recognition.md) | One-off invoices: immediate, net-of-tax, no ledger posting | Accepted |
| [ADR-005](ADR-005-layered-caching.md) | Layered caching: tenant-scoped Redis + react-query | Accepted |
| [ADR-006](ADR-006-token-based-accounting-connections.md) | Token-based connections for non-OAuth accounting providers | Accepted |
| [ADR-007](ADR-007-accounting-policy-engine.md) | Accounting policy is a first-class engine, separate from tax and jurisdiction | Accepted |
| [ADR-008](ADR-008-accrual-revenue-recognition.md) | Accrual revenue recognition (schedule at issuance), opt-in and reversible | Accepted |
| [ADR-009](ADR-009-bad-debt-treatment.md) | Bad-debt write-off splits recognized-vs-deferred; tax relief is policy-driven | Accepted |
| [ADR-010](ADR-010-multi-currency-general-ledger.md) | The GL is single-functional-currency; multi-currency GL is out of scope (use entity-per-currency) | Accepted |
