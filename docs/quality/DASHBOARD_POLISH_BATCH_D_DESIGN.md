# Dashboard Polish — Batch D Design
## Table Finish — INVESTIGATION (no production code)

> **Read-only, code-cited + live-validated, point-in-time (2026-08-15), against
> `main` @ `e5f34412`.** No production code was written for this document. This is
> the Batch D investigation deliverable; it ends with a STOP for approval.
>
> Scope: interaction + information-density on tables — **not** a redesign. Do NOT
> add column resize/reorder/saved-views/virtualization/spreadsheet behaviour,
> configurable columns, infinite scroll, drag-drop, or a new filtering/pagination/
> selection architecture. Do not modify `<Money>`. Do not solve missing backend
> data (document it).

---

## 1. Table inventory + classification

Recurso has **three** table substrates. "Do not assume every table should be
converted" — so each is classified by what it *is*, not by CSS similarity.

### Class 1 — DataTable-based worklists (**27 pages**) → the Batch D target
Every reusable list flows through `patterns/DataTable.jsx` (which composes
`ui/table.jsx`): Invoices, Customers, Subscriptions, Payments, Credit Notes,
Disputes, Coupons, Quotes, Plans, Metering, Wallets, Gifts, Referrals, Events,
Audit Log, Dunning campaigns, Cancel flows, Mandates, Offline payments, Usage
Explorer, Organizations, Churn, and more. These are genuine operator worklists
(sort / paginate / select / row-nav). **All sticky-header + accessible-name work
lands here — one change, 27 surfaces.**

### Class 2 — Hand-composed via `ui/table` primitives (~19 pages)
These import `Table/TableHead/TableRow/TableCell` directly (so they already have
correct `<thead>`/`<th scope="col">` semantics) but are **not** DataTable
worklists. Two sub-kinds:

- **Specialized accounting / report presentations — KEEP specialized:**
  `TrialBalance`, `RevenueRecognition`, `MonthEndClose`, `ReconciliationRunPage`,
  `FinanceReconciliation`, `GSTReturns`, `RevenueWaterfall`, `Collections`,
  `Dashboard`, `AskAnalytics`, `PortalDashboard`, `Security`, `ImportData`. These
  are debit/credit/balance/reconciliation/report structures. Per the brief, do
  **not** force them into a generic CRUD DataTable. They mostly render short,
  card-framed tables; sticky headers add little and would need per-page height
  work. **Left as-is** (they already get correct header semantics + the Batch C
  overline column labels).
- **Small settings/admin lists — KEEP as-is:** `Team`, `Entities`, `Developers`,
  `settings/TaxNexusSettings`, `settings/EntitiesSettings`. Small, bounded, not
  operator worklists needing sort/paginate/select. Converting them is not
  justified (would add DataTable machinery for a handful of rows).

### Class 3 — Raw `<table>` (4) → not Batch D
`AccountPage` (journal activity), `WalletPage` (transactions), `QuotePage` (line
items), `slide-overs/PricingSimulator` (pricing preview). These are hand-rolled
`<table>` **not** using `ui/table` primitives. AccountPage/Wallet are accounting
presentations; QuotePage/PricingSimulator are line-item/preview tables — all
specialized. Giving them semantic `ui/table` primitives is **Batch E**
(object-page parity; the audit already files AccountPage→primitives there). Batch
D leaves their structure alone (they received the canonical header colour in
Batch C).

---

## 2. Current-state audit (the six objectives)

### 2.1 Sticky headers — feasibility (the crux)

**Scroll model (code + live-verified).** The app shell is a fixed flex column:
`<header h-16>` (64px, non-scrolling) + `<main overflow-y-auto>` — **`main` is the
vertical scroller.** A DataTable nests its table as:

```
main(overflow-y:auto) › … › DataTable root › Card(overflow:hidden) › div.overflow-auto › <table>
                                                  toolbar ABOVE the Card,  pagination BELOW the Card
```

Two `overflow` contexts (`Card overflow-hidden`, `div overflow-auto`) sit between
`<thead>` and `main`. Consequence (a real CSS constraint): a naive
`position: sticky; top:0` on `<thead>` **cannot** stick to the page/`main` scroll —
it sticks to the nearest `overflow` ancestor, which has no bounded height and
scrolls away with the page. And `overflow-x:auto` (needed for horizontal scroll)
forces `overflow-y:auto`, so you cannot make that wrapper "horizontal-only." So
**page-scroll sticky is not achievable without layout-infrastructure changes**
(a STOP condition).

**The contained solution (LIVE-VALIDATED on the deployed Invoices table).** Because
the toolbar sits *above* the Card and pagination *below* it, the Card wraps **only
the table**. Giving the table's scroll wrapper a `max-height` turns it into the
vertical scroller; `position: sticky; top:0` on the header then pins within it —
toolbar and pagination stay put, page layout untouched. Empirically verified:

