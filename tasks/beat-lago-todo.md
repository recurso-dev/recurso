# Tasks: Beat Lago — Track A + B

> Phase 3 (Tasks) for `docs/spec_beat_lago.md` / `tasks/beat-lago-plan.md`.
> Ordered by dependency. Each increment ships as its own PR. Standing gates:
> full Go suite + E2E + invariant harness (`TestLedgerInvariants`) + OpenAPI
> drift (`TestOpenAPISpecCoversRegisteredRoutes`) green before merge; frontend
> lint/build/vitest for UI-touching increments; never bill a usage window twice;
> never weaken an invariant test; never commit secrets.

## A1 — Charge models → 7 + pricing simulator

- [ ] **A1.1 — `percentage` charge model**
  - Acceptance: a percentage charge (rate + optional fixed/min/max/free-units) rates onto an invoice line exactly.
  - Verify: rating table test; `go test ./internal/service/...`.
  - Files: `internal/core/domain/charge.go`, `internal/service/rating.go`, migration.
- [ ] **A1.2 — `graduated_percentage` charge model**
  - Acceptance: per-tier percentage + flat crosses tiers correctly; boundary units exact.
  - Verify: table test across ≥3 tiers incl. boundaries.
  - Files: `internal/service/rating.go`, `internal/core/domain/charge.go`.
- [ ] **A1.3 — `dynamic` charge model (Q2)**
  - Acceptance: usage event carries `dynamic_amount` (int64 minor); charge sums it; validation rejects negatives.
  - Verify: table test; event-ingestion test carries the field end-to-end.
  - Files: `internal/core/domain/usage_event.go`, `internal/service/rating.go`, ingestion handler.
- [ ] **A1.4 — Tier-builder UI for the 3 new models**
  - Acceptance: `PlanCharges.jsx` builds/edits percentage, graduated_percentage, dynamic visually; matches existing tier-builder patterns.
  - Verify: `npm run lint && npm run build && npm run test` in `frontend/`.
  - Files: `frontend/src/components/PlanCharges.jsx` (+ small helpers).
- [ ] **A1.5 — SDK enums for new models**
  - Acceptance: Go/TS/Python SDK charge-model enums include the 3 new values. (Merge only — no publish/tag; founder-gated.)
  - Verify: SDK builds/tests in each SDK repo.
  - Files: SDK enum/type files (three repos).
- [ ] **A1.6 — Pricing simulator (leapfrog)**
  - Acceptance: `POST /v1/plans/:id/simulate-charges` rates a proposed charge set against the tenant's last-period usage; returns invoice preview + GL preview; **read-only** (no DB write, no ledger post).
  - Verify: handler test asserts zero persistence + balanced preview; OpenAPI drift green.
  - Files: new `internal/service/pricing_simulator.go`, `internal/adapter/handler/plan.go`, `api/openapi.yaml`.

## A2 — Aggregations → 6 + p95/p99 + safe DSL  *(after A1)*

- [ ] **A2.1 — `weighted_sum` aggregation**
  - Acceptance: time-weighted sum over the period matches hand-computed fixture.
  - Verify: aggregation table test.
  - Files: `internal/service/aggregation.go` (or metering query path), domain enum.
- [ ] **A2.2 — `latest` aggregation**
  - Acceptance: returns the last event value in the window.
  - Verify: table test incl. out-of-order arrival.
  - Files: aggregation query path, domain enum.
- [ ] **A2.3 — `percentile` (p95/p99) aggregation (leapfrog)**
  - Acceptance: p95/p99 over the window; query bounded (no unbounded scan).
  - Verify: table test vs known distribution; large-window query stays bounded.
  - Files: aggregation query path, domain enum.
- [ ] **A2.4 — Sandboxed `custom` aggregation DSL (Q4)**
  - Acceptance: arithmetic over event fields only; **no** SQL/file/network/loops-unbounded; malformed/hostile input rejected.
  - Verify: evaluator unit tests + injection/abuse test; **security review (Ask-first before merge)**.
  - Files: new `internal/service/aggregation_dsl.go`, wiring in aggregation path.
- [ ] **A2.5 — Metric config UI + SDK enums**
  - Acceptance: metric editor exposes the new aggregation types; SDK enums updated (merge only).
  - Verify: frontend lint/build/test; SDK builds.
  - Files: `frontend/src/pages/Metering.jsx` (+ helpers), SDK enum files.

## A4 — Charge filters + GL dimensions + tax-aware  *(after A1; ∥ A2)*

- [ ] **A4.1 — `charge_filters` / `charge_filter_value` schema**
  - Acceptance: a charge can carry filters keyed on event properties; migration reversible.
  - Verify: migrate up/down; repository round-trip test.
  - Files: migration, `internal/core/domain/charge.go`, repo.
- [ ] **A4.2 — Filter-aware rating**
  - Acceptance: events select amounts by matching property; precedence deterministic; **filter-less charges byte-identical to today** (regression guard).
  - Verify: two-value table test + golden test proving no-filter output unchanged.
  - Files: `internal/service/rating.go`.
- [ ] **A4.3 — GL dimension on posted revenue (leapfrog)**
  - Acceptance: revenue legs carry the filter dimension; ledger stays balanced.
  - Verify: ledger test asserts dimension present + balanced; invariant harness green.
  - Files: `internal/service/ledger*.go`, rating→invoice path.
