# Spec: Beat Lago — surpass, don't match

> **Status: SPECIFY (Phase 1) — awaiting founder review before Plan → Tasks →
> Implement.** Grounded in a code-level read of `getlago/lago-api` (2026-07-20)
> and Recurso's current feature set. Every Track-A item is *parity + a leapfrog*
> (see the leapfrog thesis); Track C is the post-quarter surpass roadmap.

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

- **Fight:** *parity on the core + win on the flank* — and, per the leapfrog
  thesis below, **surpass** rather than merely match on each item.
- **Infra bets deferred:** the highest-value gaps are **pure Go/Postgres
  features**; the ClickHouse/Kafka high-volume pipeline and GraphQL are
  **volume-gated / later**, not in this program.
- **Horizon:** a focused ~1-quarter push (Track A + B, ~8 increments), each an
  independently shippable PR (the cadence proven by the BYO gateway series
  #44–#48). The bigger platform leapfrogs are captured as **Track C**, the
  post-quarter surpass roadmap.

### The leapfrog thesis (how we get *ahead*, not level)

> **Don't out-Lago Lago feature-for-feature. Reimplement each Lago feature fused
> with a moat Lago's architecture can't copy** — the double-entry ledger, all-Go
> single-binary simplicity, per-tenant BYO, the India/tax engine, the
> risk/churn ML, and residency. The *same* capability then ships **provably
> correct, per-tenant, safer, or simpler to operate**. Lago (Rails + one-org-
> per-instance + no ledger) can't follow without a rewrite.

Every Track-A item below is therefore **parity + a leapfrog**, not parity alone.

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

## The program

### Track A — parity + leapfrog on the billing core (pure Go/Postgres; this quarter)

**A1. Charge models → 7, + ledger-previewed pricing simulator.** Add
`percentage`, `graduated_percentage`, `dynamic` to `RateCharge`
(`internal/service/rating.go`) + the `ChargeModel` enum + the plan-charges
validator + the tier builder (`PlanCharges.jsx`) + the 3 SDKs. `dynamic` = a
caller-supplied precise per-event amount (the AI/API-resale case Lago markets).
**Leapfrog:** a *simulate-pricing* endpoint that rates a proposed charge set
against a tenant's **last-period real usage** and returns the exact invoices
**and GL impact** — pricing experimentation Lago can't do (no double-entry
preview). Optional India: GST-slab-aware model.

**A2. Aggregations → 6, + a safe custom DSL.** Add `weighted_sum` and `latest`.
**Leapfrog:** add **p95/p99 percentile** aggregation (SLA/latency billing AI
companies want, absent in Lago), and beat Lago's `custom_agg` (user Ruby/SQL, a
security hole) with a **sandboxed expression DSL** — custom aggregation that's
safe under `RESIDENCY_MODE=self_hosted`.

**A3. Pay-in-advance charges, + ledger-native deferred revenue.** A
`pay_in_advance` flag: events bill **immediately** (a fee/invoice line at event
time), idempotent per event (`transaction_id` guard). **Leapfrog:** the advance
charge **auto-posts deferred revenue** (DR cash / CR deferred, recognized over
the period) — rev-rec-correct out of the box; Lago charges in advance with no
double-entry deferred-revenue engine. India: advance-collect via UPI mandate.

**A4. Charge filters (dimensional pricing), + GL dimensions + tax-aware.** Price
by event **property** value (`region=eu` vs `region=us`); `charge_filter` +
`charge_filter_value`, applied in rating; filter-less charges unchanged.
**Leapfrog:** post the dimension to the **GL** (revenue by region/SKU as ledger
dimensions), and let a filter dimension drive **tax jurisdiction** (region →
rate *and* GST/VAT) — unique to Recurso's tax engine.

