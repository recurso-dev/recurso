# Dashboard Grade Audit 2 — Whole-Dashboard Reassessment

> **Investigation-only, code-cited + live-verified, point-in-time (2026-08-15).**
> No production code was modified to produce this document. It reassesses the
> whole dashboard at a higher level than the prior [DASHBOARD_QUALITY_AUDIT.md](./DASHBOARD_QUALITY_AUDIT.md)
> and does not repeat that audit's findings except where the code has
> **materially changed** since (each such item is marked *changed since prior
> audit*). Every claim is anchored to `path:line` or a live screenshot taken on
> app.recurso.dev.
>
> **Method.** Five parallel read-only investigations (information architecture,
> object-page depth, backend data availability, design/state/motion/a11y,
> workflow tracing + prior-audit synthesis) plus a first-hand live-QA walk of the
> deployed app (Home → Subscription → Payments → Reconciliation → Invoice).
> Backend claims were verified against `cmd/api/main.go` and `openapi.yaml`.

---

## Batch 3A — Financial Object Foundation (Implemented 2026-08-15)

The backend + data-model foundation that makes the money chain
(Subscription → Invoice → Payment → Journal Entry → Reconciliation) addressable
and explainable. **Read-model / addressability / connectivity only — no
accounting, posting, recognition, proration, or reconciliation-algorithm
behavior was changed.** The full accounting invariant harness, reconciliation,
trial-balance and ledger suites stay green (soaked across 6+ extra invariant
seeds). No polished object pages were built (that is Batch 3B).

### Implemented
- **Payment object** — `GET /v1/payment-attempts/:id` returns one attempt
  resolved with its invoice/customer/subscription context (read-time joins off
  the immutable invoice edge). *Addresses P1-2, GAP-1.*
- **Journal-entry connectivity** — `GeneralLedgerRow` now carries
  `debit_account_id`/`credit_account_id` (read-model projection of columns
  already in the DB), so the invoice/credit-note journal legs deep-link to their
  accounts with **no frontend change** (the components already read those
  fields). *Addresses P0-2/DATA-MODEL and the "wired but dead" trace break.*
- **Journal-entry object** — `GET /v1/ledger/transactions/:id` makes a single
  posted transaction addressable (keyed on `ledger_transactions.id`), each leg
  account-linked — the target a reconciliation discrepancy's transaction points
  to. *Addresses P1-2, GAP-2.*
- **Reconciliation-run object** — `GET /v1/finance/reconciliation/runs/:id`
  returns a recorded run **with its persisted discrepancy rows**. Migration
  `000171` adds `reconciliation_run_discrepancies`; `RecordRun` now persists the
  already-computed discrepancy list (the reconciliation algorithm is untouched).
  `discrepancies_truncated` honestly flags the live-run listing cap or a run
  recorded before persistence existed; pre-migration runs carry an empty list,
  never fabricated history. *Addresses P1-2/P1-3, GAP-3.*
- **Subscription financial summary** — `GET /v1/subscriptions/:id/financial-summary`
  returns MRR (reusing the engine's canonical `monthlyMinorUnits` definition
  verbatim — monthly-normalized list price, active-only, single-currency),
  recurring value + interval, next-invoice date/base-amount (only when the sub
  will renew; base = list price, gap-flagged), and the per-currency outstanding
  position. *Addresses P1-1, GAP-4.*
- **Cancel preview** — `GET /v1/subscriptions/:id/cancel-preview` forecasts a
  cancellation deterministically **before** the mutation (which is unchanged):
  effective time, resulting status, the still-deferred revenue an immediate
  cancel forfeits + recognizes as breakage (computed read-only from the same
  rev-rec data `UnwindOnCancel` uses), avoided future recurring, and
  `flat_fee_refund: 0` (stated explicitly — paid in advance, not refunded).
  *Addresses P0-1 (blind cancel), GAP-5.*

### API Changes (all additive, no contract broken)
Five new `GET` routes (above) + `debit_account_id`/`credit_account_id` added to
the existing `/invoices/:id/journal-entries` and `/credit-notes/:id/journal-entries`
payloads. `openapi.yaml` updated; the drift + spec-parse gate passes. The
payment detail route is `/v1/payment-attempts/:id` (not `/payments/:id`) — the
existing resource is named `payment-attempts` (its list is `/payment-attempts`),
and this avoids a gin static-vs-param clash with `/payments/offline`.

### Data-Model Changes
- Migration **000171** `reconciliation_run_discrepancies` (+ down): persists the
  per-run discrepancy rows that were previously discarded.
- `GeneralLedgerRow` gains two account-id fields (read-model struct + two SELECT
  projections + two scans; **no schema change** — the ids were already joined).
- **`PaymentAttempt.customer_id` column deliberately NOT added**: it is
  derivable at read time off the immutable invoice→customer edge, so the audit's
  suggested column is unnecessary for this read path (zero migration/backfill/
  write-path risk). Recorded as an assessment, not a gap; a column would only be
  warranted to *filter* attempts by customer at scale (future ADR).

### Accounting Safety
No posting/recognition/proration/reconciliation logic touched. `CreateTransaction(s)`,
the ledger legs, invoice recognition, and the reconciliation checks are byte-for-
byte unchanged. Verified: `go test ./...` green; full Postgres suite
(`internal/service`, `internal/adapter/db`) green including
`TestLedgerInvariants_RandomizedBillingSequences`, reconciliation, and
trial-balance; 5 extra invariant seeds (101/202/303/404/505) all balanced.

### Security / Tenant Isolation
Every new read is tenant-scoped in SQL (`WHERE … AND tenant_id = …`) and, for
the subscription reads, additionally re-checked in the service
(`TenantID != tenantID → not found`). A cross-tenant or unknown id returns a
**flat 404** (never 403, never a leak). Explicit cross-tenant tests cover all
five endpoints (handler-level) plus the payment / ledger-transaction /
reconciliation-run repos (Postgres-level).

### Tests
- Handler unit tests (owned→200, cross-tenant→404, unknown→404, bad-id→400) for
  all five endpoints (`internal/adapter/handler/object_get_test.go`,
  `cancellation_preview_test.go`, `reconciliation_runs_test.go`).
- Service tests: reconciliation discrepancy-persistence mapping; the read-only
  forfeit sum (with a nil-embedded rev-rec repo so any mutating call would
  panic — proving read-only); MRR edge cases (monthly / annual-normalized /
  canceled→0 / paused→0 / cancel-at-period-end→no-next-invoice).
- Postgres integration: `GetTransactionByID` (account-ids populated + tenant
  scope), `PaymentAttempt.GetByID` (customer/subscription resolution + tenant
  scope), `ReconciliationRun.GetByID` (summary + discrepancy round-trip + clean
  run + tenant scope; also exercises migration 000171).
- Frontend: api-client contract test for all five new methods; full vitest suite
  green (655).

### Performance
All reads are single-row or one small aggregate. No N+1: the payment/journal
reads are one `QueryRow` with joins already indexed (`payment_attempts.invoice_id`,
`ledger_transactions` PK, `da.tenant_id`); the reconciliation detail is one run
row + one indexed `reconciliation_run_discrepancies (run_id, seq)` scan; the
subscription summary is one plan read + one grouped invoice aggregate (indexed
`invoices (tenant_id, subscription_id)`); the forfeit sum walks a subscription's
active schedules (bounded). No tenant-wide scans, no recomputation loops.

