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

## Running totals
- Frontend tests: 183 → 216 (+33).

## Method
- Behavioral assertions over implementation details.
- Every added file run green before commit; each batch ships as its own
  green-CI PR via the repo's patch-flow.

See `TEST_BACKLOG.md` for the ranked remaining work, `BUGS_FOUND.md` for issues,
`PRODUCTION_READINESS.md` for the scorecard.
