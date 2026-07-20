# Spec: Beat Lago — Neutralize the core, win on the flank

> **Status: SPECIFY (Phase 1) — awaiting founder review before Plan → Tasks →
> Implement.** Grounded in a code-level read of `getlago/lago-api` (2026-07-20)
> and Recurso's current feature set.

## Objective

An Indian or global SaaS / AI company evaluating **Lago vs. Recurso** should
conclude:

1. **No billing-core reason to pick Lago** — Recurso matches every metering and
   pricing capability a buyer would test on the shortlist.
2. **Several reasons to pick Recurso** — a real double-entry ledger, India +
   global compliance depth, per-tenant bring-your-own gateway/integrations, and
   data-residency control — things Lago's architecture cannot easily follow.

Success = win the head-to-head on a feature checklist **and** on the demo, for
both the India-first buyer and the global usage-billing buyer.

### Strategy (founder-approved)

- **Fight:** *parity on the core + win on the flank.*
- **Infra bets deferred:** the highest-value gaps are **pure Go/Postgres
  features**; the ClickHouse/Kafka high-volume pipeline and GraphQL are
  **volume-gated / later**, not in this program.
- **Horizon:** a focused ~1-quarter push, ~8 increments, each an independently
  shippable PR (the cadence proven by the BYO gateway series #44–#48).

## Where Lago leads today (the gap, from lago-api source)

| Area | Lago | Recurso today |
| --- | --- | --- |
| Charge models | 7 — standard, graduated, package, volume, **percentage, graduated_percentage, dynamic** | 4 (per_unit, graduated, volume, package) |
| Aggregation types | 7 — count, sum, max, unique, **weighted_sum, latest, custom** | 4 (count, sum, max, unique) |
| Charge timing | arrears **and pay-in-advance** (bill on event) | arrears only |
| Dimensional pricing | **charge filters** (price by event property) | — |
| Progressive billing | **invoice mid-cycle on a usage threshold** | usage alerts (webhook/email) only |
| Pricing units (abstract credits) | ✅ | money-denominated wallets (deliberate) |
| Multi billing-entity | ✅ | org grouping, not per-entity invoicing |
| GraphQL API | ✅ (+ REST) | REST only |
| CPQ (order forms, quote versions) | ✅ | plain quotes |
| Custom RBAC | roles/permissions | fixed owner/admin/member |
| Integrations breadth | Anrok, Salesforce, Cashfree/Moneyhash/Flutterwave | QBO/Xero/NetSuite/Tally, TaxJar/Avalara, HubSpot + **BYO** |

## Where Recurso already wins (defend + amplify)

Double-entry ledger on TigerBeetle with a reconciliation + randomized invariant
harness; India GST / IRP e-invoicing / TDS / UPI AutoPay; per-tenant **BYO
gateway + tax/CRM/storage** (Lago is one org per instance — it cannot do this);
`RESIDENCY_MODE=self_hosted` egress lockdown; smart-retry ML dunning + churn
prediction; NL→SQL Ask-AI.

---

## The program — 8 increments

### Track A — NEUTRALIZE (billing-core parity; pure Go/Postgres)

**A1. Charge models → 7.** Add `percentage`, `graduated_percentage`, and
`dynamic` to `RateCharge` (`internal/service/rating.go`) + the domain
`ChargeModel` enum + the plan-charges validator + the dashboard tier builder
(`PlanCharges.jsx`) + the 3 SDKs. `dynamic` = a caller-supplied
`precise_total_amount` per event (the AI/API-resale case Lago markets).

**A2. Aggregations → 6.** Add `weighted_sum` and `latest` billable-metric
aggregations (`internal/service/`, the aggregation query path). `custom` (user
SQL) is a **Consider**, deferred (security surface).

**A3. Pay-in-advance charges.** A `pay_in_advance` flag on a charge: usage
events for that charge bill **immediately** (create a one-off fee/invoice line at
event time) instead of at renewal. Must post its ledger legs like every other
money movement (ADR-002) and be idempotent per event (reuse the
`transaction_id` guard).

**A4. Charge filters (dimensional pricing).** Price a metric differently by
event **property** value — e.g. `region=eu` at one rate, `region=us` at
another. New `charge_filter` + `charge_filter_value` (mirrors Lago), applied in
the rating engine. Backward-compatible: a charge with no filters rates as today.

**A5. Progressive billing.** When a subscription's period usage crosses a
**usage threshold**, generate an **interim invoice** for the usage so far
(instead of only firing an alert). Recurso already has `usage_threshold` alerts
(the health-alert scheduler) — this wires the threshold to
`InvoiceService.GenerateInvoice` for a partial period, with the
`usage_ratings` claim preventing double-billing the same window.

### Track B — WIN (extend the moats Lago can't follow)

**B1. Finish per-tenant BYO (2b-2).** Saved-card off-session charging + portal
SetupIntent resolved **per tenant** (autopay / card-on-file on the tenant's own
Stripe). Completes the multi-tenant BYO story end-to-end — a structural moat
(Lago = single org per instance). Needs the payment-method↔account data-model
decision first.

**B2. Ledger-grade correctness as a headline feature.** Surface what Lago has no
answer for: every invoice **provably balanced**, revenue-recognition tie-out to
the ledger, and a one-click **audit/close pack** (trial balance + GL export +
reconciliation status). Mostly a UX + export layer over existing services;
turns an engineering invariant into a sales differentiator.

**B3. (Consider) EU e-invoicing parity (Peppol / EN16931).** Close Lago's one
compliance lead while keeping the India IRP lead — makes "compliance depth" an
unambiguous Recurso win globally. Larger; gated on demand.

### Ordering & parallelism

A1 → A2 → A4 build on the rating/metering layer (do A1 first; A2 and A4 can
parallelize after). A3 and A5 depend on the charge/event + invoice paths (after
A1). B1 is independent (gateway BYO already shipped). B2 is independent (UX over
existing ledger). B3 last / optional.

## Tech Stack

Unchanged: Go 1.25 + Gin + PostgreSQL (authoritative) + optional TigerBeetle
mirror; React + Vite + shadcn dashboard; Mintlify docs; SDKs Node/Python/Go. No
new runtime dependencies anticipated. **No ClickHouse, Kafka, or GraphQL in this
program.**

## Commands

```
go build ./... && go test ./...                       # backend
LEDGER_INVARIANT_SEED=<n> go test ./internal/service/ -run TestLedgerInvariants
cd frontend && npm run lint && npm run build && npx vitest run
docker compose up -d --build && ./scripts/e2e_test.sh # full stack
gofmt -l internal cmd && golangci-lint run            # pre-commit runs it
```

## Project Structure (where new code lands)

```
internal/core/domain/metering.go       ChargeModel enum, charge filters, pay_in_advance
internal/service/rating.go             RateCharge: percentage/graduated_percentage/dynamic
internal/service/metering.go           aggregation types, charge-filter validation
internal/service/invoice.go            progressive billing, pay-in-advance lines
internal/service/*                      new aggregation query paths, filter rating
internal/adapter/db/migrations/0001NN_*.sql   charge_filters, pay_in_advance, thresholds
internal/adapter/handler/               charge/metric/threshold endpoint additions
cmd/api/openapi.yaml                    spec (drift test gate)
frontend/src/components/slide-overs/PlanCharges.jsx   new models + filters UI
../recurso-{node,python,go}             SDK methods (own repos, founder-gated release)
```

## Code Style

Match the house reference — the wallet Drain comment is canonical:

```go
// RateCharge prices an aggregated quantity for one currency. Adding a model is
// a new switch arm + a validate* helper; a config that passes SetPlanCharges'
// probe-rating cannot fail at invoice time (metering.go). Money is int64 minor
// units except decimal-string rates (spec_usage_billing D1).
func RateCharge(model domain.ChargeModel, amounts domain.ChargeAmounts, quantity int64) (int64, error) {
```

Conventions: sentinel errors + typed `ValidationError` strings per service;
nil-safe optional collaborators (`Set*`); repos return `(nil, nil)` on not-found
reads and `sql.ErrNoRows` on zero-row writes; comments state constraints, not
narration; every list endpoint uses `ParsePagination`/`clampLimitOffset`.

## Testing Strategy

- **Table tests** for all new money math (percentage/dynamic/graduated_percentage
  rating; weighted_sum/latest aggregation; filter selection; progressive-billing
  proration) — same rigor as `rating_test.go`.
- **Fake-repo service tests** asserting the standing invariants on every new
  invoice path: `Total == Subtotal + Tax`, `Σ line.Amount == Subtotal`, ledger
  legs balance.
- **Invariant harness** (`TestLedgerInvariants`) must stay green — any new
  money movement posts its legs or randomized reconciliation fails CI.
- **OpenAPI drift test** green — every new route documented.
- **E2E** extended for pay-in-advance + progressive-billing round-trips.
- **SDK** vitest/pytest/go-test extended per new method.

## Boundaries

- **Always:** `go test ./... && golangci-lint` before commit; route every
  invoice line through `newInvoiceLine`; post ledger legs for every money
  movement; keep residency guards on any new egress; document new routes in
  openapi.yaml.
- **Ask first:** the payment-method↔account data-model for B1; any schema change
  beyond the migrations named here; changing payment-ordering / charge-timing
  semantics; `custom` aggregation (user SQL — security review required).
- **Never:** commit secrets; weaken an existing invariant test; bill the same
  usage window twice (the `usage_ratings` claim is sacred); auto-publish SDK
  releases (npm/PyPI/GHCR); make `recurso-archive` public.

## Success Criteria

1. **Charge models:** all 7 of Lago's models supported and rated correctly,
   configurable from the dashboard tier builder and all 3 SDKs. Table-tested.
2. **Aggregations:** weighted_sum + latest supported; a metric using each rates
   onto an invoice correctly.
3. **Pay-in-advance:** an event on a pay-in-advance charge produces an immediate,
   ledger-balanced invoice line; a retried event does not double-bill.
4. **Charge filters:** two events with different property values on the same
   metric bill at their respective filtered rates; a filter-less charge is
   unchanged.
5. **Progressive billing:** crossing a usage threshold mid-period generates an
   interim invoice for usage-so-far, once, ledger-balanced; the renewal invoice
   doesn't re-bill it.
6. **B1:** a BYO tenant runs autopay / card-on-file on **their own** Stripe
   account end-to-end.
7. **B2:** a one-click close pack (trial balance + GL export + reconciliation
   status) proves every invoice balances.
8. **Parity checklist:** the marketing comparison table can honestly flip every
   Track-A row to Recurso ✅ vs Lago; no regressions (full suite + E2E +
   invariant harness green at every commit).

## Open Questions

1. **B1 data model:** do saved payment methods store which gateway *connection*
   (tenant account) they belong to? (Needed before autopay-per-tenant.)
2. **`dynamic` charge model:** confirm the field name/shape for the caller-
   supplied per-event amount (match Lago's `precise_total_amount_cents` or a
   Recurso-native name?).
3. **Progressive billing default:** opt-in per plan, or per usage-threshold? And
   does an interim invoice attempt payment immediately or accrue to renewal?
4. **`custom` aggregation** (Lago's 7th): worth the security surface, or leave it
   as the one deliberate non-parity item?
5. **B3 EU e-invoicing:** in this quarter, or a follow-on program?
