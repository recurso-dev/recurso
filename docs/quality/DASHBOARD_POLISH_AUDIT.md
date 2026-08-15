# Dashboard Polish Audit — Stripe-Grade Refinement

> **Read-only, code-cited + live-verified, point-in-time (2026-08-15).** No
> production code was modified to produce this document. This is the **polish /
> quality** phase audit — not feature work. The question it answers is *"why does
> the existing experience still feel like an admin dashboard rather than a
> world-class financial operating system, and what is the highest-leverage way to
> fix it?"*
>
> **Method.** Four parallel read-only code investigations (typography/tokens/
> hierarchy/density; DataTable + object-page consistency; states/motion/a11y/
> financial-trust; navigation/consistency/microinteractions/responsive) against
> the intent docs (`DESIGN.md`, `ART_DIRECTION.md`, `QUALITY_BAR.md`,
> `DASHBOARD_PRINCIPLES.md`, `MOTION.md`, `UX_RULES.md`, `ANTI_PATTERNS.md`) and
> the token source (`index.css`, `tailwind.config.js`), plus a first-hand live
> visual pass on app.recurso.dev (Home, Customers, Collections, and every object
> page captured this session). Every finding carries a `path:line`, a **P0–P3**
> severity, and an **owner** (FRONTEND / DESIGN SYSTEM / DATA / BACKEND /
> ACCESSIBILITY / PERFORMANCE).

---

## 1. Executive Summary

**The foundation is already excellent; the finish is not yet enforced.** Recurso
has a genuinely Stripe-caliber design *system* — one accent, a three-level shadow
ladder, a radius ladder derived from a single variable, warm-stone neutrals, a
`.money` monospaced-tabular signature, WCAG-corrected status tiers, a canonical
`StatusBadge` registry, a disciplined reduced-motion-gated motion layer,
exponent-aware money, and first-class loading/empty/error/partial states in the
shared primitives. The books are trustworthy and the money chain is fully
addressable.

**What still reads as "admin dashboard" is drift, not decoration.** The same
conceptual text role is rendered 3–8 different ways across pages; money is
rendered through **three** different paths (`<Money>`, `formatCurrency`,
`formatMinorUnits`) so the mono-tabular "money signature" is absent from most
amounts — *including the invoice total, the flagship number of the flagship
page*; the object "hero" (identity + primary amount) does not dominate its
header; list rows are loose; a per-row chevron and a link icon compete as trailing
affordances; and the loading/404/error blocks are hand-copied across ~13 pages
(with a straight-vs-curly-apostrophe divergence that proves it). None of this is
gradients, mesh, or vanity illustration — the ART_DIRECTION banned-list is
respected. **The SaaS smell here is typographic/role drift and un-enforced
primitives.**

**Therefore the fix is consolidation, not redesign.** The highest-leverage moves
are all "one change, many pages": a `<Label>`/`<Overline>` primitive for the
uppercase micro-label role (~48 sites, 8 variants); making `<Money>` the *only*
money renderer with a size prop (35 files); a shared object-page state wrapper
(`useObjectQuery` / `<ObjectNotFound>` + `<ObjectPageSkeleton>`) that collapses
~13 duplicated blocks *and* fixes the live 404-copy symptom; a sticky +
accessible-named `DataTable` header (every list at once); a shared
Details+Timeline+Audit rail (kills the "Details/Metadata/Scope" title drift and
gives three pages their missing timelines); and preserving list state on object
back-navigation (16 object pages ↔ 16 list pages). Do those and the product
crosses from "excellent billing admin" to "financial operating system" without a
single new feature.

---

## 2. Current Grade

**B+ (≈ 83 / 100) — "an excellent, trustworthy billing administration product
with a Stripe-caliber design foundation, held back from Stripe-grade by
enforcement drift and a few structural finish gaps."**

- **Foundation / tokens / primitives — A.** Token system, StatusBadge, Money
  exponent-handling, motion discipline, partial-failure degradation, focus rings,
  heading hierarchy are at or near the bar.
- **Application / consistency — B–.** The roles the system defines aren't enforced
  by components, so pages hand-roll and drift.
- **Object-page rigor — B.** Invoice is A; several peers are a tier below
  (missing rails, raw tables, verdict-not-a-banner).
- **List/table finish — B.** Genuinely good table, but no sticky header, no
  accessible name, loose density, inconsistent money/row-nav helpers.
- **Navigation continuity — B–.** Deep-links and URL-state are strong; the
  in-page back-link throws that state away.

