# Test Progress

Autonomous test-engineering run. Goal: raise the automated-test safety net toward
production grade **without rewriting the product**. Backend is already heavily
covered (319 Go test files vs 412 sources, incl. the ledger invariant harness);
the frontend is the primary gap (16/57 pages, 4/12 slide-overs, 3/9 lib modules
had tests at start).

## Baseline (run start)
- Frontend: 29 test files, 183 tests, all green.
- Backend: 319 `*_test.go` files; PG-backed suites + invariant harness green.

## Added this run

### Batch 1 — core lib (money + auth + data)
- `lib/__tests__/utils.test.js` (25 tests) — currency-exponent math is the money
  display layer used app-wide. Covers `currencyDecimals` (USD/JPY/KWD/BHD/KRW +
  fallbacks), `fromMinorUnits`/`toMinorUnits` (incl. rounding + round-trip
  property across currencies), `formatCurrency`/`formatCurrencyHeadline` (whole
  vs cents, negatives, zero-decimal currencies), `formatNumber`, `formatDate`
  (ISO/Date/invalid), `shortId`, `cn`.
- `lib/__tests__/authToken.test.js` (4 tests) — the in-memory API-key holder;
  asserts get/set/clear and, crucially, that the key is **never written to
  localStorage** (XSS-hardening contract).
- `lib/__tests__/countries.test.js` (3 tests) — ISO-3166 shape, unique codes,
  `COUNTRY_NAME` lookup completeness.

### Batch 2 — money-display components + a list page (PR #293)
- `components/ui/__tests__/money.test.jsx` (7) — `Money` renders minor units per
  currency exponent, negatives, the styled symbol span, className, nullish → $0.
- `components/patterns/__tests__/StatCard.test.jsx` (6) — label/value/hint,
  loading skeleton, delta colors, danger tone, link vs non-link tile.
- `pages/__tests__/Coupons.test.jsx` (5) — render, status filter, direct
  reactivate, deactivate-behind-confirm, empty state.

### Batch 3 — money-path slide-overs
- `slide-overs/__tests__/CreditNoteDetail.test.jsx` (7) — the Void money-path
  guards: Void offered only for an issued adjustment with a balance; hidden for
  refunds, zero-balance, and non-admins; requires confirmation. Plus
  approve/reject visibility for pending notes.
- `slide-overs/__tests__/CouponDetail.test.jsx` (4) — status badge, activate /
  deactivate toggle calls, redemption progress bar.

### Batch 4 — shared data hooks + a payments list page
- `lib/__tests__/useCustomers.test.jsx` (4) — `useCustomers`/`usePlans`/
  `useSubscriptions`: id→name maps, best-effort empty on failure, and the
  **anti-truncation guard** (asserts each hook requests `limit: 1000` so name
  resolution never silently drops past the API's default page).
- `pages/__tests__/Mandates.test.jsx` (3) — renders max/cycle money + method,
  revoke-behind-confirm (payload = mandate id), empty state.

### Batch 5 — plan pricing list
- `pages/__tests__/Plans.test.jsx` (2) — renders each plan's first-price money
  (per currency exponent) and the empty state.

### Batch 6 — growth list pages (lifecycle actions)
- `pages/__tests__/Gifts.test.jsx` (4) — renders gift codes, Cancel offered only
  for a purchased (unredeemed) gift, cancel-behind-confirm, empty state.
- `pages/__tests__/Referrals.test.jsx` (3) — renders referral codes, Qualify
  fires for a pending referral, empty state.

### Batch 7 — auth / session provider (security-critical)
- `auth/__tests__/AuthProvider.test.jsx` (6) — session resolution via /auth/me:
  authenticated success, **401 = definitive logout (no retry)**, **transient
  (503) retry-with-backoff then success**, login sets user, logout clears user +
  key + calls the API, and legacy API-key mode stays authenticated. Uses fake
  timers to exercise the backoff path deterministically.

### Batch 8 — customer detail + subscriptions list
- `slide-overs/__tests__/CustomerDetail.test.jsx` (5) — name/email render, the
  **payment-method-on-file** display (present with a card, omitted without), the
  **churn drill-in** (score + risk + drivers when a score is available), and the
  edit → `updateCustomer` save path.
- `pages/__tests__/Subscriptions.test.jsx` (3) — status badges render, row-click
  opens the detail sheet with the right subscription, empty state.

### Security fix (surfaced by the run's CI gates)
- **PR #300** — Trivy Security Scan flagged **CVE-2026-56852** (golang.org/x/text
  v0.38.0, HIGH, norm.Iter infinite loop). It was blocking *every* merge; bumped
  the indirect dep to v0.39.0. See BUGS_FOUND.md.

### Batch 9 — auth login flow + credit-notes list
- `pages/__tests__/Login.test.jsx` (4) — renders the form, logs in and navigates
  home, **advances to the 2FA step on mfa_required then loginMfa + navigate**,
  and surfaces an error (no navigate) on invalid credentials.
- `pages/__tests__/CreditNotes.test.jsx` (4) — renders reference + amount, opens
  the detail sheet on row-click, filters by customer via search, empty state.

