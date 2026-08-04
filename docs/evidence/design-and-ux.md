# Evidence — design-and-ux

> Code-cited findings extracted by repository inspection (reverse-engineered
> from the implementation) on 2026-08-04. This is the evidence base for the
> corresponding source-of-truth doc(s). Every claim cites the file that
> proves it; implementation wins. Confidence is High unless a line is marked
> `ASSUMPTION`. Regenerate by re-running the inspection agents.

I have everything needed. Here is the code-cited report.

---

# Recurso Frontend — Design System & UX Reverse-Engineering Report

Root: `/Users/swapnull/Documents/Workspace/reccurso/recur-so/frontend`. Stack confirmed: React + Vite + shadcn/Radix + Tremar + react-query (`package.json`, `CLAUDE.md`).

---

## 1. DESIGN TOKENS — what IS implemented

**Accent = emerald, confirmed.** `src/index.css:20-21` sets `--primary: 161 94% 30%` ("Deep emerald for interactive elements; #10B981 stays for charts"). `tailwind.config.js:28-31` maps `primary.DEFAULT → hsl(var(--primary))`. The `primary-hover: "#059669"` (emerald-600) legacy token exists at `tailwind.config.js:59`.

**CSS-variable token system** (`src/index.css:11-37`) — all shadcn semantic tokens defined as HSL triples on `:root`: `--background` (0 0% 100%), `--foreground` (24 10% 10%, warm stone), `--card`, `--popover`, `--secondary`/`--muted`/`--accent` (60 5% 96%), `--destructive` (0 72% 51%), `--border`/`--input` (20 6% 90%), `--ring` (161 94% 30%, emerald), `--radius: 0.5rem`. Tailwind consumes them at `tailwind.config.js:22-55`.

**Semantic colors** live in `src/components/ui/badge.jsx:12-17`: `success` = emerald-50/700 ring, `warning` = amber-50/700 ring, `destructive` = red-50/700 ring. `StatCard` (`src/components/patterns/StatCard.jsx:35-44`) uses emerald-600 (positive), red-600 (negative/danger), amber-600 (warning).

**Tremor tokens** (`tailwind.config.js:70-93`) mapped to emerald brand + zinc neutrals.

**Safelist** (`tailwind.config.js:137-143`): regex `^(bg|text|border|ring|stroke|fill)-(emerald|zinc|red|amber|blue|violet)-(50…900)$` with `hover`/`ui-selected` variants — keeps Tremor's dynamic chart/badge colors from being purged.

**Dark mode:** `darkMode: ["class"]` is set (`tailwind.config.js:7`) but the header comment (`tailwind.config.js:5-6` and `src/index.css:8`) states dark mode is **intentionally disabled** — "Light-only: no `.dark` overrides are defined." Body is hard-pinned `bg-stone-50` (`src/index.css:50`).

### Violations / notes
- **Dead `dark:` classes.** `src/App.jsx:109` and `:122` still reference `dark:bg-stone-950` / `bg-background-dark` despite dark mode being unreachable — stale.
- **Legacy token block** (`tailwind.config.js:56-68`, `background-light/dark`, `surface-*`, `text-light-*`) kept alive for "not-yet-redesigned pages" — a migration debt marker.

---

## 2. TYPOGRAPHY — what IS implemented

- **Font stack = Inter → system fallback.** `tailwind.config.js:96-99`: `sans`/`display` both `["Inter","ui-sans-serif","system-ui","sans-serif"]`.
- **tabular-nums for money:** globally on every `td` (`src/index.css:69-71`), on the `.money` class (`src/index.css:74-79`), on the chart tooltip value (`ChartTooltip.jsx:43`), and pagination counters (`DataTable.jsx:172`).
- **Monospace for money/codes:** `.money` uses `ui-monospace, "SF Mono", "Cascadia Mono", "JetBrains Mono", Menlo` (`src/index.css:75`); `kbd` uses `ui-monospace, Menlo` (`src/index.css:86`). IDs are truncated (not mono-styled) via `shortId()` (`src/lib/utils.js:94-96`).

---

## 3. COMPONENT LIBRARY — what IS implemented

**`src/components/ui/` (shadcn/Radix), 19 components:** avatar, badge, button, card, command-palette, confirm-dialog, ConsentCheckbox, dialog, dropdown-menu, input, label, money, select, separator, sheet, sonner (toasts), table, tabs, tooltip.

**`src/components/patterns/`:** CustomerSelect, DataTable, EmptyState, EntityScopeSelect, ErrorState, FormField, LoadingSkeleton, PageHeader, ProviderGuide, ReportScopeSelect, StatCard (barrel export `patterns/index.js:3-9`).

