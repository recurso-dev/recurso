# Recurso Dashboard — UI Audit (Phase 0 of the redesign mission)

> Deliverable of `DASHBOARD_REDESIGN.md` Phase 0. Produced 2026-08-12 from six
> parallel code-grounded sweeps (IA/navigation, tables, object pages, tokens,
> forms/workflows, responsive/accessibility) over `frontend/src` (~33k LOC JSX,
> 75 pages, 24 ui primitives, 13 patterns), plus first-hand reads of the shell,
> tokens, and doctrine (`DESIGN.md`, `ART_DIRECTION.md`, `UX_RULES.md`,
> `ANTI_PATTERNS.md`). Every claim cites code. No code was changed.
>
> **Relationship to the August design review** (`design-review-2026-08.md`):
> that initiative closed 13 *leaf-level, shared-component* themes (v0.11.0).
> This audit examines the *structural* layer that pass deliberately did not
> touch: information architecture, addressability, the table system, the
> object model, the token tier, responsiveness, and WCAG conformance. The two
> documents do not overlap; this one supersedes the stale audit appendix in
> `UX_RULES.md` (§Audit there still lists react-query drift that #580–#587
> closed).

---

## Verdict

**The product has excellent bones and a missing structural layer.** The
architecture pass of 2026-08 fixed what pages *show* (honest states, real
currencies, names not UUIDs). What remains is what the app *is*: a set of
lists with state-opened side sheets, unreachable by URL, unnavigable between
objects, unusable below ~1024px, unsortable everywhere, and styled by ~641
raw-palette classes because three semantic tokens don't exist.

The gap to the Stripe/Linear bar is concentrated in **six structural systems**
— shell, addressability, tokens, tables, object pages, forms — and every one
of them is fixable at the primitive level, exactly as the mission's critical
design rule demands. Almost nothing requires redesigning pages one by one.

### Scorecard (Stripe/Linear/Vercel bar = 9–10)

| Dimension | Score | One-line reason |
|---|---|---|
| Honest states (error/loading/empty) | 8 | DataTable gives all four free to 25 lists; ~6 raw tables still render failure as empty |
| Money/date/status precision | 7 | `<Money>` + tabular-nums strong; 6 date formats, 13 duplicated status maps |
| Information architecture | 4 | 44 flat nav items, 3 names per destination, no breadcrumbs (built, unused), palette drifted |
| Addressability / deep links | **2** | 1 of 22 detail surfaces has a URL (the public checkout); 2 of ~28 pages keep any state in the URL |
| Object model (detail pages) | 3 | 16 one-off sheets; zero related-object rails; zero audit trails; zero real timelines |
| Tables | 4 | Great shell, but **no sorting anywhere**, 8 pagination models, wrong row semantics |
| Forms | 5 | FormField/useFormErrors are excellent and 7%-adopted; 24 unguarded money ops; no dirty guard |
| Design-token discipline | 4 | 641 raw-palette occurrences; success/warning/info tokens missing; dead token systems |
| Responsiveness | **2** | No mobile nav at all; 87px of content at 375px; breakpoints ignore the 240px sidebar |
| Accessibility | 4 | Primary CTA fails AA contrast (3.86:1); menu focus 1.09:1; no skip link; 110 unlabeled controls |

---

## 1. Current architecture