### Frontend Integration (minimal, per scope)
`endpoints.getPayment` / `getLedgerTransaction` / `getReconciliationRun` /
`getSubscriptionFinancialSummary` / `getSubscriptionCancelPreview` added to
`api.js` with a contract test. **No pages, hooks-on-pages, breadcrumbs, columns,
or motion** — Batch 3B builds the object-page treatment on this foundation.

### Backend Gaps Remaining
- **GAP-6** (cross-object `/search`) — untouched (⌘K scaling; Batch 3B/later).
- **GAP-7** (recent-failed-payments feed with `since` + customer linkage) —
  untouched; the collections signal still covers the operational case.
- **GAP-8** (tenant-wide failed-webhook feed) — untouched.
- **Next-invoice true total** (tax + coupon + add-ons + usage + commitment) —
  still no single deterministic source; the summary exposes the **list/base**
  amount only, explicitly labeled. Documented gap, not approximated.
- **Cancel unused-time proration credit / final metered-usage figure** — the
  cancel mutation computes neither; the preview omits both rather than inventing
  money that never moves (STOP-flagged in the spec). A read-only usage estimator
  or a proration-credit policy would be a *behavior* change → future decision.

### Bugs Found / Bugs Fixed
None. This was greenfield additive read-model work; no defects surfaced in
existing code, and no behavior needed fixing.

### Follow-up Required for Batch 3B
Build the object-page treatment on these contracts (no new backend needed for
the six): Payment page (`/payment-attempts/:id`), Journal-Entry view/sheet
(`/ledger/transactions/:id`) + link the reconciliation "Transaction" cell,
Reconciliation-run page (`/finance/reconciliation/runs/:id`) with its
discrepancy rows, Subscription financial-summary strip + cancel money-preview in
an amount-anchored confirm, and surface the now-live invoice journal-leg→account
links. GAP-6/7/8 remain for their own increments.

---

## 1. Executive Summary

**The one-sentence finding:** Recurso is already a *well-built billing admin with
a real accounting spine* — what still makes it feel like "an admin dashboard"
rather than "a financial operating system" is that **the financial story
dead-ends at exactly the two places a revenue operator lives: the recurring-
revenue object (Subscription) and the accounting tail of the money chain
(Payment → Journal Entry → Reconciliation).**

What a senior finance/revenue operator would praise today:

- **A genuine accounting IA.** The `BOOKS` nav group (Reconciliation, Ledger,
  Trial Balance, Close, Rev-Rec, Entities, GST) is a deliberate controller/
  auditor surface, foregrounded on purpose (`lib/navigation.js:78-88`). Most
  SaaS billing tools have no such thing.
- **Reconciliation actually explains itself** — every discrepancy is labelled
  "what disagrees, by how much, and why" (`FinanceReconciliation.jsx:30-51,
  285-330`; live-verified: *"Customer-credit liability mismatch"*, *"Missing
  write-off transaction"* with expected/found/difference).
- **Invoice is a true reference object page** — identity, lifecycle + failure
  reason, full amount breakdown, balanced journal legs, related links, activity,
  actions (`InvoicePage.jsx`). Live-verified end-to-end.
- **One clean design system** — a single CSS-variable token source
  (`index.css:11-62` → `tailwind.config.js:20-143`), monospaced **tabular**
  money (`index.css:101-110`), a canonical `StatusBadge` registry, an
  exponent-aware `<Money>`, and a `DataTable` that is now v2 (selection, bulk,
  sort, URL-state, priority column hiding).

What that same operator would still call "an admin dashboard":

- **The Subscription — the object a revenue operator opens 20× a day — has no
  MRR, no financial summary, and shows nothing about what it posts to the
  ledger** (`SubscriptionPage.jsx` imports neither `FinancialSummary` nor
  `JournalEntries`, `:1-43`; live-verified: Overview shows *"$299.00 / month"*
  and *upcoming invoice*, but no recurring-value/MRR and no accounting section).
- **The "explain any number" chain breaks at the accounting tail.** Payment,
  Journal Entry, and Reconciliation-run are **not addressable objects** — you can
  *see* a payment attempt or a posting inline, but you cannot *open* one,
  bookmark it, or drill a reconciliation discrepancy's transaction
  (live-verified: a Payments-log row drills to the **invoice**, not a payment;
  the Reconciliation `TRANSACTION` column is an unlinked `–`).
- **The one destructive money action lacks a preview.** Immediate subscription
  cancel posts blind — no refund/credit/proration amount is shown before it
  executes (`SubscriptionPage.jsx:1169-1177,339`).
- **Navigation lacks connective tissue.** 43 sidebar items across 9 groups, **no
  breadcrumbs anywhere** (the component exists but is used on zero pages), back-
  links that **reset** a list's filter/page context, and ⌘K that live-searches
  only 3 of ~9 object types.

**Honest framing for prioritisation:** the codebase has **no catastrophic
correctness defect** — the prior "quality transformation" cleared those (the
100× money-misread P0 is fixed, `FinanceReconciliation.jsx:52-66`; truncation
protections shipped; destructive confirms are amount-anchored). So the **P0 list
in this audit is deliberately thin**, and *most of the deepest gaps are
BACKEND / DATA-MODEL, not frontend neglect* — the frontend is near the ceiling
of what the API exposes. The transformation from "admin dashboard" to "financial
operating system" is gated on roughly **six read endpoints and three data-model
additions**, plus the frontend depth work those unblock. This is a *depth and
connectivity* problem, not a rewrite.

---

## 2. Current Product Maturity

| Axis | Maturity | Evidence |
|---|---|---|
| Correctness / money safety | **High** | Exponent-aware money everywhere; reconciler + invariant harness (ADR-002); amount-anchored destructive confirms |
| Design system | **High** | One token source; tabular money; canonical primitives; `DataTable` v2 |
| List surfaces | **High** | URL-state on 16 lists; selection + bulk; server-side filters; CSV export |
| Reference object page (Invoice) | **High** | All 7 depth dimensions present (`InvoicePage.jsx`) |
| Accounting explainability (books pages) | **High at the page, broken at the link** | Reconciliation/Ledger/AccountPage strong; the cross-object *drill* dead-ends (§7) |
| **Subscription depth** | **Low–Medium** | No MRR/summary/journal (§6) |
| **Addressable accounting objects** | **Low** | No Payment / Journal-Entry / Recon-run page (§4) |
| Navigation connectivity | **Medium** | Great IA groups, but no breadcrumbs / context-preserving back / broad ⌘K (§3, §9) |
| Motion | **High / done** | 7-phase system complete, reduced-motion aware (§12) |
| Accessibility | **Medium–High** | Strong semantics; gaps: sticky headers, table names, grid nav (§15) |
| State handling | **Medium–High** | 4 states via DataTable; gaps: stale + 403 (§13) |

**Net:** a strong *billing administration* product sitting one **depth + connectivity**
layer below a *financial operating system*. The ceiling is not polish; it is
whether an operator can **understand and trace the money** without leaving the
object they're on.

---

## 3. Information Architecture

**Route surface** (`App.jsx:167-255`): ~70 authenticated routes. Object detail
routes live under **three inconsistent prefixes** — flat (`/customers/:id`),
`/finance/*` (books + reports), and orphans (`/ledger/accounts/:id:187`,
`/billable-metrics/:id:189` whose list is `/metering:188`).

**Sidebar** (`lib/navigation.js:23-114`): **43 items across 9 groups** — Home/Ask,
BILLING (8), GROWTH (2), USAGE (3), REVENUE RECOVERY (5), PAYMENTS (3), BOOKS (7),
REPORTS (7), SYSTEM (6) — plus 14 aux destinations behind Settings/header (**57
total named destinations**).

**What's genuinely good (do not "fix" for aesthetics):**

- The **BOOKS** group (`:82-88`) is a real accounting hub and is correctly
  foregrounded — the product's differentiator.
- **PAYMENTS** (`:72-74`) is its own coherent group.
- All six audit-named surfaces (Payments, Ledger, Reconciliation, Reports, Audit
  Log, Events) are reachable in **one** sidebar click.

**IA gaps (ranked by operator impact):**

1. **The finance surface is fragmented across three groups** — BOOKS, REPORTS,
   and PAYMENTS — with `/finance/*` routes split between BOOKS (`:194-197,201`)
   and REPORTS (`:198-205`). One operator role ("controller") spans three nav
   groups. *A single Finance/Accounting hub with Books vs Reports sub-sections
   would map to the role and shorten the rail.* **(FRONTEND, P2.)**
2. **Over-dense rail** — 9 groups exceeds the ~7 a scannable sidebar supports;
   REVENUE RECOVERY leaks one workflow's internals (`/dunning`, `/dunning/campaigns`,
   `/collections` as three siblings, `:62-64`). **(FRONTEND, P2/P3.)**