## 3. Target Grade

**A (≈ 95 / 100) — Stripe-grade.** Achieved when: every conceptual role is
rendered by exactly one primitive; every amount is `<Money>` with a fixed size
vocabulary and the object hero dominates its header; every list has a sticky,
named, appropriately-dense table with consistent money/row-nav; every object page
shares one skeleton/not-found/rail; navigation never loses the operator's place;
and no raw gateway code or unsanctioned status renderer survives. This is a
finish target, not a feature target — the functional bar is already met.

---

## 4. Top 25 Polish Issues (ranked by impact)

| # | P | Owner | Issue | Anchor |
|---|---|---|---|---|
| 1 | P1 | DESIGN SYSTEM | Uppercase micro-label role has ~8 inline variants and **no primitive** (~48 sites) — biggest single "generated" tell | `StatCard.jsx:69` vs `ObjectPage.jsx:129` (disagree on color) |
| 2 | P1 | DESIGN SYSTEM | Money bypasses `<Money>` in **35 files** (`formatCurrency`/`formatMinorUnits`); mono-tabular signature absent from most amounts | `InvoicePage.jsx:423` (invoice total) |
| 3 | P1 | DESIGN SYSTEM | Object-page **loading skeleton** copy-pasted byte-identical across **13 pages** | `CustomerPage.jsx:129-140` + 12 more |
| 4 | P1 | DESIGN SYSTEM | Object-page **404/error branch** copy-pasted across **~9 pages** (with a straight-vs-curly apostrophe divergence) | `PaymentPage.jsx:94` vs peers |
| 5 | P1 | DATA/FRONTEND | Live **404 copy never shows**: guard's `error || !obj` treats a resolved-but-null / non-404 payload as a generic error | all 5 object pages |
| 6 | P1 | ACCESSIBILITY | `DataTable` has **no accessible table name** (`TableCaption` exported but unused) — every list is anonymous to a screen reader | `table.jsx:73-80`, `DataTable.jsx:322` |
| 7 | P1 | FRONTEND | **No sticky table header**; acute on Invoices (up to 10k client-loaded rows) — headers gone after ~15 rows | `table.jsx:16-18,51-60` |
| 8 | P1 | FRONTEND | Object **back-link discards list state** (filters/page/search) — the affordance the UI invites lands on page-1/All | `ObjectPage.jsx:48-55`; all 16 object pages |
| 9 | P1 | DATA | `Payments` list still shows **raw gateway `failure_code`** (`insufficient_funds`, `R01`) — the exact leak fixed elsewhere | `Payments.jsx:145-146` |
| 10 | P1 | DESIGN SYSTEM | `Payments` list uses a **bespoke `StatusPill`** (raw lowercase status), bypassing canonical `StatusBadge` — list ≠ its own detail page | `Payments.jsx:25-52` |
| 11 | P1 | FRONTEND | **`font-bold`** off-convention (vocab is medium/semibold) on the invoice total + a few totals — reads louder than every other number | `InvoicePage.jsx:423` |
| 12 | P1 | FRONTEND | The object **hero (identity + amount) doesn't dominate** — the total sits in an equal-weight "Amount" card below the header; JournalEntry doesn't lead with its amount at all | `InvoicePage.jsx:422-423`, `JournalEntryPage.jsx:76` |
| 13 | P2 | DESIGN SYSTEM | **Section-title role 3-way split**: `text-sm` (ObjectSection) vs `text-base` (16 report `<h2>`) vs CardTitle | `ObjectPage.jsx:107` vs `RevenueRecognition.jsx:181` |
| 14 | P2 | DESIGN SYSTEM | Rail attribute-section **title drift**: "Details" / "Metadata" / "Scope" for the same slot | `ObjectPage` consumers |
| 15 | P2 | DESIGN SYSTEM | **Shadow ladder violated** in ~24 places (raw `shadow-sm`, `hover:shadow-md`, `shadow-lg`) vs the intended 3-token ladder | `tailwind.config.js:131`; `StatCard.jsx:63` |
| 16 | P2 | DESIGN SYSTEM | **Hero-amount size inconsistent** across object pages (3xl / 2xl / lg) | Invoice vs Payment vs Subscription |
| 17 | P2 | FRONTEND | List **row density loose** (`p-4`, ~74px live on Customers) for a financial table; `compact` mode exists, unused | `DataTable.jsx:177`, live QA |
| 18 | P2 | FRONTEND | Per-row **chevron + link-icon** double trailing affordance; the chevron is decorative (rows already clickable) | live QA (Customers) |
| 19 | P2 | FRONTEND | **Reconciliation-run verdict is a section, not an AttentionBanner**, and uses a raw `<Badge>` not `StatusBadge` — the most audit-critical page is the least rigorous | `ReconciliationRunPage.jsx:98-114` |
| 20 | P2 | DESIGN SYSTEM | **CreditNote & Dispute pages have no Timeline/Audit rail** despite full approval/resolve lifecycles | `CreditNotePage.jsx:217`, `DisputePage.jsx:170` |
| 21 | P2 | FRONTEND | **Action-feedback split**: Invoice uses inline `<Alert>`; all peers use `toast` — standardize on toast | `InvoicePage.jsx:365-372` |
| 22 | P2 | FRONTEND | **`ObjectHeader` actions lack `flex-wrap`** (PageHeader has it) → multi-button headers overflow on narrow | `ObjectPage.jsx:76` |
| 23 | P2 | FRONTEND | **Ledger list debit/credit accounts are dead-end text** while the same accounts link in the detail sheet | `Ledger.jsx:180,192` |
| 24 | P2 | FRONTEND | **`AccountPage` hand-rolls a raw `<table>`** instead of the `ui/table` primitives | `AccountPage.jsx:171-222` |
| 25 | P2/P3 | FRONTEND | **Per-row "0 · Low" risk badge** on every Customers row + **inconsistent page subtitles** (precise on Collections/Payments, generic on Customers/Home) = low-signal noise | live QA |

