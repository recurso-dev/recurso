# Recurso Dashboard — Quality Audit

> **Read-only, code-cited, point-in-time (2026-08-15).** Audit of the shipped
> React dashboard (`frontend/`) against the source-of-truth docs (`docs/DESIGN.md`,
> `UX_RULES.md`, `ANTI_PATTERNS.md`, `ACCOUNTING_PRINCIPLES.md`, `BRAND.md`,
> `DASHBOARD_PRINCIPLES.md`, `QUALITY_BAR.md`, `MOTION.md`). Method: 4 parallel
> read-only code audits (IA, DataTable, financial-UX/backend, states/a11y/tests)
> + live QA on `app.recurso.dev` (test-mode tenant). No production code was
> modified. Supersedes `frontend/DASHBOARD_AUDIT.md`, whose P0/P1 clusters have
> since shipped (noted in §Current State).

---

# Executive Summary

Recurso's dashboard is **already good** and, in its strongest pages, genuinely at
the target bar. The prior "quality transformation" cleared the dangerous defects:
the 100× money-misread P0 is fixed (`RevenueRecognition.jsx:65-68`,
`FinanceReconciliation.jsx:56-65`), the react-query/ADR-005 drift is largely
resolved, native tables now scroll responsively at the primitive
(`ui/table.jsx:6`), destructive money confirms are amount-anchored, and the code
is clean of `dark:` variants, off-scale spacing, unlabeled icon buttons, and
decorative motion.

The **InvoicePage is the reference implementation** — it answers all ten of the
QUALITY_BAR questions (identity, state, *why* it's past due with the retry
schedule, full amount breakdown, the journal entries it posted to the ledger,
linked customer/subscription, next action). **Ledger, AccountPage, Reconciliation,
and the exception-first Home** are all strong. Money is exponent-correct
everywhere; the reconciler explains every discrepancy in words.

**The ceiling now is not correctness or prettiness — it is depth, navigability,
and the table layer.** Five themes remain:

1. **The table layer under-serves operators.** DataTable is a strong presentational
   component but lacks selection, bulk actions, in-component filtering, sticky
   headers, and URL-state — 7 of the 11 QUALITY_BAR "operational surface"
   capabilities. Worse, **~12 unbounded money lists silently drop rows past 50**
   (credit notes, wallets, coupons, quotes…), which violates "never hide accounting
   numbers."
2. **The accounting end of the object chain is not addressable.** Payment,
   Journal Entry, and Reconciliation-run have no stable URL and no object page —
   all three are **BACKEND GAPs** (no single-read endpoints). The chain
   Customer → Subscription → Invoice → **Payment → Journal Entry → Reconciliation**
   dead-ends exactly where the accounting-first promise should be strongest.
3. **Subscription — the revenue-driving object — is the weakest gold page.** It
   shows no MRR/financial summary and no ledger postings; it fails the "what is the
   accounting impact?" question.
4. **⌘K indexes routes, not objects.** An operator cannot jump to a customer,
   invoice, or subscription by name — the single biggest "operating system feel"
   gap. Verified live: searching a real customer ("Initech") returns "Nothing
   matches."
5. **List state is disposable.** `useUrlState` exists and is proven on Customers,
   but every other list holds state in `useState`, so returning from a detail
   resets filters/search/page.

None of these require a rewrite. Every fix is a refinement of an existing shared
pattern (`DataTable`, `useUrlState`, the object-page framework, the command
palette) or a surgically-scoped backend read endpoint. The highest-leverage moves
improve one shared abstraction and propagate — not one-off page edits.

---

# Current State

**Architecture (unchanged, healthy).** React 19 + Vite (rolldown), shadcn/Radix +
Tremor, react-query 5 (ADR-005), 89 route-level `lazy()` splits under one
`Suspense` (`App.jsx:146`), charts isolated into their own chunk. Object-page
framework: `ObjectHeader` / `ObjectPageLayout` / `AttributeList` / `RelatedRow`
plus `FinancialSummary` / `JournalEntries` / `PaymentAttempts` / `AuditTrail` /
`ObjectTimeline`. Table pattern: `patterns/DataTable.jsx`. List-state hook:
`lib/useUrlState.js`. Motion system: complete (7 phases, `MOTION.md`), CSS + rAF,
reduced-motion-gated.

**Fixed since `frontend/DASHBOARD_AUDIT.md` (verified in current code):**

- **P0 money-misread — FIXED.** `RevenueRecognition.jsx:65-68`,
  `FinanceReconciliation.jsx:56-65` now route through exponent-aware
  `formatCurrency`; multi-currency renders "Multiple currencies", not a raw sum.
- **ADR-005 drift — mostly FIXED.** Metering, Usage, Organizations,
  OfflinePayments, Integrations, CancelFlows, Events, Churn, Wallets,
  TaxNexusSettings and **BillingSettings** (`settings/BillingSettings.jsx:83,118-121`)
  are on react-query with error+retry.
- **Native-table overflow — FIXED at the primitive** (`ui/table.jsx:6` wraps every
  `<Table>` in `overflow-auto`); raw-`<table>` object pages each add their own
  `overflow-x-auto` (`QuotePage.jsx:293`, `AccountPage.jsx:170`, `WalletPage.jsx:384`).