3. **Two audit-relevant objects can be listed but never opened** — no
   `/payments/:id`, no reconciliation-run page (§4). For an "every event is
   traceable" product this is the sharpest IA gap. **(BACKEND-gated, P1.)**
4. **Audit Log vs Events adjacency** — both in SYSTEM; the "book of record for
   money events" vs "platform config log" distinction isn't signalled
   (`:21,107,109`). **(FRONTEND, P3.)**
5. **URL-scheme inconsistency** (`/finance/*` vs flat vs orphan prefixes) makes
   deep links unpredictable. **(FRONTEND, P3.)**

---

## 4. Object Model

Convention: a first-class object = a `/thing/:id` route + Page component + the
7 depth dimensions.

| Object | List | `/:id` page | First-class? |
|---|---|---|---|
| Customer | `/customers` | `CustomerPage` (App.jsx:170) | ✅ |
| Plan | `/plans` | `PlanPage` (:173) | ✅ |
| Subscription | `/subscriptions` | `SubscriptionPage` (:176) | ✅ (shallow — §6) |
| Invoice | `/invoices` | `InvoicePage` (:178) | ✅ **reference** |
| Credit Note | `/credit-notes` | `CreditNotePage` (:208) | ✅ |
| Ledger **Account** | `/ledger` | `AccountPage` (:187) | ✅ |
| Dispute | `/disputes` | `DisputePage` (:247) | ✅ (thin) |
| Quote | `/quotes` | `QuotePage` (:212) | ✅ |
| Coupon | `/coupons` | `CouponPage` (:182) | ✅ |
| Wallet | `/wallets` | `WalletPage` (:191) | ✅ |
| Meter | `/metering` | `MeterPage` (:189) | ✅ |
| Dunning Campaign | `/dunning/campaigns` | `DunningCampaignPage` (:240) | ✅ |
| **Payment** | `/payments` | **none** | ❌ **missing** |
| **Journal Entry / Ledger posting** | `/ledger` (sheet) | **none** | ❌ **missing** |
| **Reconciliation Run** | `/finance/reconciliation` | **none** | ❌ **missing** |
| Entity | `/finance/entities` | **none** | ❌ missing |

**The three missing objects (Payment, Journal Entry, Reconciliation Run) are
precisely the accounting tail of the money chain** — the objects a controller
needs most to trace, and the ones the product's own `DASHBOARD_PRINCIPLES.md`
names as required stable URLs (`/payments/:id`, `/journal-entries/:id`,
`/reconciliation/:id`). All three are **backend-gated** (GAP-1/2/3).

---

## 5. Object Depth Matrix

Scored against the InvoicePage bar. ● present · ◐ partial · ○ absent/N-A.
"Related linked?" = are related objects real `<Link>`s, not text.

| Object | Identity | Lifecycle | Financial summary | Accounting / journal | Related linked | Activity/History | Actions | Depth vs Invoice |
|---|---|---|---|---|---|---|---|---|
| **Invoice** (ref) | ● | ● | ● | ● | ● | ● | ● | — |
| Customer | ● | ◐ | ● | ○ (ledger-acct link) | ● | ● | ● | **closest** |
| Subscription | ● | ● | ◐ (no MRR) | ○ | ● | ● | ● (richest) | **2 core gaps** |
| Credit Note | ● | ◐ | ● | ● | ● | ○ | ● | no activity rail |
| Wallet | ● | ◐ | ● | ◐ (movements, not JE) | ● | ◐ | ● | not JE-wired |
| Account (ledger) | ● | ○ N/A | ● | ● (its core) | ◐ (no owner backlink) | ○ | ○ N/A | no activity rail |
| Quote | ● | ◐ | ● | ○ N/A | ● | ○ | ● | no activity rail |
| Plan | ● | ○ | ◐ | ○ N/A | ● | ● | ◐ (edit only) | thin lifecycle |
| Meter | ● | ○ N/A | ○ N/A | ○ N/A | ● | ◐ (audit, no timeline) | ○ | light by nature |
| Coupon | ● | ◐ | ○ (no agg value) | ○ N/A | ● | ○ | ◐ | thin |
| **Dispute** | ● | ◐ | ○ | ○ (credit-note outcome unlinked) | ● | ○ | ● | **thinnest money-adjacent** |
| **Payment** | — | — | — | — | — | — | — | **no page** |
| **Journal Entry** | — | — | — | — | — | — | — | **no page** |
| **Reconciliation Run** | — | — | — | — | — | — | — | **no page** |

**Ranked object-page backlog (worst gap first, excluding the 3 missing pages):**

1. **DisputePage** — no financial summary, no accounting impact, no activity;
   accept→"issue credit" *creates a credit note that posts to the ledger* but the
   outcome is neither shown as accounting nor linked to the created credit note
   (`DisputePage.jsx:73-92,228-242`). **(FRONTEND.)**
2. **SubscriptionPage** — no MRR/recurring-value strip, no journal/accounting
   (§6). *Highest business impact of the list* because of how central the object
   is. **(BACKEND GAP-4 for MRR + FRONTEND for journal.)**
3. **CreditNotePage / QuotePage / CouponPage** — missing `ObjectTimeline`/
   `AuditTrail` despite being free drop-ins (`CreditNotePage.jsx:217-337`,
   `QuotePage.jsx:254-375`). **(FRONTEND, cheap.)**
