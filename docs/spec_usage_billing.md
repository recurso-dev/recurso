# Spec: Usage-Based Billing (Billable Metrics + Charge Models + Rating)

> **Status: DRAFT — awaiting founder decisions (D1–D6 below)**
>
> Closes the competitive gap vs. Lago identified 2026-07-18: Recurso ingests
> usage events but cannot price them. This spec adds billable metrics with
> real aggregation types, non-flat charge models on plans, and rating of
> usage into invoice lines at period close — while keeping every existing
> invariant (exact paise reconciliation, per-line GST/HSN, ledger postings).

## Why now

- Lago's entire core is metering → rating. We have metering-lite (SUM by
  free-form dimension feeding entitlement quotas) and **no path from a usage
  event to an invoice line** (`invoice_lines.go` builds from base plan +
  add-ons + ad-hoc charges only).
- Our `Price` model is flat recurring/one-time only. No per-unit, tiered,
  volume, or package pricing exists anywhere in the domain.
- Target customer overlap is total: Indian SaaS + AI/API companies need
  hybrid (flat + usage) pricing *and* GST-correct invoices. Lago gives them
  the first; only we can give them both.

## What exists today (grounding)

| Piece | Where | State |
| --- | --- | --- |
| Event ingestion | `POST /v1/usage/events`, `service/usage.go` | EXISTS — `UsageEvent{SubscriptionID, CustomerID, Dimension, Quantity, Timestamp}` |
| Aggregation | `usage.go` repo queries | SUM only, by dimension, `date_trunc` buckets |
| Plan pricing | `domain/plan.go` `Price{Currency, Amount, Type}` | Flat recurring / one-time only |
| Invoice composition | `service/invoice.go:98 GenerateInvoice` | base plan line + unbilled charges + add-ons via `newInvoiceLine` |
| Line invariants | `invoice_lines.go` | Σ line.Amount == Subtotal exactly; per-line HSN + CGST/SGST/IGST; largest-remainder discount |
| Entitlements | `entitlement.go` | `limit` grants matched to dimension by feature_key — read-side only |

## Domain model (new)

### BillableMetric

One per tenant-defined meter. `code` doubles as the event `dimension` it
aggregates (keeps existing events and the entitlement feature_key linkage
working unchanged).

```go
type BillableMetric struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    Name            string          // "API calls"
    Code            string          // "api_calls" — matches UsageEvent.Dimension; unique per tenant
    AggregationType AggregationType // count | sum | max | unique
    // FieldName: for unique — the event property whose distinct values are
    // counted (e.g. "user_id"). Empty for count/sum/max (they use Quantity).
    FieldName string
    CreatedAt time.Time
}
```

Aggregations (v1, per D5): `count` (number of events), `sum` (Σ Quantity),
`max` (max single-event Quantity), `unique` (distinct `properties[FieldName]`).
`latest` and `weighted_sum` (time-weighted, for per-second resources) are
explicitly deferred.

### UsageEvent gains properties (D2)

```go
type UsageEvent struct {
    // ... existing fields unchanged ...
    Properties map[string]string `json:"properties,omitempty"` // JSONB
}
```

