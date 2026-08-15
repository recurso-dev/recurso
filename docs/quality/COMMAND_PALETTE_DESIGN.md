# Command Palette (⌘K) — Object Search Design

> **Design + investigation doc, no production code.** Turns ⌘K from a route
> launcher into a Recurso object-navigation entry point. Every claim cites the
> real file/endpoint; backend capabilities are verified, not assumed.
> Scope of the *first* implementation: **Customers, Plans, Subscriptions** object
> search (the three the backend can search server-side today), on top of the
> existing route / create / help destinations. Invoices and Payments are
> **BACKEND GAPs** and are explicitly deferred, not faked.

---

# Current Architecture

- **Component:** `frontend/src/components/ui/command-palette.jsx`. A Radix
  `Dialog` + a hand-rolled combobox: a text `input` with
  `role="combobox" / aria-controls / aria-activedescendant / aria-autocomplete`,
  and a `role="listbox"` of `role="option"` buttons. DOM focus stays on the input
  while arrow keys move the highlighted option (the standard listbox pattern) —
  **a11y is already correct**.
- **Mount + trigger:** `components/layout/DashboardLayout.jsx:193` renders
  `<CommandPalette open onOpenChange>`; the ⌘K/Ctrl-K handler is at
  `DashboardLayout.jsx:50` (`(e.metaKey || e.ctrlKey) && e.key === "k"`).
- **Data today:** a static `DESTINATIONS` array (`command-palette.jsx:14-30`) =
  `ALL_DESTINATIONS` from the canonical `lib/navigation.js` (so the palette can't
  drift from the sidebar) + static "Create" + "Help" entries. Filtering is a
  client-side substring match over `label` (`:39-43`).
- **Keyboard:** ArrowDown/Up clamp the active index, Enter activates
  (`:70-81`); active option `scrollIntoView` (`:55-59`); Esc closes via Dialog.
  Screen-reader result count via `aria-live` (`:110-113`).
- **Navigation:** `go(item)` (`:61-68`) `navigate(item.to)` for routes or
  `window.open(item.href)` for external help links. No palette-only pages.

**Verdict:** the shell (dialog, combobox/listbox semantics, keyboard, grouping,
navigation) is a strong, reusable foundation. What's missing is *object* search.

# Current Limitations

- **Objects are invisible.** Searching "Initech" returns "Nothing matches" — the
  palette only knows route labels. Verified live in the Batch-1 audit.
- **No async data.** `results` is a pure `useMemo` over a static array
  (`:39-43`) — no react-query, no debounce, no loading/error states, no
  cancellation.
- **Flat ranking.** Substring match on `label` only; no notion of object
  relevance, recency, or exact-vs-prefix.
- **No result identity model.** Every option is `icon + label (+ external
  icon)` — there is no "type / who / state" structure for rich object results.

---

# Object Search Matrix

Server-side search verified against `cmd/api/openapi.yaml` + handlers +
repositories. "Searchable fields" = the SQL `ILIKE` columns.

| Object | Search API | Searchable fields | Result fields available | Canonical URL | Backend gap |
|---|---|---|---|---|---|
| **Customers** | `GET /v1/customers?q=` (`handler/customer.go:380`; SQL `db/customer_repository.go:271-272`) | `name`, `email` | id, **name**, email, active | `/customers/:id` (`App.jsx:170`) | none |
| **Plans** | `GET /v1/plans?q=` (`handler/plan.go:157`; SQL `db/plan_repository.go:140-141`) | `name`, `code` | id, **name**, code, interval, amount, currency | `/plans/:id` (`App.jsx:173`) | none |
| **Subscriptions** | `GET /v1/subscriptions?q=` (`handler/subscription.go:103`; SQL `db/subscription_repository.go:235-236`) | joined `customer.name`, `customer.email`, `subscription.id::text` — **NOT plan name** | id, customer_id, plan_id, **status** — **no names** (resolve via caches) | `/subscriptions/:id` (`App.jsx:176`) | (a) search misses plan name; (b) response carries no customer/plan **names** → needs the id→name caches or denormalization |
| **Invoices** | **none** — `GET /v1/invoices` accepts only `customer_id` / `subscription_id` (`handler/subscription.go:363`; openapi `:1990-2005`) | — | invoice_number, total, currency, status, amount_due (rich rows) | `/invoices/:id` (`App.jsx:178`) | **GAP: no text/`q` search** (frontend searches invoices client-side over a page-through today) |
| **Payments** | **none** — `GET /v1/payment-attempts` accepts only `status`/`page`/`per_page` (`handler/subscription.go:334`) | — | amount, currency, status, gateway, invoice_number, failure_code | **none** — no `/payments/:id` route exists | **DOUBLE GAP: no search AND no addressable object** (rows link to the invoice today, `Payments.jsx:180`) |
| Quotes *(future)* | `GET /v1/quotes?search=` (`handler/quote.go:68`; SQL `db/quote_repository.go:169-170`) | `quote_number`, `notes` | quote_number, status, customer_id | `/quotes/:id` (`App.jsx:212`) | param is `search`, not `q` (naming inconsistency) |