4. **AccountPage** — no back-link to the owning customer for AR sub-accounts even
   though the code knows it is one (`AccountPage.jsx:158-163`). **(FRONTEND, cheap.)**
5. **PlanPage** — no plan-level revenue rollup, no lifecycle. **(FRONTEND/BACKEND.)**

The depth **toolkit is fully shared** (`FinancialSummary`, `JournalEntries`,
`ObjectTimeline`, `AuditTrail`, `PaymentAttempts`, `LedgerAccountLink` all exist)
— so most of these are *omissions, not missing infrastructure*.

---

## 6. Financial Depth

**Can an operator answer "what is happening financially?" on each object?**

**Subscription — the biggest gap.** Present: plan amount/interval, current
period, upcoming invoice (`SubscriptionPage.jsx:633-640`), accrued usage
(`:851-886`), live proration preview on plan-change (`:566-587`). **Missing: MRR /
ARR, recurring-value strip, lifetime billed/collected/outstanding rollup, and any
journal/accounting impact.** Live-verified: the page shows *"$299.00 / month"* and
an *upcoming invoice*, but never states the MRR or what the subscription has
posted. **MRR is a BACKEND gap** — `CustomerFinancialSummary` (domain) carries
outstanding/past-due/billed/paid but **no MRR field**, and there is no
`/subscriptions/:id/financial-summary` (GAP-4). The journal section is a
**FRONTEND** gap (the `<JournalEntries>` component exists and is used on Invoice).

**Invoice — the model to copy.** Full amount breakdown (subtotal/tax/TDS/credit/
paid/due, `:422-510`), balanced journal legs (`:575-589`), payment attempts
(`:558-572`), status history (`:514-554`). Complete.

**Payment — no object at all.** The `PaymentAttempt` (domain) carries id,
invoice_id, gateway, method, status, `failure_code` (raw), amount, created/
settled timestamps. It has **no `customer_id`**, **no ledger-transaction linkage**,
and **no single-object read endpoint** (GAP-1). Live-verified: the Payments-log
row has no Customer column and drills to the *invoice*. A controller cannot open
one attempt's gateway trail. **(BACKEND GAP-1 + DATA-MODEL: add customer_id.)**

**Reconciliation run — summary only.** Recorded runs persist counts, not the
discrepancy rows (`reconciliation_runs` migration); there is no `/runs/:id`
(GAP-3), so a recorded run cannot be re-opened. **(BACKEND GAP-3 / DATA-MODEL.)**

---

## 7. Accounting Explainability

**"Why does the ledger look like this?"** — trace Subscription → Invoice →
Payment → Journal Entry → Ledger Account → Reconciliation, and mark where it
breaks:

| Hop | Status | Evidence | Classification |
|---|---|---|---|
| Customer → Subscription | ✅ link | `CustomerPage.jsx:261` | — |
| Subscription → Invoice | ✅ link | `SubscriptionPage.jsx:1042` | — |
| Invoice → Payment | ❌ attempts shown but **not links** (no payment page) | `InvoicePage.jsx:558-572` | BACKEND GAP-1 |
| Invoice → Journal Entry | ◐ postings shown inline, **no JE page** | `:583-589` | BACKEND GAP-2 |
| **Journal leg → Ledger Account** | ❌ **on the invoice, renders as plain text** | see below | **DATA-MODEL** |
| Ledger Account → counterpart | ✅ links | `AccountPage.jsx:166-232` | — |
| Reconciliation discrepancy → Invoice | ✅ link | `FinanceReconciliation.jsx:303-309` | — |
| Reconciliation discrepancy → **Transaction** | ❌ unlinked `–` (no JE page) | `:314-319` (live-verified) | BACKEND GAP-2 |
| Reconciliation run → re-open | ❌ history rows not addressable | `:360-406` | BACKEND GAP-3 |

