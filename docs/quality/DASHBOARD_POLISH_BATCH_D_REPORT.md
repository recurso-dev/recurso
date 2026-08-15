# Dashboard Polish — Batch D Report
## Table Finish — sticky headers + accessible names

Status: **complete, awaiting review.** The 27 DataTable worklists now keep their
column header visible while scrolling long datasets and every one has a real
accessible name. Implements the approved design (`DASHBOARD_POLISH_BATCH_D_DESIGN.md`,
PR #703). Shipped as **#704**. Contained to `DataTable` + `ui/table` + `PageHeader`
— no page-layout rewrite, no new table features, no change to pagination /
selection / filtering / sorting / DataTable behaviour; density baseline (`p-4`)
preserved. Lint, build, and the full **718**-test suite are green.

---

### 1. Table inventory + classification

| Class | Count | Examples | Batch D action |
|---|---|---|---|
| **DataTable worklists** | 27 | Invoices, Customers, Subscriptions, Payments, Credit Notes, Disputes, Coupons, Quotes, Plans, Metering, Wallets, Gifts, Referrals, Events, Audit Log, Dunning campaigns, Cancel flows, Mandates, Offline payments, Usage Explorer, Organizations, Churn… | **Sticky header + accessible name** (one change → 27) |
| **Hand-composed `ui/table` — specialized accounting/report** | ~13 | TrialBalance, RevenueRecognition, MonthEndClose, ReconciliationRunPage, FinanceReconciliation, GSTReturns, RevenueWaterfall, Collections, Dashboard, AskAnalytics, PortalDashboard, Security, ImportData | **Untouched** (already semantic; keep specialized per brief) |
| **Hand-composed `ui/table` — small settings lists** | ~5 | Team, Entities, Developers, TaxNexusSettings, EntitiesSettings | **Untouched** (not worklists; conversion unjustified) |
| **Raw `<table>`** | 4 | AccountPage, WalletPage, QuotePage, PricingSimulator | **Untouched** (accounting/line-item; primitives → Batch E) |

### 2. DataTable vs hand-rolled — decisions

- The 27 worklists all flow through `patterns/DataTable.jsx`; fixing it propagates
  to every list. **This is the whole Batch D surface.**
- The specialized accounting/report tables use `ui/table` primitives directly
  (correct `<thead>`/`<th scope=col>`), and the brief is explicit: *do not force
  accounting structures into a generic CRUD table.* They are **kept specialized**.
- The small settings lists are not operator worklists; adding DataTable machinery
  (sort/paginate/select) for a handful of rows is not justified. **Kept as-is.**
- The 4 raw `<table>`s are accounting/line-item/preview presentations; giving them
  `ui/table` primitives is **Batch E** (object-page parity) — out of scope here.

### 3. Sticky-header implementation

Root constraint (from the design investigation): the shell scrolls via
`<main overflow-y-auto>`, but two `overflow` contexts (`Card overflow-hidden`,
the `ui/table` wrapper `overflow-auto`) sit between `<thead>` and `main`, so naive
`sticky top:0` cannot pin to the page scroll, and page-scroll sticky would need a
layout rewrite (a STOP condition). **Solution (approved Option A):** because the
Card wraps *only* the table (toolbar above it, pagination below), bound the table's
own scroll wrapper so it becomes the scroller and the header sticks within it.

- **`ui/table.jsx`** — `Table` gains an optional `wrapperClassName` applied to the
  `overflow-auto` scroll container. Unset → behaviour unchanged (so specialized
  tables are unaffected).
- **`DataTable.jsx`** —
  - `wrapperClassName="max-h-[calc(100vh-15rem)]"` on the scroll wrapper: the body
    scrolls inside it; short tables don't reach the bound (no forced scroll). One
    template constant, **no per-page overrides**.
  - every header `<th>` (select-all, columns, chevron) gets
    `sticky top-0 z-20 bg-muted border-b border-border` — opaque so rows can't
    bleed through, above rows, with a travelling divider. Header row hover
    neutralised (`hover:bg-transparent`).
- **Pure CSS `position: sticky`** — no scroll listeners, observers, or per-row
  work (see §11).

**Validated on the real component** (harness with 120-row and 3-row datasets,
inside a shell replicating `header h-16` + `main overflow-y-auto`):

| Check | Result |
|---|---|
| Header pinned at wrapper top when scrolled | `thTopMinusWrapTop = 0`, opaque bg, z-20 |
| Header stays pinned while scrolled **down AND right** | `headerStillVerticallyPinned = 0` |
| Horizontal overflow contained; page never scrolls sideways | `pageHorizontalScroll = false`, `mainHorizontalScroll = false`, wrapper scrolls |
| Header tracks its body column during horizontal scroll | `headerTracksBodyColumn = true` (correct — no left pin) |
| Short table (3 rows) | no forced internal scroll (`scrollH == clientH`); pagination below; no overflow |
| Toolbar / pagination placement | outside the scroll region (above / below the Card) |

### 4. Accessible-name coverage

`DataTable` previously set **no** accessible name (audit A1). Now:
- `PageHeader`'s `<h1>` renders `id="page-title"` (via a `titleId` prop, default
  `"page-title"`).