```
maxHeight 55vh → wrapper clientHeight 442px over scrollHeight 16874px (~10k rows);
after scrolling 400px internally: header pinned, theadTopVsWrapTop = 0, stuck = true.
```

Screenshot confirmed the header sits cleanly over rows with sort arrows intact and
no page reflow. **This is contained to `DataTable` + `ui/table`, uses only
`position: sticky` + existing tokens, and does not rewrite layout.**

- **Height strategy (decision §5.1):** a viewport-relative `max-h-[calc(100vh −
  Npx)]` template constant on the scroll wrapper — self-adjusting (short tables
  don't scroll, long ones do). Because the 27 DataTables share the same page
  template (PageHeader + toolbar + pagination), one tuned constant fits all — a
  per-*template* value, **not** per-page bespoke CSS. (A flex height-chain would
  be more "correct" but requires converting every list page to a flex-fill layout
  — layout infrastructure, out of scope.)
- **Cover / z-index:** sticky `<thead>` cells need an opaque background (the card
  `bg-muted/40` header tint) + `z-index` above rows, and the header's bottom
  border must travel with it (use a box-shadow/`border-b` on the sticky cells) so
  no 1px row-bleed shows above it (a detail seen in the live test).

### 2.2 Accessible names — **the real gap**
`DataTable` sets **no accessible name** on its `<table>` (no `aria-label`, and
`ui/table`'s `TableCaption` is exported but unused) — every list is anonymous to a
screen reader (audit A1/#6). Header semantics are otherwise good (below). **Fix:**
give the table an accessible name (decision §5.2), preferring the page's existing
visible `<h1>` via `aria-labelledby`, `aria-label` only as fallback. Applied to
the 27 DataTables.

### 2.3 Header semantics — already sound (verify, don't churn)
Live-verified on the deployed table: `<thead>` present, `<th scope="col">`,
sortable headers are real `<button>`s (keyboard-operable, `aria-sort` on the
`<th>`), the select-all header cell has `aria-label="Select all rows on this
page"`, per-row checkboxes `aria-label="Select row …"`. **No semantic regression
to fix here** — Batch D verifies and preserves these; the only header-level gap is
the accessible *name* (2.2). The trailing chevron header is `aria-hidden` (good).

### 2.4 Density — baseline is `p-4` comfortable; keep breathing room
`ui/table` `TableCell` = `p-4`; `TableHead` = `h-10`; DataTable `compact` density
= `[&_td]:py-2 [&_th]:h-9` (exists, opt-in). The DataTable baseline is **already
consistent** across the 27 (all share TableCell). Per the brief — "do NOT globally
make tables smaller; financial data needs breathing room" — Batch D **does not
shrink** the default. Scope here is *consistency verification*, not a density
redesign. (The audit's "loose rows / trailing-affordance noise" is a Batch F
interaction concern, not Batch D.)

### 2.5 Money / numeric alignment — mostly good; document gaps
DataTable supports `align: "right"` per column and a `moneyColumn` helper that
bundles right-align + `<Money>` + `tabular-nums`. Money columns that use it are
consistent and jitter-free (live-verified: amounts right-aligned, tabular). **Gap
(document, likely Batch E):** Payments / Subscriptions lists hand-roll some money
cells instead of `moneyColumn` (audit A5). Batch D will *verify* alignment on the
QA pages and **document** any hand-rolled money cell rather than restructure those
lists (that's Payments-cleanup / Batch E). `<Money>` itself is untouched.

### 2.6 Horizontal overflow — present; must survive the sticky change
The `ui/table` wrapper `div.overflow-auto` already provides horizontal scroll;
columns hide responsively (`hideBelow` → `hidden sm/md/lg:table-cell`). Live-
verified: no unexpected page-level horizontal scroll. The sticky change adds a
`max-height` to that same wrapper — horizontal scroll is preserved (same element),
and sticky must be validated to behave during horizontal scroll (the header should
scroll horizontally with the body; `position: sticky; top:0` does not pin
horizontally, which is correct).

### 2.7 Loading / empty / error / selection / pagination (verify intact)
DataTable already renders first-class `error` / `loading` (`TableSkeleton`) /
`empty` vs no-results states, page-scoped bulk selection with pruning, and exact
"page / total" pagination — all **outside** the scroll region, so the sticky
change does not touch them. Batch D must confirm they still render correctly with
the bounded-height wrapper (e.g., the skeleton/empty states short-circuit before
`<Table>`, so they are unaffected).

---

## 3. What Batch D will change (proposed)

1. **`ui/table.jsx` + `DataTable.jsx` — sticky headers** for the 27 worklists:
   bounded `max-height` scroll wrapper + sticky `<thead>` cells (opaque header bg,
   `z-index`, travelling bottom border). Contained; validated; toolbar/pagination
   unaffected.
2. **`DataTable.jsx` — accessible name:** add an accessible-name mechanism
   (decision §5.2) and thread it on the 27 pages, preferring `aria-labelledby` the
   page `<h1>`.
3. **Verification passes** (no churn): header semantics (2.3), density consistency
   (2.4), money alignment (2.5), horizontal overflow (2.6), states (2.7) — fix
   only genuine regressions found; otherwise document.
4. **Focused tests** (§ Testing): accessible-name presence, sticky class/structure
   contract, `th scope="col"` / thead semantics, select-all header name,
   pagination/selection behaviour intact.

**Not changed:** the specialized accounting/report tables (Class 2), the small
settings lists (Class 2), the raw `<table>`s (Class 3), `<Money>`, filtering,
pagination architecture, selection architecture, density defaults.

---

## 4. Accessibility, performance, semantics guardrails

- **Performance:** sticky is pure CSS `position: sticky` — no scroll listeners, no
  observers, no per-row work, no DOM rewrite. The only structural change is a
  `max-height` class on one wrapper. Zero per-frame JS.
- **A11y:** keyboard nav / visible focus / `aria-sort` / checkbox semantics are
  preserved; the sticky header must not obscure a keyboard-focused row — because
  the wrapper is the scroll container, focusing a row scrolls it into the
  wrapper's viewport *below* the sticky header (standard behaviour); to verify
  live. Adequate contrast retained (existing header tint).
- **Semantics:** no element type changes; `<thead>/<th scope="col">` kept. Sticky
  is applied via className only.

---

## 5. Decisions — RESOLVED (2026-08-15)

1. **Sticky height → Option A (`calc()` template constant).** `max-h-[calc(100vh
   − Npx)]` on the DataTable scroll wrapper, N tuned once for the shared list-page
   template; self-adjusting; no per-page CSS; no layout rewrite.
2. **Accessible name → Option A (`aria-labelledby` the page `<h1>`).** `PageHeader`
   gets a stable id on its `<h1>`; `DataTable` references it, with a small
   `ariaLabel` fallback for tables without a page heading.
3. **Sticky scope → DataTable worklists only.** Specialized accounting/report and
   raw tables keep their current non-sticky, card-framed treatment.

_Original options retained below for the record._

### 5.1 Sticky-header height strategy
- **A (recommended):** `max-h-[calc(100vh − Npx)]` template constant on the
  DataTable scroll wrapper (N tuned once for the shared list-page template).
  Self-adjusting, contained, no per-page CSS, no layout rewrite. Validated.
- **B:** flex height-chain (`main` → page `flex flex-col` → table `flex-1
  min-h-0`). More "correct" but requires converting every list page to a
  flex-fill layout — **layout infrastructure** (a stated STOP condition). Not
  recommended for Batch D.

### 5.2 Accessible-name mechanism
- **A (recommended):** `aria-labelledby` — give `PageHeader` an `id` on its `<h1>`
  and let `DataTable` reference it (with a small `ariaLabel` string fallback for
  the few tables without a page heading, e.g. embedded/section tables). Matches
  the brief's "prefer the existing visible heading."
- **B:** `ariaLabel` string prop per page (simpler, but duplicates the visible
  heading text as an invisible label rather than referencing it).

### 5.3 Sticky scope
Confirm sticky headers land **only on the 27 DataTable worklists**, and the
specialized accounting/report tables and raw tables keep their current
(non-sticky, card-framed) treatment. (Recommended — matches "keep accounting
structures specialized.")

---

## 6. Documented, NOT solved in Batch D

- **Payments / Subscriptions hand-rolled money cells** → `moneyColumn` sweep is
  Batch E (Payments cleanup), not a table-finish change.
- **Raw `<table>` → `ui/table` primitives** (AccountPage/Wallet/Quote/
  PricingSimulator) → Batch E.
- **Trailing chevron + row-nav affordance / loose-row de-noise** → Batch F
  interaction, not density-consistency.
- **Any missing column due to backend gaps** → will be documented in the report,
  never fabricated.

## 7. STOP conditions (will halt + document)
Sticky needing a layout-infrastructure rewrite; a table needing a custom grid
engine; a required backend change; a page needing bespoke table CSS to compensate
for structure; DataTable accumulating page-specific flags; density hurting
readability; a11y forcing changes to unrelated components. Per §2.1, the contained
sticky avoids the first; if tuning the height constant turns out to need per-page
values, that is a STOP → I'll ship sticky only where the template constant works
and document the rest.

---

**STOP — investigation complete. Awaiting approval (esp. §5.1/§5.2/§5.3) before any
production code.**
