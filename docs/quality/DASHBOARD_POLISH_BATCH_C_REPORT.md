# Dashboard Polish — Batch C Report
## Label / Overline / Section-role consolidation

Status: **complete, awaiting review.** The dashboard's uppercase micro-label role
now has one home — `<Overline>` — so every label doing the same semantic job
belongs to the same authored system. Implements the approved investigation
(`DASHBOARD_POLISH_BATCH_C_DESIGN.md`, PR #700). Shipped as **#701**. Lint, build,
and the full **711**-test suite are green. No Money / ObjectHeader / ObjectPage
lifecycle / DataTable-behaviour / token / navigation / motion / backend changes.

---

### 1. Before / after inventory

| | Before (`main` @ `1ef831c3`) | After |
|---|---|---|
| Uppercase micro-label sites | **66** (audit estimated ~48) | 66, all resolved |
| Distinct class strings for the role | **18** (audit estimated 8) | **1** canonical token, in `<Overline>` |
| Primitives disagreeing on the role's colour | Yes — object-page family `text-subtle` vs stat/table family `text-muted-foreground` | No — one token (`text-subtle`) |
| Home for the role | none | `components/ui/overline.jsx` |

The audit's "~48 sites / 8 variants" was a low estimate; verified reality on
`main` was **66 sites in 18 distinct class strings** across 40 files (the extra
site vs the design doc's 65 — `BuyGiftModal.jsx` — sat between the two inventory
greps and was caught during migration).

### 2. 66 sites / 18 variants → final state

| Disposition | Count | How |
|---|---|---|
| Migrated to `<Overline>` | **57** | 7 shared primitives + 50 inline (49 script-migrated + BuyGiftModal) |
| `TableHead` token-aligned (column-label owner) | 1 | `tracking-wider`/`muted-foreground` → `tracking-wide`/`text-subtle` |
| Raw `<tr>` header — canonical colour only | 3 | `text-muted-foreground` → `text-subtle`; structure untouched |
| **Consolidated total** | **61** | |
| Intentionally excluded (documented, §11) | 5 | heading / marketing / accent / navigation |
| **Total** | **66** | |

All 18 legacy class strings for the role are gone from consumers; the role is
expressed by one token, owned by `<Overline>` (and mirrored once in `TableHead`,
the column-label primitive).

### 3. Exact number migrated

- **57** sites now render via `<Overline>`.
- **+1** primitive token-aligned (`TableHead`), **+3** raw `<tr>` colour-swapped
  → **61 sites** carry the canonical token.
- **5** deliberately left (documented).

### 4. Files affected (32)

**New:** `components/ui/overline.jsx`, `components/ui/__tests__/overline.test.jsx`.

**Primitives retrofitted (7 + TableHead):** `patterns/StatCard.jsx`,
`patterns/ObjectPage.jsx` (ObjectHeader kicker + AttributeList `dt`),
`patterns/FinancialSummary.jsx` (term + currency group), `patterns/PaymentAttempts.jsx`,
`ui/command-palette.jsx`, `ui/table.jsx` (TableHead token).

**Inline (21):** `pages/` — `RevenueWaterfall`, `PaymentPage`, `Organizations`,
`Security`, `AuditLog`, `UnitEconomics`, `Integrations`, `MonthEndClose`,
`Ledger`, `InvoicePage`, `SubscriptionPage`, `FinanceReconciliation`,
`AskAnalytics`, `Developers`, `Events`, `settings/BillingSettings`,
`settings/MCPSettings`; slide-overs — `PlanCharges`, `PlanDetail`, `CustomerDetail`;
`components/BuyGiftModal`. **Colour-only `<tr>`:** `pages/AccountPage`,
`pages/WalletPage`, `pages/QuotePage`.

### 5. Canonical `<Overline>` API

```jsx
import { Overline } from "@/components/ui/overline";

<Overline>Invoice</Overline>              // overline / kicker  → <div>
<Overline as="dt">Amount due</Overline>   // metadata term      → <dt>
<Overline as="p">MRR</Overline>           // stat label         → <p>
<Overline as="th" scope="col">Method</Overline>  // (available)  → <th>
```

- **One deliberate style:** `text-xs font-medium uppercase tracking-wide text-subtle`.
  Exported as `OVERLINE_CLASS` for the one primitive that can't compose the
  component (`TableHead`).
- **Only knob is `as`** (default `"div"`) — sets the element to keep semantics.
  **No `size` / `tone` / `weight` / `variant` props** (the proliferation the brief
  forbids). A caller `className` is for layout offsets (`mb-3`, `px-3`) only.
- Forwards `ref`; passes through arbitrary props (e.g. `title=` for the
  StatCard definition tooltip).

### 6. Semantic / accessibility verification

- **Elements preserved via `as`:** every attribute term stayed a `<dt>` (inside
  its `<dl>`), the command-palette/StatCard/Ledger labels stayed `<p>`, the
  PaymentPage related labels stayed `<span>`, table column labels stayed
  `<th scope="col">`. **No semantic heading was turned into a span; no span became
  a heading.** An overline is not an `<h*>`.
- **Uppercasing** remains CSS `text-transform` (the `uppercase` class inside the
  atom) — screen readers read the underlying mixed-case text; unchanged.
- **Contrast improved:** ~44 sites moved `text-muted-foreground` (5.27:1) →
  `text-subtle` (7.25:1). Nothing dropped below AA. No colour-only meaning
  introduced.
- **Form labels untouched:** the field-label role stays on `ui/label.jsx`
  (`text-sm font-medium`, sentence case). `BuyGiftModal` keeps its `Label` import
  alongside the new `Overline` — the two roles remain distinct.
- **Focus / keyboard / reduced-motion:** the atom is static non-interactive text;
  nothing added or removed on those surfaces.
- **Tests:** `overline.test.jsx` verifies default `<div>` + canonical tokens,
  polymorphic `dt`/`th`/`span`, `dt`/`th` semantics preserved, className merge,
  and ref forwarding. No CSS-class snapshot tests (per the brief).

### 7. TableHead treatment

`ui/table.jsx:56` is the **column-label owner** — every `DataTable` header already
renders through it, so this is a one-line token alignment, not a migration:
`tracking-wider text-muted-foreground` → `tracking-wide text-subtle`, matching the
atom. **Layout, density, sorting, sticky behaviour, and all DataTable
functionality are untouched** (those belong to Batch D). The 3 hand-rolled `<tr>`
headers (`AccountPage`/`WalletPage`/`QuotePage`) received the canonical colour
only; their raw-`<table>` structure is left for Batch D/E.

### 8. Lint-guard decision + evidence

**Deferred — not added.** After migration, exactly **9** raw `uppercase`+`tracking`
sites remain, and they span **6 legitimate categories**:

```
ui/table.jsx:56              → TableHead (the column-label primitive home)
pages/AccountPage.jsx:173    ┐
pages/WalletPage.jsx:379     ├ 3 structural raw <tr> headers (Batch D/E)
pages/QuotePage.jsx:288      ┘
components/layout/Sidebar.jsx:63          ┐ 2 navigation-chrome labels
components/settings/SettingsLayout.jsx:87 ┘ (off-limits this batch)
pages/Landing.jsx:81              → marketing eyebrow (out-of-dashboard + accent)
pages/portal/PortalDashboard.jsx:375  → accent text-primary (meaningful colour)
pages/ExecutiveSummary.jsx:368   → a section <h2> heading
```

Per the decision framework, a guard is only worth adding if the exceptions are
"very few and obvious" and it needs no broad allowlist. Here the exceptions are
9 across 6 rationales — a guard would require a broad, multi-reason allowlist and
would be **noisy**. **Guard deferred because the remaining legitimate patterns
would make the rule noisy.** The consolidated `<Overline>` primitive + review is
the boundary; a future guard could be revisited once the structural `<tr>` and
navigation items move onto their own primitives in later batches.

### 9. Section-title decision

**Deferred by decision** (yours). The `ObjectSection` (`text-sm`) vs report-page
(`text-base`) section-title split (audit #13) is **not** in Batch C's enumerated
vocabulary, and normalising it forces a visual size change on either the
reference object pages or the report pages. Left untouched; documented as a
separate follow-up. No `<SectionTitle>` created.

### 10. Live visual QA

Verified on the deployed Batch C build (app.recurso.dev, post-#701 merge) across
the 8 representative pages. The **Stripe-grade test** — "does every label doing
the same semantic job look like it belongs to the same system?" — passes.

| Page | Result |
|---|---|
| **Home** | KPI overlines (MRR, ACTIVE SUBSCRIPTIONS, CHURN, RECOVERED REVENUE) render as the uniform `text-subtle` overline; hierarchy title→overline→metric→caption reads cleanly |
| **Invoice** | "INVOICE" kicker, "LINE ITEMS", and every Details-rail term identical; section titles (Breakdown/Details) stay sentence-case (correctly deferred); Money hero intact |
| **Subscription** | Kicker + Overview terms + the migrated FinancialSummary strip (MRR/OUTSTANDING/PAST DUE/…) + Metadata rail all render identically — the strip now matches the attribute terms |
| **Payment** | Kicker + the migrated Related labels (INVOICE/CUSTOMER/SUBSCRIPTION) + Details rail all uniform |
| **Journal Entry** | Kicker + Details rail uniform; balanced-amount hero + DR/CR intact |
| **Reconciliation Run** | Kicker + Scope rail terms + the Discrepancies **column headers** all uniform |
| **Customer** | Kicker + Financial summary + Overview + Metadata rail all uniform |
| **Invoices (dense DataTable)** | Column headers (NUMBER/CUSTOMER/AMOUNT/STATUS/E-INVOICE/DATE) now render in the aligned `text-subtle` overline, matching the object pages; sorting arrows, density, alignment, and money right-alignment untouched |

**Specifically inspected:** uppercase-label **consistency** (colour, letter-spacing,
weight, size now identical across kickers, terms, stat labels, and column headers);
**hierarchy** (page title → sentence-case section title → overline → value →
caption is intact and legible); **alignment** (money still right-aligned/tabular);
**contrast** (the `text-subtle` overlines read crisply — the ~44 formerly
`muted-foreground` labels are now a touch darker/clearer); **density** (object-page
rails and the dense invoices table are calm, no reflow); **wrapping** (short
uppercase labels, no overflow at 1440px). No heading or screen-reader semantics
regressed; Money / ObjectHeader hero / DataTable behaviour all unchanged.

**Notes / limitations:** the dashboard is a light-theme surface (no dark toggle);
`text-subtle` is a themed token so it adapts if dark mode is enabled. Narrow-
viewport is safe by construction — overlines are short, self-contained text with
no fixed width inside the unchanged object-page/table grids. The only micro-labels
that intentionally differ are the excluded nav-chrome (Sidebar `BILLING`/`GROWTH`),
accent, heading, and marketing labels (§11) — a deliberate, documented boundary,
not drift.

### 11. Intentionally deferred / NOT migrated (with rationale)

- **Form field labels** (`ui/label.jsx`) — different role (sentence-case); untouched.
- **Badges / chips** (`Entities`, `ExecutiveSummary` chip) & **mono codes**
  (`EUEInvoiceSettings`) — not the micro-label role.
- **Secondary / context text** (~212 `text-*/muted-foreground` sites) — already
  consistent; no primitive (would be pure migration noise, per the brief).
- **`ExecutiveSummary.jsx:368`** — an `<h2>` section heading rendered in the
  micro-label style; migrating it would change heading size (section-title
  territory, deferred) — left as a heading.
- **`Landing.jsx:81`** — the marketing landing page (not the operator dashboard)
  and a branded `text-success` eyebrow.
- **`PortalDashboard.jsx:375`** — accent `text-primary` coordinated with its
  primary-tinted referral card (meaningful colour).
- **`Sidebar.jsx:63` & `SettingsLayout.jsx:87`** — navigation chrome (off-limits;
  Batch F territory).
- **Section-title size split (#13)** and **shadow ladder (#15)** — out of scope.

### 12. Tests / lint / build / CI

- `npm run lint` → clean.
- `npm run build` → clean.
- `npx vitest run` → **711 passed (711)**, 0 skipped (705 + 6 new `<Overline>` tests).
- No page test regressed (label text is still queryable; semantics unchanged).

### 13. Bugs found / fixed

- **Inventory undercount** — the design doc's 65 was 1 short; `BuyGiftModal.jsx`'s
  micro-label sat between the primitive-scope and inline-scope greps. Caught
  during migration and included; true count 66.
- No functional/product bugs surfaced (typography-only change).

---

**Stop.** Batch C is complete and self-contained. Not starting Batch D. No new
audit. The uppercase micro-label role is consolidated onto one primitive without
adding a family of specialised label components.
