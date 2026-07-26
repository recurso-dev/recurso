# Design: Collections Intelligence

## Objective

Recurso already has a sophisticated **automated** recovery engine — a contextual
multi-armed bandit picking retry intervals per (currency, error_code, method,
amount, day, customer-age) context, multi-step dunning campaigns, an ACH
settlement/return lifecycle, and a recovered-revenue recorder feeding a "Smart
Dunning" analytics page. What's missing is the **operator-facing** layer: the
machine recovers money, but a human on the finance/collections team cannot *see*
who is failing right now, *why*, what happens next, or *act* on it.

Collections Intelligence closes that gap. It makes the existing recovery state
legible and actionable without rebuilding the engine underneath.

## What already exists (do not rebuild)

- Smart-retry bandit (`internal/service/smart_retry.go`), `dunning_weights` /
  `dunning_history`, RetryWorker (`internal/adapter/worker/retry_worker.go`).
- Dunning scheduler (email escalation + uncollectible) `internal/scheduler/dunning.go`.
- Dunning campaigns (`internal/service/dunning_campaign.go` + worker + UI).
- Recovery recorder `dunning_recovery.go` → `recovered_payments`; `GetRecoveredSummary`.
- ACH `payment_attempts` lifecycle incl. late-return reopen (Inc 3c).
- "Smart Dunning" dashboard (`frontend/src/pages/DunningDashboard.jsx`) + 4
  `/analytics/dunning/*` endpoints.
- Per-invoice state already persisted: `status` (past_due/uncollectible),
  `last_payment_error`, `retry_count`, `next_retry_at`, `dunning_managed_by`,
  `amount_paid`, plus the linked `payment_attempts.failure_code`.

## Gaps this closes

1. No **at-risk / failing-invoice queue** — the raw state exists but nothing lists it.
2. Failure codes captured (`last_payment_error`, ACH return codes) but **never aggregated or shown**.
3. No **manual intervention** — no "retry now", pause dunning, or manual write-off.
4. Recovery attribution is coarse — no **funnel** (failed→in-dunning→recovered vs lost), no **revenue-at-risk**, no segment recovery-rate.

## Increments (vertical slices, each shippable + independently valuable)

### Inc 1 — Collections queue (read-only) ✅ FIRST
The at-risk worklist. `GET /v1/collections/queue` lists invoices currently in
recovery (past_due / uncollectible), each with: customer, amount (currency-aware),
days overdue, `retry_count`, `last_payment_error` (humanized), `next_retry_at`,
`dunning_managed_by` (scheduler/worker/campaign), and in-flight ACH attempt state.
Paginated (`ParsePagination`), filterable by status + managed_by, sortable by
amount / days-overdue. Dashboard page "Collections" with a DataTable, status
chips, failure-reason column, EmptyState. **No money-path.**

### Inc 2 — Failure-reason & recovery-funnel analytics (read-only)
`GET /v1/analytics/collections/failures` — breakdown by error code / gateway /
payment method (count, still-failing vs recovered, recovery-rate, at-risk amount).
`GET /v1/analytics/collections/funnel` — failed → in-dunning → recovered vs
written-off, plus **revenue currently at risk** (sum of open past_due, FX-normalized
like MRR). Surface both on the Collections page. **Report-path, no ledger writes.**

### Inc 3 — Manual intervention controls (money-path-adjacent)
Row actions on the queue: **Retry now** (`POST /v1/invoices/:id/retry-now` →
set `next_retry_at=now`, respecting the in-flight guard + UPI-mandate safety so it
never double-charges), **Pause / resume dunning** (a flag the scheduler + worker
honor), **Mark uncollectible** (manual write-off — report-path). ConfirmDialog on
each. Gated on CI; retry-now must not bypass the ACH in-flight guard.

### Inc 4 — Retry-timing intelligence (optional, research-y)
Add a day-part dimension to the bandit context (or wire the vestigial
`domain/bandit.go` time-of-day arms) and surface a "recommended retry window".
Deferrable.

## Conventions

- Backend: hexagonal (`internal/core/{domain,port}`, `internal/adapter`,
  `internal/service`). Money = minor units int64. New list endpoints use
  `ParsePagination`/`clampLimitOffset`. Every registered route MUST be added to
  `cmd/api/openapi.yaml` (CI hard-gate `TestOpenAPISpecCoversRegisteredRoutes`).
- Frontend: react-query hooks (ADR-005), patterns in `src/components/patterns/`
  (DataTable, PageHeader, StatCard, EmptyState); detail/actions via right-side
  Sheets + ConfirmDialog. Money via currency-exponent-aware `utils.js` helpers.
- Any money/report-path change (Inc 2 at-risk sums, Inc 3 write-off) is flagged
  and gated on the invariant harness + E2E before merge.

## Success criteria

- An operator can open "Collections", see every currently-failing invoice with
  why + what's next, filter/sort it, understand *which failure reasons* cost the
  most and *how much revenue is at risk*, and (Inc 3) act on individual invoices.
- No regression to the automated engine; all existing dunning tests stay green.

## Boundaries

- **Always**: add routes to openapi.yaml; gate money/report-path on green CI.
- **Ask first**: changing bandit reward math or retry cadence defaults.
- **Never**: bypass the ACH in-flight guard or UPI-mandate double-charge safety
  in any manual "retry now" path.