---

## 5. Visual Hierarchy Issues

- **[P1 · FRONTEND] The object hero doesn't dominate.** QUALITY_BAR /
  DASHBOARD_PRINCIPLES want identity + primary money to be the first thing an
  operator reads. Today the identity is in `ObjectHeader` but the **primary
  amount lives in an "Amount" `ObjectSection` card of equal visual weight to
  "Timeline"/"Audit trail" below it** (`InvoicePage.jsx:422`), and on
  `JournalEntryPage` the amount doesn't lead at all — the header title is the code
  label and the balanced amount only appears deep in the legs
  (`JournalEntryPage.jsx:76`, `JournalEntries.jsx:114`). A journal entry's
  identity *is* its balanced amount. **Fix:** promote identity+amount into the
  header band with real dominance.
- **[P2 · live] The 4-KPI card grid is the loudest element on Home and
  Collections** — four equal-weight bordered cards with large numbers. They *are*
  contextualized (deltas, captions, and on Collections info-tooltips), so this is
  polish not a P0, but it's the single most "generic dashboard" visual. Home's
  KPIs lack the info-tooltips Collections' have (inconsistency). **Fix:** tighten
  to a denser single strip and/or reduce the numeric size so the exception surface
  above them stays the hero; give Home the same definition tooltips.
- **[P2 · DESIGN SYSTEM] Equal visual weight for unequal information** is the
  through-line — sections, KPI cards, and rail widgets all render at similar
  weight, so nothing recedes. The `ObjectRail` consolidation (§21) plus a real
  hero band re-establishes the priority ladder.

## 6. Typography Issues

- **[P1 · DESIGN SYSTEM] The uppercase micro-label role has ~8 inline variants
  and no home** — drifts on weight (none/medium/semibold), tracking (wide/wider),
  and color (`text-subtle` 7.25:1 vs `text-muted-foreground` 5.27:1). The two
  flagship primitives disagree: `StatCard.jsx:69` (`text-muted-foreground`) vs
  `AttributeList` `ObjectPage.jsx:129` (`text-subtle`). This is the biggest driver
  of the "subtly-off, generated" feel.
- **[P1 · FRONTEND] `font-bold` drift** at `InvoicePage.jsx:423`,
  `Organizations.jsx:278`, `CreateQuote.jsx:389` — the app's vocabulary is
  medium/semibold; bold amounts read louder than everything.
- **[P2 · DESIGN SYSTEM] Section-title 3-way split** (`text-sm` vs `text-base` vs
  CardTitle) — object pages and report pages disagree on how big a section title
  is.
- **[P3 · ACCESSIBILITY] Off-scale sizes** (`text-[11px]`, `text-[13px]`) at
  `PaymentPage.jsx:182`, `JournalEntries.jsx:73` bypass the 12/14/18/30 scale.
- **Consistent & good:** page title (`text-2xl font-semibold tracking-tight`),
  object kicker, StatusBadge, metadata/caption. The page-title role is the model
  the others should follow.

