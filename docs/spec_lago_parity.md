# Spec: Lago Parity Program (Usage-Billing Phase 2 + Platform Tail)

> **Status: DRAFT — awaiting founder review (assumptions A1–A6, decisions D1–D8)**
>
> Successor to `spec_usage_billing.md` (v1 shipped 2026-07-18, commit
> e457bc9). Target: functional parity with Lago (getlago.com, 10.2k★,
> AGPLv3) on the billing core, while keeping Recurso's differentiators —
> GST/e-invoicing/TDS, double-entry ledger, UPI mandates, data residency —
> as invariants, not casualties.

## Objective

An Indian SaaS or AI/API company evaluating Lago vs. Recurso should find
**no billing-core reason to pick Lago**, and several India/finance reasons
to pick Recurso.

Post-v1 gap list vs. Lago, in priority order:

| # | Gap | Track |
| --- | --- | --- |
| 1 | Renewal scheduler absent; mandate-debit invoices miss metered lines (honest flag 2) | A |
| 2 | SDKs lack metering methods (honest flag 1) | A |
| 3 | Prepaid wallets / credits with usage burn-down | B |
| 4 | Minimum commitments + true-up | B |
| 5 | Usage threshold alerts | B |
| 6 | Batch + idempotent event ingestion | C |
| 7 | Audit log | C |

Success looks like: a hybrid plan (flat + usage) renews **automatically**
on schedule, drains a prepaid wallet before charging a gateway, tops the
customer up when a threshold alert fires, true-ups to the committed
minimum, and every step is visible in SDKs, dashboard, and the audit log —
with GST-correct invoices and balanced ledger postings throughout.

## Track A — Close the honest flags

### A1. Billing cycle scheduler (the missing renewal loop)

New `internal/scheduler/billing_cycle.go` (same shape as the six existing
schedulers: ticker + `stopOnce` + conditional-UPDATE claims):

- Scans for ACTIVE subscriptions with `current_period_end <= now` that are
  **not** mandate-managed (mandate debits keep their own cycle) and not
  `cancel_at_period_end` pending cancellation processing.
- Per due subscription, atomically claims it (conditional UPDATE on
  `current_period_end`, mirroring the mandate cycle-key pattern), calls
  `InvoiceService.GenerateInvoice` (which now rates usage), advances the
  period with the existing anchor-preserving `CalculateNextBillingDate`,
  and attempts payment via the stored method where one exists.
- `cancel_at_period_end` subscriptions get their final period rated, then
  transition to canceled.
- Config: `BILLING_CYCLE_INTERVAL` (default 5m), enabled by default.

### A2. Metered lines on mandate-debit invoices

The mandate-debit path builds its own invoice (`service/mandate.go`).
Reuse `InvoiceService.meteredLines` there so UPI-autopay customers —
the mainline India path — get usage lines too. The existing
`mandate_cycle_key` uniqueness plus the `usage_ratings` claim make the
two idempotency layers compose.

### A3. SDK metering methods + release

All three SDKs gain: `billableMetrics` (CRUD), `plans.setCharges/getCharges`,
`subscriptions.usageAmount`, `usage.record` properties param.
Node `recurso@1.3.0`, Python `recurso@1.2.0`, Go `v1.1.0` — prepared and
tagged; npm/PyPI publish is founder-gated (OTP/token).

## Track B — Lago Phase 2 commerce

### B1. Prepaid wallets

New domain: `Wallet` (per customer+currency) and `WalletTransaction`
(top-up / drain / expiry / void, with balance-after audit trail).

- **Denomination (D1):** money-denominated in currency minor units — not
  Lago's abstract "credits with rate conversion". Simpler, ledger-native;
  credit-unit pricing can layer on later if a customer demands it.
- **Top-ups:** `POST /v1/wallets/{id}/top-up` — via gateway charge or
  offline payment, producing a normal invoice/receipt so the money enters
  the existing ledger flow. Optional expiry date per top-up (paid credits
  vs. promotional credits with expiry, D2).