### Batch 10 — invoice money-path actions + register (with a real fix)
- `slide-overs/__tests__/InvoiceDetail.test.jsx` (+3) — extended the existing
  tax-regime suite with the money-path actions: **Download PDF**, **Send invoice
  to customer**, **Preview** all call the right endpoint.
- `pages/__tests__/Register.test.jsx` (4) — form render, register → navigate,
  too-short-password blocks the API call, and an API error shows without
  navigating.
- **Fix:** `Register.jsx` `handleChange` switched to a functional `setFormData`
  update so browser-autofill batched field changes don't drop earlier fields
  (see BUGS_FOUND.md). One-line, behavior-preserving for normal typing.

### Batch 11 — prepaid wallets + trial balance (ledger invariant)
- `pages/__tests__/Wallets.test.jsx` (2) — renders a wallet balance (money),
  empty state.
- `pages/__tests__/TrialBalance.test.jsx` (2) — the **double-entry invariant in
  the UI**: a balanced book shows "Books balance" + debit/credit totals +
  "Balanced"; an out-of-balance book shows "Out of balance" + "Unbalanced".

### Batch 12 — invoice aging report + dunning campaign detail
- `pages/__tests__/InvoiceAging.test.jsx` (2) — outstanding total + aging bucket
  labels render; all-clear empty state when nothing is outstanding.
- `slide-overs/__tests__/DunningCampaignDetail.test.jsx` (2) — loads a campaign
  by id and renders its steps + active state; deactivate calls
  updateDunningCampaign with `{ is_active: false }`.

### Batch 13 — unit economics + revenue-by-plan reports
- `pages/__tests__/UnitEconomics.test.jsx` (2) — ARPA/ARPU/LTV render from the
  API; LTV shows "—" when `has_ltv` is false.
- `pages/__tests__/RevenueByPlan.test.jsx` (2) — total MRR + per-plan segments
  (money + labels); empty state when there's no plan revenue.

### Batch 14 — revenue-by-geography + MRR waterfall
- `pages/__tests__/RevenueByGeography.test.jsx` (2) — total MRR + per-country
  segments, empty state.
- `pages/__tests__/MRRWaterfall.test.jsx` (2) — starting/ending MRR + net
  movement (+$200 from the mocked deltas); requests the default trailing-month
  range (two date args).

### Batch 15 — churn ops + revenue-recognition waterfall
- `pages/__tests__/Churn.test.jsx` (3) — high-risk customers render with score;
  **acknowledge** an alert calls acknowledgeChurnAlert(id); no-alerts state.
- `pages/__tests__/RevenueWaterfall.test.jsx` (1) — total recognized renders
  from the recognition curve (dual-query page; rollforward stubbed).

### Batch 16 — team RBAC gate
- `pages/__tests__/Team.test.jsx` (3) — members + roles render; the **Add member**
  action is shown to an owner/admin and **hidden from a plain member** (RBAC gate).

### Batch 17 — password-reset auth flows
- `pages/__tests__/ForgotPassword.test.jsx` (2) — submits the email + shows the
  generic sent confirmation, and **still shows success on a request error**
  (no account-enumeration — a security property).
- `pages/__tests__/ResetPassword.test.jsx` (2) — resets with the URL token then
  redirects to /login; shows the invalid-link state when the token is missing.