- **Amount-anchored confirms — SHIPPED** for credit-note approve/void
  (`CreditNotePage.jsx:344,363`), write-off (`Collections.jsx:176`), wallet close
  (`WalletPage.jsx:573`), plan change (`SubscriptionPage.jsx:566-587`).
- **Ledger/UUID navigability — PARTLY SHIPPED:** ledger-page postings link to
  accounts (`Ledger.jsx:32-43`); Invoices/Usage customer cells resolve to name+link;
  `SubscriptionRef` links plan names.
- **URL state — STARTED:** `Customers.jsx` fully migrated to `useUrlState`.
- **Clean sweeps:** zero `dark:` variants; zero off-scale `[Npx]` spacing; all
  icon-only buttons carry `aria-label`; `MotionNumber`/`ChartTooltip` gate on
  `useReducedMotion`.

**Live QA confirmed (test-mode tenant):** exception-first Home renders a "Needs
attention" strip (reconciliation discrepancies, overdue invoices with per-currency
totals, churn) before KPIs; InvoicePage shows the full breakdown + failure reason
+ journal entries; Reconciliation labels every discrepancy with what/how-much/why;
multi-currency ($/₹) renders correctly throughout.

---

# P0 Issues

> Fix first. By the product's own rule (`ANTI_PATTERNS.md` "Never hide accounting
> numbers. If a figure exists, it's shown"), hiding financial rows is a P0 — not
> a display nicety.

### P0-1 · ~12 unbounded money lists silently truncate at 50 rows
- **Current behavior:** these pages call their list endpoint with **no `limit`** and
  render **no pagination UI**, so they hit the tier-1 default (`limit=50`, per
  `CLAUDE.md`) and silently drop rows 51+: `Coupons.jsx:33`, `CreditNotes.jsx:25`,
  `Quotes.jsx:45`, `Gifts.jsx:56`, `Wallets.jsx:63`, `CancelFlows.jsx:31`,
  `DunningCampaigns.jsx:45`, `Referrals.jsx:54`, `Organizations.jsx:45`,
  `Churn.jsx:34`, `Metering.jsx:72`, `OfflinePayments.jsx:62/85/98`, plus
  `Usage.jsx:44` (limit 50, no pager).
- **Problem:** an operator sees a complete-looking list that is actually the first
  50 rows. On financial objects (credit notes, wallets, coupons, offline payments)
  this hides money that exists.
- **User impact:** wrong totals, missed credit notes/wallets, "where did that
  record go?" — the exact trust failure `ANTI_PATTERNS.md` forbids.
- **Recommended solution:** give each list a real pagination contract by wiring it
  to the DataTable pagination the component already renders
  (`{page,pageSize,total}`), or an Invoices-style page-through fetch for
  bounded-small sets. Standardize on one PAGE_SIZE. This is pattern work, not 12
  one-offs — see "Reusable Components".
- **Priority:** P0. **Affected:** the 12 pages above + `DataTable.jsx` (pagination
  contract). **Dependencies:** none (frontend-only; backends already paginate).

*(No other P0 remains. The prior money-misread P0 is fixed; no wrong-value or
required-state failure survives on in-shell operator pages — a real result of the
prior initiative, stated honestly.)*

---

# P1 Issues

### P1-1 · ⌘K command palette indexes routes only, not objects
- **Current:** `command-palette.jsx:14-30` indexes `ALL_DESTINATIONS` (nav + aux
  routes) + static Create/Help actions; filtering is a substring match over static
  labels (`:39-43`). **Verified live:** ⌘K → "Initech" (a real customer) →
  "Nothing matches 'Initech'."
- **Problem:** every object has a URL, but there is no way to reach one except
  drilling through a list. That is the defining capability of an "operating system."
- **Impact:** slow operation; the ⌘K affordance implies object search and doesn't
  deliver it.
- **Solution:** add object results to the palette — customers, invoices,
  subscriptions, plans by name/number/id. Ideally a lightweight backend search
  (see BACKEND GAP-6); interim, search the already-cached react-query lists client-side.
- **Priority:** P1. **Affected:** `command-palette.jsx`, `useCustomers`/list hooks.
  **Dependencies:** best with GAP-6; usable without it.

### P1-2 · Subscription is the weakest gold page — no financial summary, no accounting impact
- **Current:** `SubscriptionPage.jsx` shows amount/mo, upcoming invoice, accrued
  usage, lifecycle, and its invoices — but imports **neither `FinancialSummary` nor
  `JournalEntries`** (`:10-43`). No MRR anywhere; ledger is reachable only by hopping
  through an invoice.
- **Problem:** the object that drives recurring revenue can't answer "how much is it
  worth?" (MRR) or "what did it post?" (Q5/Q9 of the quality test).
- **Impact:** operators can't assess a subscription's financial state or verify its
  accounting without leaving the page.
- **Solution:** add `FinancialSummary` (MRR/next-invoice/lifetime-billed) + per-sub
  `JournalEntries`. Journal-entries-from-its-invoices is doable **now**; MRR needs
  BACKEND GAP-4.
- **Priority:** P1. **Affected:** `SubscriptionPage.jsx`, `FinancialSummary.jsx`,
  `JournalEntries.jsx`. **Dependencies:** MRR portion → GAP-4.

