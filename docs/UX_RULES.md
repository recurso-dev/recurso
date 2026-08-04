# Recurso — UX Rules

> The behavioral contract every screen honors. If a page is missing one of
> these, it is unfinished. Enforced in review; several already have automated
> tests.

## Every page must have

- **Loading state** — a skeleton (not a bare spinner) that mirrors the coming
  layout.
- **Error state** — a message that says what failed and offers **Retry**. Never
  a blank page. (Tested for Usage/OfflinePayments/Churn.)
- **Empty state** — an `EmptyState` with a one-line explanation and the primary
  action ("No wallets yet — Create wallet").
- **Success feedback** — a toast or inline confirmation that names what happened
  ("Published", "Invoice voided").

## Data & tables

- **Search** on any list a user scans.
- **Filters** for the dimensions that matter (status, date, customer).
- **Pagination** — server-side for anything that grows with the customer base;
  virtualization for high-cardinality lists.
- **Export** (CSV) on financial tables.
- **Bulk actions** where a user would otherwise repeat themselves.
- **Sortable** columns; **sticky headers** on long lists.
- Money is right-aligned, tabular, and currency-exponent-correct.

## Interaction

- **Keyboard**: everything operable without a mouse; visible focus rings;
  shortcuts for power users where they fit (`⌘K` search exists).
- **Confirmation dialogs** for destructive/irreversible actions — with the
  specific target named ("Void INV-1043?"), destructive styling, never the
  default focus.
- **Undo where possible**; where not, confirm first.
- **Optimistic updates** for cheap toggles, with rollback on error.
- **≤ 3 clicks** for common tasks.

## Forms

- Validate at the edge with specific, per-field messages.
- Preserve input on error.
- Disable submit while in-flight; show progress; re-enable on completion.
- Never auto-refresh a form the user is editing.

## Accessibility (WCAG 2.1 AA)

- Labelled inputs; `aria-label` on icon-only buttons.
- Focus management in dialogs/sheets (Radix handles this — use the shared
  components).
- Contrast ≥ 4.5:1 for text; status conveyed by text/icon, not color alone.
- `prefers-reduced-motion` respected.

## Responsive

- Works at 320 / 768 / 1024 / 1440.
- Wide content (tables, charts) scrolls inside its own container; the page body
  never scrolls sideways.

## Money-specific UX

- Every displayed figure is traceable to its postings (deep-link to the ledger
  where the surface supports it — "explain any number").
- Statuses read in words, not codes.
- A reconciling difference is explained, never called "unexplained"
  (`ANTI_PATTERNS.md`).

## The house patterns to reuse

`src/components/patterns/`: `DataTable`, `PageHeader`, `StatCard`, `EmptyState`,
`ErrorState`, `LoadingSkeleton`, `ConfirmDialog`, `CustomerSelect`/`CustomerName`.
Detail & create/edit views are right-side Sheets (`sm:max-w-md`). Data fetching
is react-query (ADR-005) — not hand-rolled `useEffect`.

## Related

- `DESIGN.md` — the visual system these rules live in
- `ANTI_PATTERNS.md` — the inverse (what breaks these)
