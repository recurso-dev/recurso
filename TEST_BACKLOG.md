# Test Backlog (ranked by risk × value)

Frontend is the coverage gap. Ranked highest-value first.

## P0 — money / correctness-critical
- [x] `lib/utils.js` money math (currency exponents) — **done batch 1**
- [ ] `components/ui/money.jsx` (`Money`) — renders minor units; used in every table
- [ ] `SubscriptionDetail` — plan change proration preview, add-on math, cancel-with-reason, charges
- [ ] `CreateSubscription` / `CreatePlan` / `CreateQuote` — form → minor-unit payload correctness
- [ ] `PricingSimulator` / `PlanCharges` — charge-model math (per-unit/graduated/volume/package)

## P1 — critical workflows / slide-overs
- [ ] `InvoiceDetail` — PDF/preview/send, e-invoice retry/cancel, statuses
- [ ] `CustomerDetail` — edit, archive guard, credit statement, entitlements, churn drill-in
- [ ] `PlanDetail` — edit, entitlements
- [ ] `DunningCampaignDetail`, `CancelFlowDetail` — step CRUD, active toggle
- [ ] `Collections` page — worklist actions (retry/pause/mark-uncollectible)

## P2 — list pages (filter/search/empty/error/row-click)
- [ ] Wallets, Coupons, Plans, Metering, Mandates, Gifts, Referrals, Organizations,
      OfflinePayments, CreditNotes, Ledger, Usage, Churn, DunningCampaigns, Quotes
- [ ] Finance report pages (MRRWaterfall, RevenueByPlan/Geography, TrialBalance,
      InvoiceAging, UnitEconomics, RevenueRecognition/Waterfall, MonthEndClose,
      FinanceReconciliation, GSTReturns)

## P3 — hooks / infra
- [ ] `lib/useCustomers.js` (usePlans/useSubscriptions/useCustomers) — cache shape
- [ ] `lib/queryClient.js` — retry/gcTime config
- [ ] `auth/AuthProvider` — login/logout/session-restore
- [ ] `components/patterns/*` (FormField, StatCard, LoadingSkeleton, ErrorState)

## P4 — backend edges (already strong; targeted gaps only)
- [ ] Handler validation paths that lack a table-driven test (400s, oneof enums)
- [ ] Pure helpers (pagination clamps, ID/number formatting, tax-label helpers)

## Known non-goals
- E2E/browser cross-testing beyond the existing E2E harness (infra-gated).
- Load/stress tooling (needs a separate environment).