**The "wired but dead" finding (new, not in prior audit).** `LedgerAccountLink`
was shipped and `JournalEntries.jsx:75-96` calls it — but the invoice/credit-note
`GeneralLedgerRow` payload carries account **name+code, not `debit_account_id`/
`credit_account_id`**, so on the invoice the journal legs fall back to **plain
text** (`LedgerAccountLink.jsx:16-19` documents this). The affordance *looks*
shipped (prior audit's P1-8 marked SHIPPED) but the invoice→account drill is a
**live dead end** — a trust gap because the operator sees links elsewhere and
expects them here. **Fix is DATA-MODEL** (add account ids to the journal-entries
payload), then it lights up for free.

**Separated gap classes:**

- **FRONTEND GAP:** add `<JournalEntries>` to SubscriptionPage; add
  `ObjectTimeline`/`AuditTrail` to the four pages missing them; add Dispute→
  credit-note outcome link.
- **BACKEND GAP:** GAP-1 payment read, GAP-2 journal-entry/ledger-transaction
  read + `/ledger/accounts/:id` already exists, GAP-3 per-run recon rows.
- **DATA-MODEL GAP:** account_ids on `GeneralLedgerRow`; `customer_id` on
  `PaymentAttempt`; an MRR field somewhere queryable; persisted per-run
  discrepancy rows.

---

## 8. Workflow Analysis

Legend: ✅ supported · ◐ partial · ❌ dead end. (detection / understanding /
decision / action / confirmation / result / audit-trail)

**1. Failed payment → collection → retry → recovery.**
Detection ✅ (Home payments card, live-verified). Understanding ◐ — humanized on
Collections/Home/Invoice, but **raw `failure_code` leaks** on `PaymentAttempts.jsx:88-90`
and `Payments.jsx:146`. Decision/Action ✅ (Collections retry/pause/mark-uncollectible,
amount-anchored). Confirmation ◐ — **no per-invoice "recovered" moment**; recovery
is only visible as the row later leaving the queue or an aggregate stat. Result/
Audit ✅. **Dead end** ❌ — payment attempt not addressable (GAP-1).

**2. Reconciliation → discrepancy → investigation → resolution.**
Detection ✅. Understanding ✅ (strong, labelled). Investigation ❌ — **transaction
cell unlinked** (`:314-319`); non-invoice discrepancies (`ledger_unbalanced`,
`customer_credit_liability_mismatch`) have **no drill target**. Resolution ❌ —
page is **read-only**, no "mark reviewed"/in-context next action (partly by design
per ANTI_PATTERNS "never silently correct the books", but the operator is left
without a next step). Run re-open ❌ (GAP-3).

**3. Subscription cancellation → preview → cancel → proration → accounting.**
Preview ❌ — **plan-change has a proration preview, cancel does not**
(`:1169-1177,339`); the only money-moving action with no amount. Action ◐ — plain
`Dialog`, not the amount-anchored `ConfirmDialog`. Result ◐ — resulting credit/
refund **does not link back** from the cancel flow. Accounting ❌ — invisible on
the subscription (§6). Audit ✅ (lifecycle history).

**4. Credit note → approval/rejection → accounting.** *Best-supported, effectively
end-to-end.* Role-gated approve/reject/void, all amount-anchored; dedicated
journal section; links to customer + offset invoice (`CreditNotePage.jsx:189-337`).
Only gaps: no activity rail; bulk-reject reports backend no-ops as "succeeded."

**5. New customer → subscription → invoice → payment → accounting.** Clean through
Invoice; **dead-ends at the accounting layer** exactly as §7 shows.

**Highest-leverage workflow fixes:** cancel money-preview (#3), a recovery-
confirmed moment (#1), and reconciliation transaction/run drill (#2).

---

## 9. Navigation & Discovery

**Can an operator reach any important object in 1–2 intentional actions?** Mostly
for *lists*; **not for accounting-tail objects, and not while preserving context.**

- **Breadcrumbs: built but used nowhere.** `PageHeader` supports a `breadcrumbs`
  prop (`PageHeader.jsx:15-38`); a repo-wide search finds **zero** pages passing
  it. Deep pages (`/ledger/accounts/:id`, `/dunning/campaigns/:id`) present no
  trail. **(FRONTEND.)**
- **Back-links reset list context.** `ObjectHeader` `backTo` is a **static link to
  the bare list root** (`ObjectPage.jsx:49`; e.g. `CustomerPage.jsx:162`), not
  `navigate(-1)` and not the preserved query string — so "Back to Customers"
  **discards** the filter/page/search the operator had. Only the browser Back
  button restores it, and the UI doesn't signal that. **(FRONTEND.)**
- **⌘K covers 3 of ~9 object types.** Live object search is Customers/Plans/
  Subscriptions only (`command-palette.jsx:66-68`); no jump-to-invoice,
  jump-to-payment, credit-note, dispute, quote, or ledger account. For a finance
  OS, "open INV-1234" from anywhere is table stakes. **(FRONTEND now; BACKEND
  GAP-6 to scale.)**
- **No org switcher in the header** — Organizations is buried in SYSTEM
  (`DashboardLayout.jsx:166-189`). **(FRONTEND, P3.)**

**Strengths:** URL-state on 16 lists (`useUrlState`), one-click reachability of
every books/payments surface, a proper combobox ⌘K (§15).

---

## 10. Data Density

Evaluated as "what does an operator need to *decide*", not "fit more columns."

- **Invoices** (live): Number · Customer · **Amount(total)** · Status · E-INVOICE
  · Date. *Missing the two fields that matter when working exceptions —
  **balance/amount-due** and **due date / days-overdue*** — while the India/EU-
  specific **E-INVOICE** column takes a prominent slot for most operators.
  **(FRONTEND; noise vs missing.)**
- **Subscriptions** (live): Customer · Status · Plan · **List price** · Start ·
  Next invoice. *Shows list price, not **MRR/recurring value**, and mixes /mo, /yr,
  and currencies in one column* — hard to compare. **(BACKEND GAP-4 + FRONTEND.)**
- **Payments** (live): When · Invoice · Amount · Method · Status · Reason. *No
  **Customer** column* (data-model: `PaymentAttempt` has no `customer_id`).
  **(DATA-MODEL.)**
- **Ledger / Reconciliation:** dense and correct; Reconciliation's expected/found/
  difference is exemplary.

Column verdict per list — essential-but-missing: invoice **balance + due date**,
subscription **MRR**, payment **customer**; noise-for-most: invoice **E-INVOICE**
(could be behind a jurisdiction toggle).

---

## 11. Design Language

**Typography.** Tremor 4-step scale (`tailwind.config.js:125-130`); Inter + self-
hosted JetBrains Mono; **financial numerals are tabular + monospaced** globally
(`index.css:94-96,101-110`). This already reads like finance software, not a
generic template. *Refinement, not overhaul:* a slightly larger metric step and
more deliberate numeric hierarchy on object headers would sharpen it.

**Color / surfaces / spacing.** Full semantic status tier as real tokens with
documented AA contrast (`index.css:41-53`); single radius ladder from
`--radius:0.5rem`; exactly three elevation levels. **One cohesive system — no
competing token source.** Restraint is good; no gratuitous gradients/shadows.

**Tables.** `DataTable` is v2 and strong — right-aligned numerics, priority
column hiding, sort with `aria-sort`, selection + bulk bar, real `<Link>`/`<button>`
first-cell activation, normalized pagination (`DataTable.jsx`). **Two gaps:** no
**sticky header** (`table.jsx:6` scrolls but `<thead>` doesn't stick) and no
`role="grid"`/arrow-key nav.

**Forms.** `FormSheet`/`FormField` right-side sheets, `ConfirmDialog` for
destructive, amount-anchored money confirms. Consistent.

**Does it look like a financial OS or a polished admin template?** *Polished, and
genuinely finance-aware at the primitive level (tabular money, status registry,
reconciliation clarity).* The thing that still reads "admin" is **not the visual
language — it's the shallowness behind the object headers** (a beautiful
Subscription page that doesn't tell you the MRR still feels like admin). Design is
not the bottleneck; depth is.

---

## 12. Motion

The motion system matches `MOTION.md`: **CSS + a little rAF, no framer-motion**,
reduced-motion-aware on two layers (`index.css:129-138` + `useReducedMotion.js`).
Primitives verified: `MotionNumber` (eased number changes, tabular, snaps under
reduced motion), `MotionReveal`/`MotionStagger`, `MotionState` (one-shot flash).
Applied to StatCard values, DataTable new-row reveal, StatusBadge flash-on-change,
EmptyState settle. **All 7 charter phases complete.**

**Where motion would still materially help (apply, don't invent):**

1. **Payment retry → recovered** — a settle/flash the moment an attempt succeeds
   (closes the "no recovery moment" gap in Workflow #1).
2. **Journal entry balancing** — the signature debits=credits settle, *when the new
   Subscription/Payment/JE pages ship*.
3. **Reconciliation discrepancy resolving → 0** — already the charter signature;
   extend to the (future) per-run page.
4. **Cancel preview → committed** — a brief confirm transition once the preview
   exists.
5. **⌘K result progressive reveal** — subtle, already partly present.

**Where motion would harm:** list re-sorts/filter changes (jitter on dense
financial tables), and any money value mid-flight that could be misread — keep
`MotionNumber` snapping under reduced motion (it does).

**Motion is effectively done; it is not a transformation lever.**

---

## 13. States

`DataTable` gives every list loading/empty/error for free; object/report pages
compose `ErrorState`/`EmptyState`/skeletons directly. Anti-zeros stance is real —
Dashboard and Collections surface systemic failures instead of a page of `$0`
(`Collections.jsx:166-208`, `Dashboard.jsx:86-116`).

**Missing/inconsistent states:**

- **No "stale/refetching" signal** anywhere — react-query `isFetching` is never
  surfaced, so finance screens swap data silently. For money software, "these
  numbers are updating" matters. **(FRONTEND.)**
- **No reusable permission/403 state** — `SubscriptionPage.jsx:216-224` hand-rolls
  403 copy; other pages fall to a generic ErrorState that can leak raw backend
  strings. **(FRONTEND.)**
- **"Processing/optimistic" is toast-only**, not inline on the row/object.

---

## 14. Trust & Safety

Strong baseline: exponent-aware money, amount-anchored destructive confirms,
per-record idempotency on bulk actions, partial-failure-first bulk runner,
role-gated credit-note approvals, `AuditTrail` on most objects.

**Trust gaps (ranked):**

1. **Immediate cancel with no preview** — a money-moving / credit-issuing action
   executes without showing the resulting amount (`:1169-1177,339`). **This is the
   clearest safety gap.** **(P0; BACKEND GAP-5 + FRONTEND.)**
2. **"Wired but dead" journal→account links on the invoice** — the UI implies a
   drill that silently isn't there (§7). Misleading traceability. **(P0/P1;
   DATA-MODEL.)**
3. **Raw `failure_code` to operators** on two surfaces while humanized elsewhere —
   an ANTI_PATTERNS "raw code to operator" violation and a consistency bug.
   **(P2; FRONTEND, cheap.)**
4. **Reconciliation resolution has no in-context action** — correct not to auto-
   fix, but the operator should at least be able to *record* a review. **(P2.)**

The UI never fakes success on ambiguous money state (bulk runner is honest;
Collections/Home degrade to "unavailable"). No fabricated metrics found.

---

## 15. Accessibility

**Strong:** semantic `<table>`/`<th scope>`; DataTable `aria-sort` + real
button/link controls; skeletons announce `aria-busy`; bulk bar is a labelled
`role="region"`; Radix Dialog/Sheet give focus-trap+restore; **⌘K is a proper
combobox** (`role=combobox`, `aria-activedescendant`, listbox options, live count,
arrow/Enter/Escape — `command-palette.jsx:192-298`).

**Gaps (user-impact order):**

1. **No sticky table headers** — long financial tables lose column context on
   scroll (usability + cognitive-a11y). **(FRONTEND.)**
2. **Tables have no accessible name** — `TableCaption` exists (`table.jsx:73-80`)
   but DataTable never sets one; screen-reader users get an unnamed table.
   **(FRONTEND, cheap.)**
3. **No `role="grid"`/arrow-key cell nav** — linear tab through dense tables.
   **(FRONTEND, lower priority.)**
4. **No reusable 403 affordance** (§13).

---

## 16. Responsive Design

**CODE-VERIFIED (not live pixel-tested):** breakpoint usage is real —
`DataTable` toolbar stacks `flex-col sm:flex-row`, column `hideBelow` priority-
hides at sm/md/lg, sheets are `w-full sm:max-w-md`, all tables wrapped in
`overflow-auto` with `min-w-[…]` so wide tables scroll rather than crush.

**LIVE-VERIFIED:** desktop/laptop widths only (this audit's screenshots). **Tablet
and mobile were NOT live-verified** — the 320–375px floor and fixed-width filter
Selects flagged in the prior audit remain **code-plausible-but-unconfirmed**. Do
not claim mobile QA that was not performed.

---

## 17. Performance

No real performance defect found; the earlier hidden-tab chart-freeze and
pagination-truncation issues were fixed in prior batches.

- **Home** splits the heavy 10-source overview from a light `["dashboard-collections"]`
  query so the attention surface paints early (`Dashboard.jsx`). Good.
- **⌘K** debounces 200ms, min-length 2, per-type limit 6, `staleTime` 30s, reads
  cached lists via `getQueryData`, cancels via AbortSignal — verified in prior
  batch (no duplicate requests).
- **Watch items (not defects):** several object pages fan out many parallel
  queries (SubscriptionPage), and the Subscriptions list computes on a
  `limit:1000` fetch — fine at demo scale, worth server-side aggregation when MRR
  lands. **(BACKEND, future.)**

---

## 18. Stripe-Quality Principles

Not "clone Stripe" — the principle behind each, and Recurso's domain answer:

| Stripe solves… | Principle | Recurso's equivalent (own domain) |
|---|---|---|
| Every object opens and every field drills | **Total addressability** | Give Payment, Journal Entry, Reconciliation-run real pages; make journal legs link (§4, §7) |
| The object tells you its financial state at a glance | **Depth at the header** | MRR + financial summary + journal on Subscription (§6) |
| You can trace a charge → balance → payout | **Follow-the-money chain** | Customer→Subscription→Invoice→Payment→JE→Reconciliation with no dead end (§7) |
| Destructive/money actions preview the result | **No blind money moves** | Cancel money-preview (§8, §14) |
| Jump to anything instantly | **Command-driven speed** | ⌘K over invoices/payments/credit-notes (§9) |
| You always know where you are | **Context you never lose** | Breadcrumbs + context-preserving back (§9) |
| The UI states what's happening | **Legible state** | Stale/refetching + 403 states (§13) |
| Restraint; nothing decorative | **Calm density** | *Already largely met* (§11) |

Recurso already matches Stripe on *restraint, money typography, and reconciliation
clarity*. It trails on *addressability, object depth, and follow-the-money
connectivity* — which is the whole game for a financial OS.

---

## 19. P0 / P1 / P2 / P3 Ranking

### P0 — Trust / correctness (can cause unsafe operation or misleading understanding)
- **P0-1 · Immediate cancel executes a money/credit action with no preview**
  (`SubscriptionPage.jsx:1169-1177,339`). BACKEND GAP-5 + FRONTEND.
- **P0-2 · Journal-leg→account links are silently dead on the invoice** (payload
  lacks account ids) — implies a traceability that isn't there. DATA-MODEL.
- **P0-3 · Raw gateway `failure_code` shown to operators** on `PaymentAttempts.jsx:88-90`,
  `Payments.jsx:146`. FRONTEND (cheap).
> *P0 is intentionally short — no catastrophic correctness bug exists; these are
> safety/trust edges, not broken books.*

### P1 — Financial-OS depth (prevents understanding/tracing the money lifecycle)
- **P1-1 · Subscription has no MRR / financial summary / journal** (§6). BACKEND
  GAP-4 (MRR) + FRONTEND (journal/summary).
- **P1-2 · Payment / Journal-Entry / Reconciliation-run not addressable** (§4).
  BACKEND GAP-1/2/3 + DATA-MODEL.
- **P1-3 · Reconciliation discrepancy transaction + run not drillable** (§8).
  BACKEND GAP-2/3.
- **P1-4 · Dispute outcome (issued credit note) not shown/linked as accounting**
  (`DisputePage.jsx:73-92`). FRONTEND.

### P2 — Operator productivity (slows daily work)
- **P2-1 · ⌘K covers only 3 object types** (§9). FRONTEND / BACKEND GAP-6.
- **P2-2 · No breadcrumbs; back-links reset list context** (§9). FRONTEND.
- **P2-3 · List columns show identity, not decisions** — invoice balance/due-date,
  subscription MRR, payment customer (§10). FRONTEND + DATA-MODEL.
- **P2-4 · No recovery-confirmed moment in the failed-payment workflow** (§8).
  FRONTEND.
- **P2-5 · Finance IA fragmented across BOOKS/REPORTS/PAYMENTS** (§3). FRONTEND.
- **P2-6 · Missing `ObjectTimeline`/`AuditTrail` on CreditNote/Quote/Coupon/Dispute**
  (§5). FRONTEND (cheap).

### P3 — Design / polish
- **P3-1 · No sticky table headers** (§11, §15). FRONTEND.
- **P3-2 · Tables lack accessible name** (§15). FRONTEND.
- **P3-3 · No stale/refetching + no reusable 403 state** (§13). FRONTEND.
- **P3-4 · Over-dense 9-group sidebar; org switcher buried; Audit-Log/Events
  adjacency** (§3, §9). FRONTEND.
- **P3-5 · Numeric hierarchy on object headers; jurisdiction-gate the E-INVOICE
  column** (§10, §11). FRONTEND.

---

## 20. Top 10 Gaps Preventing Stripe-Grade

**1. The money chain dead-ends: Payment, Journal Entry, and Reconciliation-run
are not addressable objects.**
*Why it matters:* "every important event is traceable" is the product's own
promise; it breaks at the accounting tail. *User impact:* a controller cannot
open a payment, a posting, or re-open a recorded reconciliation run.
*Class:* **BACKEND** (GAP-1/2/3) + **DATA-MODEL** (persist per-run rows, add
`customer_id`) + FRONTEND (3 new pages on the existing `ObjectPage` framework).
*Complexity:* L. *Dependencies:* the three read endpoints. *Solution:* add
`GET /payments/:id`, `GET /ledger/transactions/:id` (+ `/journal-entries/:id`),
`GET /finance/reconciliation/runs/:id`; then build three pages reusing
`FinancialSummary`/`JournalEntries`/`ObjectTimeline`.

**2. The Subscription — the central recurring-revenue object — has no MRR,
financial summary, or accounting impact.**
*Why:* a revenue operator's home object should state its recurring value and what
it posts. *Impact:* the page feels like admin metadata, not a financial object.
*Class:* **BACKEND** (GAP-4 MRR) + **FRONTEND** (drop in `<FinancialSummary>` +
`<JournalEntries>` — both exist). *Complexity:* M. *Deps:* MRR field /
`/subscriptions/:id/financial-summary`. *Solution:* add MRR to a queryable
summary; add a financial-summary strip and a journal section to `SubscriptionPage`.

**3. Immediate cancel posts blind — the one money-moving action with no preview.**
*Why:* a financial OS never moves money without showing the amount. *Impact:*
operators cancel without seeing the refund/credit/proration. *Class:* **BACKEND**
(GAP-5 cancel-preview) + **FRONTEND**. *Complexity:* M. *Deps:* mirror the existing
plan-change preview endpoint for cancel. *Solution:*
`GET /subscriptions/:id/cancel-preview` → refund/credit/ledger effect; render it
in an amount-anchored `ConfirmDialog`.

**4. Journal legs don't link to ledger accounts on the invoice (wired but
dataless).**
*Why:* the "explain any number" drill silently dies exactly where it's most
expected. *Impact:* misleading — links work on `/ledger` but not from the invoice.
*Class:* **DATA-MODEL** (add `debit_account_id`/`credit_account_id` to
`GeneralLedgerRow`) + FRONTEND (already ready via `LedgerAccountLink`).
*Complexity:* S. *Deps:* payload change on `/invoices/:id/journal-entries`.

**5. ⌘K searches only 3 of ~9 object types.**
*Why:* command-driven speed is table stakes for operators. *Impact:* can't jump to
an invoice/payment/credit-note. *Class:* **FRONTEND** now (client-side over cached
lists) / **BACKEND** GAP-6 to scale. *Complexity:* M. *Solution:* add invoice/
credit-note/dispute groups to the palette; a `GET /search` later for scale.

**6. No breadcrumbs and back-links that reset list context.**
*Why:* "you never lose your place" is core to feeling fast. *Impact:* returning
from a detail discards filters/page. *Class:* **FRONTEND**. *Complexity:* S–M.
*Solution:* pass `breadcrumbs` (the prop exists) on object pages; make
`ObjectHeader` back preserve the list's query string (or `navigate(-1)`).

**7. List columns are built for identity, not decisions.**
*Why:* operators triage from the list. *Impact:* Invoices lack balance + due-date;
Subscriptions show list-price not MRR; Payments lack a customer column.
*Class:* **FRONTEND** + **DATA-MODEL** (`customer_id` on payment, MRR).
*Complexity:* S–M. *Solution:* add the decision columns; gate E-INVOICE by
jurisdiction.

**8. Raw gateway `failure_code` leaks to operators on two surfaces.**
*Why:* ANTI_PATTERNS forbids raw codes; it's also inconsistent with Collections/
Home. *Impact:* an operator sees `authentication_required` vs a clean label.
*Class:* **FRONTEND** (cheap). *Complexity:* S. *Solution:* route
`PaymentAttempts.jsx`/`Payments.jsx` through the shared `humanizeFailure`.

**9. Dense financial tables lose their headers on scroll; tables are unnamed to
AT.**
*Why:* scanning a long ledger without column headers is error-prone. *Impact:*
usability + a11y on the surfaces operators stare at longest. *Class:* **FRONTEND**.
*Complexity:* S. *Solution:* sticky `<thead>` in the `Table` primitive; wire
`TableCaption`/`aria-label` through `DataTable`.

**10. State legibility gaps: no "refetching/stale" signal, no reusable 403.**
*Why:* money software must say what's happening. *Impact:* silent data swaps;
raw backend strings on forbidden. *Class:* **FRONTEND**. *Complexity:* S–M.
*Solution:* surface react-query `isFetching` as a subtle indicator; add a
`PermissionState` variant to `ErrorState`.

---

## 21. Target Recurso Experience — "The Recurso Dashboard Experience"

Not marketing — the actual interaction model, built on Recurso's domain:

**The spine.** Every session starts at an **exception-first Home** ("what needs my
attention now?" — already shipped) and moves *through objects*, never through
dead-end reports:

```
Home (exception)
  → open the object it names (invoice / subscription / customer)
    → read its financial state at the header (amount, MRR, balance, status, why)
      → move to any related object as a link
         (Customer → Subscription → Invoice → Payment → Journal Entry
          → Ledger Account → Reconciliation)
        → take the one supported action, previewed with its amount + accounting effect
          → see the result and its posting confirmed inline (motion)
            → and the whole thing is in the audit trail
```

**The three rules that make it feel like an OS, not an admin panel:**

1. **Every object opens, and every number drills.** No inline-only postings, no
   unlinked transaction ids, no "list but can't open." A payment, a journal entry,
   and a reconciliation run are things you can *open, bookmark, and share*.
2. **The header answers "what's happening financially" before you scroll.** MRR on
   a subscription, balance + due date on an invoice, gateway + reconciliation
   state on a payment.
3. **Money moves are always previewed and always traceable afterward.** No blind
   cancels; every consequential action states what/amount/resulting-state/
   accounting-impact/reversibility, and links to what it produced.

An operator should be able to start from a failed-payment on Home and, without
typing a URL or losing their list, walk to the invoice, see why it failed, retry
it, watch it recover, and open the resulting ledger posting — then ⌘K to the next
customer by name. **That continuous, addressable, previewed, traceable flow is the
product.**

---

## 22. Recommended Roadmap (5 phases, optimised for transformation)

> Phases are sequenced by *leverage and unblocking*, not PR count. Phases 1–3 are
> the transformation; 4–5 are the finish. Each phase is gated on the backend work
> named in it — file those endpoints first (they are small, additive reads).

**Phase 1 — Financial object depth (make the objects tell their money story).**
Subscription MRR + financial-summary strip + journal section; decision-oriented
list columns (invoice balance/due-date, subscription MRR); dispute→credit-note
accounting linkage; add the missing `ObjectTimeline`/`AuditTrail` drop-ins.
*Backend needed:* GAP-4 (MRR / subscription financial-summary). *Frontend:* reuse
existing primitives.

**Phase 2 — Accounting connectivity (make the money chain fully addressable).**
Ship Payment, Journal-Entry, and Reconciliation-run object pages; light up
journal-leg→account links on the invoice; make reconciliation transaction cells
and run-history rows drillable. *Backend needed:* GAP-1/2/3 + DATA-MODEL
(account_ids on `GeneralLedgerRow`, persisted per-run rows, `customer_id` on
payment).

**Phase 3 — Operational workflows (make money moves safe and closed-loop).**
Cancel money-preview via amount-anchored confirm; a "payment recovered" confirmation
moment; a reconciliation "record review" affordance; humanize `failure_code`
everywhere. *Backend needed:* GAP-5 (cancel-preview).

**Phase 4 — Navigation & discovery (make it feel fast, never lose your place).**
Breadcrumbs on object pages; context-preserving back; expand ⌘K to invoices/
credit-notes/disputes; consolidate the finance IA (Books vs Reports sub-sections),
surface an org switcher. *Backend optional:* GAP-6 (`/search`) to scale ⌘K.

**Phase 5 — Interaction & density refinement.**
Sticky table headers + accessible table names; stale/refetching + reusable 403
states; numeric hierarchy on object headers; jurisdiction-gate the E-INVOICE
column; apply the motion moments (§12) to the new object pages. *Frontend only.*

---

## 23. Backend Gaps (unchanged since prior audit unless noted — none should be faked)

> **Batch 3A update:** GAP-1..GAP-5 are now **CLOSED** (endpoints shipped — see the
> Batch 3A section above). GAP-6/7/8 remain open. The table below is the original
> reassessment; the Status column reflects Batch 3A.

| # | Missing | Unblocks | Status |
|---|---|---|---|
| GAP-1 | `GET /payment-attempts/:id` | Payment object page; Invoice→Payment link | ✅ **CLOSED** (Batch 3A) |
| GAP-2 | `GET /ledger/transactions/:id` + account-ids on journal payload | Journal-Entry page; recon transaction drill; invoice JE→account | ✅ **CLOSED** (Batch 3A) |
| GAP-3 | `GET /finance/reconciliation/runs/:id` (per-run rows) | Re-openable recorded runs | ✅ **CLOSED** (Batch 3A; migration 000171 persists rows) |
| GAP-4 | `GET /subscriptions/:id/financial-summary` + MRR | Subscription depth; MRR columns | ✅ **CLOSED** (Batch 3A; MRR reuses `monthlyMinorUnits`) |
| GAP-5 | `GET /subscriptions/:id/cancel-preview` | Cancel money-preview | ✅ **CLOSED** (Batch 3A; mutation unchanged) |
| GAP-6 | `GET /search` (cross-object) | ⌘K at scale | ⬜ open |
| GAP-7 | `GET /payments?status=failed&since=…` with customer linkage | distinct recent-failure feed | ⬜ open |
| GAP-8 | `GET /webhooks/failures` (or `/events?delivery=failed`) | failed-webhook attention tile | ⬜ open |

*(All eight re-verified against current `main.go`/`openapi.yaml` — accurate, none
changed.)*

---

## 24. Frontend Gaps (no backend dependency — can ship now)

1. Add `<FinancialSummary>` + `<JournalEntries>` to `SubscriptionPage` (journal
   only where postings are already returned; MRR gated on GAP-4).
2. Add `ObjectTimeline`/`AuditTrail` to CreditNote, Quote, Coupon, Dispute pages.
3. Dispute → link the credit note it issued; show it as accounting.
4. Humanize `failure_code` on `PaymentAttempts.jsx:88-90`, `Payments.jsx:146`.
5. Breadcrumbs on object pages (`PageHeader` prop already exists, unused).
6. Context-preserving back in `ObjectHeader` (`ObjectPage.jsx:49`).
7. Expand ⌘K groups to invoices/credit-notes/disputes (client-side over cached
   lists).
8. Decision columns: invoice balance + due-date; subscription MRR (post-GAP-4);
   jurisdiction-gate E-INVOICE.
9. Sticky `<thead>` in `table.jsx`; `TableCaption`/`aria-label` through `DataTable`.
10. `isFetching` stale indicator; reusable `PermissionState`/403 in `ErrorState`.
11. AccountPage → back-link to owning customer for AR sub-accounts.
12. Finance IA consolidation; org switcher in header.

---

## 25. Data-Model Gaps (schema/payload changes, distinct from missing endpoints)

1. **`GeneralLedgerRow` lacks `debit_account_id`/`credit_account_id`** — journal
   legs can't link to accounts on the invoice/credit-note (§7). Additive payload
   field; unblocks the drill for free.
2. **`PaymentAttempt` has no `customer_id`** — Payments list can't show a customer
   column and can't power a per-customer failed-payment feed (§10, GAP-7).
3. **No MRR field anywhere queryable** — neither the subscription nor the customer
   financial summary carries recurring value (§6, GAP-4).
4. **`reconciliation_runs` persists summary counts, not discrepancy rows** — a
   recorded run can't be re-opened (§4, GAP-3).

---

## 26. Definition of "Stripe-Grade" (for Recurso's domain)

Recurso is "Stripe-grade" when a senior finance/revenue operator can, without
typing a URL or losing their place:

1. **Open anything.** Every object — including Payment, Journal Entry, and a
   Reconciliation run — has a stable URL, a page, and the seven depth dimensions.
2. **Read the money at the header.** MRR on a subscription, balance+due on an
   invoice, gateway+reconciliation state on a payment — before scrolling.
3. **Follow the money with no dead end.** Customer → Subscription → Invoice →
   Payment → Journal Entry → Ledger Account → Reconciliation, each hop a link.
4. **Never move money blind.** Every consequential action previews its amount and
   accounting effect, and links to what it produced.
5. **Move at the speed of thought.** ⌘K opens any object by name/number;
   breadcrumbs and context-preserving back mean you never lose your list.
6. **Always know the state.** Loading, refetching/stale, empty, partial, error,
   forbidden, processing, and completed are each communicated — never a silent
   swap, never a false "all clear."
7. **Trust the books.** The reconciler explains every discrepancy in words, every
   number is exponent-correct and explainable, and every action is in the audit
   trail. *(Already met — this is the foundation the rest builds on.)*

Recurso already owns #7 and most of the visual/typographic bar. Grade is reached
by closing #1–#6 — which is what the roadmap above does.

---

*End of audit. No production code was modified. Reviewer note: items marked
"changed since prior audit" (P0-1 truncation, P1-8 journal-leg linking, wider
URL-state adoption, the `SubscriptionPage:1070` bug, the reconciliation caption)
were re-verified as fixed in current code and are not re-litigated here; the prior
audit's core thesis — the accounting tail is not addressable and Subscription is
the weakest gold page — still holds and is the spine of this reassessment.*