## 7. Spacing Issues

- **[P3 · DESIGN SYSTEM] Container padding disagrees:** `ObjectSection` body is
  `px-6 py-4` (`ObjectPage.jsx:110`) while `CardContent` is `p-6` (`card.jsx:44`),
  so report pages (raw Card) sit looser than object pages (ObjectSection) —
  vertical rhythm shifts as you navigate between them.
- **[P3 · DESIGN SYSTEM] AttentionBanner spacing inconsistent:** some pages pass
  `className="mb-6"`, others pass nothing (`CreditNotePage`/`DisputePage` vs
  `InvoicePage`/`SubscriptionPage`).
- Section-to-section gaps and `AttributeList gap-y-4` are appropriately dense —
  no excessive-whitespace problem at the page level.

## 8. Density Issues

- **[P2 · FRONTEND] List rows are loose for financial software.** Default
  `comfortable` (`p-4`, ~74px live on Customers). The `compact` density exists
  (`DataTable.jsx:177`) but no money-dense list opts in. Stripe/Ramp ledgers run
  denser; more rows per viewport aids scanning. **Fix:** consider `compact` as the
  default for money tables (or tighten `comfortable`).
- **[P2 · live] Trailing-affordance noise** (chevron + link icon) and per-row
  risk badges reduce information-per-pixel. Removing the decorative chevron and
  de-badging the "0 · Low" risk column recovers scan speed.
- **Good:** density is *high without chaos* on the object pages —
  `FinancialSummary` and `AttributeList` are dense and calm; the density problem
  is concentrated in list rows and trailing affordances.

## 9. Table Issues

(Owner: DESIGN SYSTEM / ACCESSIBILITY / FRONTEND — `DataTable.jsx`, `table.jsx`,
`cells.jsx`, `columns.jsx`.)

- **[P1] No accessible table name** (`A1`) — `TableCaption` is exported but never
  used; thread an `aria-label`/caption from `DataTable`.
- **[P1] No sticky header** (`A2`) — add `sticky top-0 z-10 bg-*` to the header
  row; acute on Invoices' large client-loaded lists.
- **[P2] `moneyColumn` helper bypassed** off Invoices (`A5`) — Payments/
  Subscriptions hand-roll money cells, losing bundled alignment + `sortValue`.
- **[P2] Most lists aren't sortable** (`A4`) — only Invoices; Payments/
  Subscriptions/Customers expose zero sortable columns.
- **[P2] `rowHref` not used on Payments** (`A3`) — uses `onRowClick`+navigate,
  losing ⌘/middle-click + copy-address that every other list has.
- **[P2] Numeric count left-aligned** on Customers "Subscriptions" (`A7`) — should
  right-align like money.
- **[P2] Filter-control placement inconsistent** (`A9`) — Payments puts its filter
  in the PageHeader actions slot; others in the DataTable toolbar.
- **[P2] Skeleton→row layout shift** (`A8`) — skeleton row padding/alignment
  doesn't mirror real cells.
- **[P3] Raw failure code in Payments list** (`A11`), **empty action-column
  header** (`A13`), **UUID read aloud in row checkbox label** (`A6`), **no grid
  keyboard nav** (`A12`, deliberate).
- **Strengths (keep):** empty/loading/error states first-class; bulk bar page-
  scoped + pruned; pagination boundary exact; `aria-sort` + focus rings; IDs
  render secondary. This is a genuinely good table — these are the last mile.

## 10. Object-Page Issues

- **Consistency matrix deviations (P2 · DESIGN SYSTEM):** rail title drift
  (Details/Metadata/Scope); action-feedback split (Invoice `<Alert>` vs toast);
  ReconciliationRun raw `<Badge>` vs `StatusBadge`; AttentionBanner `mb-6`
  inconsistency; `flashOnChange` present on most badges but absent on Customer.
- **Pages below the Invoice standard (see §22).**
- **Bespoke sub-components that should be primitives:** Subscription's MRR strip
  (`SubscriptionPage.jsx:678-725`) is hand-rolled rather than a shared
  `FinancialSummary`-family component; AccountPage's raw `<table>`.

## 11. Navigation Issues

- **[P1 · FRONTEND] `backTo` discards list state** (`N1`) — all 16 object pages
  hardcode a static list root; the in-page back-link lands on page-1/All while the
  list persisted filters in the URL. `navigate(-1)` appears nowhere. **This is the
  single most impactful nav gap.** Prefer restoring the referrer's search string
  (contextual back) over adding breadcrumbs.