**Verification of the named set:** DataTable ✓, PageHeader ✓, StatCard ✓, EmptyState ✓, ErrorState ✓, LoadingSkeleton ✓ (exports `Skeleton`, `TableSkeleton`, `CardGridSkeleton`). **ConfirmDialog ✓ but lives in `ui/confirm-dialog.jsx`, not `patterns/`** and is not re-exported by `patterns/index.js`. There is **no dedicated `ErrorState`-style page-level "LoadingSkeleton" mismatch** — all present.

**Sheet pattern:** `src/components/ui/sheet.jsx:35` — right side default is `w-3/4 … sm:max-w-md`. Detail/create views (12 slide-overs in `src/components/slide-overs/`) use it; `sm:max-w-md` appears in 22 files across components/pages. ConfirmDialog also uses `sm:max-w-md` (`confirm-dialog.jsx`).

---

## 4. CHARTS — what IS implemented

Single shared chart module: `src/components/charts/ChartTooltip.jsx`.
- `makeChartTooltip(valueFormatter)` (`:18-54`) — the shared premium tooltip (elevated card, per-series colored dot from recharts payload, right-aligned tabular value). Filters null series to avoid NaN rows (`:23`).
- `chartCategoryColors = ["emerald","blue","amber","violet","red"]` (`:61`) — restricted to safelisted names, emerald-first.
- `chartDefaults` (`:84-87`): `showAnimation: motionOK`, `animationDuration: 900`.
- **Visibility-gated animation (the recent fix):** `motionOK` (`:77-82`) is true only when `document.visibilityState === "visible"` **and** the user hasn't set `prefers-reduced-motion`. The comment (`:70-76`) explains the bug: Chrome freezes rAF in hidden/occluded windows, so charts booted in a background tab froze at frame-1 (sub-pixel bars). Hidden boot now renders full-height immediately.

**Tremor consumers (6 pages):** Usage, DunningDashboard, RevenueWaterfall, Dashboard, AskAnalytics, ExecutiveSummary (grep `@tremor/react`). `chartDefaults`/`makeChartTooltip` are consumed by those same chart pages.

---

## 5. MONEY FORMATTING — what IS implemented (`src/lib/utils.js`)

- `currencyDecimals(currency)` (`:17-26`) — derives exponent from `Intl.NumberFormat(...).resolvedOptions().maximumFractionDigits`; falls back to 2. Not hardcoded `/100`.
- `fromMinorUnits(amountMinor, currency)` (`:32-34`) and `toMinorUnits(amount, currency)` (`:40-43`) — currency-exponent-aware (4200 JPY→4200, 4200 KWD→4.2).
- `formatCurrency(amountMinor, currency)` (`:50-55`) — Intl currency format on the major value.
- `formatCurrencyHeadline` (`:64-72`) — drops `.00` on whole amounts for KPI tiles, keeps non-zero cents; explicitly warns table cells to keep `formatCurrency`/`Money` for alignment.
- `Money` component (`src/components/ui/money.jsx:6-26`) — renders `formatToParts` with the currency symbol in a muted `.money-symbol` span.
- Also `formatNumber`, `formatDate`, `shortId` (`:77-96`).

---

## 6. PAGE INVENTORY

- **`src/pages/` top-level: 60 page files** (`ls` = 63 entries minus `__tests__/`, `portal/`, `settings/` subdirs).
- **`src/pages/portal/`: 6** — CustomerPortal, PortalDashboard, PortalLogin, PortalPaymentMethod, PortalRedeem, PortalVerify.
- **`src/pages/settings/`: 9** — BillingSettings, EntitiesSettings, EUEInvoiceSettings, GSTSettings, InvoiceBranding, IRPSettings, MCPSettings, TaxNexusSettings, USTaxSettings.
- **Total ≈ 75 page components.**

**Routing (`src/App.jsx`):** All pages **lazy-loaded** via `lazy(() => import(...))` (`:22-96`) except `Login` (eager, `:4`) which is on the critical path. Wrapped in `<Suspense fallback={<PageFallback/>}>` (`:124`, spinner). Public routes: `/login`, `/register`, `/forgot-password`, `/reset-password`, `/verify-email`, `/accept-invite`, `/checkout/:id`, `/portal/*` (`:126-140`). Protected routes nested under `<PrivateRoute>` → `<DashboardLayout>` (`:143-211`). Fallback `*` → redirect to `/` (`:214`).

---

## 7. STATE MANAGEMENT — react-query (ADR-005)

**Shared client** `src/lib/queryClient.js:8-17`: `staleTime 60_000`, `gcTime 5×60_000`, `refetchOnWindowFocus:false`, `retry:1` — comment ties it to the rate-limit incident.

### AUDIT — pages that BYPASS react-query (hand-rolled `useState`+`useEffect`+fetch, no `useQuery`/`useMutation`). Verified by set-difference of the two greps:

| Page | Bypass? | Error state RENDERED? |
|---|---|---|
| Metering (`pages/Metering.jsx`) | ✅ yes | partial (has catch; local error) |
| Usage (`pages/Usage.jsx`) | ✅ yes | yes (catch → state) |
| AuditLog (`pages/AuditLog.jsx`) | ✅ yes | **yes** — passes `error` to DataTable (`:34,105-106`) |
| Organizations (`pages/Organizations.jsx`) | ✅ yes | partial |
| OfflinePayments (`pages/OfflinePayments.jsx`) | ✅ yes | partial |
| Integrations (`pages/Integrations.jsx`) | ✅ yes | partial |
| CancelFlows (`pages/CancelFlows.jsx`) | ✅ yes | partial |
| Events (`pages/Events.jsx`) | ✅ yes | partial |
| Checkout (`pages/Checkout.jsx`) | ✅ yes (public) | yes |
| Churn (`pages/Churn.jsx`) | ✅ yes | partial |
| Wallets (`pages/Wallets.jsx`) | ✅ yes | partial |
| FinanceReconciliation (`pages/FinanceReconciliation.jsx`) | ✅ yes | partial |
| AskAnalytics (`pages/AskAnalytics.jsx`) | ✅ yes (AI streaming) | yes |
| settings/TaxNexusSettings | ✅ yes | partial |
| settings/BillingSettings | ✅ yes | **NO error render** — only `loading` state (`:79,103`), no catch/ErrorState |
| portal/CustomerPortal | ✅ yes (dead? no data calls) | n/a |
| portal/PortalDashboard | ✅ yes | yes |
| portal/PortalPaymentMethod | ✅ yes | yes |
| portal/PortalVerify | ✅ yes | minimal |
| portal/PortalLogin, PortalRedeem | ✅ yes | minimal |

**Correction to the task list: `Security` (`pages/Security.jsx`) does NOT purely bypass** — it uses **both** `useQuery` and `useEffect` (appears in both grep lists), so it is react-query-backed. Same for other dual-use pages (Customers, Dashboard, Invoices, Ledger, Plans, Settings, Subscriptions, Team) whose `useEffect` is for non-fetch concerns.

**Worst offender:** `settings/BillingSettings.jsx` — bypasses react-query AND has no error path at all.

---

## 8. UX STATES AUDIT

**DataTable built-in states** (`src/components/patterns/DataTable.jsx:91-104`): error → `<ErrorState message onRetry>`; else loading → `<TableSkeleton rows=6>`; else empty (`data.length===0`) → `<EmptyState>`; else the table. So **any page routing data through DataTable gets error/loading/empty for free** (e.g. AuditLog).

**Pattern components:** `ErrorState` (retry button, `ErrorState.jsx:33-37`), `EmptyState`, `TableSkeleton`/`CardGridSkeleton` (`LoadingSkeleton.jsx`), `StatCard` has a `loading` skeleton (`StatCard.jsx:56-59`). ConfirmDialog has a `busy` state (`confirm-dialog.jsx:19`).

**Gap:** bypass pages that don't render through DataTable (BillingSettings, several portal pages, TaxNexusSettings custom tables) hand-roll only `loading` and often swallow errors in `catch` without an `ErrorState` render.

---

## 9. ACCESSIBILITY — what IS implemented

- **91 `aria-label`s** across pages/components (grep) — icon buttons are labeled.
- **DataTable keyboard nav:** clickable rows get `tabIndex={0}`, `role="button"`, `onKeyDown` handling Enter/Space, and `focus-visible:ring-2 ring-inset ring-ring` (`DataTable.jsx:124-139`).
- **Dialog focus/keyboard:** Radix Dialog underlies `confirm-dialog.jsx` (focus trap, Esc). Sheet is Radix Dialog too.
- **Status is not color-only:** `Badge` variants pair color with a text label (`badge.jsx:12-17`) — success/warning/destructive always carry text, not just hue. StatCard pairs delta color with an arrow icon (`StatCard.jsx:44`).

### Violations
- Native `<table>` pages (Team, Security, DunningDashboard, etc.) bypass DataTable's row keyboard/focus affordances — no `tabIndex`/`role`/`onKeyDown` on those rows.

---

## 10. RESPONSIVE AUDIT

DataTable wraps in `<Card className="overflow-hidden">` (`DataTable.jsx:91`) — clips, no horizontal scroll, but it's the sanctioned pattern.