**A5. Progressive billing, + risk-driven thresholds.** Crossing a usage
threshold generates an **interim invoice** for usage-so-far (today only fires an
alert), via `InvoiceService.GenerateInvoice` for a partial period, guarded by the
`usage_ratings` claim. **Leapfrog:** make thresholds **churn/credit-ML-driven** —
bill risky customers progressively **sooner** (Lago's thresholds are static),
tying progressive billing to the risk moat.

### Track B — finish the moats (this quarter)

**B1. Finish per-tenant BYO (2b-2).** Saved-card off-session charging + portal
SetupIntent resolved **per tenant** (autopay on the tenant's own Stripe).
Completes multi-tenant BYO end-to-end — Lago is one org per instance. Needs the
payment-method↔connection data-model decision first (Open Q1).

**B2. Ledger-as-a-feature: the close pack.** A one-click **audit/close pack** —
trial balance + GL export + reconciliation status — proving **every invoice
balances** and revenue-recognition ties out. UX + export over existing services;
turns the engineering invariant into a sales weapon Lago has no answer to.

### Track C — platform leapfrogs (post-quarter surpass roadmap)

Bigger bets that turn parity gaps into *leads*; sequenced after Track A/B.

**C1. MCP server — agent-operable billing (leapfrog GraphQL).** Rather than only
add GraphQL (parity), expose billing as **Model-Context-Protocol tools** over
the existing typed SDKs + per-instance OpenAPI, so **AI agents can operate the
billing system** — beyond Lago's *conversational* assistant. REST + (optional)
GraphQL remain; MCP is the differentiator.

**C2. Ledger-backed pricing units.** Offer abstract credits, but every issue/burn
is a **double-entry posting** against a credit-liability account — **auditable,
rev-rec-correct credits** with GST-on-advance handled. Lago's credits are a bare
balance.

**C3. Multi-entity with per-entity double-entry books.** Beyond invoicing from
multiple legal entities: each entity gets its own chart of accounts / trial
balance, **consolidated with intercompany elimination**; **per-GSTIN** entities
for India. Accounting-grade multi-entity Lago can't match.

**C4. Finance-grade RBAC.** Custom roles/permissions **+ segregation-of-duties +
every action in the append-only audit log** → SOC2-ready. Lago has RBAC but not
immutable-audit + SoD.

**C5. Correctness-at-scale (only when volume demands).** Push Postgres far
(partitioned event tables, COPY batch, in-process streaming aggregation) as a
**single Go binary**; add an optional columnar mirror later — but keep the
**ledger as source of truth at every scale**. Positioning: *"metered billing
provably balanced at 10k events/sec, in one binary + Postgres"* — scale **and**
correctness, which Lago's split Rails/Kafka/ClickHouse stack can't guarantee.

**C6. (Consider) EU e-invoicing (Peppol / EN16931)** + **vault-driven
integration framework** (new integration = a config/vault entry, not a
hand-written adapter; per-tenant, extending the BYO pattern) + **ledger-previewed
quotes/CPQ** (a quote shows the exact invoices + rev-rec schedule it produces).

### Ordering & parallelism

Track A: A1 first (rating layer); A2 + A4 parallelize after; A3 + A5 follow (need
the charge/event + invoice paths). Track B: B1 independent (gateway BYO shipped);
B2 independent (UX over the ledger). Track C is the post-quarter roadmap — C1
(MCP) and C4 (RBAC) are the highest-leverage next; C5 is volume-gated.

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

*Parity criteria (neutralize):*
1. **Charge models:** all 7 of Lago's models rated correctly, configurable from
   the tier builder + all 3 SDKs. Table-tested.
2. **Aggregations:** weighted_sum + latest supported; a metric using each rates
   onto an invoice correctly.
3. **Pay-in-advance:** an event on a pay-in-advance charge produces an immediate,
   ledger-balanced invoice line; a retried event does not double-bill.
4. **Charge filters:** two events with different property values bill at their
   respective filtered rates; a filter-less charge is unchanged.
5. **Progressive billing:** crossing a threshold generates one interim invoice
   for usage-so-far, ledger-balanced; the renewal doesn't re-bill it.

*Leapfrog criteria (surpass — these are the "ahead of Lago" proofs):*
6. **Pricing simulator:** simulating a charge set against last-period real usage
   returns the exact invoices **and** GL impact (A1).
7. **Rev-rec-correct advance billing:** a pay-in-advance charge auto-posts
   deferred revenue and recognizes it over the period, provable on the ledger (A3).
8. **Dimensional GL:** filtered revenue posts by dimension to the ledger and a
   filter can drive tax jurisdiction (A4).
9. **Risk-driven progressive billing:** a high-risk customer is billed
   progressively sooner than a low-risk one on the same threshold config (A5).
10. **Safe custom aggregation:** the sandboxed DSL evaluates without arbitrary
    SQL/code execution (A2) — passes a security review.
11. **B1:** a BYO tenant runs autopay on **their own** Stripe end-to-end.
12. **B2 close pack:** one click proves every invoice balances + rev-rec ties out.

*Program-level:*
13. **Comparison flip:** every Track-A row in the marketing table honestly flips
    to Recurso ✅ vs Lago — and at least 5 rows read as a Recurso *lead*, not
    parity (the leapfrogs). No regressions; full suite + E2E + invariant harness
    green at every commit.

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