- **[P3 · DESIGN SYSTEM] `breadcrumbs` prop is dead code** (`N2`) —
  `PageHeader.jsx:15-38` implements it; zero callers. Given the strong active-nav
  signal and N1, fix contextual back and **delete** the unused branch rather than
  wire decorative breadcrumbs.
- **[P2 · FRONTEND] Dead-end technical IDs** (`N3`/`N4`) — Ledger list accounts
  are unlinked text (linked in the detail sheet); scattered bare sliced UUIDs in
  AuditLog/Events/DunningDashboard/Disputes that are neither `CopyableId` nor a
  `Link` where a target page exists.
- **[P2 · FRONTEND] No org switcher / no notification unread indicator** in the
  chrome (`N5`) — "which org am I in" isn't surfaced (org-switch is backend-gated;
  the bell has no badge).
- **Good:** deep-link/refresh safety, `useUrlState` on 16 lists, active-rail
  "where am I" signal.

## 12. Interaction Issues

- **[P2 · ACCESSIBILITY] Native `title=` misused for real content** (`M2`) —
  `Metering.jsx:226` hides a meter's aggregation *definition* in `title`;
  `DunningDashboard.jsx:443` hides chart values in `title` — neither is
  keyboard/touch reachable. A real `Tooltip` (exists at `ui/tooltip.jsx`) is
  accessible.
- **[P2 · FRONTEND] Feedback channel inconsistent** (`M3`) — ~78 pages use toast;
  InvoicePage alone uses an inline `Alert`. Standardize on toast.
- **Good (the bar to hold):** `CopyableId` copy feedback (Check swap + `aria-live`
  "Copied"); `button.jsx` hover/active/focus/disabled/loading states; row
  `cursor-pointer` + real first-cell link; pagination "start–end of total".

## 13. Loading-State Issues

- **[P1 · DESIGN SYSTEM] The object-page loading skeleton is copy-pasted
  byte-identical across 13 pages** (`C1`) — collapse into `<ObjectPageSkeleton>`.
- **[P2 · PERFORMANCE] Skeleton→content layout shift** on tables (`A8`).
- **[P2 · FRONTEND] Stale/`isFetching` never surfaced** — only `Ledger.jsx:74`
  reads it; `DataTable` has no `fetching` prop, so post-mutation swaps and
  background refetches are invisible. Low urgency (staleTime 60s), but the one
  missing state.
- **Good:** skeletons everywhere, announced as `role=status aria-busy`.

## 14. Error-State Issues

- **[P1 · DATA/FRONTEND] 404-specific copy never renders live** (`§4 #5`) — the
  guard `error || !obj` catches resolved-but-null / non-404 payloads as generic
  errors, so "…not found" never shows (verified app-wide incl. the reference
  InvoicePage). **Frontend-fixable** in a shared wrapper: treat a resolved-but-null
  payload as an explicit not-found. (A true HTTP-404 from the read endpoints is a
  backend change — note and move on.)
