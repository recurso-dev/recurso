# Recurso Dashboard — Quality Transformation Audit

**Scope:** the operator dashboard (`frontend/`). **Method:** static inspection of
routes, pages, primitives, and backend OpenAPI, plus live visual/keyboard QA on
`app.recurso.dev`. **Stance:** the codebase is mature and well-abstracted — this
is *refinement and gap-closing against a world-class bar*, not a rebuild. Rank
tags: **P0** critical · **P1** high · **P2** medium · **P3** polish.

> **Bottom line.** The dashboard is already strong: an exponent-aware money
> layer, an object-page framework, a consistent state system, a real design
> system, and a complete motion layer. The gap to "financial operating system"
> is concentrated in **five places**: (1) money that can be *misread* on two
> finance screens [P0]; (2) **Payment / Journal-Entry / Reconciliation are not
> addressable objects** — the charter's own chain breaks, and this is
> **backend-blocked** [P1 + BACKEND GAP]; (3) the **Subscription** page lacks
> financial consequence [P1]; (4) money **confirmations don't state the amount**
> and immediate-cancel has no preview [P1]; (5) **DataTable** loses list context
> on back-nav and lacks selection/bulk/consolidation [P1/P2].

---

## 1. Current architecture

- **Stack:** React 19, react-router 8, `@tanstack/react-query` 5 (shared-cache
  data layer, ADR-005), Radix primitives (9 packages), Tremor 3.18 (charts on
  recharts/d3), Tailwind 3.4, CVA, `lucide-react` pinned 0.294, Sonner, Vite 8
  (rolldown).
- **Shape:** `App.jsx` = **96 routes**, ~**85 pages** (`src/pages`, +9 settings).
  Auth/portal render outside the shell; the operator app is one
  `DashboardLayout` (sidebar + topbar + `<main>`), all pages `React.lazy`.
- **Perf posture (already good):** route-level code-splitting; deliberate
  `vite.config.js` manual chunking that claims React/ui vendors before the
  `charts` chunk (~968 kB) so the charting stack stays out of eager chunks.
- **Motion:** a complete, reduced-motion-aware CSS+rAF motion system
  (`frontend/MOTION.md`) — tokens, primitives, signature ledger/reconciliation
  moments. This axis is **done**.

## 2. Current design system

- Warm-stone neutral palette + a semantic **status tier** (`success`/`warning`/
  `info`/`destructive`, dual-role as text and as `/10` tint) — the root fix for
  the old raw-palette sprawl. ~33 `:root` tokens; motion tokens
  (`fast/normal/slow` + `ease-standard`/`ease-out-soft`); elevation
  (`shadow-raised`/`shadow-overlay`); `focus-visible:ring-2 ring-ring`
  consistently; `.money`/`tabular-nums` for figures.
- **Money is exponent-aware** (`ui/money.jsx`, `lib/utils.js`): `Intl.formatToParts`
  drives the currency exponent (JPY 0, KWD 3), symbol styled separately —
  a genuinely strong, correct foundation.
- Preserve and extend this direction. Do not fork a second design system.

## 3. Existing reusable components (reuse these)

- **Object framework:** `patterns/ObjectPage` (`ObjectHeader`/`ObjectPageLayout`/
  `ObjectSection`/`AttributeList`/`RelatedRow`/`RelatedEmpty`), `ObjectTimeline`,
  `AuditTrail`, `AttentionBanner`, `FinancialSummary`, `JournalEntries`,
  `PaymentAttempts`.
- **Data/forms:** `DataTable`, `cells`/`columns` (incl. `MoneyCell`), `FormField`,
  `FormSheet`, `CustomerSelect`, `SubscriptionRef` (new).
- **UI (28):** `button`, `card`, `badge`, `status-badge`, `dialog`, `sheet`,
  `confirm-dialog`, `command-palette`, `money`, `copyable-id`, `checkbox` (new),
  `select`, `input`, `textarea`, `switch`, `table`, `tabs`, `tooltip`, `alert`,
  `sonner`, motion primitives (`MotionNumber`/`Reveal`/`State`), …
- **State:** `EmptyState`, `ErrorState`, `LoadingSkeleton` (`TableSkeleton`/
  `CardGridSkeleton`), `ErrorBoundary` (now design-system + chunk-recovery).

