# ADR-008: Accrual revenue recognition (schedule at issuance), opt-in and reversible

## Status
Accepted

## Date
2026-08-04

## Context
Under the original model, a subscription invoice's recognition schedule is built
when the invoice is **paid** (cash recognition): revenue is deferred until cash
arrives, then recognized over the period. GAAP accrual accounting instead
recognizes revenue when it is **earned** — at issuance — regardless of payment.
Tenants that must report on an accrual basis need the schedule built at issuance
so revenue recognizes over the service period and the month-end tie-out is
structurally zero. Changing recognition timing touches the most sensitive part
of the system (the ledger), so it must be introduced without altering any
existing tenant's books until they opt in, and it must be reversible.

This decision builds on ADR-007's accounting-policy engine (the `cash` vs
`accrual` axis) and is coupled to ADR-009 (bad debt): once revenue is recognized
on an unpaid invoice, a later write-off must expense Bad Debt rather than reverse
Deferred.

## Decision
Introduce accrual as an **opt-in, default-off** recognition method, rolled out in
independently-correct increments.

- `InvoiceService.SetAccrualRecognition(on)` (wired from
  `RECURSO_ACCRUAL_RECOGNITION=true` in `cmd/api/main.go`) makes
  `GenerateInvoice` build the recognition schedule at **issuance** for
  subscription invoices, instead of at payment. Default off = the cash model, so
  production is byte-identical until a deployment opts in.
- The deferred tie-out identity is `ledger_closing == scheduled +
  awaiting_payment`. `SumUnscheduledDeferral` (the awaiting-payment bucket) was
  changed to **exclude invoices that already have an active recognition
  schedule** — without this, an accrual invoice would be double-counted (in both
  scheduled and awaiting-payment) and the tie-out would go negative. This is the
  load-bearing correctness fix.
- Rollout order is deliberate: (1) policy seam + Bad Debt account, (2) write-off
  bad-debt split (ADR-009) — shipped *before* the switch so accrual never drives
  Deferred negative on a write-off, (3) the schedule-at-issuance switch behind
  the flag, (4) the tie-out exclusion + a close-pack tie-out proof, (4b) a
  `cmd/backfill_schedules` tool to create issuance schedules for a tenant's
  existing open invoices when it flips to accrual.
- Migration is gradual, not big-bang: legacy tenants stay cash; the flag is
  enabled first on an internal tenant, then demo, then early adopters.

## Alternatives Considered

### Flip everyone to accrual at once
- Pros: one model to maintain
- Cons: rewrites every existing tenant's recognition timing in one deploy; no
  rollback; a bug would corrupt live books
- Rejected: incremental, reversible, per-tenant opt-in is mandatory for a ledger

### Recognize at issuance but keep the old tie-out math
- Cons: `SumUnscheduledDeferral` would double-count scheduled invoices and the
  close-pack tie-out would read negative — a false discrepancy on healthy books
- Rejected: the exclusion is required

### A parallel "accrual ledger" alongside the cash ledger
- Pros: total isolation
- Cons: two sources of truth to reconcile; enormous surface area
- Rejected: one ledger, policy-selected recognition timing

## Consequences
- Accrual is available end-to-end and opt-in; a tenant enables it by setting
  `RECURSO_ACCRUAL_RECOGNITION=true` and running `cmd/backfill_schedules
  --tenant=<id> --apply` to schedule existing open invoices.
- The tie-out is proven zero under accrual and still ties under cash:
  `internal/service/closepack_accrual_tieout_pg_test.go`
  (`ledger 100000 = scheduled 100000 + awaiting 0`).
- End-to-end accrual write-off behavior (recognize part, then write off) is
  proven by `internal/service/accrual_writeoff_pg_test.go`.
- The full randomized invariant harness (10 seeds) stays green with the flag off,
  and the `recognized_exceeds_invoice` invariant (see ADR-009) guards against
  over-recognition once it is on.
- Open follow-up: **accounting-model versioning** (V1 cash / V2 accrual stamped
  on journals) for historical reproducibility — recording *which* rules produced
  a given set of entries. Tracked separately; when built it supersedes the
  implicit "current policy" assumption here.
