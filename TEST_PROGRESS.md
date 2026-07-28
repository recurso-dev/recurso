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

## Running totals
- Frontend tests: 183 → 274 (+91). Test files: 29 → 47.
- Shipped as 8 green-CI PRs (#292–#298, + this).

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
