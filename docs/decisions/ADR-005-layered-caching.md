# ADR-005: Layered caching — tenant-scoped Redis for reports, react-query for the dashboard

## Status
Accepted

## Date
2026-07-20

## Context
Two cost centers: server-side, the finance reports (trial balance, deferred
rollforward, rev-rec report/waterfall, `/analytics/*`) re-aggregate the whole
ledger per request; client-side, the SPA refetched everything on every
navigation (~5× necessary request volume — the pressure that made the
rate-limit incident user-visible). One page load fires many API calls; users
navigate constantly.

## Decision
Two independent layers with deliberately short TTLs:

1. **Server:** heavy read-only reports sit behind `CacheMiddleware` — Redis,
   5-minute TTL, key `cache:<url>:<tenant-id>`. If no tenant is resolved the
   middleware **skips caching entirely** (the previous ClientIP fallback key
   could serve one user's payload to another behind shared NAT if ever
   mounted pre-auth). Reconciliation is deliberately uncached: its "Run
   again" button must actually re-run.
2. **Client:** react-query with 60s `staleTime` / 5min `gcTime`, no
   focus-refetch. Reference data (customers/plans/subscriptions) lives in
   shared hooks (`useCustomers`/`usePlans`/`useSubscriptions`) so all pages
   share one fetch; server-driven lists key by `(page, q, status)` with
   `placeholderData` to avoid skeleton flashes. Mutations invalidate by key
   prefix.

## Alternatives Considered

### HTTP caching (Cache-Control / ETags)
- Pros: standards-based, benefits SDK consumers too
- Cons: per-user auth'd JSON caches poorly at intermediaries; ETag
  revalidation still costs a round-trip per request; doesn't reduce request
  *count*, which was the actual problem
- Deferred: worth adding for SDK/API consumers later; not a substitute

### Cache invalidation on write (server-side)
- Pros: no staleness window
- Cons: every money path would need to know every dependent report; the
  reports' inputs change via daily workers, so a 5-minute window is
  imperceptible
- Rejected: TTL is the right tool at this write/read ratio

### SWR (client library alternative)
- Roughly equivalent; react-query chosen for its explicit query-key
  invalidation model and devtools. Not a deep commitment — the shared hooks
  isolate the choice.

## Consequences
- Staleness contract: dashboards may lag writes by ≤60s (client) and reports
  by ≤5min (server). Anything that must be read-your-write goes through
  invalidation (see `fetchCustomers` in `Customers.jsx`) or bypasses caching.
- The API defaults list endpoints to `limit=10` — shared hooks must pass
  explicit limits (a silent-truncation bug shipped before this was written
  down; see the hooks' comments).
- New heavy read-only endpoints should join the `reportCache` group;
  anything with a "run now" semantic must not.