Required for `unique`; also the foundation for charge filters and grouping
later (Lago's group-by). Optional on ingest — existing callers unaffected.

### Charge

A usage price attached to a plan. Flat subscription fees stay on `Price`
untouched — a plan with charges is "hybrid" (flat fee in advance + usage in
arrears).

```go
type Charge struct {
    ID           uuid.UUID
    PlanID       uuid.UUID
    MetricID     uuid.UUID
    ChargeModel  ChargeModel // per_unit | graduated | volume | package
    // Per-currency pricing properties, keyed by ISO code (mirrors how Price
    // rows give a plan multi-currency support). See D6.
    Amounts   map[string]ChargeAmounts // JSONB
    HSNCode   string                   // per-charge HSN/SAC; empty → plan HSN → tenant SAC (existing fallback chain)
    CreatedAt time.Time
}

type ChargeAmounts struct {
    // per_unit: rate per unit as a DECIMAL STRING (see D1), e.g. "0.0035"
    UnitAmount string `json:"unit_amount,omitempty"`
    // package: price (minor units) per bundle of PackageSize units, round up
    PackageAmount int64 `json:"package_amount,omitempty"`
    PackageSize   int64 `json:"package_size,omitempty"`
    // graduated & volume
    Tiers []ChargeTier `json:"tiers,omitempty"`
}

type ChargeTier struct {
    UpTo       *int64 `json:"up_to"`                 // nil = infinity (last tier)
    UnitAmount string `json:"unit_amount"`           // decimal string
    FlatAmount int64  `json:"flat_amount,omitempty"` // minor units, charged when tier is entered
}
```

Charge model semantics (Lago/Stripe-compatible):

- **per_unit**: `total = round(quantity × unit_amount)`
- **graduated**: each tier prices only the units that fall inside it
  (0–100 @ ₹1, 101–∞ @ ₹0.50 → 150 units = 100×1 + 50×0.5); tier
  FlatAmount added once when any unit lands in the tier.
- **volume**: the whole quantity is priced at the single tier it reaches.
- **package**: `ceil(quantity / package_size) × package_amount`.

### Rating math & precision (D1)

Per-unit rates are **decimal strings** persisted as PG `numeric`, computed
with `math/big.Rat`, and **rounded half-up to int64 minor units once per
line**. Rationale: AI/API pricing is sub-paise ("₹0.0035 per call") and
int64 minor units cannot express it; int64 micro-units would leak a second
unit system into every API. The int64-paise invariant is preserved where it
matters — **every `InvoiceItem.Amount` stays int64 minor units**, so
Σ lines == Subtotal, GST, and ledger postings are untouched.

## Rating at period close (the core)

In `GenerateInvoice` (`invoice.go`), after the base-plan line, for each
charge on the subscription's plan:

1. Aggregate the subscription's events for the metric over the **elapsed
   period** `[PreviousPeriodStart, PreviousPeriodEnd)` (arrears — D3).
2. Price the aggregate per the charge model in the invoice currency.
3. Emit one `newInvoiceLine` per charge — description
   `"{metric.Name} — {n} {unit} @ {period}"`, quantity = aggregated units,
   HSN from the charge (fallback chain as today), tax via the existing
   per-line GST resolution. Zero-usage charges emit **no line**.

Invariants preserved by construction: lines flow through `newInvoiceLine`,
totals accumulate exactly as the existing paths do, discount distribution
and GST recompute already operate on arbitrary line sets.

**Double-billing guard:** a `usage_ratings` row
`(subscription_id, charge_id, period_start, period_end, invoice_id)` with a
unique constraint on `(subscription_id, charge_id, period_start)` makes
rating idempotent — a retried generation can't bill the same window twice.

**Final invoice:** cancellation/expiry rates the partial last window
`[PeriodStart, canceled_at)` on the terminal invoice. Trial-only
subscriptions with usage rate at first real invoice.

**First invoice:** an acquisition invoice has no elapsed window — usage
lines start from the first renewal (matches Lago).

## API surface (v1)

```
POST/GET/PUT/DELETE  /v1/billable-metrics[/:id]
PUT/GET              /v1/plans/:id/charges          (full-replacement, like entitlements)
GET                  /v1/subscriptions/:id/usage-amount   (live pre-invoice preview: per-charge aggregate + priced amount)
POST /v1/usage/events                                (gains optional `properties` object)
```

OpenAPI + docs (`advanced/usage-billing.mdx` rewrite) + Node/Python/Go SDK
methods in the same release. Dashboard: metrics CRUD page + charges section
on the plan editor + priced preview on the subscription Usage tab
(follow-up PR, not gating).

## Migrations

- `000093_billable_metrics` — metrics table (unique `(tenant_id, code)`).
- `000094_plan_charges` — charges + `usage_ratings` tables.
- `000095_usage_event_properties` — `ALTER TABLE usage_events ADD COLUMN properties JSONB`.

## Founder decisions

| # | Question | Recommendation |
| --- | --- | --- |
| D1 | Sub-unit rate representation | **Decimal strings / PG numeric, round once per line** (alt: int64 micro-units) |
| D2 | Add `properties` JSONB to events now | **Yes** — needed for `unique`, cheap now, painful later |
| D3 | Usage billing timing | **Arrears on the renewal invoice** (hybrid: flat fee advance + usage arrears on one invoice; alt: separate usage-only invoice) |
| D4 | Charge models in v1 | **per_unit, graduated, volume, package**; `percentage` (needs a monetary base) deferred |
| D5 | Aggregations in v1 | **count, sum, max, unique**; `latest`/`weighted_sum` deferred |
| D6 | Multi-currency charges | **Per-currency `Amounts` map on the charge** (mirrors Price-per-currency; alt: single-currency charges) |

## Build order

1. Domain types + migrations + metric CRUD (+ tests)
2. Aggregation queries (count/max/unique alongside existing sum) (+ tests)
3. Pure pricing library — `RateCharge(model, amounts, quantity) int64` with
   exhaustive table tests incl. rounding edges
4. Rating in `GenerateInvoice` + `usage_ratings` guard + final-invoice path
   (+ tests against Σ-line and GST invariants)
5. Handlers, OpenAPI, SDK methods, docs
6. Dashboard UI (follow-up)

Out of scope (Phase 2 spec): prepaid wallets/credit burn-down, minimum
commitments + overage, usage threshold alerts, charge filters/group-by,
ClickHouse-scale ingestion.
