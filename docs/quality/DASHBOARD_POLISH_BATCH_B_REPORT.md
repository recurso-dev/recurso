# Dashboard Polish — Batch B Report
## State & Error Consolidation

Status: **complete, awaiting review.** One canonical object-page lifecycle now
governs every object detail page. Lint, build, and the full 703-test suite are
green. No backend, motion, token, color, or Money changes.

---

### 1. Scope

**In scope (done):** one canonical query-state model (`useObjectQuery`), one
loading state (`ObjectPageSkeleton`), one not-found state (`ObjectNotFound`),
one error state (`ObjectPageError`), one safe `errorMessage` helper, and a
root-level fix for the live 404-copy bug — then migration of all standard
object detail pages onto them.

**Deliberately not in scope (unchanged):** Label/Overline, DataTable sticky
headers / accessible table names, breadcrumbs, back-navigation architecture,
ObjectHeader action wrapping, Recent Objects, unified search, failed-webhook
feed, backend gaps, motion, token/color redesign, the Money system, and any new
feature. These belong to later batches.

### 2. Architecture

Three thin, additive layers on top of the existing design system — no second
page framework, no duplicate `ObjectHeader`, no page-specific state components:

```
src/lib/httpError.js       ← status/error classification + safe errorMessage()
src/lib/useObjectQuery.js  ← classifier over ONE object GET (wraps react-query)
src/components/patterns/ObjectPage.jsx
   ├─ ObjectPageSkeleton   ← canonical loading
   ├─ ObjectNotFound       ← canonical not-found (built on ErrorState)
   └─ ObjectPageError      ← canonical error    (built on ErrorState)
```

`useObjectQuery` **wraps** `useQuery` — react-query is preserved, not replaced.
The state components render through the existing `ErrorState` primitive (which
gained one optional `action` slot) and the existing `Skeleton`. Nothing here
introduces a new color, shadow, typography ramp, card, or motion.

### 3. `useObjectQuery` contract

```js
const { object, loading, notFound, isError, error, refetch, query } =
  useObjectQuery(queryKey, queryFn, options);
```

Classification is mutually exclusive, evaluated in order:

| State      | Condition                                                              |
|------------|-----------------------------------------------------------------------|
| `loading`  | `query.isPending` — no resolved result yet (incl. a disabled query)   |
| `notFound` | HTTP 404, API `not_found` code, **or** a resolved-but-null object     |
| `isError`  | settled with an error that is not a not-found                         |
| `object`   | the resolved object (null in every non-success state)                 |

Key decision: `loading` keys off **`isPending`, not `isLoading`**. A query that
is still disabled (`enabled: Boolean(id)` before an id arrives) reports
`isLoading === false` while holding no data — the old code would have rendered
its success body against an `undefined` object. `isPending` stays true until a
real result exists, so a page never touches object fields before they exist.
`object` is `null` in loading / not-found / error, so pages guard their states
before dereferencing.

### 4. State components

- **`ObjectPageSkeleton({ hasRail })`** — mirrors the object-page geometry
  (kicker + title + hero, two-column main, optional metadata rail) so data
  landing doesn't reflow the page. `role="status"`, `aria-busy`, a single
  `sr-only "Loading…"`; all geometry is `aria-hidden`. No full-page spinner.
