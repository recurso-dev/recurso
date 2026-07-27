# Design: pagination consistency

Status: proposed (inventory done, no behavior changed yet).
Backlog: `docs/backlog.md` P3.13. Silent truncation has bitten twice
(CLAUDE.md), so this is a design doc first — changing list defaults changes
behavior for existing clients.

## Current inventory (2026-07-28)

Three generations of list-endpoint behavior coexist:

1. **Hardcoded `limit := 10` + `page` param** — the oldest tier: customers,
   subscriptions, plans (`handler/customer.go:296` and siblings). Default 10
   rows; callers who forget `limit` silently get a truncated view.
2. **`parseLimitOffset(c, max, default)`** — mid-generation: mandate, ledger,
   coupon, credit-note, dispute, quote, offline-payment handlers, mostly
   `(1000, 1000)`.
3. **`ParsePagination(c)` / `clampLimitOffset`** — the current convention
   (collections, gifts, referrals): shared parsing, clamped, documented.
4. **Unbounded** — several older list endpoints accept no pagination at all
   and return every row (acceptable at today's volumes, a cliff later).

## Constraints

- Changing an unbounded endpoint to bounded **silently truncates** existing
  clients — the exact bug class this repo has been bitten by. Any change must
  be additive or versioned.
- The dashboard (react-query hooks in `src/lib/useCustomers.js` etc.) often
  relies on current defaults; it must be audited endpoint-by-endpoint before
  server changes.

## Proposal (incremental, non-breaking)

1. **Document reality first**: annotate every list operation in
   `cmd/api/openapi.yaml` with its actual default and max (`limit`/`offset`
   or `page` params). Zero behavior change; kills the surprise factor. This
   is the bulk of the work and is safe to do piecemeal.
2. **New endpoints**: `ParsePagination` only (already the convention).
3. **Tier-1 endpoints** (default 10): raise the default to a saner 50 with
   the same `page` semantics — strictly more data, nothing truncated, and the
   dashboard already passes explicit limits where it matters.
4. **Unbounded endpoints**: add `limit`/`offset` support (clamped at 1000,
   matching #224) while keeping the **no-param behavior unchanged** for one
   release; announce a future default in the changelog before flipping it.
5. Never change a default and a max in the same release; one variable at a
   time keeps client breakage diagnosable.

## Non-goals

- Cursor pagination: not needed at current volumes; revisit if any table
  crosses ~10^6 rows per tenant.