## 4. Existing inconsistencies

| Finding | Rank | Evidence |
|---|---|---|
| ~16 pages **hand-roll tables** instead of `DataTable` (dup empty/loading/error/a11y) | P2 | Team, Entities, Collections:381-473, +reports (TrialBalance, MonthEndClose, RevenueRecognition, …) |
| **Pagination UX diverges** — legacy `hasNext: len>=PAGE_SIZE` vs `total/pageSize`; PAGE_SIZE 10 vs 50 | P1 | Customers/Subscriptions/Plans (legacy) vs Payments:181-186 |
| **Filter UX diverges** — pill/segmented vs Select dropdowns; client-vs-server filtering with different truncation | P2 | Invoices:270 / Coupons:158 (pills) vs Subscriptions:210 / Payments:153 (Selects) |
| **Local `fmtMoney` dialects** (currency-code suffix, not shared symbol) | P3 | `slide-overs/CustomerDetail.jsx:19-25`, `Wallets.jsx:35`, `WalletPage.jsx:573` |
| Duplicated `BAR_COLORS` viz palette | P3 | `RevenueByPlan.jsx:15`, `RevenueByGeography.jsx:14` |
| Other **hand-rolled checkboxes** (post shared `Checkbox`) | P3 | `Security.jsx:608`, `slide-overs/PlanCharges.jsx:692`, +~6 |
| `AUX_DESTINATIONS` hand-maintained title map (drift risk) | P3 | `lib/navigation.js:118-133` |

## 5. UX problems

- **List context lost on back-nav** [P1] — page/search/filter/sort live in page
  `useState` (Customers:41-43, Subscriptions:29-33), so returning from a detail
  resets to page 1 / All. Only Invoices' aging filter survives (URL). No scroll
  restoration.
- **Money confirmations omit the amount** [P1] — see §12.
- **Immediate subscription cancel has no proration/refund preview** [P1] — §12.
- **Raw minor units printed as bare integers** [P0] — §5-critical below.

### P0 — money can be misread by 100×
`RevenueRecognition.jsx:64-67` (multi-currency `fmt`) and
`FinanceReconciliation.jsx:55,62` (`formatMinorUnits`/`formatDifference`) render
minor-unit integers **as plain numbers** — e.g. `990000` shows as "990,000",
not `$9,900`. On finance-critical screens this is a magnitude misread. **P0**:
convert to major units per row-currency (or label the unit explicitly). Never
show a raw minor-unit money figure to a user.

## 6. Information-architecture problems

The charter's chain — Customer → Subscription → Usage → Invoice → **Payment** →
**Journal Entry** → **Reconciliation** — breaks at the accounting end:

| Object | Detail URL | Status |
|---|---|---|
| Customer / Subscription / Invoice / Plan / Coupon / Dispute / Credit Note / Quote / Wallet / Meter / Dunning campaign / Cancel flow | `/:id` | ✅ exists |
| **Payment** | — | ❌ **no `/payments/:id`** — a tenant-wide log that redirects to the invoice (`Payments.jsx:180`) |
| **Journal Entry / transaction** | — | ❌ only `/ledger/accounts/:id`; the individual posting has no page and is a dead-end |
| **Reconciliation run** | — | ❌ recorded runs render in a table with no per-run URL (`FinanceReconciliation.jsx:336-357`) |
| **Usage record** | — | ❌ `/usage` list only; no addressable node between Subscription and Invoice |

**These four are the single biggest gap to "financial operating system."** Three
are **backend-blocked** — see §15.

## 7. Object-model problems

- **Subscription page omits financial consequence** [P1] — no `JournalEntries`,
  no `FinancialSummary`/MRR (`SubscriptionPage.jsx` imports lack them), while the
  gold Invoice page has both. The object that drives recurring revenue can't
  answer "what did this post / what is it worth." Invoice is the *only* page
  that satisfies all seven depth questions.
- **Ledger postings are navigational dead-ends** [P2, frontend-fixable] —
  debit/credit accounts render as `slice(0,8)…` not links to
  `/ledger/accounts/:id` (`Ledger.jsx:229,245`; `AccountPage.jsx:195`), and no
  posting links back to its source invoice/payment via `reference_id`.