### P1-3 · DataTable selection + bulk actions — SHIPPED (Batch 2), partially
- **Shipped:** `DataTable.jsx` now has opt-in `selectable` / `selectedIds` (a Set) /
  `onSelectionChange` + a `renderBulkActions` bar. Semantics are deliberately
  **page-scoped** — the header checkbox is "select all rows on this page" (labeled
  as such), never "all matching records" (no backend supports that); selection is
  pruned to the current result set on page/filter/search change and retained on a
  same-page refetch. Row checkboxes `stopPropagation` so selecting never navigates;
  indeterminate state via ref; keyboard-native `Checkbox`.
- **Observable bulk runner** (`lib/useBulkAction.js` + `patterns/BulkActionDialog.jsx`):
  confirm (states the scope, e.g. "Send 24 invoices") → progress (n/total) → result
  as a first-class `all_succeeded` / `partial` / `all_failed` state; failed rows stay
  listed and retryable, retry re-runs only the failed ids reusing a per-record
  `Idempotency-Key` (added to `sendInvoice`/`rejectCreditNote`) so a retry can't
  double-act.
- **Safe consumers shipped:** Invoices → bulk **Send** (non-money, resendable),
  Credit Notes → bulk **Reject** (idempotent, no money moves).
- **Deliberately NOT shipped as bulk** (bulk-API audit): credit-note *approve*
  (refund-type is irreversible gateway money), write-off / mark-uncollectible
  (irreversible books), **all Dispute outcomes** (per-case judgment, irreversible).
  Collections bulk pause/resume + retry-now are safe and high-value but need
  Collections migrated from its hand-rolled table to DataTable first — **follow-up**.
- **BACKEND GAP:** there is **no batch/multi-id endpoint** anywhere (only single-record
  POSTs, looped client-side with per-record idempotency). A genuinely atomic
  all-or-nothing bulk op (e.g. bulk refund) would need a real transactional batch
  endpoint. Also missing: invoice-void, invoice-finalize, un-write-off, dispute
  un-resolve, dispute submit-evidence, collections snooze/mark-contacted.

### P1-4 · List context is lost on back-navigation everywhere but Customers
- **Current:** `useUrlState.js` is clean and proven, but **only `Customers.jsx`
  adopts it**. Subscriptions (`:29-33`), Plans (`:29-32`), Invoices (`:86-87`),
  Payments (`:59-60`), Coupons, Disputes, Quotes, Events, AuditLog all hold
  page/search/filter in `useState`.
- **Problem:** open a detail, hit Back → filters/search/page reset to defaults.
- **Impact:** operators re-filter constantly; deep links to a filtered view aren't
  shareable.
- **Solution:** let DataTable optionally bind list state to the URL via `useUrlState`
  so migration is one prop, then propagate — gold-standard lists first.
- **Priority:** P1. **Affected:** `DataTable.jsx`, `useUrlState.js`, list pages.
  **Dependencies:** none.

### P1-5 · Immediate subscription cancel has no money preview
- **Current:** the cancel dialog (`SubscriptionPage.jsx:1130-1184`) offers a "cancel
  immediately" checkbox (`:1161-1169`) and posts `immediately:true` (`:339`) with
  **no proration/refund preview** — the only destructive money action lacking an
  amount. (Plan-change already previews proration at `:566-587` — the ready pattern.)
- **Problem:** violates DASHBOARD_PRINCIPLES "communicate amount affected / resulting
  state / accounting impact" for consequential actions.
- **Impact:** operator cancels blind to the refund/credit consequence.
- **Solution:** show a proration/refund preview in the dialog when "immediately" is
  chosen, reusing the plan-change preview UI. Needs BACKEND GAP-5.
- **Priority:** P1. **Affected:** `SubscriptionPage.jsx`. **Dependencies:** GAP-5.

### P1-6 · Payment, Journal Entry, and Reconciliation-run are not addressable objects
- **Current:** `DASHBOARD_PRINCIPLES.md:44-49` requires `/payments/:id`,
  `/journal-entries/:id`, `/reconciliation/:id`. None exist. Payment rows redirect
  to the invoice (`Payments.jsx:180`); a journal posting opens a Sheet, not a URL
  (`Ledger.jsx:382`); reconciliation runs render in a table with no per-run URL
  (`FinanceReconciliation.jsx:359-410`).
- **Problem:** the chain dead-ends at the accounting layer — you can't link,
  bookmark, or share a payment / journal entry / recon run.
- **Impact:** the accounting-first promise is weakest exactly where it should be
  strongest.
- **Solution:** build the three object pages on the existing framework — **after**
  the backend read endpoints land (GAP-1/2/3). Do **not** fabricate them.
- **Priority:** P1 (backend-blocked). **Affected:** new pages + `App.jsx` routes.
  **Dependencies:** GAP-1/2/3.

### P1-7 · Home attention strip omits failed payments and failed webhooks
- **Current:** the "Needs attention" strip (`Dashboard.jsx:388-461`) surfaces
  reconciliation discrepancies, overdue invoices, disputes, and churn — but not
  **failed payments** or **failed webhooks**, both named in
  `DASHBOARD_PRINCIPLES.md:61-64` and both one query away
  (`getPaymentAttempts?status=failed` `api.js:149`; events/webhooks via `/events`).