- **`ObjectNotFound({ objectLabel, identifier, backTo, backLabel })`** — states
  the object type, the identifier when available, that it wasn't found, and the
  way back. **No Retry** (retrying a not-found can't help). Never leaks tenant
  or security detail — a cross-tenant object reads identically to a deleted one
  ("…doesn't exist, or you may not have access to it").
- **`ObjectPageError({ objectLabel, error, onRetry, backTo, backLabel })`** —
  distinct title ("Couldn't load this X"), **Retry** plus contextual back nav.
  Message comes only from `errorMessage()` — never a raw backend error.

Contextual back navigation is a shared internal `StateBackLink` (renders nothing
without a target), so not-found and error stay consistent.

### 5. `errorMessage` behavior

`errorMessage(error, fallback?)` returns safe, operator-facing copy and **never**
echoes a raw error object, stack, SQL, gateway code, or status number:

| Input                                   | Output                                             |
|-----------------------------------------|----------------------------------------------------|
| 401 / 403                               | "You don't have permission to view this."          |
| status ≥ 500 (even if a message leaked) | generic fallback (internal detail suppressed)      |
| known API `{error:{message}}` (4xx)     | that message, trimmed (operator-written copy)      |
| network failure (no response)           | generic fallback                                    |
| unknown / no envelope                   | generic fallback                                    |

Detection is centralized in `httpStatus()` / `apiError()` — no page hand-rolls
`error.response.status`. These key off the app's **actual** shapes: the axios
instance attaches the server response unchanged (no response interceptor), and
the API envelope is `{ error: { code, message } }`; a network failure has no
`response`.

### 6. Pages migrated (15)

Canonical financial objects first, then the remaining standard object pages:

| Page | objectLabel | back |
|---|---|---|
| InvoicePage | invoice | /invoices |
| PaymentPage | payment | /payments |
| JournalEntryPage | journal entry | /ledger |
| ReconciliationRunPage | reconciliation run | /finance/reconciliation |
| SubscriptionPage | subscription | /subscriptions |
| CustomerPage | customer | /customers |
| PlanPage | plan | /plans |
| MeterPage | meter | /metering |
| CreditNotePage | credit note | /credit-notes |
| DisputePage | dispute | /disputes |
| CouponPage | coupon | /coupons |
| QuotePage | quote | /quotes |
| DunningCampaignPage | campaign | /dunning/campaigns |
| CancelFlowPage | cancel flow | /cancel-flows |
| WalletPage | wallet | /wallets |

Not a blind codemod — each page was migrated by hand and its domain specifics
preserved (see §15). Secondary queries stay on plain `useQuery`
(ReconciliationRun's team members, CancelFlow's stats, Wallet's transactions);
only the **primary object GET** moved to `useObjectQuery`.

### 7. Before / after

**Before:** each page hand-rolled `if (isLoading) return <Skeleton/>` and a
combined `if (error || !obj) return <ErrorState .../>`, with per-page 404 copy
and per-page status sniffing. A missing object and a server failure looked the
same, and the copy varied page to page.

**After:**

```jsx
const { object: invoice, loading, notFound, isError, error, refetch } =
  useObjectQuery(["invoice", id], () => endpoints.getInvoice(id).then(r => r.data.data),
                 { enabled: Boolean(id) });

if (loading)  return <ObjectPageSkeleton />;
if (notFound) return <ObjectNotFound objectLabel="invoice" identifier={id?.slice(0,8)}
                                     backTo="/invoices" backLabel="Invoices" />;
if (isError)  return <ObjectPageError objectLabel="invoice" error={error}
                                      onRetry={refetch} backTo="/invoices" backLabel="Invoices" />;
```

One vocabulary, one set of components, identical semantics on every page.

### 8. 404 root cause + fix

**Root cause.** There is no axios response interceptor, so a rejected request
carries the raw response, and the queryClient retries once. The per-page guard
`if (error || !obj)` collapsed *three distinct outcomes* — a resolved-null
object, an HTTP 404, and a genuine 5xx/network failure — into one generic error
branch. A real 404 (and any endpoint that unwrapped a missing object to `null`)
therefore rendered "Couldn't load this…" instead of a not-found.

**Fix — at the shared abstraction, not per page.** `isNotFound({error, data,
resolved})` treats a request as not-found when the HTTP status is 404, the
envelope code is `not_found`, **or** the query resolved successfully to a
null/undefined object. `useObjectQuery` routes that to `ObjectNotFound`; only a
non-not-found error reaches `ObjectPageError`. A genuine server/network failure
still renders the error state with Retry — the two are never conflated again.

### 9. Tests

New, deterministic, no weakening or skips:

- **`src/lib/__tests__/httpError.test.js`** — `httpStatus`/`apiError`/`isNotFound`
  (incl. resolved-null and resolved-undefined) and `errorMessage`
  (404 / 400-with-message / 401-403 / 500-with-leaked-message / network /
  unknown / null-object).
- **`src/lib/__tests__/useObjectQuery.test.jsx`** — loading → resolved object;
  resolved-null → notFound (not error); real 404 → notFound; 500 → isError;
  refetch exposed.
- **`src/components/patterns/__tests__/ObjectStates.test.jsx`** — skeleton
  `role="status"` + `sr-only`; not-found heading + back + **no** retry; error
  Retry + safe message + preserves a known 4xx message.

Migrated-page tests updated: seven pages (Coupon, Dispute, Quote, CreditNote,
CancelFlow, DunningCampaign, Wallet) had "missing object" tests that mocked an
HTTP **404** yet asserted the old generic *error* copy — i.e. the assertions
encoded the bug. They now assert the canonical not-found copy. This is a
correction, not a weakening: the same 404 input, the correct expected output.

### 10. Lint / build / CI

Run from `frontend/`:

- `npm run lint` → clean (0 problems).
- `npm run build` → clean.
- `npx vitest run` → **703 passed (703)**, 0 skipped.

Fixes made to reach green: removed now-unused `useQuery` imports (Coupon,
DunningCampaign, Quote), named the test-wrapper component to satisfy
`react/display-name`.

### 11. Accessibility QA

- Loading announced once via `role="status"` + single `sr-only "Loading…"`; the
  skeleton geometry is `aria-hidden` so it isn't announced shape-by-shape.
- Error uses `ErrorState`'s `role="alert"` live region.
- Not-found and error each present a clear `<h?>`-level heading.
- Retry is a real `<button>` (focusable, keyboard-activatable); back is a real
  link.
- No meaning is carried by color alone — every state has an icon + heading +
  text.

### 12. Responsive QA

`ObjectPageSkeleton` uses the same `grid-cols-1 lg:grid-cols-3` as the real
layout (collapses to one column on narrow). Not-found/error are centered
`flex flex-col items-center` with a `flex flex-wrap` action row and no
fixed-width children, so Retry + back wrap rather than overflow. Verified on the
desktop render (see §13); narrow safety holds by construction (no fixed widths,
wrapping action row).

### 13. Live visual QA

Rendered the three canonical states in the real design system (real
`ErrorState`/`Skeleton`, real tokens) via a throwaway Vite entry, at desktop
width, then removed the harness. Confirmed:

- **Loading** — skeleton reproduces the object-page geometry (kicker/title/hero
  + two-column main + rail); no full-page spinner.
- **Not found** — "Invoice not found", identifier `(7f3a9c2b)` shown, single
  "Invoices" back link, **no Retry button**.
- **Error (500)** — "Couldn't load this invoice", "Something went wrong on our
  end. Please try again.", **Retry + Invoices** back link. The injected raw
  `pq: deadlock detected` is **not** shown — the safe generic copy replaces it.
- **Error (network)** — generic safe fallback, Retry + back.

The full state machine (404 → not-found, resolved-null → not-found, 500 →
error+Retry, 4xx-message preserved, network → generic) is proven deterministically
by the §9 tests. **Remaining:** end-to-end QA of the real pages at deliberately
invalid IDs on the deployed build (Invoice / Subscription / Payment / Journal
Entry → not-found; a forced server failure → error+Retry) — a credentialed
click-through best done against app.recurso.dev after merge.

### 14. Bugs found + fixed

1. **Disabled-query undefined leak (real robustness bug).** Keying `loading` off
   `isLoading` meant a page rendered before its route id resolved (disabled
   query) would fall through to the success body with an `undefined` object and
   crash (`Cannot read properties of undefined`). Fixed by keying off
   `isPending`. Surfaced by the PageSmoke guard.
2. **The 404-copy bug** — resolved-null / 404 rendered generic error. Fixed at
   the abstraction (§8).
3. **`SubscriptionPage` identifier collision** — the page already had a
   `const [loading]` busy-state; destructured `loading: objectLoading` to avoid
   a redeclare parse error.
4. **Test assertions encoding the old bug** — corrected (§9).

### 15. Domain-specific states preserved

Migration did not flatten page-specific behavior:

- **ReconciliationRunPage** keeps its own "no discrepancies" domain empty state
  (a *resolved* run with zero discrepancies ≠ a run that wasn't found) and its
  secondary team-members query.
- **WalletPage** keeps its combined `walletLoading || txsLoading` gate so the
  page doesn't render half-loaded.
- **CancelFlowPage** keeps its best-effort stats query and "no sessions yet"
  copy; **DunningCampaignPage / CancelFlowPage** keep their "no steps yet"
  `RelatedEmpty`.
- **CouponPage** keeps its "no subscriptions using this coupon yet" related
  empty. All page-level "empty related list" states are untouched — only the
  object's own loading/not-found/error moved.

### 16. Findings deferred / remaining state inconsistencies

- **AccountPage** — not migrated. It has a non-standard structure (raw `<table>`,
  no single canonical object GET matching the pattern); forcing it onto the
  abstraction would be a bad fit. Flagged for a later batch (table/structure
  work), not Batch B.
- **List/index pages** keep their existing list-level loading/empty/error
  patterns — Batch B is the *object* lifecycle only; list-state consolidation is
  a separate concern.
- **Deployed end-to-end not-found/error click-through** (§13) remains as the
  post-merge verification step.

---

**Stop.** Batch B is complete and self-contained. Not starting Batch C, D, E, or
F. Awaiting review.