- **Drain at invoice time:** after `GenerateInvoice` commits, wallet
  balance applies to the invoice **before** gateway charge and before
  adjustment credit notes (D3 ordering: wallet → credit notes → gateway),
  recorded as a `WalletTransaction` + ledger posting (DR Customer Credit /
  liability, CR AR — reusing account code 2xxx Customer Credit).
- **GST on top-ups (D4, open):** prepaid top-ups are advances; GST
  treatment of advances for services needs CA sign-off. v1 behaviour:
  top-up receipts carry no tax; tax is charged on the consumption invoice
  as today. Flagged for the CA review already on the founder list.
- **Auto-recharge:** threshold + amount on the wallet; the billing-cycle
  scheduler triggers top-up when balance falls below threshold (uses the
  stored payment method; skipped when none).

### B2. Minimum commitments + true-up

`Commitment` on a subscription: `amount_minor` per period (per currency).
At period close, if flat + metered subtotal < commitment, a **true-up
line** ("Commitment shortfall") fills the difference on the same invoice —
taxed at the plan HSN. Overage needs nothing: usage charges already bill
past any included tier. Commitment visible in `usage-amount` preview.

### B3. Usage threshold alerts

`UsageAlert`: subscription (or plan-default) + metric + thresholds
(absolute quantity or % of entitlement limit / commitment). Evaluated by a
scheduler sweep (reuse the health-alert scheduler cadence); firing emits a
webhook event `usage.alert.triggered` + notification (email/in-app via the
existing notifier), with once-per-period-per-threshold dedup.

## Track C — Scale + trust tail

### C1. Ingestion hardening

- `POST /v1/usage/events/batch` — up to 500 events, per-item results.
- **Event idempotency (D5):** optional `transaction_id` per event; unique
  `(subscription_id, transaction_id)` partial index; duplicate returns the
  original event id (Lago-compatible semantics). Retry-safe SDK ingestion.
- Index review: `usage_events (subscription_id, dimension, timestamp)`
  covering index for the aggregation hot path.
- ClickHouse/events-processor: **explicitly deferred** until a customer
  exceeds Postgres comfort (~thousands of events/sec); noted as roadmap.

### C2. Audit log

`audit_logs` append-only table: tenant, actor (user/API key), action,
entity type+id, before/after JSON summary, IP, timestamp. Written from the
mutation paths of config-grade entities (plans, charges, metrics, coupons,
entitlements, webhooks, team, wallets, commitments) via a small
`audit.Record(ctx, ...)` helper. `GET /v1/audit-logs` (filter by entity,
actor, time) + dashboard page. No deletes, no updates — 409 on attempts.

## Explicitly out of scope

SOC 2, CRM connectors (Salesforce/HubSpot), marketplace listings, Lago AI,
white-label embedded billing, ClickHouse — business-gated or
volume-gated, tracked on the roadmap, not in this program.

## Tech stack

Unchanged: Go 1.25, Gin, PostgreSQL (authoritative) + optional
TigerBeetle mirror, React dashboard, Mintlify docs. No new runtime
dependencies anticipated (D6: confirm none slip in without asking).

## Commands

```
go build ./...                                   # build
go test ./... -count=1                           # full test suite
go test ./internal/service/ -run TestWallet      # focused
gofmt -l internal cmd && golangci-lint run       # lint (pre-commit runs it)
make demo                                        # seeded local stack (Docker)
```

## Project structure (where new code lands)

```
internal/core/domain/        wallet.go, commitment.go, usage_alert.go, audit.go
internal/core/port/          *_repository.go per aggregate
internal/adapter/db/         repositories + migrations/0000{96..}_*.sql
internal/service/            wallet.go, commitment true-up in invoice.go,
                             alert evaluation, audit helper
internal/scheduler/          billing_cycle.go, alert sweep
internal/adapter/handler/    wallet.go, audit.go, metering additions
cmd/api/main.go              wiring + routes
cmd/api/openapi.yaml         spec (sync copy to recurso-docs/api-reference/)
frontend/src/pages/          Wallets, AuditLog; charges UI on plan editor
../recurso-{node,python,go}  SDK methods (own repos, own releases)
```

## Code style

Match the house style exactly — this snippet is the reference:

