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

## Running totals
- Frontend tests: 183 → 244 (+61).

## Method
- Behavioral assertions over implementation details.
- Every added file run green before commit; each batch ships as its own
  green-CI PR via the repo's patch-flow.

See `TEST_BACKLOG.md` for the ranked remaining work, `BUGS_FOUND.md` for issues,
`PRODUCTION_READINESS.md` for the scorecard.
