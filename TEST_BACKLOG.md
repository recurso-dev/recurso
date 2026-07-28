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
- [x] Coupons, Mandates, Plans — done batches 2/4/5
- [x] Invoices, Audit Log, Disputes, Ask AI — done (feature work)
- [ ] Wallets, Metering, Gifts, Referrals, Organizations, OfflinePayments,
      CreditNotes, Ledger, Usage, Churn, DunningCampaigns, Quotes
- [ ] Finance report pages (MRRWaterfall, RevenueByPlan/Geography, TrialBalance,
      InvoiceAging, UnitEconomics, RevenueRecognition/Waterfall, MonthEndClose,
      FinanceReconciliation, GSTReturns)

## P3 — hooks / infra
- [x] `lib/useCustomers.js` (usePlans/useSubscriptions/useCustomers) — done batch 4
- [x] `components/patterns/StatCard` — done batch 2
- [ ] `lib/queryClient.js` — retry/gcTime config
- [ ] `auth/AuthProvider` — login/logout/session-restore
- [ ] `components/patterns/*` (FormField, LoadingSkeleton, ErrorState)

## P4 — backend edges (already strong; targeted gaps only)
- [ ] Handler validation paths that lack a table-driven test (400s, oneof enums)
- [ ] Pure helpers (pagination clamps, ID/number formatting, tax-label helpers)

## Known non-goals
- E2E/browser cross-testing beyond the existing E2E harness (infra-gated).
- Load/stress tooling (needs a separate environment).
