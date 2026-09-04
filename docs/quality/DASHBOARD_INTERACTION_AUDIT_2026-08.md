# Recurso — Interaction & Navigation Assessment (final Stripe-grade gate)

> **Read-only, code-cited + live-traced on the deployed app (app.recurso.dev,
> build post-#708), point-in-time 2026-08-15, `main` @ `5715b4a7`.** No production
> code written; no PRs opened. This is **not** a visual-polish audit — the sole
> question is whether Recurso feels like a *financial operating system* an operator
> can move through quickly, confidently, and without losing context. Batches A–E
> are complete and merged.

---

## 1. Current Stripe-grade assessment

**Score: 90 / 100 — "close, but not there."** (Original polish audit graded ~83;
A–E lifted it to ~90.)

**Strongest areas (at or near the bar):**
- **Financial trust — A (no P0s found).** Money is exponent-aware and rendered
  through the `<Money>` signature everywhere that matters; the books reconcile;
  amounts are anchored in destructive confirms; no fabricated totals; error/404/
  not-found states are safe (`useObjectQuery`, Batch B). The trust foundation is
  solid — the remaining gap is *workflow*, not *safety*.
- **Object pages — A−.** Identity → status (`StatusBadge`) → financial hero
  (`<Money>`) → metadata → actions is consistent; exception states banner
  (`AttentionBanner`); Details/Timeline/Audit rails where data exists; extensive
  object-to-object links. Live-verified across Invoice/Subscription/Payment/
  Journal/Reconciliation/Credit Note/Customer/Account.
- **Destructive actions — A.** Subscription cancel (live) captures a reason, offers
  period-end vs immediate explicitly, and gates on a **"Review impact"** consequence
  preview before confirming — answers "what happens if I do it?" the Stripe way.
- **Worklist mechanics — B+.** Sticky headers + accessible names (Batch D),
  first-class loading/empty/error states, page-scoped selection + bulk, exact
  pagination, money right-aligned/tabular.
- **States & object addressability — A−.** Every object is a deep-linkable page;
  canonical loading/not-found/error; lists keep their own state in the URL.

**Remaining weaknesses (why it's not yet Stripe-grade):**
1. **Navigation continuity breaks on the way *back*.** The in-page back-link
   discards the operator's list state (filters/search/page/sort). Confirmed live.
2. **Global search can't find the objects operators search most** (invoices,
   payments). The command palette returns "Nothing matches" for a visible invoice.
3. **Worklists are barely sortable** — 24 of 25 list pages expose zero sortable
   columns.

These three are the "still an admin dashboard" tells: the operator *does* lose
their place, and *can't* jump to an invoice by number.

---

## 2. Top 10 remaining problems (ranked by operator impact)

| # | Sev | Finding | Anchor |
|---|---|---|---|
| 1 | **P1** | **Back-link discards list state.** ObjectHeader renders a static `<Link to={backTo}>` to a hardcoded list root; filtering Invoices→opening one→clicking "← Invoices" lands on unfiltered `/invoices` (live-confirmed). The list *itself* restores state from the URL — only the back affordance drops it. | `patterns/ObjectPage.jsx:63-68`; every object page's `backTo="/…"` |
| 2 | **P1** | **Command palette can't find invoices / payments / credit-notes / disputes.** It indexes only customers/plans/subscriptions; `⌘K` "INV-000009" → "Nothing matches" for an invoice visible on-screen (live). | `ui/command-palette.jsx:67-129` |
| 3 | **P1/P2** | **Sorting absent on ~24/25 list pages.** Only Invoices marks columns `sortable`; operators can't sort Payments/Subscriptions/Customers/etc. by amount, age, or status. | `patterns/DataTable.jsx` (mechanism exists); `sortable:true` in 1 page |
| 4 | **P2** | **`ObjectHeader` actions don't wrap.** Actions container is `flex shrink-0` (no `flex-wrap`); 3-button headers (Invoice Send/Preview/Download) can overflow on narrow. `PageHeader` already wraps. | `patterns/ObjectPage.jsx:~95` |
| 5 | **P2** | **Real content hidden in native `title=`** (not keyboard/touch reachable): dunning success-rate chart values; a metric expression. A real `Tooltip` exists (`ui/tooltip.jsx`). | `DunningDashboard.jsx:443`, `Metering.jsx:226` |
| 6 | **P2/P3** | **No stale / refetching indicator.** `DataTable` has no `fetching` state, so post-mutation swaps / background refetches are invisible (low urgency — `staleTime` 60s). | `patterns/DataTable.jsx` |
| 7 | **P3** | **Dead `breadcrumbs` prop.** Implemented in `PageHeader`, **0 callers** — decide (delete) rather than wire; active-nav + contextual-back answer "where am I". | `patterns/PageHeader.jsx:15,18` |
| 8 | **P3** | **Two generic page subtitles** ("Manage your customer base…", "Track and manage your recurring subscriptions") vs the operational voice on Payments/Collections. | `Customers.jsx`, `Subscriptions.jsx` |
| 9 | **DATA MODEL** | **Dispute audit + object timelines (credit-note/dispute) unavailable** — `disputes` isn't on the audit allowlist; no domain events emitted for either. (From Batch E; documented, not fabricated.) | `internal/adapter/middleware/audit.go` |
| 10 | **P3** | **A few dead-end technical IDs** remain (bare sliced UUIDs neither `CopyableId` nor a `Link`) in Audit Log / Events — mostly addressed; low residue. | `AuditLog.jsx`, `Events.jsx` |

**No P0 (trust/financial-safety) findings.**

---

## 3. Workflow traces

**1. Customer → Subscription → Invoice → Payment → Journal → Ledger Account →
Reconciliation.** *Works:* every hop is a real link (Customer→subscriptions,
Subscription→invoices, Invoice→journal entries + payment, Journal→account,
Account→postings, Reconciliation→invoice/transaction/account); each object answers
identity/status/amount/what-happened. *Friction:* at every *list* hop the back-link
resets the list (P1 #1); no palette shortcut to jump mid-chain (P1 #2). *Break:*
**frontend.**

**2. Failed payment → attention → invoice → customer → payment → recovery.**
*Works:* Home surfaces "N invoices failing collection" (with per-row invoice links)
+ churn + reconciliation exceptions; invoice shows the dunning banner + payment
attempts; Collections/Dunning carry the recovery worklist. *Friction:* minor —
returning to the Collections worklist after drilling loses filter state (P1 #1).
*Break:* **frontend.**

**3. Subscription → financial position → MRR → next invoice → cancellation →
preview → confirm.** *Works — strongest workflow.* Financial summary (MRR / billed-
each-period / next-invoice / outstanding / past-due / lifetime) is dense and clear;
cancel captures reason + period-end choice + a **"Review impact"** preview before
confirming. *Friction:* none material. *Break:* n/a.

**4. Reconciliation → run → discrepancy → transaction → journal → invoice/account.**
*Works — audit-grade.* Run banners its exception (Batch E), each discrepancy row
explains what/why and links the invoice + transaction; the transaction opens the
journal entry; the entry links its accounts. *Friction:* minor. *Break:* frontend
(none blocking).

**5. Invoice → amount → payment → journal → ledger accounts → reconciliation.**
*Works:* Invoice → journal entries (posted to ledger) → account; Invoice → payment.
*Friction:* no *direct* invoice→reconciliation link (recon→invoice exists, not the
reverse) — minor, arguably fine. *Break:* frontend/none.

**6. Search → object → action → return to worklist.** *Weakest workflow — broken at
both ends.* The palette can't find invoices/payments (P1 #2), and the return-to-
worklist affordance loses state (P1 #1). An operator who searches an invoice number
gets "Nothing matches," and one who drills from a worklist can't get back to their
filtered view. *Break:* **frontend** (palette coverage + back-nav), with an optional
backend search-endpoint enhancement.

---

## 4. Shared-fix opportunities (one change → many surfaces)

- **Context-preserving back navigation — the single highest-leverage fix.** Change
  `ObjectHeader`'s back-link once (restore the referrer's search string, or
  `navigate(-1)` when the referrer is the owning list) → **all 16 object pages ↔ 16
  list pages** keep the operator's place. The list state already lives in the URL;
  only the affordance needs to carry it.
- **Command palette object coverage — one component, global lookup.** Extend
  `command-palette.jsx`'s search sources to invoices / payments / credit-notes /
  disputes (reusing the same list endpoints the pages already use). One change makes
  the objects operators search most actually findable.
- **Worklist sorting — one config sweep.** The `DataTable` sort mechanism already
  exists; adding `sortable: true` to the money/date/status columns across the list
  pages (or a sensible default for money/date columns) lights up sorting everywhere.
- **`title=` → `Tooltip`** and **`ObjectHeader` action `flex-wrap`** are tiny
  shared touches (2 sites and 1 primitive respectively).

---

## 5. Backend / data-model blockers (kept separate from frontend)

- **DATA MODEL — Dispute audit trail:** `disputes` is absent from the audit
  middleware allowlist (`internal/adapter/middleware/audit.go`), so no `AuditTrail`
  data exists for disputes. Adding it is backend work.
- **DATA MODEL — Object timelines for credit-notes/disputes:** no domain events are
  emitted for these entity types, so `ObjectTimeline` would be permanently empty.
- **BACKEND (optional optimization) — global object search:** the palette can be
  extended frontend-side (invoices already load-all + client-filter, per
  `Invoices.jsx:115-122`), but a dedicated lightweight search endpoint would scale
  better for large tenants. *Not a blocker for Batch F* — do the frontend extension;
  note the endpoint as a future optimization.
- **Reconciliation single currency** — ADR-010 decision (not a gap).

**None of these block the recommended Batch F.**

---

## 6. Recommended Batch F (3–4 tightly-scoped increments)

1. **Context-preserving object back-navigation.** One shared `ObjectHeader` change
   so the back-link returns to the *owning list with its state* (filters/search/
   page/sort). Delete the dead `breadcrumbs` prop in the same increment. *(Fixes #1,
   #7; the single biggest "feels like an OS" lever.)*
2. **Command-palette object coverage.** Add invoices / payments / credit-notes /
   disputes to `⌘K` search (reuse existing endpoints; client-filter like the list
   pages), so the objects operators actually look up are findable by number.
   *(Fixes #2; document the server search endpoint as a future backend optimization.)*
3. **Worklist sorting sweep.** Enable `sortable` on money/date/status columns across
   the DataTable list pages (mechanism already exists). *(Fixes #3.)*
4. *(Optional, small)* **Interaction consistency:** `ObjectHeader` actions
   `flex-wrap`; the two `title=` values → real `Tooltip`; the two generic page
   subtitles → operational voice; a subtle DataTable `fetching` indicator.
   *(Fixes #4, #5, #6, #8.)*

Sequence: #1 and #2 are the operator-confidence wins; #3 is the worklist win; #4 is
optional consolidation. All are shared/config changes — no new design system, no
motion, no token work.

---

## 7. Explicit DO-NOT-FIX (intentional or low-value)

- **Specialized accounting tables** (reconciliation discrepancies, account/wallet/
  quote ledgers) — kept specialized on purpose; do **not** migrate to DataTable.
- **Journal Entry / Ledger Account minimal rails** — an immutable posting and a
  chart-of-accounts entry have no lifecycle/status; the thin rail is correct.
- **Reconciliation "Scope" rail title + single-currency amounts** — domain-
  meaningful / ADR-010; leave.
- **Dispute audit + object timelines** — backend/data-model, not a frontend fix;
  do not fabricate.
- **Breadcrumbs as a feature** — don't wire them; the active-nav rail + contextual
  back are the real answer. Just delete the dead prop.
- **Native browser Back** already preserves list state — the fix is the *in-page*
  affordance, not a new global back control.
- **Money / tokens / typography / motion / `useObjectQuery` / DataTable pagination-
  selection-sticky** — done; do not re-touch.
- **No new primitive** unless ≥2 surfaces need it (none of the above require one).

---

## 8. Final recommendation

**Close, but not there.** Recurso has crossed the hardest threshold — **trust**: no
P0s, the books reconcile, money reads as money, destructive actions preview their
consequences, and every object is a first-class, safely-stated page. What still
separates it from Stripe-grade is **operator flow**, concentrated in three shared,
non-cosmetic gaps: the back affordance loses the operator's place, global search
can't find the core financial objects, and worklists don't sort. These are exactly
the frictions that make a finance operator feel they're *fighting an admin tool*
rather than *working in an operating system* — and all three are one-change-many-
surfaces fixes with no backend dependency.

**Do Batch F (increments 1–3, optionally 4) and Recurso reaches the Stripe-grade
bar (~95).** Until then it is an excellent, trustworthy financial product held back
from "operating system" by navigation continuity and search reach — not by anything
that would make an operator distrust the numbers.

**STOP — assessment complete. No production code; no PRs; Batch F not started.**
