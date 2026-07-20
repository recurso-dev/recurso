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
