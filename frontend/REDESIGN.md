# Recurso Dashboard — Redesign Foundation

Phase 1 of a from-scratch admin dashboard rewrite. This document is the contract
for the page-rewrite agents: **copy these patterns, don't reinvent them.**

## Aesthetic

Stripe / Linear-style **light enterprise UI**.

- White / `zinc-50` surfaces, subtle `zinc-200` borders, near-black text.
- **Emerald `#10B981` is the single accent** (primary buttons, active nav,
  positive deltas, focus rings) — ties the app to the marketing site.
- Dense professional tables, generous whitespace, small uppercase section
  labels + large numerals for metrics.
- Typography: **Inter**, tight tracking.
- **No dark mode.** Do not add `dark:` variants. The old dark theme is retired.

## Stack added in this phase

| Dependency | Purpose |
|---|---|
| `shadcn/ui` (hand-vendored, JSX) | Component primitives in `src/components/ui/` |
| `@tremor/react` | Charts + KPI visuals (`AreaChart`, etc.) |
| `class-variance-authority`, `clsx`, `tailwind-merge` | `cn()` + variant styling |
| `tailwindcss-animate`, `@headlessui/tailwindcss` | animations + Tremor variants |
| `sonner` | Toasts (`@/components/ui/sonner`) |
| `@radix-ui/*` | Primitives behind dialog/sheet/select/dropdown/tabs/etc. |
| `lucide-react` (already present) | **The only icon set.** No material-symbols. |

## Configuration

- **Path alias `@/` → `src/`** — set in `vite.config.js` and `jsconfig.json`.
  Import everything new via `@/...` (e.g. `@/components/ui/button`).
- **`components.json`** — shadcn config, `"tsx": false` (JSX mode), base color zinc.
- **Design tokens** — CSS variables in `src/index.css` (`:root`), consumed by
  `tailwind.config.js` semantic colors. `--primary` / `--ring` are emerald.
- **`src/lib/utils.js`** — `cn()` plus shared `formatCurrency`, `formatNumber`,
  `formatDate`. **Money from the API is in minor units (cents)** — always pass
  raw integer amounts to `formatCurrency(amount, currency)`.

> Legacy color tokens (`bg-background-light`, `text-light-primary`, etc.) are kept
> in the Tailwind config, remapped to light values, so the ~25 not-yet-rewritten
> pages keep rendering. Delete them once every page is migrated.

## Where things live

```
src/
  lib/utils.js                      cn(), formatCurrency, formatDate, formatNumber
  components/
    ui/                             shadcn primitives (button, card, table, dialog,
                                    sheet, input, label, select, badge, tabs,
                                    dropdown-menu, separator, avatar, tooltip, sonner)
    patterns/                       ← the reusable page building blocks (USE THESE)
      PageHeader.jsx                title + description + breadcrumbs + actions
      StatCard.jsx                  KPI tile: label + big numeral + delta
      DataTable.jsx                 the canonical list table (search/filter/paginate)
      EmptyState.jsx                empty list/section
      ErrorState.jsx                fetch failure + retry
      LoadingSkeleton.jsx           Skeleton, TableSkeleton, CardGridSkeleton
      FormField.jsx                 label + control + description + error
      index.js                      barrel — import { PageHeader, DataTable, ... }
    layout/
      DashboardLayout.jsx           app shell (top bar, search, user menu)
      Sidebar.jsx                   grouped nav (Core / Growth / Finance / System)
```

> **Skeleton naming note:** the filesystem is case-insensitive and a legacy
> `ui/Skeleton.jsx` exists, so the shadcn skeleton primitive lives in
> `patterns/LoadingSkeleton.jsx` as `export function Skeleton`, not `ui/skeleton.jsx`.

## Reference implementations (study these)

- **`src/pages/Dashboard.jsx`** — KPI row (`StatCard`), Tremor `AreaChart`,
  recent-invoices `Table`. Wires real endpoints; shows `—` / empty states when a
  metric endpoint is missing (never invents data).
- **`src/pages/Customers.jsx`** — the **list-page template**: `DataTable` with
  server-side search (`q`), status filter, pagination, row → detail slide-over,
  `Badge` statuses.
- **`src/pages/CreateCustomer.jsx`** — the **form template**: a `Sheet`
  slide-over with `FormField` + `Input`/`Select`, inline validation, unchanged
  create-customer payload.

## Recipe: build a new LIST page

```jsx
import { PageHeader, DataTable } from "@/components/patterns";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

const columns = [
  { key: "name", header: "Name", cell: (r) => <span className="font-medium">{r.name}</span> },
  { key: "status", header: "Status", cell: (r) => <Badge variant="success">{r.status}</Badge> },
  { key: "amount", header: "Amount", align: "right", cell: (r) => formatCurrency(r.total) },
];

<PageHeader title="Invoices" description="…" actions={<Button><Plus/>New</Button>} />
<DataTable
  columns={columns}
  data={rows}
  loading={loading} error={error} onRetry={refetch}
  onRowClick={openDetail}
  search={{ value: q, onChange: setQ, placeholder: "Search…" }}
  toolbar={/* filter chips/selects */}
  empty={{ title: "No invoices yet", description: "…", action: <Button/> }}
  pagination={{ page, onPrev, onNext, hasNext: rows.length >= PAGE_SIZE }}
/>
```

## Recipe: build a new FORM (slide-over)

```jsx
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter } from "@/components/ui/sheet";
import { FormField } from "@/components/patterns";
import { Input } from "@/components/ui/input";

<Sheet open onOpenChange={(o) => !o && navigate(back)}>
  <SheetContent side="right" className="w-full sm:max-w-lg">
    <SheetHeader><SheetTitle>Add …</SheetTitle></SheetHeader>
    <form id="my-form" onSubmit={submit} className="flex-1 overflow-y-auto px-6 py-6 space-y-6">
      <FormField label="Name" htmlFor="name" required error={errors.name}>
        <Input id="name" value={form.name} onChange={…} />
      </FormField>
    </form>
    <SheetFooter>
      <Button variant="outline" onClick={close}>Cancel</Button>
      <Button type="submit" form="my-form" disabled={loading}>Save</Button>
    </SheetFooter>
  </SheetContent>
</Sheet>
```

## Conventions the rewrite agents MUST follow

1. **Import via `@/`**; pull page blocks from `@/components/patterns`, primitives
   from `@/components/ui/*`.
2. **Icons: `lucide-react` only.** Remove any `material-symbols` / `<Icon>` usage
   in pages you rewrite.
3. **Emerald is the only accent.** Use semantic classes (`bg-primary`,
   `text-primary`, `focus-visible:ring-ring`) or the `Badge`/`Button` variants —
   don't hardcode other brand colors.
4. **No `dark:` variants.** Light UI only.
5. **Money is in minor units** — format with `formatCurrency(amountMinor, currency)`.
6. **Every list page** = `PageHeader` + `DataTable`, with loading / error /
   empty handled by the DataTable props (don't roll your own table shell).
7. **Every form** = `Sheet` + `FormField`; keep the existing API payload identical,
   restyle only.
8. **Missing data ≠ fake data.** If an endpoint doesn't exist, render `—` or an
   `EmptyState`. Never invent numbers.
9. **Preserve** `src/App.jsx` routing, `src/auth/AuthProvider`, `src/lib/api.js`.
10. **Toasts:** new pages may use `import { toast } from "@/components/ui/sonner"`.
    The legacy `useToast()` context still works and is used by not-yet-migrated pages.
```