- **Raw-UUID / non-linked related refs in list cells** [P2] — `Invoices.jsx:181`
  (customer as `slice(0,8)`), `Usage.jsx:206`, `Disputes.jsx:116` (invoice_id not
  linked), `Metering.jsx:328/568`, `Mandates.jsx:333`. (Own-id-with-Copy is fine
  and excluded.) Same anti-pattern already fixed on the invoice→subscription ref.

## 8. Accessibility problems

- **No permission / 403 state** [P2] — RBAC exists (Team roles) but a forbidden
  response falls through to generic `ErrorState` with a raw backend string; no
  "you don't have access" affordance. Whole missing state category.
- **Chart a11y thin** [P3] — chart wrappers use `role="img"` but labels are
  generic/occasionally inaccurate (`Usage.jsx:415`) and there's no table/text
  alternative for the series.
- Otherwise **strong**: skip link + focusable `<main>`, `header`/`nav[aria-label]`/
  `aside`/`main` landmarks, aria-labels on all icon buttons, `FormField` label
  association + `role="alert"` errors + first-error focus, `aria-sort` + real
  button/Link controls in `DataTable`, Radix focus trap/restore. `DataTable` has
  no `role="grid"`/arrow-key nav (minor).

## 9. Responsive problems

- **320px is unsupported** [P2] — comments reference only 375px QA; `md:`/`xl:`
  breakpoints sparse (essentially two-tier). Wide journal/line-item tables set
  `min-w-[640px]` (`AccountPage`, `WalletPage`) / `min-w-[560px]` (`QuotePage`)
  and fixed-width filter Selects, so 320px forces horizontal scroll. They live in
  `overflow-auto` (scroll, don't break) — but the floor should be lowered
  intentionally.
- Shell itself is sound: `min-w-0 flex-1`, `max-w-[1400px]` (a max, centered),
  `Table` wrapped in `overflow-auto`, wrapping filter toolbar.

## 10. Performance problems

- **No list virtualization** [P3] — Events/Usage/AuditLog render every fetched
  row; safe only while callers keep `limit` small (they do). A guard/virtual
  window is the remaining scalability risk.
- **Public/pre-shell bare spinners** [P3] — `App.jsx:116-130` (`PageFallback`,
  auth-loading) and `portal/PortalDashboard.jsx:294` use whole-page spinners vs
  the app's skeleton system.
- `charts` chunk ~968 kB is already isolated and lazy — acceptable.
- Motion is transform/opacity-only with proper rAF cleanup and a global
  reduced-motion kill-switch — no leaks, no continuous loops.

## 11. Missing states

- **Permission / 403** (§8) [P2]. **Public/pre-shell skeletons** (§10) [P3].
- Otherwise complete: loading (skeletons), empty (first-run vs no-results with
  contextual docs link), error (real backend messages — §12), success (toasts).

## 12. Missing interactions / financial safety

- **DataTable v2 missing capabilities** [P1/P2]: URL-persisted state [P1],
  row **selection + bulk actions** [P2], column visibility/resizing [P2/P3],
  in-component filter config + export [P2], saved views [P3], keyboard grid nav
  [P3].
- **Financial confirmations lack the concrete amount** [P1] — credit-note
  approve (`CreditNotePage.jsx:344`), write-off (`Collections.jsx:176`), void
  (`:363`), dispute-credit (`DisputePage.jsx:266`) state the *consequence* but
  not the *number*. `WalletPage.jsx:567-576` already proves the amount-anchored
  pattern; plan-change (`SubscriptionPage.jsx:566-587`) proves the full
  proration preview. **Immediate cancel has no money preview** [P1].
- Currency-aware input `step` [P3] — `step="0.01"` is unconditional; wrong for
  JPY/KWD (conversion via `toMinorUnits` is already correct).

## 13. Missing financial context

- Subscription financial depth (§7) [P1]. Ledger posting → source/account links
  (§7) [P2]. Amount-anchored confirmations + cancel preview (§12) [P1]. Raw
  minor-unit fixes (§5) [P0]. Reconciliation drill (blocked, §15).

## 14. Motion opportunities

Motion is **complete** (7 phases, `MOTION.md`). Remaining is *application*, not
system: when the new **Payment / Journal-Entry / Reconciliation** object pages
are built (post-backend), give them the existing lifecycle motion
(`MotionState` status flash, `JournalEntries` sequential balance, reconciliation
`MotionNumber → 0`). No new motion primitives needed.

## 15. Backend gaps (do NOT fake — §22 of the charter)

The three highest-IA-value object pages are **blocked on the API**:

1. **Payment object** — only `GET /v1/payment-attempts` (list),
   `getInvoicePaymentAttempts` (per-invoice), `/v1/payments/offline` exist. **No
   `GET /v1/payments/{id}` / `GET /v1/payment-attempts/{id}`.**
   *Needs:* a single-payment/attempt read endpoint returning the attempt +
   gateway events + settlement + linked invoice/refund/dispute, to back
   `/payments/:id`.
2. **Journal-Entry / transaction detail** — ledger exposes `/accounts`,
   `/entries` (by `account_id`), `/trial-balance`, `/export`,
   `/deferred-rollforward`. **No single-transaction endpoint.**
   *Needs:* `GET /v1/ledger/transactions/{id}` (or entries by `reference_id`) to
   back a journal-entry page. *Frontend-only interim:* link posting accounts and
   posting→source (invoice/payment) via existing `reference_id` — do this now.
3. **Reconciliation run detail** — `GET /v1/finance/reconciliation/runs`
   persists only a **summary** row (counts), not the per-run discrepancies. **No
   `GET .../runs/{id}` and no stored discrepancy detail.**
   *Needs:* persist per-run discrepancies + `GET .../runs/{id}` to back
   `/reconciliation/:id`.

Until these ship, **do not fabricate** these pages. Build the frontend-only
value around them (links, subscription depth, confirmations, DataTable, money).

## 16. Recommended implementation order

Gold-standard-first (charter): Customer, Subscription, Invoice, Payment, Ledger —
but sequenced by *leverage × unblocked*. Each item is its own small green-CI PR
(lint/build/test + keyboard/responsive/reduced-motion/visual QA).

**Now — P0 (correctness/safety, frontend-only):**
1. Kill raw-minor-unit rendering (`RevenueRecognition`, `FinanceReconciliation`) → real money. [§5]

**Batch A — P1 financial safety & depth (frontend-only):**
2. Amount-anchor every destructive money confirm; add proration/refund preview to immediate cancel (reuse plan-change preview). [§12]
3. Bring **Subscription** page to Invoice depth: add `FinancialSummary` (MRR/next-invoice) + per-subscription `JournalEntries` (from its invoices). [§7]
4. **DataTable v2 – phase 1:** URL-persisted list state (page/search/filter/sort) via a shared hook → back-nav restores context; standardize on `total/pageSize` pagination + one PAGE_SIZE. [§5/§12]

**Batch B — P2 navigability & consistency (frontend-only):**
5. Make ledger postings navigable: link accounts (`/ledger/accounts/:id`) and posting→source; link the remaining raw-UUID list refs (Invoices customer, Disputes invoice, Usage, Metering, Mandates). [§7]
6. **DataTable v2 – phase 2:** row selection + bulk-action bar; migrate the hand-rolled tables; promote filter config + export into the component. [§4/§12]
7. Add a **permission/403** state to `ErrorState`/query error handling. [§8]
8. Lower the responsive floor to **320px** (let filter Selects shrink; confirm wide tables scroll cleanly). [§9]

**Batch C — P3 polish:**
9. Route local `fmtMoney` through `Money`; currency-aware input `step`; migrate remaining hand-rolled checkboxes to `Checkbox`; dedupe `BAR_COLORS`; auto-derive `AUX_DESTINATIONS`; public/pre-shell skeletons; chart a11y labels + table alt; list-virtualization guard.

**Backend track (parallel, unblocks the chain):** file the three §15 gaps; when
`GET /payments/{id}`, `GET /ledger/transactions/{id}`, and per-run reconciliation
land, build the **Payment**, **Journal-Entry**, and **Reconciliation** object
pages with the existing object-framework + motion.

---

### Already shipped during this initiative (context)
Design-system + reliability fixes already merged: shared `Checkbox` +
`ConsentCheckbox` migration (#675), `ErrorBoundary` design-system + stale-chunk
recovery (#674), `SubscriptionRef` plan-name links (#676), sidebar rail
indicator (#673), and the full motion system (#664–#672).