- `DataTable` names its `<table>` with `aria-labelledby="page-title"` by default —
  referencing the page's existing visible heading (e.g. the table is announced
  "Invoices, table"). An `ariaLabel` prop is the fallback for a table that has no
  page heading (then `aria-labelledby` is omitted).
- **Coverage:** all **25** real DataTable pages render a `PageHeader`, so every
  worklist is named for free — no per-page edits, no visually-hidden duplicate
  titles, no generic "Data table" (unit-tested: a `/^data table$/i` name is
  absent).

### 5. Header semantics (verified intact — no churn)

`<thead>` present; `<th scope="col">` on every header cell; sortable headers are
real `<button>`s (keyboard-operable) with `aria-sort` on the `<th>`; the select-all
header cell keeps `aria-label="Select all rows on this page"`; per-row checkboxes
keep `aria-label="Select row …"`; the trailing chevron header stays `aria-hidden`.
No element types changed; sticky is className-only. Focused tests assert
`th scope=col`, `<thead>`, and the select-all name.

### 6. Density changes

**None to the baseline** — per the brief ("do not globally make tables smaller;
financial data needs breathing room"). `ui/table` `TableCell` stays `p-4`,
`TableHead` `h-10`; the opt-in `compact` density is unchanged. The DataTable
baseline was already consistent across the 27. The sticky header row keeps the
same height; only its background became opaque (was `bg-muted/40` tint → solid
`bg-muted`) so it can cover scrolling rows — a near-identical faint gray.

### 7. Horizontal-overflow findings

The `ui/table` wrapper already provides horizontal scroll (`overflow-auto`) with
responsive column hiding (`hideBelow` → `hidden sm/md/lg:table-cell`). Adding
`max-height` to that same element does not affect the horizontal axis. Validated:
with a forced-narrow wrapper (520px) over a 1000px table, the **wrapper** scrolls
horizontally while `main`/document do **not** (`pageHorizontalScroll = false`), and
the sticky header scrolls horizontally with the body while staying vertically
pinned. No page-level horizontal scroll introduced.

### 8. Pages affected

**Changed (3 files + 1 test):** `ui/table.jsx` (wrapperClassName),
`patterns/PageHeader.jsx` (`id`/`titleId` on `<h1>`), `patterns/DataTable.jsx`
(sticky header, bounded wrapper, accessible name), and
`__tests__/DataTableFinish.test.jsx`. **Effect:** the 27 DataTable worklist pages
(via the shared component) + every `PageHeader`. No page files edited individually.

### 9. Live visual QA

Verified on the deployed Batch D build (app.recurso.dev, post-#704). Each
DataTable page was scrolled and measured (accessible name resolution, sticky
pin, `th scope`, page horizontal scroll); the specialized tables were checked to
be **unchanged**.

| Page | Accessible name | Sticky header | th scope | Page h-scroll | Notes |
|---|---|---|---|---|---|
| **Invoices** (long) | "Invoices" | pinned (`stickyPinned:true`) | col | none | header holds over scrolled rows; sort arrows intact |
| **Subscriptions** | "Subscriptions" | pinned | col | none | `.money` signature present |
| **Customers** | "Customers" | pinned | col | none | multi-line name+email rows render cleanly |
| **Payments** | "Payments" | sticky (maxH 564px) | col | none | |
| **Credit Notes** | "Credit Notes" | sticky | col | none | |
| **Events** (usage/events) | "Events" | pinned | col | none | |
| **Reconciliation** (specialized) | — (none) | **static (unchanged)** | col | none | `wrapMaxH:none`, not sticky — scope containment proven |
| **Journal Entries / Ledger** (specialized) | — | static (unchanged) | col | none | same hand-composed `ui/table` pattern as Reconciliation |
| **Home** | — | n/a | — | none | no DataTable (failing-collection list is bespoke) — unaffected |

**Datasets:** long (Invoices, 100s of rows — internal scroll + sticky) and short
(Payments/Credit Notes — no forced scroll, pagination below) both correct.
**Horizontal overflow:** contained to the wrapper on every page; the page/`main`
never scrolls horizontally. **Scope containment:** the specialized `ui/table`
accounting tables carry `wrapMaxH:none` / `thPosition:static` / no `aria-labelledby`
— provably untouched.

### 10. Accessibility verification

- Keyboard nav + visible focus preserved (sortable `<button>`s, row links,
  checkboxes unchanged).
- `aria-sort`, checkbox semantics, `th scope=col`, `<thead>` intact.
- Accessible table name added (§4); no keyboard trap introduced (CSS-only change).
- The sticky header does **not** obscure a keyboard-focused row — verified live:
  after scrolling down and focusing a far-above row, the browser's focus scroll
  lands it at top 269 vs the sticky-header bottom 265 (`obscured: false`). Chrome's
  native scroll-into-view respects the sticky header, so no `scroll-padding` is
  needed.
- Contrast retained (header uses the existing muted tint, now opaque).

### 11. Performance observations

Zero runtime cost: sticky is `position: sticky` (CSS), the only structural change
is one `max-height` class on an existing wrapper. **No** scroll listeners, **no**
IntersectionObserver/ResizeObserver, **no** per-row React work, **no** DOM rewrite,
**no** layout thrashing. Rendering work per row is unchanged.

### 12. Tests / lint / build / CI

- `npm run lint` → clean.
- `npm run build` → clean.
- `npx vitest run` → **718 passed (718)**, 0 skipped (711 + 7 new Batch D tests;
  all 28 existing DataTable tests still pass — pagination/selection intact).
- New `DataTableFinish.test.jsx`: accessible name via `aria-labelledby`,
  `ariaLabel` override, no generic name, bounded scroll wrapper, sticky+opaque
  header cells, `th scope=col`/`thead`, select-all checkbox name.

### 13. Bugs found / fixed

- No product/functional bug surfaced (interaction-only change). A transient
  "row peeking above the sticky header" seen mid-scroll was verified to be a
  scroll-animation artifact, not a gap (`thTopMinusWrapTop = 0` when settled).

### 14. Intentionally deferred capabilities

- **Sticky on the specialized accounting/report tables** — kept non-sticky
  (card-framed, mostly short); would need per-page height work. Deferred.
- **Raw `<table>` → `ui/table` primitives** (AccountPage/Wallet/Quote/
  PricingSimulator) → Batch E.
- **`moneyColumn`/`rowHref` sweep on Payments/Subscriptions** (hand-rolled money
  cells, audit A5) → Batch E (Payments cleanup), not a table-finish change.
- **Trailing-chevron / loose-row de-noise** → Batch F interaction.
- **Column resize/reorder, saved views, virtualization, configurable columns,
  infinite scroll, spreadsheet behaviour** → explicitly out of scope (separate
  product decisions).
- **Backend-gated columns** — none fabricated; none required for Batch D.

---

**Stop.** Batch D is complete and self-contained. Not starting Batch E. No new
full-dashboard audit.