**No global `/v1/search` endpoint exists** (verified: zero `/search` routes in
`main.go`/`openapi.yaml`). Search must be a **per-object fan-out**.

**None of the searchable list repos return a total count for the filtered query**
(they `LIMIT/OFFSET` a slice, no `COUNT`), so per-group counts in the palette are
"first N", never "N of M".

**First-implementation set = Customers + Plans + Subscriptions** (searchable +
addressable). Invoices and Payments are deferred to their backend gaps. Quotes is
the cheapest future add.

---

# UX Model

Mental model: **⌘K → search Recurso → recognize the object → see its state →
open it.** The palette stays the calm, dense, keyboard-first surface it is today;
object results slot in as new groups above the existing navigation.

**Empty query (palette just opened):**
```
Search Recurso…

RECENT              (only if Recent Objects ships — see decision)
  Invoice  INV-000009 · Initech Systems      $99.00 · Past due
  Customer Acme Corporation

GO TO               (existing nav destinations)
CREATE              (existing)
HELP                (existing)
```

**With a query ("initech", debounced):**
```
Search Recurso…                                       initech

CUSTOMERS
  Initech Systems                          jane@initech.com
SUBSCRIPTIONS
  Growth · Initech Systems                          Active
GO TO
  (any nav label matching "initech" — usually none)
```