- **Problem / impact:** the two most operationally-urgent exceptions for a billing
  operator aren't on the attention surface.
- **Solution:** add failed-payment and failed-webhook tiles to the strip.
- **Priority:** P1. **Affected:** `Dashboard.jsx`. **Dependencies:** none.

### P1-8 · Invoice and Subscription journal legs don't link to their ledger accounts
- **Current:** `InvoicePage.jsx:583-588` renders `JournalEntries`, but
  `JournalEntries.jsx` has **no `Link`** — debit/credit accounts are plain text.
  The `/ledger` page already solved this (`Ledger.jsx:32-43`).
- **Problem:** from an invoice's postings you can't click through to the account —
  breaking "every figure traces to its postings" (`UX_RULES.md`).
- **Solution:** promote `LedgerAccountLink` to a shared helper and use it inside
  `JournalEntries`. Frontend-only, quick.
- **Priority:** P1 (low-effort). **Affected:** `JournalEntries.jsx`, `Ledger.jsx`
  (extract helper). **Dependencies:** none.

---

# P2 Issues

- **No sticky table headers anywhere** (`DataTable.jsx:236`, `ui/table.jsx` lack
  `sticky top-0`). Long lists scroll their headers away. *Fix once in the shared
  header.*
- **CSV export missing on most financial tables** — only Invoices, Usage,
  TrialBalance, MonthEndClose, AskAnalytics export. Payments, Credit Notes, Wallets,
  Ledger, Subscriptions, Disputes, Collections don't (`UX_RULES.md` requires export
  on financial tables). *Promote a `csvExport` toolbar slot into DataTable.*
- **Pagination-shape divergence** — modern `{page,pageSize,total}` (`Payments.jsx:183`)
  vs legacy `hasNext` (Customers/Subscriptions/Plans) vs `PER_PAGE+1`
  (Disputes/Mandates) vs `len===PAGE_SIZE` (Events/AuditLog/Ledger); PAGE_SIZE
  varies 10/25/50/100. *Standardize on the total-based contract + PAGE_SIZE 50.*
