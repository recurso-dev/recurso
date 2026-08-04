# Recurso — UX Rules

> The behavioral contract every screen honors, plus a code-cited audit of where
> the current UI does and doesn't meet it (§Audit). A page missing a required
> state is unfinished. Implementation wins over this doc.

## Every page must have

- **Loading** — a skeleton, not a bare spinner (`patterns/LoadingSkeleton.jsx`:
  `Skeleton`/`TableSkeleton`/`CardGridSkeleton`).
- **Error** — a message + **Retry**, never a blank page (`patterns/ErrorState.jsx`).
- **Empty** — an `EmptyState` with a one-line explanation and the primary action.
- **Success feedback** — a toast (`ui/sonner`) or inline confirmation.

`patterns/DataTable.jsx:91-104` supplies error → loading → empty → data **for
free** to any list routed through it. Pages that don't use DataTable must
hand-roll all four.

## Data & tables

Search · filters · pagination (server-side for growth-unbounded lists;
virtualization for high-cardinality — *intended, not yet implemented*, see
Audit) · export (CSV) on financial tables · bulk actions where repetition would
occur · sortable columns · sticky headers · right-aligned tabular money with the
correct currency exponent.

## Interaction

Keyboard-operable with visible focus; `⌘K` command palette
(`ui/command-palette.jsx`). Destructive actions use `ConfirmDialog`
(`ui/confirm-dialog.jsx`) with the specific target named, destructive styling,
never default focus. Undo where possible; optimistic updates with rollback;
≤ 3 clicks for common tasks.

## Forms

Validate at the edge with per-field messages (`patterns/FormField.jsx`); preserve
input on error; disable submit in-flight; never auto-refresh a form the user is
editing.

## Accessibility (WCAG 2.1 AA)

Labelled inputs; `aria-label` on icon buttons (91 present); Radix focus
management in Dialog/Sheet; contrast ≥ 4.5:1; status conveyed by text+icon, not
color alone (`ui/badge.jsx:12-17`); `prefers-reduced-motion` respected
(chart animation gates on it).

## Responsive

Works at 320/768/1024/1440; wide content scrolls inside its own
`overflow-x-auto` container; the page body never scrolls sideways.

## Money-specific UX

Every figure traces to its postings (deep-link to the ledger where supported);
statuses in words, not codes; a reconciling difference is explained, never
"unexplained" (`ANTI_PATTERNS.md`).

## House patterns to reuse

`src/components/patterns/` (DataTable, PageHeader, StatCard, EmptyState,
ErrorState, LoadingSkeleton, CustomerSelect/Name, FormField) and
`ui/confirm-dialog.jsx`. Detail & create/edit views are right-side Sheets
(`sm:max-w-md`). Data fetching is react-query (ADR-005), not hand-rolled
`useEffect`.

---

## Audit — inconsistencies to fix (code-cited)

- **State handling (ADR-005 drift):** ~12 pages bypass react-query with
  hand-rolled `useEffect`+fetch (Metering, Usage, Organizations, OfflinePayments,
  Integrations, CancelFlows, Events, Churn, Wallets, FinanceReconciliation,
  TaxNexusSettings, portal/*). **Worst: `settings/BillingSettings.jsx`** — no
  error path at all (only `loading`, `:79,103`). *Correction:* `Security.jsx`
  does **not** purely bypass.
- **Responsive violations:** native `<Table>` without `overflow-x-auto` — Team
  (`:122`), Security (`:403`), DunningDashboard (`:272`); clipping
  `overflow-hidden` — RevenueRecognition (`:166`), FinanceReconciliation
  (`:177`).
- **No virtualization yet:** DataTable renders all rows; native-`<Table>` pages
  are unpaginated (silent-truncation risk with backend default limits). The
  "virtualized / dense mode" table intent is not implemented.
- **Test holes:** no dedicated test for AcceptInvite, CancelFlows, CreateCoupon,
  CreateCreditNote, CreatePlan, CreateQuote, ExecutiveSummary, Integrations,
  Ledger, Profile, RevenueRecognition, Security (PageSmoke covers them).
  **Portal:** only PortalDashboard tested; PageSmoke's glob **excludes
  `portal/*`** — 5 portal pages untested.

Recommendation order + full detail: `docs/evidence/design-and-ux.md`; tracked in
`../REMEDIATION.md`.

## Source of truth

- **Code:** `frontend/src/components/{patterns,ui,charts}/`,
  `frontend/src/pages/`, `frontend/src/App.jsx`.
- **ADR:** ADR-005 (layered caching / react-query contract).
- **Evidence file:** `docs/evidence/design-and-ux.md`.
- **Related:** `DESIGN.md`, `ANTI_PATTERNS.md`, `DOCUMENTATION_RULES.md`.