```go
// Drain applies the wallet balance to a committed invoice, oldest
// non-expired top-up first. Best-effort at generation time: a failure
// leaves the invoice at its full amount (recoverable), never fails
// invoice creation — mirroring credit-note application (ENG-153).
func (s *WalletService) Drain(ctx context.Context, tenantID uuid.UUID, inv *domain.Invoice) (int64, error) {
```

Conventions: sentinel error values + typed `ValidationError` strings per
service; nil-safe optional collaborators on services; repositories return
`(nil, nil)` for not-found on reads and `sql.ErrNoRows` on zero-row
writes; comments state constraints, not narration; money is int64 minor
units except decimal-string rates (spec_usage_billing D1).

## Testing strategy

- Table tests for all pure money math (wallet ordering, true-up, alert
  threshold evaluation) — same rigor as `rating_test.go`.
- Fake-repo service tests for every invoice-path change asserting the
  standing invariants: `Total == Subtotal + Tax`, `Σ line.Amount ==
  Subtotal`, `Σ line tax == TaxAmount`, ledger legs balance.
- Scheduler claim tests proving at-most-once per period under concurrent
  ticks (pattern from the mandate cycle-key tests).
- SDK: vitest / pytest / go test suites extended for each new method.
- Every migration has a working `.down.sql`.

## Boundaries

- **Always:** run `go test ./... && golangci-lint` before commit; route
  every invoice line through `newInvoiceLine`; post ledger entries for
  every money movement; keep residency guards on any new egress.
- **Ask first:** new runtime dependencies; schema changes beyond the
  migrations named here; changing payment-ordering semantics (D3);
  anything touching gateway charge flows; publishing SDK releases.
- **Never:** commit secrets; weaken an existing invariant test; bill the
  same usage window twice (usage_ratings claim is sacred); auto-publish
  to npm/PyPI/GHCR; make recurso-archive public.

## Success criteria

1. A hybrid subscription with no mandate renews **unattended**: scheduler
   fires, invoice carries flat + metered lines, payment attempts, period
   advances with anchor preserved. Proven by an integration test.
2. Mandate-debit invoices carry metered lines; idempotent under retry.
3. Wallet flow end-to-end: top-up → invoice drains wallet before gateway
   → ledger balanced → transactions listable. Zero-balance and expiry
   edge cases tested.
4. Commitment shortfall lines appear exactly when subtotal < commitment,
   never otherwise; preview shows the projected true-up.
5. Alerts fire once per threshold per period, delivered via webhook +
   notification.
6. Batch ingest of 500 events with duplicate `transaction_id`s stores
   each unique event exactly once.
7. Every config mutation appears in the audit log; the log rejects
   updates/deletes.
8. All three SDKs expose every v1 metering endpoint + wallets; releases
   tagged, publish commands documented for the founder.
9. Full test suite + lint green at every commit; docs + OpenAPI synced;
   CHANGELOG Unreleased updated per track.

## Founder decisions

| # | Question | Recommendation |
| --- | --- | --- |
| D1 | Wallet denomination | **Money (minor units), not abstract credits** |
| D2 | Promotional credits with expiry in v1 | **Yes** — expiry date per top-up, promotional flagged non-refundable |
| D3 | Payment application order | **Wallet → adjustment credit notes → gateway** |
| D4 | GST on wallet top-ups | **No tax on top-up receipt; tax at consumption** — pending CA sign-off (already founder-gated) |
| D5 | Event idempotency key | **Optional `transaction_id`, unique per subscription** |
| D6 | New runtime dependencies | **None expected; any addition needs explicit approval** |
| D7 | Scheduler default state | **On by default** (`BILLING_CYCLE_INTERVAL=5m`; disable with `0`) |
| D8 | Track order | **A → B1 → B2 → B3 → C1 → C2** (flags first, then wallets — the most-asked-for Lago feature) |

## Open questions

1. D4 needs the CA's word eventually — is "no tax on top-up" acceptable
   as the shipping default until then?
2. Should auto-recharge failures (no payment method / gateway decline)
   notify the tenant, the customer, or both? (Default: both.)
3. Alert channels: webhook + email + in-app now; SMS too? (Default: no —
   SMS costs money and Twilio config is optional.)