**Native `<Table>` pages WITHOUT an `overflow-x-auto` wrapper** (confirmed — these 18 files import `ui/table` directly; only 6 files use `overflow-x-auto`: ImportData, RevenueWaterfall, Collections, TrialBalance, MonthEndClose, AskAnalytics, Entities):
- **Team** (`pages/Team.jsx:122` bare `<Table>`) — no overflow wrapper. **Violation.**
- **Security** (`pages/Security.jsx:403`) — bare `<Table>`. **Violation.**
- **DunningDashboard** (`pages/DunningDashboard.jsx:272`) — bare `<Table>`. **Violation.**
- **RevenueRecognition** (`pages/RevenueRecognition.jsx:182`) inside `<Card className="overflow-hidden">` (`:166`) — clips wide tables, no scroll. **Violation.**
- **FinanceReconciliation** (`pages/FinanceReconciliation.jsx:187`) inside `overflow-hidden` Card (`:177`) — the clip you flagged. **Violation.**

`overflow-x-auto` wrappers exist only in ImportData, RevenueWaterfall, Collections, TrialBalance, MonthEndClose, AskAnalytics, Entities, and `slide-overs/PlanDetail.jsx`.

---

## 11. VIOLATIONS — house-standard breaches

**Oversized components (verified line counts):**
- `components/slide-overs/SubscriptionDetail.jsx` — **1011 lines** ✅ (largest in repo).
- `pages/Developers.jsx` — **948 lines** ✅.
- `components/slide-overs/PlanCharges.jsx` — **777 lines** ✅.
- Also large: `Integrations.jsx` (691), `Security.jsx` (669), `CustomerDetail.jsx` (687), `Metering.jsx` (627), `Dashboard.jsx` (615), `portal/PortalDashboard.jsx` (569), `Wallets.jsx` (561), `AskAnalytics.jsx` (532), `OfflinePayments.jsx` (525), `InvoiceDetail.jsx` (650), `PlanDetail.jsx` (532).

**`console.*`:** 9 real calls (`grep`), **all `console.error` in error handlers** — none are stray `console.log`. Files: Referrals:72, Quotes:65/74, CreateCreditNote:70, Gifts:74, CreateSubscription:116, Developers:242, Settings:70/92, plus legit `ErrorBoundary.jsx:16`. (The `IntegrationConnections.jsx:56` "console." grep hit is a false positive — a UI copy string "admin console.".) Low severity, but house standard from `CLAUDE.md` prefers no stray console.

**`eslint-disable exhaustive-deps`:** **0 occurrences** in pages/components — clean.

**Tables without pagination:** DataTable's `pagination` prop is optional (`DataTable.jsx:38,156`). All native-`<Table>` pages (Team, Security, DunningDashboard, RevenueRecognition, FinanceReconciliation, TrialBalance, MonthEndClose, Entities, Developers, etc.) render unpaginated — combined with the backend's inconsistent default limits (`CLAUDE.md`, "silent truncation"), these can silently show partial data.

---

## 12. TEST COVERAGE

**`src/pages/__tests__/`:** dedicated tests exist for most top-level pages plus `PageSmoke.test.jsx`.

**Pages WITHOUT a dedicated test** (`comm` set-difference):
AcceptInvite, CancelFlows, CreateCoupon, CreateCreditNote, CreatePlan, CreateQuote, ExecutiveSummary, Integrations, Ledger, Profile, RevenueRecognition, Security.

**PageSmoke glob** (`pages/__tests__/PageSmoke.test.jsx:66`): `import.meta.glob(["../*.jsx", "../settings/*.jsx"])` — smoke-mounts every top-level page and every settings page. **Notably excludes `../portal/*.jsx`** — portal pages get no smoke coverage.

**Settings tests:** only `InvoiceBranding.test.jsx` and `USTaxSettings.test.jsx` are dedicated (`pages/settings/__tests__/`); the other 7 settings pages rely on PageSmoke only. BillingSettings has a dedicated test in `pages/__tests__/BillingSettings.test.jsx`.

**Portal tests:** only `pages/__tests__/PortalDashboard.test.jsx`. The other 5 portal pages (CustomerPortal, PortalLogin, PortalPaymentMethod, PortalRedeem, PortalVerify) have **no test and are excluded from PageSmoke's glob** — a real coverage hole.

**Slide-overs:** SubscriptionDetail and PlanCharges have dedicated tests (`components/slide-overs/__tests__/`), but the other 10 slide-overs do not.

---

## Recommendation priorities (audit-derived)
1. **BillingSettings** — add error rendering (only bypass page with zero error path).
2. **Portal pages** — add react-query or at least error states; add tests + fold into PageSmoke glob (currently untested + unglobbed).
3. **Responsive** — wrap Team/Security/DunningDashboard native tables and the two `overflow-hidden` finance tables in `overflow-x-auto`.
4. **Decompose** SubscriptionDetail (1011), Developers (948), PlanCharges (777).
5. **Migrate** the 15+ react-query-bypass pages onto `useQuery` per ADR-005 to get caching + DataTable's free error/loading/empty states.
6. Remove dead `dark:`/`bg-background-dark` classes in `App.jsx` and the legacy token block once redesign is complete.