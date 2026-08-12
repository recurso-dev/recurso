# Recurso Dashboard — Stripe-Grade Product Design Mission

> **Standing directive** (founder, 2026-08-12). This is the charter for the
> dashboard redesign initiative. The audit it mandates lives in
> `DASHBOARD_UI_AUDIT.md`. Related doctrine: `DESIGN.md` (tokens/legality),
> `ART_DIRECTION.md` (identity), `UX_RULES.md` (behavioral contract),
> `ANTI_PATTERNS.md`.
>
> **Prime directive: do not beautify the existing UI. Establish a design
> system and migrate the existing UI onto it.** Stripe-grade quality comes
> from consistency across many screens, not from making one page beautiful.
>
> **Execution model — four passes, iterative loop**
> (audit → redesign system → implement → screenshot → critique → fix → repeat):
>
> | Pass | Job | Output |
> |---|---|---|
> | 1 | Product/design audit | `DASHBOARD_UI_AUDIT.md` |
> | 2 | Design system | tokens + primitives + shell |
> | 3 | Dashboard | Home + navigation + tables + object pages |
> | 4 | Ruthless QA | screenshots → critique → fixes |
>
> The thesis stays: **a financial operating system — an instrument panel for
> financial infrastructure, not a colorful SaaS analytics product.**

---

You are the principal product designer and frontend architect responsible for
transforming the existing Recurso dashboard into a world-class financial
operating system.

This is NOT a greenfield redesign.

A substantial amount of the product is already implemented. Your job is to
inspect the existing product, understand what already works, preserve business
behavior, and systematically elevate the UX/UI, information architecture,
interaction design, visual hierarchy, accessibility, responsiveness, and design
consistency.

The target quality bar is Stripe Dashboard / Linear / Vercel-class product
quality. Do NOT blindly copy Stripe. Study the principles behind products like
Stripe: extremely strong information hierarchy · restrained visual language ·
excellent typography · predictable navigation · dense but readable
financial/data interfaces · consistent tables · excellent object/detail pages ·
contextual actions · progressive disclosure · excellent empty/loading/error
states · keyboard accessibility · responsive behavior · reusable design
primitives · consistent spacing and interaction patterns · very little
decorative UI · every pixel has a functional purpose.

Recurso must ultimately feel like a serious financial infrastructure product,
not a generic SaaS dashboard.

## Primary objective

Transform the dashboard from "an application that exposes our features" into
"a coherent financial operating system that users can operate confidently every
day." The user should immediately feel: trustworthy · precise · fast · calm ·
professional · financially serious · technically sophisticated · predictable.
The interface should communicate the same level of confidence as the accounting
engine underneath it.

## Phases

- **Phase 0 — audit first, no code changes.** Explore the repository, the
  design docs, the component library, every route/workflow/table/form/
  modal/chart/state, responsive behavior, duplicated patterns, inconsistencies,
  and frontend tech debt. Identify what can be improved globally rather than
  page-by-page. Deliverable: `docs/DASHBOARD_UI_AUDIT.md` (architecture,
  design system, components, routes, patterns, inconsistencies, UX problems,
  a11y problems, responsive problems, highest-leverage improvements,
  recommended design-system changes, migration plan). Do not assume the
  existing implementation is wrong. Do not start redesigning until the audit
  is complete.
- **Phase 1 — design system.** Centralized tokens: typography (display, page
  title, section title, body, body-small, label, caption, numeric/financial,
  table, code), spacing scale, restrained semantic colors (background, surface,
  elevated surface, border, divider, text tiers, accent, success, warning,
  danger, info), border rules (avoid card soup — not every section needs a
  card), small controlled radius scale, shadows used extremely sparingly,
  standardized iconography (one family; size/stroke/alignment rules).
- **Phase 2 — information architecture.** Navigation matches the operator's
  mental model of a billing/accounting system, not the backend's structure.
  Only categories that actually exist. Obvious primary workflows, minimal
  cognitive load, deep navigation, preserved context, clear current location,
  works at small widths.
- **Phase 3 — dashboard home.** Answers: what is happening / is anything
  wrong / how is the business performing / what needs my attention / what can
  I do next. Header → executive signal → trends → attention/exceptions →
  activity → navigation into workflows. Operational, not decorative.
- **Phase 4 — tables.** One unified table system: alignment, typography, row
  height, density, sorting, filtering, search, pagination, selection, bulk
  actions, status badges, all four states, responsive behavior. No one-off
  table implementations.
- **Phase 5 — object pages.** A flexible object-page system with consistent
  primitives: identity header (status, primary + contextual actions), summary
  attributes, activity/timeline, related objects, financial information,
  technical IDs/metadata, audit information. Do not force every object into
  one rigid layout.
- **Phase 6 — forms & workflows.** Field grouping, labels, descriptions,
  validation, error messages, defaults, keyboard navigation, confirmation,
  destructive-action consequences, success states. Progressive disclosure for
  complex flows.
- **Phase 7 — empty/loading/error/permission states** as first-class design.
  Empty states explain what the page is, why it's empty, and what to do next.
- **Phase 8 — microinteractions.** Hover/focus/pressed states, optimistic
  updates where safe, inline success feedback, contextual menus, keyboard
  shortcuts, restrained transitions, skeletons, URL-preserved filters.
  Animation only to communicate state or continuity.