- **~21 hand-rolled tables** duplicate empty/loading/error/a11y (Collections, Team,
  Entities, TrialBalance, MonthEndClose, RevenueRecognition, FinanceReconciliation,
  DunningDashboard, GSTReturns, Developers, Dashboard, settings/*, portal/*). *Migrate
  list/worklist tables to DataTable; columnar reports may keep bespoke layout but
  should route through DataTable states.*
- **Reconciliation → transaction is a dead end** (`FinanceReconciliation.jsx:314-319`
  renders the txn ref as bare `shortId`, no link); non-invoice ledger refs render as
  raw UUID text (`Ledger.jsx:442-443`). *Blocked on a journal-entry URL (GAP-2) for
  full drill; can link known invoice refs now.*
- **Required-state gaps on hand-rolled pages:** `Notifications.jsx:79` (bare-text
  loading) + `:81-88` (error via EmptyState with **no Retry**); `BillingSettings.jsx:114`
  (bare spinner load); `Profile.jsx:66-73` (error banner with **no Retry**, no load
  skeleton).
- **`FinanceReconciliation.jsx:68-104`** — primary report fetch still hand-rolled
  `useEffect` (has error+retry, so a purity drift, not a missing state).
- **`PortalPaymentMethod.jsx` has zero test coverage** and is excluded from PageSmoke
  (`PageSmoke.test.jsx:64`). It handles payment-method/mandate setup — highest-value
  test gap.
- **Bug — `SubscriptionPage.jsx:1070`:** `const last = i === lifecycle.length - 1`
  references the *actions* object `lifecycle` (`.length` undefined) instead of
  `lifecycleHistory`; the timeline connector renders on the final item too.
- **Copy bug — `FinanceReconciliation.jsx:270-271`:** subtitle still reads "Amounts
  are in minor units" though amounts now render as real currency (`:321-327`).
  Verified live. Misleading on a finance screen — remove it.
- **Invoice list row density** — rows are tall (~56px) relative to the density
  principle (`DESIGN.md §2.4`). Optional compact/dense row mode for finance operators.

---

# Design System Gaps

The design system is in **strong** shape — the earlier token/spacing/dark-mode
violations are gone. Residual, all P3:

- **Raw hex confined to chart/widget palettes:** `Dashboard.jsx:62-75` (status +
  aging chart hex arrays — the duplicated `BAR_COLORS` family); `CreateSubscription.jsx:98`
  and `Checkout.jsx:194` (third-party Stripe/Razorpay widget `color`). None are
  semantic UI surfaces. *Route the Dashboard palette through chart tokens; widget
  themes are external-API-bound and acceptable.*
- **No net-new component styles needed.** House library (Button/Card/DataTable/Sheet/
  ConfirmDialog/StatCard/PageHeader/EmptyState/ErrorState/LoadingSkeleton) covers the
  surface. The gaps are *capabilities inside existing components*, not new components
  (see "Reusable Components").
- **Documentation placement inconsistency (meta):** `CLAUDE.md` and the READ list
  point to `docs/DASHBOARD_PRINCIPLES.md`, `docs/QUALITY_BAR.md`, `docs/MOTION.md`,
  but `DASHBOARD_PRINCIPLES.md` + `QUALITY_BAR.md` live untracked in the repo root
  and `MOTION.md` exists in both root (57 lines) and `frontend/` (78 lines, the real
  one). *Consolidate the source-of-truth docs into `docs/` and delete the stray
  copies.*

---

# Information Architecture Gaps

- **Object chain dead-ends at the accounting layer** — Customer → Subscription →
  Invoice work; **Payment → Journal Entry → Reconciliation** don't (P1-6; GAP-1/2/3).
- **Usage is unaddressable** between Subscription and Invoice — `/usage` is a list
  only (`App.jsx:183`), no usage-record URL. Product decision: addressable node vs
  accept inline-on-subscription.
- **Customer → Payments not surfaced** — `CustomerPage.jsx:252-340` links
  subscriptions/invoices/credit-notes/wallets/ledger but not payment attempts.
- **Nav shell is strong** — one canonical `NAV_GROUPS` (`lib/navigation.js:23-114`)
  drives sidebar, mobile drawer, top-bar label, and palette, grouped by operator
  mental model (Billing / Growth / Usage / Revenue Recovery / Payments / Books /
  Reports / System). Skip link + focusable `<main>` present (`DashboardLayout.jsx:77-82,199`).

---

# Object UX Gaps

Object-page section coverage (per `DASHBOARD_PRINCIPLES.md:23-34`) for the 5 gold
objects. ● present · ◐ partial · ✕ absent · — no page.

| Section | Customer | Subscription | Invoice | Payment | Ledger/Journal |
|---|:--:|:--:|:--:|:--:|:--:|
| Identity | ● | ● | ● | — | ◐ (Sheet) |
| State | ● | ● | ● | — | ◐ |
| Financial summary | ◐ (no MRR) | ✕ **no summary** | ● | — | ✕ |
| Relationships | ● | ◐ (no payment/ledger) | ◐ (postings unlinked) | — | ◐ |
| Lifecycle | ◐ | ● | ● | — | ✕ |
| Activity | ● | ● | ● | — | ✕ |
| History | ● | ● | ● | — | ✕ |
| Actions | ● | ● | ● | — | ✕ |
| Technical info | ● | ● | ● | — | ◐ |
| Audit trail | ● | ● | ● | — | ✕ |

**Invoice passes all ten** (reference page). **Customer** strong (add MRR).
**Subscription** is the priority gold work (financial summary + journal — P1-2).
**Payment** and **Journal Entry** are backend-blocked (P1-6).

---

# Financial UX Gaps

- **Money presentation — no correctness violations.** All shared helpers are
  exponent-aware (`utils.js:17-72`); the only `/100` is a test fixture
  (`StatCard.test.jsx:65`); `.toLocaleString()` hits are all on counts, not money.
  Local money formatters (`Wallets.jsx:33`, `WalletPage.jsx:573`, `DunningDashboard.jsx:29`,
  `slide-overs/CustomerDetail.jsx:19`) are exponent-aware **style dialects** (append
  a currency code instead of the shared symbol) — P3 consistency, not bugs.
- **Explainability — strong.** Invoice surfaces *why* it's past due with the retry
  schedule (`InvoicePage.jsx:309-329`, verified live); Reconciliation labels every
  discrepancy with what/how-much/why (`FinanceReconciliation.jsx` DISCREPANCIES map
  `:30-51`, verified live); Ledger's entry-detail Sheet shows both legs + reference
  (`Ledger.jsx:382-457`).
- **Payment failure reason is shown but raw** — `Payments.jsx:138-143` /
  `PaymentAttempts.jsx:85-89` render the gateway `failure_code` with no
  human-readable mapping (a mild "raw code to operator" concern, `ANTI_PATTERNS.md`).
  *Map codes to plain language; ideally on the payment object (GAP-1).*
- **Confirms — mostly best-in-class.** Amount + accounting impact + reversibility
  stated for credit-note approve/void, write-off, wallet close, plan change, dispute
  accept. **One gap:** immediate cancel (P1-5).
- **Accounting impact missing on Subscription** (P1-2) and **not deep-linkable from
  invoice postings** (P1-8).

---

# DataTable Gaps

`DataTable.jsx` capability matrix vs QUALITY_BAR "operational surface":

| Capability | Status |
|---|---|
| Empty / loading / error states | ● (strongest axis, `:221-233`) |
| Sortable columns | ● (`:249-264`, `aria-sort`) |
| Pagination *rendering* | ● (`{page,pageSize,total}` + legacy, `:158-179`) |
| Search (controlled) | ◐ (renders; caller owns state) |
| Row activation / keyboard | ◐ (first cell focusable; no `role="grid"` roving) |
| Row selection (checkboxes) | ● **SHIPPED** (Batch 2) — opt-in, page-scoped, pruned, keyboard/indeterminate |
| Bulk actions | ● **SHIPPED** (Batch 2) — `renderBulkActions` bar + observable runner + partial-failure dialog |
| In-component column filters | ✕ (opaque `toolbar` node per page) |
| Sticky header | ✕ |
| Column config / visibility | ✕ |
| URL-state binding | ◐ caller wires `useUrlState` — now adopted on all major lists (Batches 1–2) |

**Remaining shared-component gaps: in-component filters, sticky header, column
config.** Selection + bulk actions and URL-state adoption are now done. (Silent
truncation, P0-1, is fixed.)

---

# Search / Command Palette Gaps

- **Objects are not searchable** (P1-1) — palette indexes static route labels only
  (`command-palette.jsx:14-43`); verified live "Nothing matches 'Initech'".
- **No recent/frequent objects, no "jump to id"** — an operator's fastest path
  ("go to INV-000009 / this customer") doesn't exist.
- Palette wiring itself is clean (⌘K/Ctrl-K, `DashboardLayout.jsx:48-57`), derived
  from the same `NAV_GROUPS` source of truth — so adding an object-results section
  is additive, not a rebuild.

---

# Motion Gaps

- **System complete** (`MOTION.md`, 7 phases). Adopted across 16 files; primitives
  gate on `useReducedMotion` (`MotionNumber.jsx:26,37` snaps under reduced motion);
  no decorative or count-up-on-money motion (DESIGN.md §8 clean).
- **Remaining is application, not system:** when the Payment / Journal-Entry /
  Reconciliation object pages are built (post-backend), give them the existing
  lifecycle motion (`MotionState` status flash, `JournalEntries` sequential balance,
  reconciliation `MotionNumber → 0`). **No new primitives needed.**

---

# Accessibility Gaps

Posture is **good** — Radix focus management, skip link + focusable `<main>`, icon
buttons labeled, contrast tokens corrected to AA, reduced-motion honored. Residual,
all P3:

- **SelectTriggers without explicit `aria-label`** (they do carry a `SelectValue`
  placeholder): `Plans.jsx:148,160`, `Subscriptions.jsx:211,225,239`, `Quotes.jsx:208`.
- **No `role="grid"` / arrow-key roving** in DataTable — one focus point per row is
  AT-safe but not grid-navigable (lowest priority; current a11y is sound).
- **Public/portal pages** use inline error `catch`, not the ErrorState pattern
  (separate mini-app; acceptable).

---

# Responsive Gaps

- **Table overflow — solved** (primitive `ui/table.jsx:6` + explicit wrappers on
  raw-`<table>` object pages). No native table clips the body.
- **Fixed-width filter Selects** (`w-[130px]`–`w-[220px]`) on Events, Subscriptions,
  Plans, Payments, Usage, Quotes, RevenueRecognition, MonthEndClose, Developers —
  a multi-Select toolbar crowds at 320px but wraps (`flex-wrap`). P3; matches the
  prior audit's "lower the floor to 320px intentionally," not a hard break.
- **Wide raw tables** set `min-w-[560/640px]` and scroll inside their own
  `overflow-x-auto` — acceptable.
- *Note:* the browser-automation screenshot tool captures a fixed viewport, so
  responsive floors (320/375/768) must be validated with real devices / devtools
  during implementation QA — code-level analysis only for this audit.

---

# Performance Gaps

- **Healthy baseline:** 89 `lazy()` route splits under one `Suspense`
  (`App.jsx:146`); charts isolated per `vite.config.js` manual chunking.
- **No list virtualization** — Events/Usage/AuditLog/Ledger render all fetched rows.
  Safe while `limit` stays small; becomes a concern once P0-1's pagination raises row
  counts. *Add a virtualization guard (or keep page sizes modest) when unbounded
  lists get real paging.* P3.
- **Public bare-spinner fallbacks** (`App.jsx:117,128`, `Checkout.jsx:249`) — pre-shell/
  public, low impact.

---

# Testing Gaps

- **`PortalPaymentMethod.jsx` — zero coverage** (no dedicated test, excluded from
  PageSmoke `PageSmoke.test.jsx:64`). Handles payment-method/mandate setup — **P2**,
  highest-value gap.
- **Smoke-covered only (no behavioral test):** AcceptInvite, CreatePlan, CreateQuote,
  ExecutiveSummary, Integrations, Profile, RevenueRecognition, Security, and settings
  `EntitiesSettings/EUEInvoiceSettings/GSTSettings/IRPSettings/MCPSettings`. P3.
- **No visual-regression coverage** anywhere — visual QA is manual. P3 (out of scope
  to add now; note it).
- **New rule for this initiative:** any page that gains an `endpoints.*` call must
  extend its test mock (`CLAUDE.md`), and every money-path change ships a test that
  fails on the old code (`ANTI_PATTERNS.md`).

---

# Backend/API Gaps

> Per `CLAUDE.md` and the charter: **do not fake.** These block concrete UX and must
> ship server-side first. Each verified against `cmd/api/main.go` + `openapi.yaml`.

**GAP-1 · Single payment / payment-attempt read — MISSING (P1).**
Exists: `GET /v1/payment-attempts` (list, `main.go:1847`), `/v1/invoices/:id/payment-attempts`
(`:1845`), `/v1/payments/offline` (`:2123`). Missing: `GET /v1/payment-attempts/{id}`
(or `/v1/payments/{id}`). *Required data:* one attempt + its gateway event trail
(auth/capture/settle/return/refund), settlement timestamps, human-decoded decline
reason, linked invoice/refund/dispute and **Code-3 ledger transaction id**. *Unblocks:*
`/payments/:id` object page, plain-language failure reason, payment→ledger drill.
*Frontend dep:* new `PaymentPage.jsx` + route; `Payments.jsx:180` / `PaymentAttempts.jsx`
retarget to `/payments/:id`.

**GAP-2 · Single journal-entry / ledger-transaction read — MISSING (P1/P2).**
Exists: `/ledger/accounts`, `/ledger/entries` (by `account_id`), trial-balance, export,
deferred-rollforward (`main.go:1983-1988`); per-source `/invoices/:id/journal-entries`,
`/credit-notes/:id/journal-entries`. Missing: `GET /v1/ledger/transactions/{id}` (both
legs of one posting) and/or `GET /v1/ledger/entries?reference_id=…`. *Required data:*
a transaction id → full balanced posting (legs, accounts, code, amount, reference_id +
reference *kind*). *Unblocks:* `/journal-entries/:id`; links for non-invoice ledger
refs (`Ledger.jsx:442-443`); subscription postings (GAP-4). *Interim (now):* invoice
legs/refs can already be linked (P1-8) without this.

**GAP-3 · Per-run reconciliation detail — MISSING (P1).**
Exists: `GET /v1/finance/reconciliation` (ephemeral run, `main.go:1991`),
`POST /runs` (record summary, `:1992`), `GET /runs` (list, `:1993`). Missing:
`GET /v1/finance/reconciliation/runs/{id}` returning the **stored per-run discrepancy
rows** (the store persists only summary counts). *Required data:* each run's discrepancy
rows (type, invoice_id, transaction_id, expected/found/diff). *Unblocks:* `/reconciliation/:id`
history drill (`FinanceReconciliation.jsx:339-411` rows are currently dead ends). The live
run view already explains discrepancies well — only history drill is missing.

**GAP-4 · Per-subscription financial summary / MRR — MISSING (P1).**
Exists: `GET /v1/customers/:id/financial-summary` (`main.go:1830`) — but returns
outstanding/past-due/billed/paid, **not MRR** (`FinancialSummary.jsx:14`);
`/subscriptions/:id/usage-amount` (`:1872`), `/preview-change` (`:1834`). Missing:
`GET /v1/subscriptions/:id/financial-summary` (recurring value/MRR, lifetime billed,
next-invoice amount+date, postings) — and no MRR field on the customer summary either.
*Unblocks:* Subscription financial depth (P1-2); MRR on any object page.

**GAP-5 · Subscription cancel preview — MISSING (P1).**
Exists: `/subscriptions/:id/preview-change` (plan change only, `main.go:1834`). Cancel
posts blind (`POST /subscriptions/:id/cancel`, `:2096`). Missing:
`GET /v1/subscriptions/:id/cancel-preview?immediately=true` → refund/credit amount,
unused-time proration, resulting ledger effect. *Unblocks:* the immediate-cancel money
preview (P1-5). The plan-change preview is the pattern to mirror.

**GAP-6 · Object search endpoint — MISSING (P1, optional).**
No cross-object search endpoint. *Would enable:* ⌘K object search (P1-1) at scale.
*Interim:* client-side search over cached lists works for small tenants; a
`GET /v1/search?q=` (customers/invoices/subscriptions/plans by name/number/id) scales it.

---

# Reusable Components We Should Build

*(Improve existing shared abstractions — do not add parallel ones.)*

1. **DataTable v2 — extend the one component** (covers P0-1, P1-3, P1-4, and several
   P2s at once):
   - Row **selection + bulk-action bar** (`selectable`/`selectedIds`/`onSelectionChange`/`bulkActions`).
   - **Real pagination contract** (`{page,pageSize,total}`) as the default, killing
     silent truncation when lists adopt it.
   - Optional **URL-state binding** (`urlState` prop → internal `useUrlState`), so
     back-nav restores context with one prop.
   - **Sticky header**; **CSV export** toolbar slot; **declarative filter config**
     (`filters:[{key,options,serverSide}]`) to end the pill-vs-Select divergence.
   - (Later) user column visibility; `role="grid"` keyboard layer.
2. **`LedgerAccountLink` → shared helper** (extract from `Ledger.jsx`), used in
   `JournalEntries` so invoice/subscription postings link to `/ledger/accounts/:id` (P1-8).
3. **`SubscriptionFinancialSummary`** — compose from `FinancialSummary` once GAP-4
   lands (P1-2); reuse `JournalEntries` for its postings.
4. **Object-results section in the command palette** (P1-1) — additive to
   `command-palette.jsx`, backed by cached lists (or GAP-6).
5. **Permission/403 state** in `ErrorState`/query-error handling — a shared honest
   state for forbidden resources.
6. **Payment / Journal-Entry / Reconciliation object pages** — only after GAP-1/2/3;
   built on the existing object framework + motion, no new primitives.

# Components We Should NOT Build

- **No new design system, tokens, color, button variant, shadow, or radius** — the
  house library is sufficient; gaps are capabilities *inside* it.
- **No new motion primitives** — the system is complete.
- **No parallel/second table component** — extend `DataTable`, don't fork it.
- **No fabricated Payment / Journal-Entry / Reconciliation pages** before their
  backend reads exist.
- **No vanity metrics** on Home — keep it exception-first; add *attention* tiles, not
  more charts.
- **No column reorder/resize or saved-views** yet — speculative until users ask.
- **No list virtualization yet** — a guard, not a virtual-scroll engine, until page
  sizes actually grow.

---

# Gold-Standard Page Strategy

Solve the *system* on the five gold pages, then propagate — do not redesign every
page.

- **Invoice — DONE. The reference page.** Passes all ten questions. Only refinement:
  link its journal legs (P1-8).
- **Ledger + AccountPage — strong.** Keep as the "explain any number" surface; single
  journal-entry URL awaits GAP-2.
- **Customer — strong.** Add MRR to `FinancialSummary` (GAP-4); optionally surface
  payments.
- **Subscription — the priority gold work (P1-2).** Bring to Invoice depth: financial
  summary (MRR/next-invoice) + journal entries. This is where the reusable "financial
  depth" pattern gets proven for propagation.
- **Payment — backend-blocked (GAP-1).** Build last, on the framework, once the read
  endpoint lands.

Sequence by *leverage × unblocked*: fix the shared table layer and Subscription depth
now (unblocked); file the backend gaps in parallel; build the accounting object pages
when their endpoints ship.

---

# Recommended Implementation Sequence

Each item is its own small green-CI PR (lint + build + vitest, plus keyboard /
responsive 320-1440 / reduced-motion / visual QA). Frontend-only unless noted.

**Batch 1 — Table integrity + quick wins (P0 + high-leverage P1, unblocked):**
1. **Kill silent truncation (P0-1)** by giving `DataTable` the default pagination
   contract and wiring the ~12 offenders to it (pattern-driven, not 12 one-offs).
2. **DataTable URL-state binding + pagination standardization (P1-4)** — one prop;
   migrate the gold-standard lists first.
3. **Link invoice/subscription journal legs (P1-8)** — extract & reuse `LedgerAccountLink`.
4. **Two concrete fixes:** `SubscriptionPage.jsx:1070` lifecycle bug; stale "minor
   units" caption (`FinanceReconciliation.jsx:270-271`).

**Batch 2 — Operator power (P1):**
5. **DataTable selection + bulk-action bar (P1-3)**; opt-in on Invoices, Collections/
   Dunning, Disputes, Credit Notes.
6. **⌘K object search (P1-1)** — client-side over cached lists (backend GAP-6 later).
7. **Home attention tiles (P1-7)** — failed payments + failed webhooks.

**Batch 3 — Subscription depth + confirms (P1, partially backend-gated):**
8. **Subscription journal entries (P1-2, frontend-now)** from its invoices; **MRR/
   financial summary when GAP-4 ships.**
9. **Immediate-cancel money preview (P1-5)** when GAP-5 ships.

**Batch 4 — Consistency & polish (P2/P3):**
10. Sticky headers; CSV export slot; migrate hand-rolled tables; fix Notifications/
    Profile/BillingSettings states; PortalPaymentMethod test; local `fmtMoney` →
    `Money`; SelectTrigger aria-labels; consolidate source-of-truth docs into `docs/`.

**Backend track (parallel):** file GAP-1..6; when the payment / journal-entry /
reconciliation-run reads land, build those three object pages.

---

# Risks

- **DataTable v2 is a shared-surface refactor** — every list depends on it. Mitigate:
  add capabilities additively (new optional props, existing call sites unchanged),
  land behind opt-in, migrate page-by-page, keep the full vitest suite green each PR.
- **Pagination change can alter what a list shows** — moving from "first 50" to real
  paging changes visible rows and totals; verify each migrated list against the
  backend count and keep CSV export covering the full set (Invoices' page-through is
  the precedent).
- **Backend-gated work must not be faked** — if an endpoint slips, the dependent
  frontend (Payment page, MRR, cancel preview) waits; ship the unblocked portion and
  record the gap.
- **Money-path tests** — any change touching a figure needs a test that fails on the
  old code; never weaken the invariant harness or a test to pass CI.
- **Scope creep toward "prettiness"** — optimize for operate-faster-and-safer, not
  screenshots; prefer improving a shared pattern over touching many pages.

---

# Definition of Done

A page is done when an operator can confidently answer the QUALITY_BAR questions —
**what is this, what state, why, what changed, how much money, what it affects, what's
next, what can I do, what's the accounting impact, how do I verify** — and every
change clears these gates:

- `npm run lint && npm run build && npx vitest run` green (no weakened tests/lint).
- Keyboard-operable with visible focus; ⌘K reaches it.
- Responsive at 320 / 375 / 768 / 1024 / 1440 (real-device/devtools QA).
- `prefers-reduced-motion` respected; no information gated behind animation.
- All four list states present (loading skeleton / error+retry / empty / success).
- Money exponent-correct; every figure traces to its postings; destructive money
  actions state amount + resulting state + accounting impact + reversibility.
- **No faked data or invented backend/accounting behavior** — a missing capability is
  recorded as a BACKEND GAP, not fabricated.

Success is not lines changed. It is: an operator understands and operates the
financial system faster and more safely, and the books still reconcile.
