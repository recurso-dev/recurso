# Dashboard Polish — Batch F Report
## Context-preserving back-nav · Command-palette object coverage · Worklist sorting

Status: **F1, F2, F3 complete and merged; awaiting review.** The three approved
increments of the Stripe-grade gate are shipped as three green-CI PRs. No new
design system, no page-specific primitives, no accounting/posting changes.
Existing abstractions reused throughout: `DataTable`, `useUrlState` /
`useResetPageOnChange`, `ObjectHeader` / `ObjectPage`, `<Money>`,
`StatusBadge`, the command-palette fan-out, and the API/handler conventions.
Full frontend suite green (**750**); backend suite green (F2).

| Increment | PR | Nature |
|---|---|---|
| **F1** — Context-preserving object back-navigation | #710 | Frontend, shared layer |
| **F2** — Command-palette invoice + payment search | #711 | Additive backend `q` + palette |
| **F3** — URL-persisted worklist sorting | #712 | Frontend, shared layer |

---

## F1 — Context-preserving back navigation

**Problem.** An object page's back-link returned to the bare list root, dropping
the originating worklist's filters/search/page/sort. Returning from an invoice
snapped the operator back to page 1 / All.

**Fix (single shared layer, no per-page wiring).**
- `DataTable` stashes the originating list URL (`pathname + search`, i.e. the
  full filter/search/page/**sort** state) as React-Router navigation
  `state.from` on every row activation — both the row `<Link>` and the
  programmatic `navigate()`.
- `ObjectPage`/`ObjectHeader` resolve the back destination via a local
  `useListBackDestination(backTo)`: if `state.from` exists **and** its path
  matches the header's declared `backTo` list, return the full stateful URL;
  otherwise fall back to the static list root. Applied at both the header
  back-link and the `StateBackLink`.
- **Directly-opened object URLs** (no `state.from`) fall back to the list root —
  correct, since there is no origin to restore.
- **Browser Back** is unaffected (it uses history, not the link).
- The **dead `breadcrumbs` prop** on `PageHeader` (0 callers) was removed along
  with its nav block and unused imports — no global breadcrumbs introduced.

**Tests.** `BackNavigation.test.jsx` (6): back-link restores single- and
multi-filter + search + sort + page; falls back on direct open, cross-object,
and sibling sub-path (`/payments/offline` vs `/payments`); full
filtered-list → row-click → object → back round-trip.

---

## F2 — Command-palette object coverage (Invoices + Payments)

Extends ⌘K search to **Invoices** and **Payments (payment attempts)**. Credit
notes and disputes were explicitly deferred (no server-side `q`; documented as a
backend gap, not faked).

### Backend changes (smallest additive design — no signature or accounting change)

`q` search is genuinely server-side; the palette never loads arbitrary lists to
filter client-side.

- **`InvoiceRepository`** (interface + PG impl): new
  `SearchPaginated(tenantID, search, limit, offset)` and
  `CountSearch(tenantID, search)` — tenant-scoped, case-insensitive
  `invoice_number ILIKE`, newest-first, empty search returns no rows. Reuses
  `scanInvoiceList` + `hydrateLineItems`.
- **`PaymentAttemptRepository`**: new
  `SearchList(tenantID, search, limit, offset)` mirroring `List` (same
  `COUNT(*) OVER()` shape), matching `invoice_number ILIKE OR
  gateway_payment_intent_id ILIKE` — both operator-visible, unambiguous handles.
- **Service**: `SearchInvoicesPaginated` and `SearchPaymentAttempts`
  (nil-safe) added; new method threaded through the `paymentAttemptLister`
  interface.
- **Handlers**: `ListInvoices` / `ListPaymentAttempts` branch to the search path
  only when `?q` is a non-empty trimmed string — the existing plain-list and
  filter paths are byte-for-byte unchanged (no filter regression).
- **OpenAPI**: `q` query param added to `/v1/invoices` and
  `/v1/payment-attempts` (drift gate satisfied).
- No generic `/search` endpoint. Existing resource routes only.

### Palette coverage (reuses the existing fan-out)

- Two new `buildSection` groups (Invoices, Payments) alongside the unchanged
  Customers / Plans / Subscriptions. Debounce (200 ms), min-length (2),
  `AbortSignal` cancellation, deterministic ranking, grouped results,
  partial-failure isolation, and canonical navigation are all preserved.
- Invoice results deep-link to `/invoices/:id`; payment results to
  `/payments/:id` (the canonical payment-attempt object).
- The palette `Option` now renders `<Money>` for the amount (exponent-aware)
  plus the existing `StatusBadge`; invoice/payment icons added
  (`FileText` / `CreditCard`).
- Getters forward the `AbortSignal` (invoice/payment list calls) and send both
  `limit` and `per_page` (the two list endpoints read `per_page`; customers/
  plans/subscriptions read `limit`).

### Tests

- **PG repository tests** (`palette_search_pg_test.go`): case-insensitive match,
  **tenant isolation**, newest-first, `CountSearch`, pagination limit/offset,
  empty-`q` for both invoices and payments (match by invoice number **and**
  gateway ref).
- **Palette tests**: invoice/payment match with `q`/`limit`/`per_page`/`signal`
  assertions, Enter → canonical object, multiple groups, partial failure;
  Customers/Plans/Subscriptions regression guard.

### Live production QA (post-deploy)

Verified on `app.recurso.dev` / `api.recurso.dev` after the Cloud Run API deploy
landed:
- Backend `q` filters correctly: `q=INV-000009` → exactly `INV-000009`;
  `q=<nomatch>` → 0 (no false positives); payment `q=INV-DEMO-000018` → the one
  matching attempt.
- Palette (⌘K) for `INV-DEMO-000018` surfaces both an **Invoices** result
  (`$99.00 · Open`) and a **Payments** result (`Payment · INV-DEMO-000018 ·
  $99.00 · Processing`) with Money + StatusBadge + icons; empty groups drop out.
- Deep-links confirmed: invoice → `/invoices/6b15a655-…`; payment →
  `/payments/c69b11fb-…`.

---

## F3 — Worklist sorting sweep

Adds **honest, URL-persisted** column sorting to the client-side (fully-loaded)
list pages only. Server-paginated lists are left untouched.

### The shared mechanism (no new sort UI, no DataTable rewrite)

- **`src/lib/tableSort.js`**:
  - `useTableSort()` — persists the active sort as one `?sort=key:dir` URL param
    (default empty is omitted; writes use `replace` so sort clicks never flood
    history). Returns `{ sort, sortKey, onSortChange }`.
  - `sortRows(rows, sort, columns)` — sorts the **complete** in-memory set using
    each column's existing `sortValue` accessor (or `row[key]`). Returns the
    input untouched when there is no active sort or the column is unknown /
    non-sortable.
  - `compareValues` — the null-safe comparator (numbers numerically, else
    `localeCompare`, **nulls always last**).
- Pages feed DataTable's **existing controlled sort mode** (`sort` /
  `onSortChange`) and sort the full set **before** `pageSlice` — so the ordering
  spans the whole loaded set, never just the current page.
- DataTable's built-in client sort was refactored to call the same
  `sortRows`/`compareValues` — **one comparator**, so controlled and
  uncontrolled sorts order identically.
- **Page reset**: changing sort resets pagination to page 1 via the existing
  `useResetPageOnChange` (the `sortKey` is added to its dep list). Skip-first-run
  means a sort carried in the URL survives mount — so **F1 back-nav restores the
  exact ordering**.

### Lists made sortable

Every one loads the **bounded-complete** set (`fetchAllPages`, or Wallets' ≤500
single fetch) **with a truncation banner** — the "complete-enough + disclosed"
standard the pre-existing Invoices amount sort already shipped on.

| List | Load | Sortable columns |
|---|---|---|
| **Invoices** | page-through (≤10k) | Number, Customer, Amount, Status, Date — *moved from local `internalSort` → URL-persisted* |
| **Credit Notes** | page-through | Customer, Amount, Balance, Status, Created |
| **Coupons** | page-through | Code, Status |
| **Wallets** | ≤500 single fetch | Customer, Balance |
| **Gifts** | page-through | Status, Duration, Purchased |
| **Referrals** | page-through | Status, Reward, Created |

### Lists left backend-gated (NOT sorted)

Genuinely server-paginated (page is in the query key, each page a separate
fetch). Sorting one server page would misorder the set, so these are left as-is
and require a backend `ORDER BY` contract:

> **Customers, Payments, Subscriptions, Disputes, Quotes, Mandates, Events,
> Audit Log, Offline Payments, Plans, Ledger.**

**Required backend contract** to make any of these sortable honestly: accept a
`sort`/`order` query param (whitelisted columns + direction), apply it in the
SQL `ORDER BY` alongside the existing `LIMIT/OFFSET`, and keep the total count
stable. Until then the UI must not pretend to sort them.

### Honesty decisions

- **Coupons "Discount" is deliberately unsortable** — the column mixes a percent
  (percent coupons) and a money amount (fixed coupons); no single ordering is
  truthful. Noted inline in the column def.
- **Cross-currency amount/balance sorts** (Invoices Amount, Wallets Balance,
  Referrals Reward, Credit Notes Amount/Balance) compare **raw minor units**, so
  a mixed-currency tenant sees currencies interleaved by nominal minor amount.
  This is the same tradeoff the pre-existing Invoices amount sort already
  shipped; single-currency tenants (the common case) sort exactly. Documented
  inline. A future refinement could group by currency then amount, or move the
  sort server-side once those lists gain the `ORDER BY` contract.

### Tests

`tableSort.test.jsx` (19): comparator numeric-vs-lexical + nulls-last both
directions; `sortRows` asc/desc, unknown & non-sortable column, no-mutation,
`row[key]` fallback; `parseSort` validation; `useTableSort` URL round-trip
(read, write, clear, preserves `q`/`status`/`page`). DataTable's existing
controlled + uncontrolled sort tests (aria-sort, toggle asc→desc→none,
onSortChange delegation) cover the refactor.

### Live production QA (post-deploy)

Verified on `app.recurso.dev` after the F3 Cloudflare deploy:
- **Invoices** — clicking Amount cycles `?sort=amount:asc` → `:desc` (URL-persisted);
  the ordering spans the whole loaded set (every `$29.00` grouped at the top on
  asc; `₹234,820.00` on top on desc — the documented cross-currency behavior),
  with the header showing the ascending/descending arrow + `aria-sort`.
- **F1 × F3** — from a sorted list, opening an invoice then clicking the
  ObjectHeader back-link returns to `/invoices?sort=amount:desc` with the sort
  fully restored (router `state.from` carries the complete list URL).
- **Wallets** — exactly `Customer` and `Balance` are sortable (`aria-sort="none"`
  → `"ascending"` on click; URL `?sort=balance:asc`).
- **Customers** (server-paginated, gated) — **zero** sortable headers, no
  `aria-sort`: correctly left untouched, no fake single-page sort.

---

## Cross-cutting

### URL-state behavior

Sort now lives in the URL (`?sort=key:dir`) exactly like search/filter/page, so:
the sort is shareable/bookmarkable; F1 back-nav restores it; a refresh keeps it;
and changing it resets to page 1 without clobbering the other params (all writes
go through `useUrlState`'s `replace` + param-preserving setter).

### Accessibility

- Sortable headers are real `<button>`s inside `<th aria-sort>` (ascending /
  descending / none) — unchanged DataTable behavior, now exercised by six more
  lists. Keyboard: Tab to the header button, Enter/Space to cycle. Reduced-motion
  and focus rings unchanged.
- F1 added no new landmarks; the back-link remains a single `<Link>`. F2's
  palette a11y (roles, active-descendant) is unchanged.

### Performance / query behavior

- **F2**: search is a single indexed `ILIKE` per resource, `LIMIT 6`, tenant-
  scoped — no unbounded scans. The palette fans out the same queries it already
  did, plus two.
- **F3**: sorting is `O(n log n)` over the already-in-memory bounded set (the
  same set the page already filters and sums each render). No new fetches, no
  extra round-trips; passing a freshly-sorted array to DataTable matches the
  pre-existing filter-every-render posture (row ids are stable, so no false
  new-row reveals).

### Bugs found / fixed during the work

- F1: automation `<Link>` clicks via synthetic coordinates didn't trigger SPA
  navigation (a test-harness artifact, not a regression) — verified real DOM
  clicks navigate and carry state.
- F2: param-name mismatch (list endpoints read `per_page`, palette sent only
  `limit`) — palette now sends both. `AbortSignal` wasn't forwarded by the two
  getters — fixed.

### Remaining backend gaps (future work, not in scope)

- **Server-side `q`** for **credit notes** and **disputes** (to extend palette
  coverage) — same additive shape as F2.
- **`ORDER BY` contract** for the eleven server-paginated lists above (to extend
  honest sorting server-side).
- **Totals/pagination** on `/coupons`, `/gifts`, `/referrals`, `/credit-notes`,
  and **offset** on `/wallets` — today the dashboard page-throughs to a bounded
  cap and discloses truncation; a proper total + server pagination would remove
  the cap and enable server-side sort.

### Before → after (Stripe-grade assessment)

- **Navigation.** Before: object → list dropped context; operators re-filtered
  constantly. After: the worklist you left is the worklist you return to
  (filters, search, page, and sort) — the Stripe/Linear expectation.
- **Findability.** Before: ⌘K found customers/plans/subscriptions only; invoices
  and payments (the two most-searched financial objects) were unreachable by
  number. After: both are first-class, server-filtered, deep-linking to the
  canonical object with amount + status inline.
- **Worklists.** Before: only Invoices could sort (and it lost the sort on
  navigation). After: the six fully-loaded worklists sort by their real decision
  columns and the sort persists — while the lists that *can't* be sorted
  honestly are left untouched with the backend contract written down, not faked.

The dashboard moves another step from "generic SaaS admin" toward "financial
operating system": every worklist is orderable where it's honest, every object
is reachable, and every context survives a round-trip.