- **[P2 · FRONTEND] Generic, non-actionable error copy is the norm** — `ErrorState`
  default "We couldn't load this data…" plus ~15 pages' "Failed to load X" with no
  *why*/*what next*, and the message string is duplicated ~15×. Centralize an
  `errorMessage(err, fallback)` helper and give ErrorState a more actionable
  default.
- **[P3 · FRONTEND] No reusable 403/permission state** — folded into the generic
  ErrorState (minor, tenancy returns not-found).
- **Good:** partial-failure degradation is excellent (Home per-tile `.catch`,
  Collections refuses to show failed analytics as `$0`).

## 15. Empty-State Issues

- **Largely good — no significant findings.** Empty vs **no-results** is correctly
  distinguished per page (`Customers.jsx:201`, `Subscriptions.jsx:254`,
  `Plans.jsx:177`…), and object sections conditionally render so a healthy object
  stays calm. **[P2]** the one systemic gap is not the *rendering* but the
  *duplication* — the empty-related-list markup repeats across 10 object pages
  (`C4`); a `<RelatedList emptyLabel>` wrapper consolidates it.

## 16. Accessibility Issues

- **[P2] `Payments` StatusPill** renders raw lowercase status, bypassing the
  sanctioned `StatusBadge` (`Payments.jsx:43-52`).
- **[P1 (table)] No accessible table name** (`A1`); **[P2] UUID read aloud in row
  checkbox label** (`A6`); **[P3] empty action-column header** (`A13`).
- **[P3] `aria-label` on non-interactive `<span>` arrows** (`SubscriptionPage.jsx:1188`,
  `InvoicePage.jsx:535`) — inconsistently announced; prefer visually-hidden text or
  `aria-hidden`.
- **[P3] Raw `<Dialog>` without `DialogDescription`** (subscription cancel, invoice
  preview) — Radix logs a missing-`aria-describedby` warning.
- **Strong baseline (keep):** focus-visible rings throughout, clean h1/h2
  hierarchy, `aria-sort`/`aria-busy`, FormField `aria-describedby`+`aria-invalid`,
  StatusBadge always text-not-color, labeled icon buttons, contrast tokens verify
  out. This is above-average; the gaps are the table name + the last StatusPill.

## 17. Responsive Issues

(Owner: FRONTEND. CODE-VERIFIED except where noted LIVE.)

- **[P2] `ObjectHeader` actions lack `flex-wrap`** (`R1`) — `PageHeader` has it;
  multi-button headers (Invoice Send/Preview/Download; Subscription Change-plan/
  Pause/Cancel) push horizontally on narrow. One-line primitive fix.
- **LIVE-verified good:** at 560px the Reconciliation-run page collapses to a
  single column with the wide table scrolling in its own container, no body
  overflow (this session).
- **CODE-verified good:** table horizontal-scroll wrappers, responsive column
  hiding, `FinancialSummary` grid degrade, `ObjectPageLayout` rail-collapse, sheet
  widths `w-full sm:max-w-*`, `AttributeList dd break-words` for long IDs. No
  overflow beyond R1.

## 18. Motion Issues

- **The motion layer is disciplined and correctly reduced-motion-gated on two
  layers.** No perpetual/decorative motion, no marketing count-ups (MotionNumber
  holds on mount, animates only real changes, snaps under reduced motion).
- **[P2 · PERFORMANCE, low] MotionNumber on every Home KPI** — well-built, but on
  frequent refreshes an operator may see a ~450ms roll on settled values;
  ledger-verified figures (recon, aging) are correctly *not* animated. Leave as-is
  or gate KPI animation to first-paint only.
- **Keep:** `StatusBadge flashOnChange` on object headers (one restrained flash on
  a real transition, off in lists); reconciliation verdict `MotionReveal`. The
  animated/static line is drawn sensibly. **Motion is essentially at target — do
  not add more.**

## 19. Financial-Data Presentation Issues

- **[P1 · DATA] `Payments` list shows raw gateway `failure_code`** (`insufficient_funds`,
  `R01`, `do_not_honor`) — the exact leak already fixed in `PaymentAttempts`. One
  edit (`humanizeFailure`) closes the last instance.
- **[P1 · DESIGN SYSTEM] Money signature absent from most amounts** — three render
  paths; the invoice total renders sans-serif non-tabular. Making `<Money>` the
  only renderer restores the mono-tabular "this is money" cue everywhere.
- **[P3 · DATA] Reconciliation-run amounts hardcoded USD** — the recorded run
  doesn't persist a reporting currency (documented in-code); correct on the live
  page. Mislabels for a non-USD functional-currency tenant. **Backend-gated** (add
  currency to the run) — note and move on.
- **[P3] One percentage (revenue-chart pill) relies on `title` only** for its
  definition; the rest carry visible definitions.
- **Sound:** exponent-aware money (JPY/KWD correct, not `/100`); signed
  negatives/credits intentional; KPI definitions as tooltips; FX-uncomparable
  currencies excluded *and disclosed*, never silently summed; no fabricated
  totals; zeros render as real zeros or em-dashes deliberately.

## 20. Consistency Violations

1. **Money:** three render paths (`<Money>` 24 files / `formatCurrency` 35 /
   `formatMinorUnits`) — the invoice total and recon amounts don't use `<Money>`.
2. **Status:** `Payments` `StatusPill` vs canonical `StatusBadge` (list ≠ detail
   for the same object).
3. **Uppercase micro-label:** ~8 class-string variants for one role.
4. **Section title:** `text-sm` vs `text-base` vs CardTitle.
5. **Rail attribute title:** Details / Metadata / Scope.
6. **Action feedback:** Invoice `<Alert>` vs toast (78 pages).
7. **Elevation:** 3-token ladder vs raw `shadow-sm`/`hover:shadow-md`/`shadow-lg`
   (~24 places).
8. **Hero-amount size:** 3xl / 2xl / lg across object pages.
9. **Page subtitle voice:** precise-operational (Collections, Payments) vs generic-
   marketing (Customers "Manage your customer base", Home "A snapshot of your
   billing performance").
10. **`flashOnChange`** on most object badges but not Customer.

## 21. Repeated Patterns That Should Be Consolidated

Ordered by pages affected (each is a "one change, many pages" lever):

1. **`useObjectQuery` / `<ObjectPageSkeleton>` + `<ObjectNotFound>`** → collapses
   the duplicated loading skeleton (**13 pages**) + the 404/error branch (**~9
   pages**), *and* fixes the live 404-copy symptom. **Largest single win.**
2. **`<Money size>` as the sole money renderer** → **35 files**; simultaneously
   fixes the money-signature split, `font-bold` drift, hero-amount size
   inconsistency, and the recon third-path.
3. **`<Label>` / `<Overline>` primitive** for the uppercase micro-label role →
   **~48 sites, 8 variants**.
4. **Sticky + accessible-named `DataTable` header** (one change to `table.jsx` +
   `DataTable.jsx`) → **every list** (fixes A1+A2).
5. **Shared `ObjectRail` (Details + Timeline + Audit)** → consistent rails on 4+
   pages; kills the Details/Metadata/Scope drift; gives CreditNote/Dispute/
   Reconciliation their missing timeline+audit.
6. **Context-preserving object back-navigation** (`ObjectHeader`) → **16 object
   pages ↔ 16 list pages**.
7. **`<RelatedList>`** wrapping `RelatedRow`/`RelatedEmpty` → **10 object pages**.
8. **`SectionTitle`** primitive (or force report pages through `ObjectSection`) →
   ~16 report `<h2>`s + Dashboard cards.
9. **`errorMessage(err, fallback)`** helper → ~15 list pages + every mutation
   toast.
10. **`moneyColumn` + `rowHref` sweep** on Payments/Subscriptions; **StatusBadge**
    on Payments; **`humanizeFailure`** on Payments; **normalize shadow ladder**
    (global `shadow-sm → shadow-raised`).

## 22. Pages Below the Invoice Reference Standard (ranked, worst first)

1. **ReconciliationRunPage** — the most audit-critical page is the least rigorous:
   raw `<Badge>` instead of `StatusBadge`, **no AttentionBanner** for a run with
   discrepancies (the archetypal exception state), "Scope" title drift, USD-
   hardcoded amounts. `ReconciliationRunPage.jsx:98-114,142-155`.
2. **CreditNotePage** — full approve/reject/void lifecycle but **no Timeline and
   no Audit rail**; a reviewer can't see who approved/when in-context.
   `CreditNotePage.jsx:217-247`.
3. **DisputePage** — open→resolved lifecycle but **no Timeline/Audit rail**.
   `DisputePage.jsx:170-196`.
4. **AccountPage** — **hand-rolls a raw `<table>`** for journal activity instead of
   the shared primitives; "Metadata" title drift. `AccountPage.jsx:171-222`.
5. **Payments (list)** — bespoke `StatusPill` + raw `failure_code` + `onRowClick`
   instead of `rowHref`; the list of a first-class object is a tier below its own
   detail page. `Payments.jsx:25-52,145,183`.
6. **JournalEntryPage** — Details-only rail, amount doesn't lead (lowest concern;
   an immutable posting justifies a thin rail, but its identity *is* its amount).

## 23. Highest-Impact Improvements

Ranked by IMPACT × FREQUENCY × TRUST × (low) EFFORT:

1. **`<Money size>` everywhere** (#2/#16) — every amount gains the money signature
   and a fixed size vocabulary; the hero total stops looking like body text.
   *Highest trust + frequency.*
2. **`useObjectQuery` state wrapper** (#3/#4/#5) — one abstraction removes ~22
   duplicated blocks and fixes the live 404 copy.
3. **`<Label>`/`<Overline>` primitive** (#1) — kills the single biggest "generated"
   tell across ~48 sites.
4. **Sticky + named table header** (#6/#7) — every list becomes scannable and
   screen-reader-named at once.
5. **Context-preserving back-nav** (#8) — the operator never loses their place;
   affects the whole list↔object workflow.
6. **Payments cleanup: StatusBadge + humanizeFailure + rowHref** (#9/#10) — closes
   the last raw-code leak and the last unsanctioned status renderer, aligns list
   with detail.
7. **Shared ObjectRail + object-page consistency pass** (#14/#19/#20) — brings the
   below-standard pages up to Invoice.

## 24. Low-Value Changes to Explicitly Avoid

- **Do NOT add global breadcrumbs.** The active-nav rail already answers "where am
  I"; N1 (context-preserving back) is the real fix. Delete the dead `breadcrumbs`
  prop rather than wire it.
- **Do NOT add more motion.** The motion layer is at target; adding transitions to
  historical timelines or hover states would slow operators.
- **Do NOT redesign tokens, colors, or the shadow/radius ladders** — they're
  correct; the work is *enforcing* them, not changing them.
- **Do NOT introduce new primitives that duplicate existing ones** (no parallel
  Money/StatusBadge/Table/Card). Every recommendation above *consolidates into*
  the existing system.
- **Do NOT chase per-page cosmetic tweaks** (individual color/spacing nudges) when
  a shared primitive fixes the class. Prefer one fix over twenty.
- **Do NOT add grid keyboard nav / role=grid** — the first-cell real link makes Tab
  work; full grid nav is disproportionate effort.
- **Do NOT touch the backend-gated items** (org switcher, per-run reconciliation
  currency, a true HTTP-404 from read endpoints) in a polish batch — document and
  move on. The 404 *copy* is fixable frontend-side without them.
- **Do NOT add features** (search/recent-objects/webhook feed/KPIs/ARR) — out of
  scope by definition.

## 25. Recommended Implementation Sequence

Small, verified, high-leverage increments — each its own green-CI PR with live QA,
compared against the Invoice reference. **Do not start until authorized.**

- **Polish Batch A — The Money Signature & Object Hero.** `<Money size>` as the
  sole renderer (start with the invoice total); promote identity+amount into a
  dominant header band on all object pages; retire `font-bold`; normalize
  hero-amount size. *Fixes #2, #11, #12, #16, #19-signature. Highest trust/impact.*
- **Polish Batch B — State & Error Consolidation.** `useObjectQuery` /
  `<ObjectPageSkeleton>` + `<ObjectNotFound>` (treat resolved-null as not-found —
  fixes the live 404 copy); `errorMessage` helper; adopt across all object pages.
  *Fixes #3, #4, #5, #14, removes ~22 duplicated blocks.*
- **Polish Batch C — The Label & Section Roles.** `<Label>`/`<Overline>` primitive
  + one `SectionTitle`; retrofit primitives (StatCard, AttributeList,
  FinancialSummary) and worst offenders; normalize the shadow ladder. *Fixes #1,
  #13, #15, and the color-token label drift.*
- **Polish Batch D — Table Finish.** Sticky + accessible-named header;
  `moneyColumn` + `rowHref` sweep; tighten row density; right-align numeric
  counts; fix skeleton layout shift; consistent filter placement. *Fixes #6, #7,
  #17, A3–A9.*
- **Polish Batch E — Object-Page Parity & Payments Cleanup.** Shared `ObjectRail`
  (Details+Timeline+Audit) on CreditNote/Dispute/Reconciliation/Account;
  ReconciliationRun verdict→AttentionBanner + StatusBadge; AccountPage→primitives;
  Payments StatusBadge + humanizeFailure + rowHref; standardize action feedback on
  toast. *Fixes #10, #19, #20, #21, #23, #24, and §22.*
- **Polish Batch F — Navigation & Interaction.** Context-preserving object
  back-nav; link the Ledger-list accounts + dead-end IDs (`LedgerAccountLink`/
  `CopyableId`); `flex-wrap` ObjectHeader actions; move the two misused `title=`
  definitions to real Tooltips; de-noise the risk badge + trailing chevron;
  align page subtitles to the operational voice. *Fixes #8, #18, #22-list, #23,
  #25, N3/N4/M2/R1.*

Sequence rationale: A and B are the two highest-trust, most-visible wins and set
the money + state foundation; C establishes the role primitives everything else
composes; D–F propagate consistency to tables, object-page parity, and
navigation. Each batch is a "one change, many pages" lever, not a page-by-page
grind.

---

## Appendix — What is already at target (do not touch)

Token system (one accent, 3-shadow/radius ladders, warm neutrals, `.money`
signature); `StatusBadge` registry; `Money` exponent handling; the reduced-motion-
gated motion layer; partial-failure degradation (Home/Collections anti-`$0`);
focus rings + heading hierarchy + FormField a11y; empty-vs-no-results distinction;
deep-link/refresh safety + `useUrlState`; the Invoice reference object page; the
core `DataTable` behavior (bulk, pagination boundary, aria-sort). The gap between
today and Stripe-grade is **enforcement and consolidation of this good system —
not a rebuild.**
