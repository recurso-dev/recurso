# ADR-002: Ledger posting — best-effort legs, reconciliation as detection, invariant harness as prevention

## Status
Accepted

## Date
2026-07-20

## Context
Recurso keeps a double-entry ledger (Postgres authoritative; TigerBeetle
optional mirror). Every invoice-creating flow must post an invoice leg
(DR Accounts Receivable / CR Deferred or Revenue) and, on settlement, a cash
leg. Two flows (mid-cycle upgrade proration, mandate debits) shipped without
their invoice leg — AR/Deferred drifted permanently and only the reconciler
noticed, in production (audit finding F1, originally archive PR #82).

The design question: should ledger postings be transactional with the
business write (fail the charge if the ledger write fails), or best-effort
with detection?

## Decision
Three layers:
1. **Best-effort posting after the business write is durably committed.** A
   ledger post failure is logged loudly ("reconciliation needed") but never
   fails the customer-facing operation. Postings are idempotent per event via
   the unique index on `(reference_id, code)` — a replay can't double-post.
2. **Reconciliation as detection.** `ReconciliationService.Run` compares
   billing records against the ledger per tenant and reports
   `missing_invoice_transaction`, `ledger_unbalanced`, `abnormal_account_balance`.
3. **Invariant harness as prevention.** A CI property test
   (`ledger_invariant_pg_test.go`) drives randomized real billing sequences
   (8 seeds × 25 ops: subscribe, upgrade, downgrade, one-off, recognize,
   cancel+unwind) and asserts reconciliation is clean after *every* step. The
   E2E suite ends with the same zero-discrepancy gate over real HTTP.

## Alternatives Considered

### Transactional posting (ledger write in the same DB transaction)
- Pros: drift is impossible by construction
- Cons: couples every money path to ledger schema health; a ledger bug turns
  into a billing outage; TigerBeetle mirror can't join a Postgres transaction
- Rejected: availability of charging beats synchronous bookkeeping;
  reconciliation + idempotency give eventual audit-grade correctness

### Outbox/event-driven posting
- Pros: decoupled, retryable
- Cons: new infrastructure (outbox table, dispatcher), ordering concerns; the
  actual historical failures were *missing call sites*, which an outbox does
  not prevent
- Rejected for now: the harness attacks the real failure mode directly

## Consequences
- Every new invoice-creating flow MUST call `LedgerService.RecordInvoice`
  after commit — and if it forgets, the invariant harness or the E2E
  reconciliation gate fails CI on some seed. That guard, not review vigilance,
  is the enforcement mechanism.
- Reconciliation discrepancies in production are actionable operator signals,
  not noise; historical drift rows persist until data is repaired or reseeded.
- `MandateService`/`SubscriptionService` gained nil-safe `SetLedgerService`
  wiring (the codebase's optional-dependency idiom).
