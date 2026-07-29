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

## Recommended next execution order (for the next session)
1. `Developers` (API-key revoke = security-relevant) + `OfflinePayments` (money).
2. `Metering`/`Usage` (usage-based billing surface).
3. `Organizations` (multi-tenant), `Security`/`Profile` (account settings).
4. Remaining finance reports (RevenueRecognition, MonthEndClose, GSTReturns).
5. `SubscriptionDetail` dedicated suite (cancel-reason flow, plan-change preview).
6. Pattern components + `lib/queryClient.js`.

## P4 — backend edges (already strong; targeted gaps only)
- [ ] Handler validation paths that lack a table-driven test (400s, oneof enums)
- [ ] Pure helpers (pagination clamps, ID/number formatting, tax-label helpers)

## Known non-goals
- E2E/browser cross-testing beyond the existing E2E harness (infra-gated).
- Load/stress tooling (needs a separate environment).
