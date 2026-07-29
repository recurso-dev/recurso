# Test Backlog (ranked by risk × value)

Frontend is the coverage gap. Ranked highest-value first.

## P0 — money / correctness-critical
- [x] `lib/utils.js` money math (currency exponents) — done batch 1
- [x] `components/ui/money.jsx` (`Money`) — done batch 2
- [ ] `SubscriptionDetail` — plan change proration preview, add-on math, cancel-with-reason, charges (partially exercised via feature tests; needs a dedicated suite)
- [ ] `CreateSubscription` / `CreatePlan` / `CreateQuote` — form → minor-unit payload correctness (CreateSubscription has a test; Plan/Quote create paths still open)
- [ ] `PricingSimulator` / `PlanCharges` — charge-model math (NOTE: math is backend — client only displays; low frontend value)

## P1 — critical workflows / slide-overs
- [ ] `InvoiceDetail` — PDF/preview/send, e-invoice retry/cancel, statuses
- [ ] `CustomerDetail` — edit, archive guard, credit statement, entitlements, churn drill-in
- [ ] `PlanDetail` — edit, entitlements
- [ ] `DunningCampaignDetail`, `CancelFlowDetail` — step CRUD, active toggle
- [ ] `Collections` page — worklist actions (retry/pause/mark-uncollectible)
- [x] `CreditNoteDetail` — void guards, approve/reject — done batch 3
- [x] `CouponDetail` — toggle — done batch 3
- [x] `QuoteDetail` — lifecycle actions — done (feature work)

## P2 — list pages (filter/search/empty/error/row-click)
- [x] Coupons, Mandates, Plans, Gifts, Referrals, Subscriptions, CreditNotes,
      Wallets, Churn, Team — done (batches 2–17)
- [x] Invoices, Audit Log, Disputes, Ask AI — done (feature work)
- [ ] Metering, Organizations, OfflinePayments, Ledger, Usage, DunningCampaigns,
      Quotes, Integrations, Security, Notifications, Profile
- [x] Finance reports done: TrialBalance, InvoiceAging, UnitEconomics,
      RevenueByPlan, RevenueByGeography, MRRWaterfall, RevenueWaterfall
- [ ] Finance reports remaining: RevenueRecognition, MonthEndClose, GSTReturns,
      FinanceReconciliation (has a test already), ExecutiveSummary

## P3 — hooks / infra
- [x] `lib/useCustomers.js` — done batch 4
- [x] `components/patterns/StatCard` — done batch 2
- [x] `auth/AuthProvider` — done batch 7 (+Login/Register/Forgot/Reset in 9/10/17)
- [ ] `lib/queryClient.js` — retry/gcTime config
- [ ] `components/patterns/*` (FormField, LoadingSkeleton, ErrorState, DataTable
      pagination controls)
- [ ] `Developers` page — webhooks + API-key create/revoke + deliveries (complex,
      multi-endpoint; security-relevant — good next target)
- [ ] `Ledger` page — coupled accounts+entries queries with account selection
- [ ] `SubscriptionDetail`, `CancelFlowDetail`, `PricingSimulator` slide-overs

## P4 — backend handler validation (DB-free RBAC/input guards)
- [x] dispute, credit-note, team, tax-nexus, organization, tenant handlers —
      done batches 25–28 (15 tests). Pattern: construct the handler with nil/empty
      service deps, `jsonCtx(...)`, set `tenant_id`/`user_role` on the context,
      call the method, assert 400/403 (these paths return before the service).
- [ ] Remaining write handlers to sweep the same way: gateway_connection (has a
      test already), integration_connection (has test), sso (has test), plus
      coupon/plan/subscription/quote create handlers' binding validation.

## Recommended next execution order (for the next session)
1. **Coverage tooling decision** — add `@vitest/coverage-v8` (frontend) + a CI
   coverage gate. This is the single highest-value item: it converts "file
   presence" into measured line/branch coverage and pinpoints true gaps. Requires
   a package.json/lockfile change (deferred here to avoid an unreviewed dep bump).
2. More backend handler-validation (coupon/plan/subscription create binding
   paths) — cheap, DB-free, same pattern as batches 25–28.
3. Backend service-layer edge tests behind the invariant harness (PG-gated).
4. Low-value frontend remainder (settings/create-form pages) — only if pursuing
   a coverage-% target; otherwise skip (PageSmoke already mounts them).
5. E2E flows beyond the existing harness (infra-gated).

## Decisions recorded this run
- Skipped installing coverage tooling mid-run (lockfile/CI change needs review).
- Kept every batch DB-free where possible so tests are fast + deterministic.
- Fixed defects when safe (cancel-400, x/text CVE, Register autofill) and
  de-flaked one test (AskAnalytics localStorage race), each with a regression
  assertion. See BUGS_FOUND.md.

## P4 — backend edges (already strong; targeted gaps only)
- [ ] Handler validation paths that lack a table-driven test (400s, oneof enums)
- [ ] Pure helpers (pagination clamps, ID/number formatting, tax-label helpers)

## Known non-goals
- E2E/browser cross-testing beyond the existing E2E harness (infra-gated).
- Load/stress tooling (needs a separate environment).
