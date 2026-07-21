# Spec: Usage-billing & finance roadmap

> A pure-Go/Postgres roadmap to make Recurso's usage-based billing and finance
> surface best-in-class. Two tracks ship this quarter — **Track A** deepens the
> rating layer, **Track B** hardens the finance/ledger surface. Larger platform
> bets are captured as **Track C** (post-quarter).

## Objective

Give AI/API companies a metering-to-invoice pipeline that is expressive
(many charge models and aggregations), precise (exact money math on a
double-entry ledger), and provable (reconciliation + a month-end close pack).
Success = every rating model and aggregation below is implemented, exact, and
covered by the invariant harness, with no regressions to existing billing.

## Tech stack

Go (gin, repository pattern, golang-migrate), Postgres (+ TigerBeetle ledger
mirror), React/Vite dashboard. Money is `int64` minor units. OpenAPI drift and
the ledger invariant harness are hard CI gates.

## Track A — the rating layer (this quarter)

- **A1 — Charge models 4 → 7 + pricing simulator.** Add `percentage`,
  `graduated_percentage`, `dynamic` to the existing per-unit/graduated/volume/
  package. Plus a read-only `POST /v1/plans/:id/simulate-charges` that rates a
  proposed charge set against sample usage and returns a balanced GL preview.
- **A2 — Aggregations 4 → 6.** Add `latest` and `percentile` (p95/p99 for
  SLA/latency billing) to count/sum/max/unique. (`weighted_sum` and a sandboxed
  `custom` DSL are deferred pending design/security review.)
- **A3 — Pay-in-advance.** Rate a charge per event at ingestion and capture it
  as a pending unbilled charge folded onto the next invoice; non-cumulative
  models only; arrears-excluded; idempotent.
- **A4 — Charge filters (dimensional pricing).** Price distinct values of one
  event property differently; one invoice line per value plus a default line;
  one rating claim per charge/period (double-billing guard intact). Follow-ups:
  post the dimension to the GL, and resolve tax jurisdiction from a filter.
- **A5 — Progressive billing.** Interim invoices when accumulated usage crosses
  a threshold, reconciled against the `usage_ratings` claim via a per-period
  billed-amount watermark (design decision pending).

## Track B — finance/ledger surface

- **B1 — Per-tenant autopay.** Charge saved cards via the tenant's own gateway
  connection across retry/renewal/wallet flows (needs the saved-PM ↔ connection
  data-model decision).
- **B2 — Ledger close pack.** One endpoint + dashboard page aggregating trial
  balance, reconciliation status, and the deferred-revenue rollforward, with a
  ready-to-close verdict. **(Shipped.)**

## Track C — platform bets (post-quarter)

MCP agent-operable billing, ledger-backed credits, per-entity double-entry
books, finance-grade RBAC + separation-of-duties, correctness-at-scale on a
single binary, EU e-invoicing / CPQ. Out of scope this quarter.

## Code style

Follow the surrounding code: `RateCharge` switch arms with `validate*` helpers,
nil-safe `Set*` optional wiring, exact `big.Rat` rounded half-up once per line,
migrations as the next sequential number with up+down.

## Testing strategy

Table tests per rating model/aggregation; DB-backed tests for SQL (skip without
`TEST_DATABASE_URL`, run in CI); the ledger invariant harness
(`TestLedgerInvariants`) and OpenAPI drift are mandatory gates for any
invoice-creating or route change.

## Boundaries

- **Always:** run the full suite + invariant harness before merge; keep money in
  minor units; one rating claim per usage window.
- **Ask first:** ledger-posting semantics, the double-billing guard, charge-flow
  changes, new migrations on the money path.
- **Never:** bill the same usage window twice; weaken an invariant test; commit
  secrets.

## Success criteria

Each increment ships behind the CI gates with per-model exactness tests and the
invariant harness green. Plan: `tasks/usage-billing-plan.md`; tasks:
`tasks/usage-billing-todo.md`.

## Open questions

Tracked in the plan doc: pay-in-advance collection cadence, charge-filter GL
dimension + tax hook, the progressive-billing watermark model, and the
autopay saved-PM ↔ connection data model.