- [ ] **A4.4 — Filter-resolved tax jurisdiction (leapfrog)**
  - Acceptance: a filter dimension can drive `TaxResolver` jurisdiction selection.
  - Verify: test resolves jurisdiction from filter value.
  - Files: `internal/service/tax_resolver.go` hook, rating path.
- [ ] **A4.5 — Filter builder UI + SDK**
  - Acceptance: dashboard builds filters visually; SDK types updated (merge only).
  - Verify: frontend lint/build/test.
  - Files: `frontend/src/components/PlanCharges.jsx`, SDK types.

## A3 — Pay-in-advance + deferred revenue  *(after A1)*

- [ ] **A3.1 — `Charge.pay_in_advance` flag**
  - Acceptance: charge can be marked pay-in-advance; migration reversible.
  - Verify: migrate up/down; repo round-trip.
  - Files: migration, `internal/core/domain/charge.go`, repo.
- [ ] **A3.2 — Immediate rating on event → invoice line/fee**
  - Acceptance: a pay-in-advance event produces an immediate line; retried event (same `transaction_id`) no-ops (Q3/usage_ratings claim).
  - Verify: event→line test; retry idempotency test.
  - Files: `internal/service/rating.go`, event path, invoice service.
- [ ] **A3.3 — Deferred-revenue ledger legs (leapfrog)**
  - Acceptance: posts **DR cash / CR deferred revenue**, balanced, idempotent by `transaction_id`.
  - Verify: ledger balanced test; **invariant harness green**.
  - Files: `internal/service/ledger*.go`, rev-rec service.
- [ ] **A3.4 — Recognition schedule over the period**
  - Acceptance: deferred revenue recognizes to earned revenue on schedule; totals tie out at period end.
  - Verify: schedule test proves recognized == billed at close; harness green.
  - Files: rev-rec service, scheduler/worker.

## A5 — Progressive billing + risk-driven thresholds  *(after A1)*

- [ ] **A5.1 — Threshold-crossing → interim invoice**
  - Acceptance: crossing a `usage_threshold` generates one interim invoice for the partial period; usage_ratings claim prevents re-bill; renewal doesn't double-bill the same window.
  - Verify: cross→one-invoice test; renewal-after-interim no-double-bill test; **harness green**.
  - Files: threshold evaluator, `internal/service/invoice*.go`.
- [ ] **A5.2 — Attempt payment on interim (Q3)**
  - Acceptance: interim invoice attempts payment via stored method when `attempt_payment` on; togglable.
  - Verify: paid-path + toggle-off (invoice only, no charge) tests.
  - Files: threshold evaluator, payment/collection service.
- [ ] **A5.3 — Risk-driven threshold evaluation (leapfrog)**
  - Acceptance: threshold consults churn/credit score; high-risk billed sooner than low-risk on identical config; degrades to static threshold when signal absent.
  - Verify: two-customer test (same config, different risk → different timing); signal-absent fallback test.
  - Files: threshold evaluator, risk-score source.
- [ ] **A5.4 — Threshold config UI**
  - Acceptance: dashboard configures progressive thresholds + attempt_payment + risk opt-in.
  - Verify: frontend lint/build/test.
  - Files: relevant plan/subscription config component.

## B1 — Per-tenant BYO autopay  *(independent; confirm Q1 first)*

- [ ] **B1.0 — Confirm Q1 data model (gate)**
  - Acceptance: user confirms `gateway_connection_id` on stored payment method. **Ask-first — money path.**
  - Verify: decision recorded in spec/plan.
  - Files: n/a (decision).
- [ ] **B1.1 — `gateway_connection_id` on stored PM**
  - Acceptance: saved PMs reference the connection that created them; migration reversible.
  - Verify: migrate up/down; repo round-trip.
  - Files: migration, PM domain + repo.
- [ ] **B1.2 — Per-tenant SetupIntent (portal save-card)**
  - Acceptance: portal saves a card on the tenant's own Stripe via the resolver; env fallback when unset.
  - Verify: BYO-tenant + env-fallback tests.
  - Files: `internal/adapter/handler/checkout.go`/portal, gateway resolver.
- [ ] **B1.3 — Per-tenant autopay in retry/renewal/wallet workers**
  - Acceptance: the four Stripe-SDK sub-flows (spec_byo_gateway.md 2b-2) charge saved cards via the tenant connection.
  - Verify: worker tests per flow; existing E2E green. **Ask-first before merge (charge flow).**
  - Files: retry/renewal/wallet workers, resolver.

## B2 — Ledger close pack  *(independent; ship early)*

- [ ] **B2.1 — Close-pack endpoint**
  - Acceptance: one endpoint returns trial balance + GL export + reconciliation status; proves every invoice balances and rev-rec ties out.
  - Verify: endpoint test on seeded data; OpenAPI drift green.
  - Files: new service + handler, `api/openapi.yaml`.
- [ ] **B2.2 — "Close" dashboard page + export**
  - Acceptance: dashboard renders the close pack and exports (CSV/PDF); reconciliation status accurate.
  - Verify: frontend lint/build/test.
  - Files: new `frontend/src/pages/Close.jsx`, nav wiring.

## Suggested execution order
`A1 → (A2 ∥ A4) → A3 → A5`, with **B2** shipped early and **B1** after Q1 is
confirmed. Hold at each checkpoint in `beat-lago-plan.md`.
