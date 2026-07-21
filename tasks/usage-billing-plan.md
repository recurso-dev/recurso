# Plan: Usage-billing roadmap — Track A + B (this quarter)

> Phase 2 (Plan) for `docs/usage_billing_roadmap.md`. Task list:
> `tasks/usage-billing-todo.md`. Track C is out of scope this quarter. Each
> increment ships as its own PR (BYO-series cadence). Awaiting Plan review
> before Implement.

## Assumed defaults for the open questions (confirm at each increment)

| # | Question | Assumed default (this plan) |
| --- | --- | --- |
| Q1 | Saved PM ↔ gateway connection (B1) | Add `gateway_connection_id` to the stored payment method; the BYO resolver charges via that connection. **Confirm before B1.** |
| Q2 | `dynamic` charge field shape | `dynamic_amount` (int64 minor units) supplied per event; the charge bills the exact per-event total. |
| Q3 | Progressive-billing semantics | Opt-in **per usage_threshold**; interim invoice generated **and** payment attempted via the stored method (like a renewal); `attempt_payment` togglable. |
| Q4 | `custom` aggregation | **Yes** — sandboxed expression DSL (arithmetic over event fields), never raw SQL/code. |
| Q5 | EU e-invoicing | Track C (post-quarter). |

## Dependency graph & order

```
A1 (charge models + simulator)  ─┬─→ A2 (aggregations + DSL)   ─┐
   [foundation: rating layer]    ├─→ A4 (charge filters)        ├─→ A3 (pay-in-advance)
                                 │                              └─→ A5 (progressive billing)
B1 (BYO autopay)  ── independent (gateway BYO shipped) ──────────→
B2 (close pack)   ── independent (UX/export over ledger) ────────→
```

- **A1 first** — it touches `RateCharge` / `ChargeModel` / `ChargeAmounts`, the
  shared spine everything else rates through.
- **A2 and A4 parallelize** after A1 (both extend metering/rating, different files).
- **A3 and A5** follow A1 (they add money movements through the invoice + ledger
  path; land them after the rating layer is stable).
- **B1, B2** are independent of Track A — can interleave / run in parallel.

## Per-increment approach, risks, verification

### A1 — Charge models → 7 + pricing simulator
- **Build:** `ChargeModel` consts (`percentage`, `graduated_percentage`,
  `dynamic`) + `ChargeAmounts` fields; `RateCharge` arms + `validate*` helpers
  (`internal/service/rating.go`); tier-builder UI (`PlanCharges.jsx`); SDK enums.
  Leapfrog: `POST /v1/plans/:id/simulate-charges` → rates a proposed set against
  the tenant's last-period usage, returns invoices + GL preview (new service +
  handler + OpenAPI).
- **Risk:** `dynamic` needs a per-event amount channel (usage event carries
  `dynamic_amount`); the simulator must be **read-only** (no persistence, no
  ledger write).
- **Verify:** rating table tests per model; simulator returns a balanced preview;
  `go test ./... && golangci-lint`; OpenAPI drift green.

### A2 — Aggregations → 6 + p95/p99 + safe DSL
- **Build:** metric aggregation types (`weighted_sum`, `latest`, `percentile`)
  in the aggregation query path; sandboxed expression evaluator for `custom`.
- **Risk:** percentile over large windows (bounded query); the DSL sandbox must
  have **no** SQL/file/network reach — security review (Boundaries: Ask-first).
- **Verify:** table tests per aggregation; DSL injection test; a metric of each
  type rates onto an invoice.

### A3 — Pay-in-advance + deferred revenue
- **Build:** `Charge.pay_in_advance` (migration); event path rates immediately →
  invoice line/fee → ledger legs **DR cash / CR deferred revenue**, idempotent
  via `transaction_id`; recognition over the period via the rev-rec service.
- **Risk:** double-billing (guarded by the usage_ratings/transaction claim);
  deferred-revenue schedule correctness (invariant harness must stay green).
- **Verify:** immediate line on event; retried event no-ops; ledger balanced;
  deferred revenue recognized on schedule; invariant harness green.

### A4 — Charge filters + GL dimensions + tax-aware
- **Build:** `charge_filters` + `charge_filter_value` (migration); rating selects
  amounts by event property; revenue posts with the dimension; a filter dimension
  can resolve tax jurisdiction (TaxResolver hook).
- **Risk:** filter precedence/overlap rules; keep filter-less charges byte-for-
  byte unchanged (regression guard).
- **Verify:** two property values bill at their rates; filter-less unchanged; GL
  carries the dimension; jurisdiction resolves from the filter.

### A5 — Progressive billing + risk-driven thresholds
- **Build:** threshold-crossing → `InvoiceService.GenerateInvoice` for the partial
  period (usage_ratings claim prevents re-bill); threshold evaluation consults the
  churn/credit score (risk-ML) to bill risky customers sooner.
- **Risk:** interim + renewal must not double-bill the same window; risk signal
  availability (degrade to static threshold when absent).
- **Verify:** threshold cross → one interim invoice; renewal doesn't re-bill;
  high-risk billed sooner than low-risk on identical config; ledger balanced.

### B1 — Per-tenant BYO autopay (2b-2)
- **Build:** confirm Q1 data model; resolve the saved-card charger + portal
  SetupIntent per tenant across retry/renewal/wallet workers + portal (the four
  concrete Stripe-SDK sub-flows from `spec_byo_gateway.md` 2b-2).
- **Risk:** money path — charge-flow boundary (Ask-first); saved PMs tied to the
  connection that created them.
- **Verify:** a BYO tenant stores a card + runs autopay on their own Stripe;
  falls back to env when unset; existing E2E green.

### B2 — Ledger close pack
- **Build:** one endpoint aggregating trial balance + GL export + reconciliation
  status; a dashboard "Close" page + export.
- **Risk:** low (read/export over existing services).
- **Verify:** the close pack proves every invoice balances + rev-rec ties out;
  reconciliation status accurate.

## Checkpoints
- After **A1**: rating layer stable, simulator read-only — proceed to A2/A4.
- After **A3**: deferred-revenue posting proven on the invariant harness —
  proceed to A5.
- Every increment: full suite + E2E + invariant harness + OpenAPI drift green
  before merge; frontend lint/build/vitest for UI-touching increments.

## Sequencing recommendation
Ship **A1 → (A2 ∥ A4) → A3 → A5**, with **B2** early (fast, high-marketing-value)
and **B1** once Q1 is confirmed. Re-evaluate Track C after A/B land.