## Running totals
- Frontend tests: 183 → 316 (+133). Test files: 29 → 63.
- Shipped as 17 green-CI test PRs (#292–#299, #302–#309, + this) plus a HIGH-CVE
  security fix (#300) and a batched-autofill form fix.
- **316 frontend tests across 63 files.** Backend: 319 test files + harness.

### Batch 18 — API client contract
- `lib/__tests__/api.test.js` (4) — locks the HTTP method + URL for critical
  endpoints (mocks axios, asserts the sent path): resource URLs
  (`/customers/:id`, `/invoices/:id/pdf` as a blob), money-path mutations
  (credit-note void/pdf, subscription cancel body, dispute resolve), lifecycle
  actions (approve, quote delete, coupon toggle, mandate revoke), and list
  query-param pass-through. Catches endpoint path typos — a real bug class.

### Batch 19 — developer API keys (security)
- `pages/__tests__/Developers.test.jsx` (3) — lists API keys by prefix, creates
  a new key (createKey), and **revokes a key only after confirmation**
  (revokeKey(id)). All the page's other reads (webhooks/events/deliveries) are
  stubbed empty so the keys tab renders in isolation.

### Batch 20 — metering + offline payments
- `pages/__tests__/Metering.test.jsx` (2) — renders billable metrics; empty state.
- `pages/__tests__/OfflinePayments.test.jsx` (2) — renders a recorded offline
  payment amount (money); empty state.

### Batch 21 — organizations + GST returns
- `pages/__tests__/Organizations.test.jsx` (2) — renders orgs; empty state.
- `pages/__tests__/GSTReturns.test.jsx` (2) — build-return action present;
  building GSTR-1 calls getGSTR1 and shows the result payload.

### Batch 22 — quotes + dunning campaigns lists
- `pages/__tests__/Quotes.test.jsx` (2) — quotes render with number + status; empty.
- `pages/__tests__/DunningCampaigns.test.jsx` (2) — campaigns render with active
  state; empty state.

### Batch 23 — subscription detail (core money-path slide-over)
- `slide-overs/__tests__/SubscriptionDetail.test.jsx` (3) — renders the
  subscription; the Cancel button opens the cancel dialog and loads the reason
  catalog; and the confirm stays **disabled until a reason is chosen** — the
  #290 cancel-with-reason guard, now verified directly in the component (not just
  via the endpoint payload).

### Batch 24 — cancel-flow detail (save-offer config)
- `slide-overs/__tests__/CancelFlowDetail.test.jsx` (2) — loads a flow by id,
  renders its steps (offer headline in the summary); deactivate calls
  updateCancelFlow({is_active:false}).

## Running totals (updated)
- Run total: **183 → 340 (+157) across 24 test PRs**, plus a HIGH-CVE fix and an
  autofill form fix. Test files: 29 → 73.
- Pages tested: ~45/57. Slide-overs: **11/12** (only PricingSimulator +
  CancelFlowStepConfig config helpers remain). lib: 9/9.

### Batch 25 — backend handler-validation (money-path RBAC + input guards)
- `internal/adapter/handler/dispute_validation_test.go` (4) — ResolveDispute
  rejects a bad id (400) and an outcome outside accept/reject (400); a valid
  'reject' passes validation; ListDisputes rejects an unknown status (400).
- `internal/adapter/handler/credit_note_validation_test.go` (3) — VoidCreditNote
  rejects a bad id (400) and a non-admin (403); ApproveCreditNote is
  admin/owner-only (403). All validate BEFORE the service, so they run with nil
  service deps and **no database** — fast, deterministic RBAC/guard coverage.
- Backend test files: 319 → 321. These are the first tests to directly cover the
  handler-layer authorization + request-validation guards on the money-path
  endpoints added this session.

## Honest assessment (end of run)
The remaining untested files are **low-risk, low-incremental-value**: settings/
account pages (Profile, Security, Integrations, ExecutiveSummary), on-demand
report pages already exercised via siblings (RevenueRecognition, MonthEndClose),
create-forms whose only logic is `toMinorUnits` (already exhaustively unit-tested
in batch 1) behind Radix selects that are impractical to drive in jsdom, and two
config-helper components. Every page also mounts under `PageSmoke`. Per the
directive's own criterion ("keep adding tests until additional tests provide
little incremental value"), the high-confidence frontier has been reached for the
frontend. **Highest-value NEXT work is infrastructure, not more page tests:**
1. Add `@vitest/coverage-v8` + a CI coverage gate (deferred here to avoid a
   lockfile change mid-run) — this turns "file presence" into measured line/branch
   coverage and surfaces true gaps.
2. Backend: targeted table-driven handler-validation tests (400/oneof paths) —
   the one area with headroom despite 319 existing test files.
3. E2E flows beyond the existing harness (infra-gated).
- Remaining pages (lower-risk): ExecutiveSummary, Integrations, Profile, Security,
  Usage, RevenueRecognition, MonthEndClose, AcceptInvite, Create{Coupon,CreditNote,
  Plan}. Slide-overs: CancelFlowDetail, CancelFlowStepConfig, PricingSimulator,
  SubscriptionDetail (SubscriptionDetail is the highest-value remaining — complex,
  multi-endpoint; its cancel-with-reason flow is verified indirectly via the
  #290 payload but a dedicated suite is worthwhile).
- NOTE: line-coverage tooling (`@vitest/coverage-v8`) is not installed — a hard
  coverage % isn't measured. Adding it (+ a CI coverage gate) is a follow-up
  (deferred to avoid a lockfile change mid-run). Confidence is tracked by
  risk-weighted surface coverage above, not a %.

## Auth surface — now fully covered
AuthProvider (session/retry/401) · Login (+2FA) · Register (+autofill fix) ·
ForgotPassword (no-enumeration) · ResetPassword (token flow) · Team RBAC gate ·
API-key-never-in-localStorage. This was the single highest-risk untested area at
the start of the run.

## What this run hardened (behavioral, not coverage padding)
- **Money display end-to-end**: `utils` exponent math + the `Money` component —
  the layer every currency amount passes through (USD/JPY/KWD/BHD).
- **Money-path UI guards**: credit-note **Void** is provably gated (issued
  adjustment + balance + admin + confirm) and mandate **revoke** requires confirm.
- **Security contract**: the API key is never persisted to localStorage.
- **Silent-truncation guard**: shared name-resolution hooks always request the
  full set (`limit:1000`).
- **List-page contracts**: render / status-filter / empty across representative
  pages (Coupons, Mandates, Plans) on top of the pages already covered earlier
  (Invoices, Audit Log, Disputes, Ask AI, Dashboard).

## Method
- Behavioral assertions over implementation details.
- Every added file run green before commit; each batch ships as its own
  green-CI PR via the repo's patch-flow.

See `TEST_BACKLOG.md` for the ranked remaining work, `BUGS_FOUND.md` for issues,
`PRODUCTION_READINESS.md` for the scorecard.