Group order is fixed and object-first: **Recent → Customers → Subscriptions →
Plans → (Invoices/Payments when unblocked) → Go to → Create → Help.** Customers
and subscriptions lead because "find who / find their subscription" is the most
common operator entry point. Each group shows at most **6** results; a group that
hit the cap shows a muted "Search all in {Customers}" row that deep-links to the
list page pre-filtered by the query (reuses the list pages' `?q=` from Batch 1).

Illustrative only — the real hierarchy is defined below from the actual model.

# Result Information Hierarchy

Every result answers **What / Who / State** and is visually type-distinct (icon +
group). Reuse existing primitives: `StatusBadge` (`ui/status-badge.jsx`, the only
sanctioned status renderer) and `<Money>` (`ui/money.jsx`, exponent-aware).

| Object | Icon | Primary (identity) | Secondary (who/what) | State (right) |
|---|---|---|---|---|
| Customer | `Users` | **{name}** | {email} | active/inactive (subtle) |
| Subscription | `Repeat` | **{plan name}** | {customer name} | `<StatusBadge status>` |
| Plan | `Package` | **{name}** | {code} · {amount}/{interval} | — |
| Invoice *(gap)* | `FileText` | **{invoice_number}** | {customer name} | `<Money total>` · `<StatusBadge>` |
| Payment *(gap)* | `CreditCard` | **{amount}** | {customer / invoice} | `<StatusBadge>` (Failed pops) |

**Name resolution:** subscription search rows carry only `customer_id`/`plan_id`.
Resolve to names via the **already-loaded** shared caches `useCustomers()` /
`usePlans()` (`lib/useCustomers.js`) — display-only reuse of the existing id→name
pattern. If a cache is cold (fresh session), fall back to a short id and let it
fill in; this is a display detail, not a search dependency. (See BACKEND GAP:
denormalized names on the subscription-search response would remove this reliance.)

Never render every result identically — the icon + group + identity structure
makes the type readable at a glance.

---

# Search / Ranking Strategy

- **Minimum query length: 2.** Below that, show Recent + nav destinations only
  (no object queries) — one char is noise and floods the fan-out.
- **Debounce: ~200 ms** on the query before firing object queries. Nav/route
  filtering stays instant (it's local).
- **Server-side relevance** comes from the repos' `ILIKE '%q%'` ordering; on top,
  the client applies a light **prefix boost** (results whose primary field
  *starts* with the query rank above mid-string matches) — cheap, in-memory, over
  ≤6 rows per group.
- **De-duplication:** results are grouped by object type and keyed by canonical
  URL; a Recent entry that also appears in live results is shown once (Recent
  wins its slot; it's removed from the live group).
- **Per-group cap: 6**, with a "Search all in {group}" deep-link when capped.

# API Architecture

**Per-object fan-out with react-query — NOT "download everything and filter".**

- One react-query per searchable type, keyed by the debounced query:
  `["palette","customers",q]`, `["palette","plans",q]`, `["palette","subscriptions",q]`,
  each calling the existing getter with a small limit, e.g.
  `getCustomers({ q, limit: 6 })` / `getPlans({ q, limit: 6 })` /
  `getSubscriptions({ q, limit: 6 })` (`lib/api.js:119/125/140`).
- **`enabled: q.length >= 2`** so no request fires for short/empty queries.
- **Parallel** (three independent queries) — total latency ≈ the slowest single
  request, not the sum.
- **Caching / SWR:** react-query caches per `(type,q)` with a short `staleTime`
  (~30 s) so re-typing a prior query is instant and re-opening the palette within
  the window doesn't refire. `placeholderData: keepPreviousData` keeps the last
  results visible while the next query loads (no flicker to empty).
- **Cancellation:** react-query marks the previous `(type,q)` query inactive on
  key change; to actually **abort** the in-flight HTTP request, thread the
  queryFn's `AbortSignal` into axios (`api.get(url,{ params, signal })`). The
  axios client (`lib/api.js`) does not pass a signal today → a small **frontend**
  change (not a backend gap) needed for true cancellation.
- **Why 3, not 5:** Invoices and Payments have no search endpoint, so the fan-out
  is only three requests, each capped at 6 rows — bounded and fast. "Do not make
  five slow requests per keystroke" is satisfied by debounce + min-length +
  three-type fan-out + cancellation.

The existing full-set caches (`useCustomers`/`usePlans`/`useSubscriptions` fetch
`limit:1000`) are **display-resolution** infrastructure and are NOT the search
path — the palette must not lean on them for search (they don't scale past 1000
and are the very "download everything" pattern to avoid).

---

# Backend Gaps

Documented, not faked. Each blocks a concrete palette capability.

**GAP-1 · Invoice text search — MISSING.**
- Endpoint: `GET /v1/invoices` (`handler/subscription.go:363`).
- Required capability: a `q` param matching `invoice_number` (and ideally
  amount/status).
- Proposed query: `GET /v1/invoices?q=INV-000009&limit=6`.
- Pagination/limit: honor `limit` (cap ~50); return newest-first.
- Response shape: existing invoice rows already carry invoice_number/total/
  currency/status — sufficient for rich results; add `pagination.total` for a
  "Search all" count.
- Authorization: same tenant scoping as the list endpoint (session/API key →
  tenant filter). No new surface.
- Frontend dependency: add an `Invoices` group + `getInvoices({q,limit})`.

**GAP-2 · Payment search + addressable Payment object — MISSING (double).**
- Endpoints: `GET /v1/payment-attempts` (no `q`), and there is **no
  `GET /v1/payment-attempts/{id}` and no `/payments/:id` route**.
- Required: (a) `q` on the attempts list (match invoice_number / gateway ref /
  failure_code); (b) a single-attempt read + a `/payments/:id` object page (the
  Payment-object gap from the dashboard audit).
- Proposed: `GET /v1/payment-attempts?q=…&limit=6`; `GET /v1/payment-attempts/{id}`.
- Response shape: attempt rows carry amount/currency/status/invoice_number/
  failure_code (enough for results once search exists).
- Frontend dependency: a `Payments` group **and** a canonical URL to link to;
  until the object page exists, a payment result has nowhere canonical to go.
- **Deferred until the Payment object ships** (tracked in the dashboard audit).

**GAP-3 · Denormalized names on subscription search — nice-to-have.**
- `GET /v1/subscriptions` rows carry `customer_id`/`plan_id` but no names
  (`db/subscription_repository.go:28`). The palette resolves via caches today.
- Proposed: include `customer_name` + `plan_name` on the row (or a
  `?expand=customer,plan`) so subscription results are self-contained and don't
  depend on the 1000-row customer cache being warm.
- Also: subscription search matches customer name/email/id but **not plan name**
  (`:236`); matching plan name would let "search Pro" surface Pro subscriptions.

**GAP-4 · Global search / unified `q` — optional, for scale.**
- No `GET /v1/search?q=`. Per-object fan-out is fine at current scale; a single
  ranked endpoint would reduce round-trips and enable cross-type relevance later.
- Also unify param naming: Quotes uses `search`, everything else uses `q`
  (`handler/quote.go:68`).

**No total count on filtered searches** (all searchable repos) → the palette can
show "first N" and a "Search all" deep-link, not "N of M".

---

# Failure Model

Because each object type is its own react-query, **failure is per-group**:

- Customers succeeds, Subscriptions fails → the Subscriptions group renders a
  quiet inline row ("Couldn't search subscriptions — retry") while Customers +
  the rest stay fully usable. The palette **never** blanks on one failure.
- A permission error (403) on a group is treated like any other group failure:
  that group shows an inline "You don't have access to subscriptions" (from the
  canonical error envelope's message) and the others work. (Today all `/v1`
  lists are tenant-scoped and available to any authenticated session; if RBAC
  narrows a resource later, the per-group model already degrades correctly.)
- The static Go-to / Create / Help groups never depend on the network, so the
  palette is always at least a route launcher.

# Performance Model

Targets: feels instantaneous; no request storm.

- **Debounce 200 ms + min length 2** → keystrokes don't each fire a fan-out.
- **≤3 parallel requests, `limit:6` each** → small, bounded payloads.
- **AbortSignal cancellation** → stale in-flight requests are aborted on the next
  keystroke; results can't arrive out of order and clobber newer ones (react-query
  also drops inactive-key results).
- **keepPreviousData + staleTime 30 s** → no flash-to-empty; re-typed queries are
  cache hits.
- **Client prefix-rank over ≤18 rows** → negligible.
- Route/create/help filtering stays a synchronous `useMemo` (unchanged).

# Accessibility

Extend the existing (correct) pattern; add nothing bespoke.

- Keep the `role="combobox"` input + `role="listbox"` + `role="option"` +
  `aria-activedescendant` model. Flatten all result groups into one ordered option
  list for arrow-key traversal; group headers are non-interactive `role="presentation"`
  labels (not focus stops).
- **Esc** closes (Dialog), **Enter** opens the active result, **Arrow Up/Down**
  move across groups, wrapping is not required (clamp, as today).
- `aria-live` announces the total result count and transitions to "Searching…" /
  "No matches" so a screen-reader user hears async state.
- Visible focus: the active option keeps `ring-2 ring-inset ring-ring`.
- **Reduced motion:** the Dialog's open/close animation is `tailwindcss-animate`,
  already neutralized under `prefers-reduced-motion` globally (`index.css`) — no
  per-result animation is added.

# Navigation

Every object result calls `navigate(canonicalUrl)` to the **existing** object
page: `/customers/:id`, `/plans/:id`, `/subscriptions/:id` (and `/invoices/:id`,
`/quotes/:id` when their groups ship). **No palette-only pages, no special
routes** — the palette is purely a faster door to the object-page architecture
already in place. The "Search all in {group}" row navigates to the list page with
`?q=<query>` (the URL-state contract from Batch 1), so it lands pre-filtered.

# Recent Objects — Decision

**Recommendation: ship a light Recent Objects list, but in `sessionStorage`, and
only in a second batch after core search lands.**

- **Value:** genuinely high for operators — re-opening the invoice/subscription
  you were just on is the most common repeat action. Not "for completeness".
- **Storage:** `sessionStorage` (per tab, cleared on close/logout), **not**
  `localStorage` — object references are tenant data, and `localStorage` persists
  across users on a shared machine (the same reason `authToken` is memory-only,
  `lib/authToken.js`). Store `{type, id, url, label, subtitle}` — no amounts or
  PII beyond a name already visible in the UI.
- **Recorded when:** an object page mounts (a small hook on
  Customer/Subscription/Invoice/Plan pages pushes its identity).
- **Max entries:** ~6, **dedup by url**, most-recent-first.
- **Stale/deleted:** validate lazily — a Recent result that 404s on open is
  removed silently; Recent entries are never trusted as live state (they show the
  label captured at visit time, not current status).
- **Permission changes:** if the object is no longer accessible, opening it hits
  the normal 403/empty path and the entry is pruned.
- If any of the above feels heavier than the value in review, **defer Recent
  entirely** — core object search stands on its own.

# Motion

Only what the Dialog already provides: open/close via `tailwindcss-animate`
(reduced-motion-aware). Result set changes **replace** in place — no per-item
stagger or count-up. A pending group may show a single small `Loader2` spinner
in its header. **No new motion system, no decorative animation.**

# Testing Strategy

Component tests (mock the three getters), covering:
- **Object results render** with correct identity / secondary / status / icon and
  the correct canonical `href` per type.
- **Grouping + order** (Customers/Subscriptions/Plans headers; nav groups after).
- **Debounce + min-length:** typing 1 char fires no object query; typing ≥2 fires
  after the debounce; rapid typing collapses to the last query.
- **Partial failure:** one group's query rejects → its inline error shows, the
  other groups still render results.
- **Empty / no-results / loading** states.
- **Keyboard:** arrow traversal spans groups; Enter navigates to the active
  result's URL; Esc closes; group headers are skipped.
- **Query cancellation / stale results:** an earlier slow query resolving after a
  newer one must not overwrite the newer results.
- **Existing behavior intact:** route / create / help destinations still filter
  and navigate.
- **Name resolution:** a subscription result shows the customer + plan name from
  the caches, and degrades to a short id when a cache is cold.
- (If Recent ships) record-on-visit, dedup, max-entries, stale-entry pruning.

# Rollout Plan

- **Batch A — core object search (frontend-only, unblocked).** Fan-out infra
  (debounced parallel react-query, `enabled` gating, `keepPreviousData`,
  AbortSignal wiring in `lib/api.js`), the Customers / Subscriptions / Plans
  groups with the result hierarchy (reusing `StatusBadge`/`Money` + the id→name
  caches), grouped keyboard traversal, per-group failure, "Search all" deep-links.
  Existing route/create/help unchanged.
- **Batch B — Recent Objects (optional).** sessionStorage recents + record-on-
  visit hook, per the decision above.
- **Backend track (parallel, unblocks more groups).** GAP-1 invoice `q` search →
  Invoices group; GAP-2 payment search + Payment object page → Payments group;
  GAP-3 denormalized subscription names + plan-name matching; GAP-4 optional
  unified `/v1/search`. Quotes group is a cheap add once we accept the `search`
  vs `q` param difference (or GAP-4 unifies it).

# Definition of Done

- ⌘K searches **Customers, Plans, Subscriptions** server-side; each result shows
  **what it is, who it belongs to, and its current state**, and opens the
  **canonical object URL**.
- The fan-out is debounced, min-length-gated, parallel, capped, cached, and
  **cancellable**; no request storm; no "download everything".
- **Partial failure** degrades per-group; the palette never blanks on one error.
- Full **keyboard** operation, **listbox/combobox** semantics, screen-reader
  announcements, Esc/Enter/arrows, visible focus, **reduced-motion** respected.
- Existing **route / create / help** navigation still works.
- Uses the existing Recurso primitives and design system; **no new design or
  motion system**.
- Backend gaps (Invoices, Payments, subscription names, global search) are
  **documented, not faked**.
- `npm run lint && npm run build && npx vitest run` green, with the tests above.

---

## Source of truth
- **Frontend:** `components/ui/command-palette.jsx`,
  `components/layout/DashboardLayout.jsx`, `lib/navigation.js`, `lib/api.js`,
  `lib/useCustomers.js`, `components/ui/{status-badge,money}.jsx`, `App.jsx`.
- **Backend search:** `internal/adapter/handler/{customer,plan,subscription,quote}.go`
  + `internal/adapter/db/{customer,plan,subscription,quote}_repository.go`;
  `cmd/api/openapi.yaml`.
- **Related:** `docs/quality/DASHBOARD_QUALITY_AUDIT.md` (⌘K + Payment-object gaps).