- **Phase 9 — responsiveness.** A deliberate strategy per screen; tables
  collapse/scroll/prioritize; nav adapts; dialogs become mobile-friendly; no
  horizontal page overflow.
- **Phase 10 — accessibility.** WCAG 2.2 AA: keyboard, focus, contrast,
  semantics, ARIA, labels, form errors, dialogs, menus, tables, charts, no
  color-only indicators.
- **Phase 11 — visual QA (mandatory).** Run the app; inspect every major page
  at desktop/tablet/mobile widths in populated/empty/loading/error states with
  keyboard navigation; screenshot before/after; fix; repeat.

## Critical rules

- **Design:** never redesign pages independently — primitives → patterns →
  page templates → migrate pages. If fixing one component improves 20 pages,
  fix the component.
- **Engineering:** never rewrite working business logic for prettiness.
  Preserve APIs, data models, accounting/billing behavior, routing semantics,
  permissions, integrations. Separate UI refactoring from business-logic
  changes.
- **Product:** never invent features to make a page look complete. If
  something is missing, document it.

## Design bar (ask per screen)

Hierarchy (understood in 3 seconds?) · Density (right for financial ops?) ·
Consistency (same product as every other screen?) · Actionability (obvious what
I can do?) · Trust (would I run millions through this?) · Precision (numbers,
dates, statuses unambiguous?) · Restraint (can anything be removed?) ·
Accessibility (no mouse needed?) · Responsiveness (excellent at every
viewport?).

## Execution order

Repository audit → UI audit → design tokens → core primitives → navigation →
dashboard shell → dashboard home → tables → object pages → forms →
modals/drawers → states → responsive → accessibility → visual QA → final
consistency pass. After each major stage: run tests, lint, verify the app,
inspect the UI, fix regressions. Do not accumulate a huge unverified redesign.

## Definition of done

Not "CSS looks nicer." Done means: coherent design language; shared primitives;
intuitive navigation; excellent tables; consistent object pages; scannable
financial data; obvious workflows; polished states; intentional responsive
behavior; working keyboard navigation; strong accessibility; eliminated visual
inconsistencies; existing functionality intact; automated tests passing; the
product feels like one coherent system. Before declaring completion, perform a
final "design director" review of the entire dashboard and fix anything that
looks amateur, inconsistent, generic, overly decorative, or unnecessarily
complicated. Do not stop at "good enough."

---

## Execution log (2026-08-12)

Every phase above shipped as green-CI PRs, in charter order. The audit
(`docs/DASHBOARD_UI_AUDIT.md`) is the findings ledger; this is the receipts.

| Stage | PRs | What landed |
|---|---|---|
| 0 · Audit | #604 | `DASHBOARD_UI_AUDIT.md` — 6 gap systems, 12 WCAG findings, 8 shipped bugs, migration plan |
| 1 · Tokens | #605 | `--success/--warning/--info/--foreground-subtle/--canvas`; `--primary` darkened in-family to 5.60:1 (a11y fix, **not** the deferred rebrand); radius/shadow ladders; menu focus ring; dark-mode config removed |
| 2 · Primitives + codemod | #606 #607 | StatusBadge registry (retired 14 duplicated maps), Alert, Textarea, CopyableId, `formatDateTime`; 641→0 raw-palette occurrences; ESLint palette guard ON |
| 3 · Shell + IA | #608 | `lib/navigation.js` canon (sidebar/palette/topbar all derive), mobile drawer, skip link, real 404, IA regroup, one h1 per page |
| 4 · Tables | #609 | DataTable v2: sorting (`aria-sort`, server+client), honest pagination totals, real row semantics (no `role="button"` on `<tr>`), column priority, cells/columns helpers |
| 5 · Addressability | #610 #611 #612 | Backend GET /invoices/:id + /credit-notes/:id; six real `/x/:id` routes; `location.state.openInvoiceId` dead; rowHref link rows; customer-scoped list filters |
| 6 · Object pages | #613 #614 #615 #616 | ObjectPage system + AuditTrail; Customer, Subscription (1011-line sheet deleted), Invoice (643-line sheet deleted) full pages; subscription-scoped invoices |
| 7 · Forms & safety | #617 #618 #619 #621 | All six audit §7 bugs dead; confirm-guards on one-click money ops; FormSheet (form/autofocus/dirty-guard) + 10 sheet migrations; accent-palette guard gap closed |
| 8 · Home | #620 | Operations console: honest 30d comparisons, activity feed with object links, 5/4/3 operational row, xl stacking |
| 9 · Accessibility | #622 | 65+ labels associated, palette = real combobox/listbox, every chart has a text alternative, `type="button"` sweep, `h-dvh` shell |
| 10 · Visual QA | #623 | Live pass at 320/375/768/1440 against a seeded stack; topbar chip + PageHeader action-wrap fixes |

Deliberately not done (with reasons): the emerald→blue rebrand (founder-deferred,
DESIGN.md §13); MRR delta on Home (no history endpoint — a fabricated trend is
worse than none); coupon max-redemptions and plan description fields (backend
has no such concepts — the lying inputs were removed instead); per-object event
timelines (needs an `object_id` filter on `/events`).