- **Stack:** Vite + React 18, react-router 8 (flat `<Routes>` in `App.jsx:132-224`),
  TanStack react-query 5 (ADR-005; universal since #580–#587), Tailwind 3.4 +
  shadcn/Radix primitives, Tremor 3.18 for all 11 charts, lucide-react 0.294
  (lockfile-pinned; `package.json` carries a caret — tighten), self-hosted
  Inter + JetBrains Mono via @fontsource. Light-only.
- **Shell:** `DashboardLayout.jsx` = fixed `w-60` sidebar + topbar (own h1,
  search/⌘K, test-mode chip, bell, avatar) + `main` capped `max-w-[1400px]`.
  `SettingsLayout` is the only nested layout besides the shell.
- **Data conventions:** react-query everywhere, minor-unit money through
  exponent-aware helpers, `{data:}` envelopes.
- **Test surface:** 93 vitest files / 430+ tests; PageSmoke covers most pages
  (portal/* excluded — gap).

## 2. Existing design system

- **Tokens** (`index.css:11-37`): complete shadcn set — warm-stone neutrals,
  deep-emerald `--primary`, one `--radius`. **Missing: `--success`,
  `--warning`, `--info`, surface/elevation tiers, text tiers.** `--secondary`,
  `--muted`, `--accent` are three names for one value.
- **Dead systems:** 11 legacy color names in `tailwind.config.js:56-68` (0
  uses), the entire Tremor token tree (0 uses of `*-tremor-*` classes), all
  three declared shadow tokens (0 uses), the declared 4-step type scale (0
  uses). The app runs on Tailwind defaults instead.
- **Doctrine quality is high** — DESIGN.md's principle order
  (Correctness > Trust > Readability > Density > Beauty), spacing scale, page
  hierarchy, and copy rules are exactly the mission's bar. Two staleness bugs:
  card radius documented as 8px-current/12px-target while `card.jsx:9` already
  ships `rounded-xl` (12px, hardcoded not tokenized); UX_RULES' audit appendix
  predates v0.11.0.
- **Constraint to respect:** the emerald→blue rebrand is founder-deferred
  (DESIGN.md §13). *Note:* fixing `--primary`'s AA contrast failure by
  darkening within the emerald family (§8) is an accessibility fix, not the
  rebrand — flagged as a decision below.

## 3. Existing reusable components

- **`patterns/`** (13): DataTable, PageHeader (breadcrumbs built, used 0
  times), StatCard (definition tooltips), EmptyState, ErrorState,
  LoadingSkeleton (Table/CardGrid), FormField (exemplary ARIA wiring),
  CustomerSelect/CustomerName, EntityScopeSelect + ReportScopeSelect (two
  entity pickers, no shared context), ProviderGuide.
- **`ui/`** (24): button (CVA, loading, `[&_svg]:size-4`), badge (8 variants —
  raw-palette internals), card, sheet, dialog, confirm-dialog, select,
  dropdown-menu, tabs, input, password-input, label, switch, table, money,
  code-sample, copyable-secret, command-palette (static list), tooltip, kbd
  (CSS), sonner, avatar, separator, ConsentCheckbox (off-pattern).
- **Missing primitives the audit repeatedly hit:** Alert/Callout (the single
  biggest drift generator), StatusBadge (13 duplicated maps), Textarea (10
  verbatim class-string copies), FormSheet, DateCell/MoneyCell/IdCell,
  CopyableId (≥6 reimplementations), ObjectLink, ActivityTimeline,
  RelatedObjects, AttributeList, PlanSelect/InvoiceSelect/SubscriptionSelect,
  CountrySelect/CurrencySelect/StateSelect, SegmentedControl/filter-chips,
  Checkbox, AuthShell, NotFound page.

## 4. Existing routes

74 declarations, 61 authenticated destinations, 2 nested layouts, **one**
authenticated `:id` route (`/quotes/:id/edit`) and zero read `:id` routes.
Full inventory in the IA sweep; the structural facts:

- **Creation is deep-linkable; reading is not.** Six `/x/new` routes render a
  bare `<Sheet open>` over an empty main region, while all 22 detail surfaces
  are `useState`-opened (`CustomerDetail`… `Organizations`). The only
  URL-addressable object in the product is the *public* checkout.
- **`location.state.openInvoiceId`** is the sole cross-page detail handoff —
  and it only matches rows on the currently loaded page (`Invoices.jsx:163-174`),
  so links from Ledger/Reconciliation/Dunning/Home fail silently past page 1.
- **No 404**: `path="*"` silently redirects to Home (`App.jsx:223`) — which is
  how a broken link shipped unnoticed (§7 bug list).
- `/security` and `/team` are settings-family pages routed as siblings of
  `/settings`, breaking the sub-nav when navigated to.

## 5. Existing UX patterns (what already works — protect these)

- The canonical page: PageHeader → StatCards → chart → DataTable → Sheet, with
  one `mb-6` header rhythm and a single-sourced page shell (`DashboardLayout.jsx:194`).
- DataTable's free error→loading→empty→data contract, with contextual docs
  links on empty states.
- FormField's `aria-describedby`/`aria-invalid`/`role=alert` wiring +
  `useFormErrors` focus-first-error (adopted in 3 forms — the pattern to
  spread, not to invent).
- 18 destructive ops already guarded with consequence-naming ConfirmDialogs
  (wallet close, write-off, gift cancel are model copy).
- Reduced-motion: global CSS block **plus** the JS chart gate — above the bar.
- 100% Radix overlays (focus trap/Esc free), zero `window.confirm`, zero
  custom modals, all 105 badges carry text, single icon family with zero
  inline SVG, self-hosted fonts, `scope="col"` on headers, global
  `tabular-nums` on `td`.
- ⌘K palette exists; Test-mode chip mirrors gateway mode; StatCard
  `definition` tooltips explain metrics.

## 6. Inconsistencies (quantified)

| Axis | Count | Detail |
|---|---|---|
| Raw palette | **~641** occurrences, 92/133 files | red 218 · stone 141 · emerald 141 · amber 115; destructive intent runs **13:1 raw-to-token**; 18 hardcoded hex (incl. cool *slate* leaking into a warm-stone system); 8 `dark:` stragglers |
| Tinted status panels | 107 lines, 84 distinct class strings | same error banner at 4 radii, 4 paddings, 2 font sizes — **no `ui/alert.jsx` exists** |
| Page title | 8 variants | canonical `PageHeader` vs 7 hand-rolls; topbar renders a *second, smaller* h1 |
| Section title | 10 variants | 16px in 13 places, 14px in 12 — a coin flip |
| Eyebrow label | **21 variants** | two leaders differ by 1px; three tracking values |
| CardTitle | sizeless | 31 uses inherit context-dependent size (`card.jsx:26`) |
| Metric size | 2 | StatCard 30px vs 9 hand-rolled 24px |
| Date formats | **6** | no `formatDateTime` helper exists; 4 raw `toLocaleString` variants |
| Status→variant maps | **13 duplicates** | `past_due` renders raw snake_case on 4 pages (capitalize applied inconsistently) |
| Pagination models | **8** | two have a boundary bug (Next enabled into an empty page); page sizes 10→250; `pagination.total` supported, passed by 0 pages |
| Sheet widths | md/lg/xl/2xl + omitted `side` | `sm:max-w-md` convention now the minority; no detail uses SheetFooter |
| Filter-control idioms | 5 | segmented / pills / ui-select / native select / tabs, in 4 different placements |
| Radius | 6 in use, 3 token-derived | `rounded-xl` (24 incl. Card) and `rounded` (35) don't move with `--radius` |
| Shadows | 5 ad-hoc values | the 3 declared tokens have 0 uses; dropdown vs its own sub-menu at different elevations |
| Icon default size | 16px (252) vs 20px (22) competing | buttons enforce 16; nothing else enforced |
| Off-scale spacing | ~259 (14%) | almost all `*-0.5/1.5/2.5/5` — the code has de-facto chosen half-steps the doc forbids; zero arbitrary bracket values |
| Copy-ID logic | ≥6 reimplementations | plus `Field`/`Section`/`DetailRow` re-declared per slide-over |
| Toast conventions | 3 | wrapper / raw sonner (4 files) / local banners; 11 mutation-bearing files show no toast at all |
| Duplicated pickers | 6 hand-rolled customer dropdowns | no Plan/Invoice/Subscription picker primitives |

## 7. UX problems

### Shipped bugs found by this audit (fix regardless of redesign)

1. **Collections → customer link is dead**: `Collections.jsx:400` targets
   `/customers/:id`, which doesn't exist; the `*` route silently lands on
   Home. (P0 — an operator clicking a name in the collections queue is dumped
   on the dashboard.)
2. **`openInvoiceId` only works on page 1** (`Invoices.jsx:163-174`) — cross-
   page invoice links fail silently for older invoices.
3. **Fetch failure renders as empty** on Team (`Team.jsx:146-158`; error is
   toast-only) and TaxNexusSettings (3 queries, zero `isError` handling) —
   "no team members" and "couldn't load" are opposite statements on a
   permissions/compliance screen.
4. **Ledger entries' error path is unreachable** and Retry refetches the
   *accounts* query (`Ledger.jsx:319-320`); `keepPreviousData: true` is a dead
   v4 option under react-query 5, so every page turn flashes a skeleton.
5. **CreateCoupon collects `max_redemptions` and `active` and sends neither**
   (`CreateCoupon.jsx:123-138`); **CreatePlan collects `description` and drops
   it**. Users set a redemption cap that never applies.
6. **Three silent mutation failures** (console.error only): gift purchase
   (money!), API-key create, referral create.
7. `/collections` has no topbar title (missing `TITLES` key → "Recurso");
   `/dunning` lacks `end` matching so two nav items highlight on
   `/dunning/campaigns`; avatar fallback is hardcoded `"AD"`
   (`DashboardLayout.jsx:166`).
8. **Keyboard double-fire**: DataTable row `onKeyDown` never checks
   `e.target === e.currentTarget`, so Enter on a nested action button fires
   the action *and* opens the row — on all 11 clickable lists.

### Structural problems

- **Nothing is addressable.** No object URL, no shareable filter/period/entity
  state (2 of ~28 stateful pages use searchParams), no browser-back semantics,
  no second-tab compare, no ⌘-click on any row. An auditor cannot share an
  audit-log query; a multi-entity controller re-picks the entity on every page.
- **The object graph is unnavigable.** One working object→object link in the
  app (ledger→invoice). Invoice→customer, customer→subscriptions/invoices,
  subscription→plan/invoices don't exist in any form; `CustomerName` is a
  `<span>` on seven surfaces. Detail sheets contain exactly one `useNavigate`
  across the whole directory — dead ends by construction.
- **Detail sheets have outgrown their objects**: Subscription (1,011 lines,
  8 peer action buttons, nested Dialog), Invoice (actions scattered across 3
  scroll positions; preview escapes to a `max-w-3xl` Dialog from a 448px
  sheet), Plan (777-line pricing editor at 512px), Customer (687 lines and
  still missing its subscriptions/invoices). Seven object types have **no**
  detail at all (Mandate/Dispute/Payment/Gift/Referral/Entity/Metric) while
  `GET /mandates/:id` sits implemented and unused.
- **No per-object provenance** anywhere, despite `/audit-logs` already
  accepting `entity_type`+`entity_id` — a one-component gap in an
  "every number explainable" product.
- **No sorting anywhere** — not one of 53 tables. Every DataTable footer says
  "Page 3" because `total` is never passed.
- **24 unguarded consequential ops**, topped by credit-note **approve**
  (moves money; its sibling *void* is guarded), quote convert/send, invoice
  send, bill-usage-now / advance-invoice / one-off charge (all three are on
  MCP's tier-3 money list — confirm-guarded for AI agents, unguarded for
  humans), cancel-IRN (statutory), business-country auto-save-on-select
  (switches the tax regime instantly, mixed save models in one card).
- **No unsaved-changes guard anywhere** — Esc/backdrop discards a half-built
  multi-line quote silently. 17 sheet forms have no `<form>` (Enter dead), and
  zero create/edit sheets autofocus their first field.
- **Home is a feature index, not an operations console** — it has good
  exception surfacing (overdue/churn callouts) but no period comparison, no
  activity feed with object links, and its layout collapses at `lg` (§9).
- Three surfaces render the same event stream (/events, Developers tab,
  /notifications — which fakes `read: false` client-side); ⌘K is a hand-coded
  route list already 18 destinations out of sync and can't search records.

## 8. Accessibility problems (WCAG 2.2 AA)

| # | Failure | Criterion | Scale |
|---|---|---|---|
| A1 | No skip link; `main` has no id; ~62 nav links precede content | 2.4.1 (A) | every page |
| A2 | **Primary button fails contrast**: white on `--primary` = 3.86:1 | 1.4.3 | every CTA |
| A3 | Menu/Select/palette keyboard focus = background-only at **1.09:1** | 2.4.7 | every dropdown |
| A4 | `text-stone-400` nav group headings/icons = 2.52:1; Badge `default` 3.41:1; `muted-foreground` on `bg-muted` 4.35:1 (TabsList) | 1.4.3/1.4.11 | shell + ~20 sites |
| A5 | 110 of 204 form controls unlabeled (80 orphan `<Label>`s) | 1.3.1/4.1.2 | 15+ files |
| A6 | `role="button"` on `<tr>` flattens row semantics; nested controls illegal; no link semantics on rows | 4.1.2 | 11 lists |
| A7 | Skeletons/ErrorState silent (`aria-live`×1 in app); fetch/pagination unannounced | 4.1.3 | every list |
| A8 | Two `<h1>`s per page (56 pages); h1→h3 skips via Empty/ErrorState | 1.3.1 | 56 pages |
| A9 | ⌘K palette: no listbox/option roles, focus never moves, count unannounced | 4.1.2 | palette |
| A10 | Charts: bare SVG, no text alternative (11 charts; 2 good counter-examples in-repo) | 1.1.1 | 6 pages |
| A11 | 320px reflow impossible (fixed sidebar + `h-screen overflow-hidden`); 200% zoom clips | 1.4.10/1.4.4 | shell |
| A12 | 11 raw buttons missing `type="button"` (two on money paths); 2 icon-only controls with no name; ~20 `title`-as-name | 4.1.2 | scattered |

Above the bar already: FormField wiring, reduced-motion (CSS+JS), Radix
overlays, badge text pairing, `scope="col"`, 143 `focus-visible:` sites.

## 9. Responsive problems

| # | Problem | Evidence |
|---|---|---|
| R1 | **No mobile shell.** Unconditional `w-60` sidebar, no drawer/hamburger; 4 visibility-toggling responsive classes in the whole layout | at 375px: **87px** of content; at 320px the nav is 75% of the viewport |
| R2 | Breakpoints ignore the sidebar: `sm:grid-cols-2 lg:grid-cols-4` (9 pages + the shared skeleton) puts 30px money in ~172px tiles at a 1024px viewport | `$1,234,567.89` needs ~210px |
| R3 | DataTable toolbar: non-wrapping flex of fixed-width selects — Subscriptions' 476px toolbar overflows until ~810px | `DataTable.jsx:87` + `w-[150px]`×3 |
| R4 | Sheets are `w-3/4` on mobile → **233px** usable form width in the app's primary create/edit surface | `sheet.jsx:35`; Dialog does it right |
| R5 | 25 of 75 pages have zero responsive classes; `md:` appears 12 times and `xl:` **zero** — effectively a one-breakpoint design | grep totals |
| R6 | 23 fixed-px widths (two 240px entity pickers, month/year input pairs); a few bare multi-col grids inside 384px sheets | inventory in sweep |
| R7 | Good: every table already scrolls in its own container; 33/35 wide grids carry prefixes; charts are Tremor-responsive with no min-widths | protect |

## 10. Highest-leverage improvements

Ordered by (screens lifted × severity) ÷ effort. Each is a primitive/shell
change, not a page redesign.

1. **Mobile shell + skip link + 404** — `hidden lg:flex` sidebar + Sheet
   drawer + hamburger; `<main id>` + skip link; NotFound page; drop the
   duplicate topbar h1 (breadcrumbs replace it); derive avatar initials.
   *Fixes R1/R2's root, A1, A8, A11, the silent-redirect bug class.*
2. **Token tier completion** — add `--success/--warning/--info` (+ subtle
   bg/border ramps); darken `--primary` within emerald to ≥4.5:1 (decision
   flag: a11y fix, not the deferred rebrand); real focus treatment for
   menu/select/palette items; retire `text-stone-400` for meaningful content;
   radius/shadow ladders token-derived; type scale in `theme.fontSize`;
   tighten the safelist + add a lint rule so drift can't compile silently.
3. **`ui/alert.jsx`** — absorbs ~107 tinted panels / ~350 raw-palette
   occurrences in one component; then the mechanical neutral/red codemod
   (~250 more), and detox badge/StatCard/Sidebar (the three most-copied files).
4. **DataTable v2** — sorting (`aria-sort`, server+client modes); row-link
   semantics (`rowHref` → real `<Link>` in the primary cell; kill
   `role="button"` on `<tr>`; fix the double-fire); one pagination contract
   (`page/pageSize/total` + `usePagedQuery` with the `+1` over-fetch and
   URL sync); wrapping toolbar; `aria-busy`/`role=alert` states; column
   priority/min-width; density; TableFooter totals + expandable rows (unblocks
   the 6 raw tables that had reasons); StatusBadge + DateCell/MoneyCell/IdCell.
   *Lifts all 25 DataTable pages + converts ~12 raw tables.*
5. **Addressability** — query-param object URLs (`?customer=<id>` …) resolved
   the way `openInvoiceId` already resolves, then delete `location.state`
   handoffs; searchParams for filters/period/entity/tab on the 12 finance
   reports and list pages (the Ledger idiom, generalized); make
   `CustomerName` a link (7 surfaces at once); fix the Collections dead link.
6. **Object-page system** — `ObjectHeader/ActionBar/AttributeList/
   MoneySummary/ActivityTimeline/RelatedObjects/ObjectLink/CopyableId`
   zone primitives; per-object audit trail via the existing
   `/audit-logs?entity_type&entity_id` (zero backend); migrate Invoice,
   Customer, Subscription, Plan, CreditNote first; Sheets remain for peek +
   create/edit. *Backend notes (documented, not fabricated): `GET
   /invoices/:id` and `/credit-notes/:id` don't exist; `GET /subscriptions/:id`
   and `/mandates/:id` exist unused; events lack an `object_id` filter.*
7. **FormSheet + guards** — `<FormSheet>` (form element, footer, autofocus,
   busy, dirty-guard via `useUnsavedGuard`); spread `useFormErrors` from 3 →
   46 forms; ConfirmDialogs with amounts on the 24 unguarded ops (mirror the
   18 good ones); one toast convention; fix the data-dropping payloads;
   Plan/Invoice/Subscription/Country/Currency/State pickers to kill the last
   raw-UUID inputs.
8. **IA regroup** — one name per destination (sidebar = topbar = page = ⌘K),
   palette derived from `NAV_GROUPS` + record search (all endpoints exist),
   collapse-by-group nav, Referrals→next-to-Gifts, AuditLog→System,
   `/security`+`/team` nested under settings, one events surface, breadcrumbs
   on (already built).
9. **Home as operations console** — keep the exception callouts, add real
   comparison periods and an activity feed with object links (unblocked by #5/#6).
10. **Visual QA loop** (Pass 4) — 3 widths × 4 states × keyboard per page,
    screenshot before/after, design-director pass.

## 11. Recommended design-system changes

**Add:** `--success/--warning/--info` (+ `-subtle` bg tiers) · text-tier vars
(or codify stone-500/600 rules) · elevation ladder (`raised/overlay/popover`)
pointed at by all primitives · token-derived `rounded`/`rounded-xl` · a real
`theme.fontSize` scale (caption 12 / body 14 / body-lg 16 / title 18 /
section 16-semibold / metric 30 / plus mono-numeric + code roles) · Alert ·
StatusBadge · Textarea · Checkbox · FormSheet · SegmentedControl/filter-chips ·
DateCell/MoneyCell/IdCell · CopyableId · ObjectLink · AttributeList ·
ActivityTimeline · RelatedObjects · AuthShell · NotFound · object pickers ·
`formatDateTime`.

**Change:** `--primary` darkened within emerald for AA (decision) · menu/select
focus states visible · sheet `w-full sm:max-w-md` · CardTitle gets a size ·
StatCard `p-5`→scale · icon default codified at 16px/14px-inline · KPI grid
ladder shifted (`sm:2 xl:4`) or container queries · legalize the de-facto
half-steps (2/6/10/20px) in DESIGN.md **or** codemod them — either way, doc
and code must agree.

**Remove:** dead legacy color names, dead Tremor token tree, dead shadow
tokens (replace with the ladder), `darkMode: ["class"]`, the 8 `dark:`
stragglers, the drift-legitimizing safelist breadth, `BuyGiftModal` (legacy
raw-UUID twin of the good Gifts flow), the topbar h1 + its two title maps,
`location.state` detail handoffs (after URLs land).

**Docs to sync:** DESIGN.md card-radius note; UX_RULES stale audit appendix;
CLAUDE.md sheet-width convention (or enforce it).

## 12. Migration plan

Mapped to the mission's four passes. Each stage = green-CI PRs; run
lint/tests/app-verify after every stage; no big-bang.

**Pass 2 — design system (Phases 1–2):**
2a. Tokens: semantic colors + contrast fixes + focus states + radius/shadow
    ladders + type scale + safelist/lint. (No visual redesign — discipline.)
2b. Alert + StatusBadge + Textarea + cell primitives + CopyableId +
    `formatDateTime`; detox badge/StatCard/Sidebar; neutral/red codemod.
2c. Shell: mobile drawer + skip link + 404 + topbar cleanup + avatar + nav
    regroup/rename + palette derived from nav + breadcrumbs on.
    *Exit criteria: app usable at 375px; axe-clean shell; zero raw palette in
    ui/ + patterns/ + layout/.*

**Pass 3 — dashboard (Phases 3–8):**
3a. DataTable v2 + usePagedQuery + URL state; migrate the 25 DataTable pages
    and convert the 12 unjustified raw tables; fix the P0 table bugs.
3b. Addressability: object URLs, CustomerName links, kill location.state,
    searchParams on finance reports.
3c. Object-page system + first five objects (backend prerequisites filed
    separately: invoices/:id, credit-notes/:id, events object filter).
3d. FormSheet + useFormErrors spread + destructive guards + toast convention
    + pickers; fix data-dropping forms.
3e. Home as operations console.
    *Exit criteria: every object shareable by URL; sortable tables; zero
    unguarded money ops; Enter submits every form.*

**Pass 4 — ruthless QA (Phases 9–11):**
4a. Responsive sweep at 320/768/1024/1440 per page (the primitives will have
    done most of it; fix residuals).
4b. A11y verification: axe + keyboard walk per page; charts get text
    alternatives; palette gets combobox semantics.
4c. Visual QA loop with before/after screenshots; final design-director pass
    against the Design Bar (hierarchy/density/consistency/actionability/
    trust/precision/restraint/accessibility/responsiveness).

**Sequencing rule:** 2a blocks everything (tokens first); 3a and 3b can run in
parallel after 2c; 3c depends on 3b; nothing in Pass 3 starts before Pass 2
exits green.
