# Dashboard Polish — Batch E Design
## Object-Page Parity & Payments Cleanup — INVESTIGATION (no production code)

> **Read-only, code-cited, point-in-time (2026-08-15), against `main` @ `ec6dc181`.**
> No production code was written for this document. Batch E investigation
> deliverable; ends with a STOP for approval.
>
> Principle: *same semantic object → same interaction & visual language* — **not**
> *every page identical*. Specialized accounting/financial presentations are
> preserved, not forced into generic CRUD abstractions. No backend work; data-model
> limits are documented, never faked. No new page-specific primitives.

---

## TRACK A — Raw / specialized tables

Four raw `<table>`s exist. Each is scored against the brief's five questions.

| Table | Worklist? | DataTable adds value? | Migration preserves semantics? | Reduces bespoke code? | Loses specialized presentation? | **Verdict** |
|---|---|---|---|---|---|---|
| **AccountPage** — journal activity (Date · Posting · Against · **Debit** · **Credit**) | No (read-only ledger postings, "first 100" cap) | No | No — DataTable can't model the debit-or-blank / credit-or-blank ledger columns | No | **Yes** (debit/credit split is specialized) | **Keep specialized** |
| **WalletPage** — transactions (Date · Movement · Detail · …) | No (chronological wallet ledger) | No | No | No | Yes | **Keep specialized** |
| **QuotePage** — line items (Description · Qty · Unit price · Amount) | No (document line items) | No | No | No | Yes — richer than Invoice's breakdown; a tabular Qty/unit layout is the right presentation | **Keep specialized** |
| **PricingSimulator** — computed pricing preview (Metric · Model · Qty · …), in a slide-over | No (transient computed preview) | No | No | No | Yes | **Keep specialized** |

**None warrant DataTable** (none are operator worklists; all are specialized
accounting / document / preview presentations that already use `<Money>`,
`tabular-nums`, the Batch-C-aligned header, hover, and `divide-y`). Forcing them
into DataTable or even a straight `ui/table`-primitive swap would (a) change their
intentional compact density (`px-6 py-2.5` vs `ui/table`'s `p-4`) and (b) not model
the debit/credit ledger columns. This is exactly the "don't migrate because it
contains `<table>`" case.

**The one genuine seam is accessibility**, not visuals: the raw `<th>`s lack
`scope="col"` and the `<table>`s have no accessible name (the DataTables got both
in Batch D). **Proposed minimal action (Track A):** add `scope="col"` to each
header cell and an accessible name (`aria-labelledby` the enclosing `ObjectSection`
title, or an `aria-label`) to each `<table>`. Zero visual/density change; closes
the a11y seam; preserves the specialized presentation. **No** `AccountTable` /
`WalletTable` / `QuoteTable` / `PricingTable` primitive.

_(Optional, for approval: PricingSimulator is a transient in-slide-over preview —
lowest stakes; a11y hardening there is nice-to-have. Recommend hardening all four
for consistency.)_

---

## TRACK B — Payments cleanup

`Payments.jsx` (the payments log — a real DataTable). Money is **already** `<Money>`
(line 119) — no money gap here. Two genuine gaps:

### B1. Status — bespoke `StatusPill` → canonical `StatusBadge`
`Payments.jsx:43` defines a bespoke `StatusPill` rendering the **raw lowercase
status** (`{status}`) via a local `STATUS_TONE` map (line 139 uses it) — bypassing
the sanctioned `StatusBadge`, so the list disagrees with the payment's own detail
page and every other object. **Fix:** replace `StatusPill` with `StatusBadge`
(delete the pill + `STATUS_TONE`). Payment statuses seen: `failed`, `returned`,
`processing`, `initiated`, `succeeded`. `StatusBadge` REGISTRY already maps
`processing→info`, `succeeded→success`, `failed→destructive`. **Missing:**
`returned` and `initiated`. Per the registry's own docstring ("extend REGISTRY here
instead" of per-page maps), add:
- `returned: "destructive"` (an ACH/card return is a failure-adjacent state — matches the current pill's destructive tone)
- `initiated: "neutral"` (an in-progress start; `neutral` = the current pill default)

This is a clean extension of the existing primitive — **not** a new
`PaymentStatusBadge`. Labels humanize correctly ("Returned", "Initiated",
"Processing", "Succeeded", "Failed").

### B2. Failure codes — raw `failure_code` → shared `humanizeFailure`
`Payments.jsx:146` renders the **raw gateway code** (`R01`, `insufficient_funds`) in
`font-mono text-destructive` as the primary explanation — the exact leak already
fixed elsewhere. The shared `humanizeFailure` (`src/lib/failureLabels.js`) is used
by `PaymentAttempts`, `Collections`, and `PaymentPage`. **Fix:** lead with
`humanizeFailure(p.failure_code)` (human reason — what happened / what it means),
keeping the raw code as quiet technical detail in a `title=` attribute — mirroring
`PaymentAttempts.jsx:87-99` exactly. No SQL/exception/gateway string is ever the
primary explanation. **No payment behaviour change.**

### B3. Money — already canonical (no change)
`Payments.jsx:119` already uses `<Money amountMinor currency>`. No `formatCurrency`/
`formatMinorUnits` in the file. Nothing to migrate.

---

## OBJECT-PAGE PARITY (vs the Invoice / Subscription reference)

Survey of the 8 object pages (StatusBadge / AttentionBanner / rail primitives /
rail-section title):

| Page | Status | Attention | Timeline | Audit | Rail title | Assessment |
|---|---|---|---|---|---|---|
| **Invoice** (ref) | StatusBadge | ✓ | ObjectTimeline | AuditTrail | "Details" | reference |
| **Subscription** (ref) | StatusBadge | ✓ | ObjectTimeline | AuditTrail | **"Metadata"** | title drift only |
| **Payment** | StatusBadge | ✓ | ObjectTimeline | — | "Details" | audit absent = backend-gated (see below) |
| **Journal Entry** | — | — | — | — | "Details" | **intentional** — immutable posting, no lifecycle/status |
| **Reconciliation Run** | **raw `<Badge>`** | **none** | — | — | **"Scope"** | **fix** (see R-1) |
| **Credit Note** | StatusBadge | ✓ | — | **— (has real data)** | "Details" | **add AuditTrail** (see R-2) |
| **Dispute** | StatusBadge | ✓ | — | — | "Details" | audit/timeline **backend-gated** (see R-2) |
| **Account** | (type Badge — correct) | — | — | — | **"Metadata"** | raw table (Track A) + title drift; no status (a ledger account has none) |

### R-1. Reconciliation Run
- **raw `<Badge variant=…>` → `StatusBadge`** for the header verdict. Add
  `reconciled: "success"` + `discrepancies: "destructive"` to REGISTRY; render
  `<StatusBadge status={balanced ? "reconciled" : "discrepancies"} label={count…}/>`
  (StatusBadge's `label` prop carries the count). Aligns the most audit-critical
  page with every other object's status rendering.
- **Add an `AttentionBanner`** when `total_discrepancies > 0` (the archetypal
  exception state — Invoice/Subscription/Payment all banner their exceptions),
  keeping the discrepancies table below. The verdict stops being a quiet section.
- **"Scope" rail title:** see the rail-title decision below.
- **USD-hardcoded amounts are NOT a defect** — `ReconciliationRunPage.jsx:33`
  documents ADR-010 (the ledger is single functional currency), so discrepancy
  amounts are correctly in the functional currency. Leave; documented.

### R-2. Credit Note / Dispute — Timeline + Audit (backend reality)
The audit wanted a Timeline+Audit rail here. Backend check:
- **Audit** (`AuditTrail` → `GET /audit-logs?entity_type=…`) is recorded by a
  route-**allowlisted** middleware. The allowlist **includes `credit-notes`** but
  **not `disputes`**.
- **Timeline** (`ObjectTimeline` → `GET /events?object_id=…`) — **no domain events
  are emitted** for credit notes or disputes.

Therefore:
- **Credit Note → add `<AuditTrail entityType="credit-notes" entityId={id}/>`** to
  its rail — real data (approve/reject/void are logged), reuses the existing
  primitive, closes audit #20's audit gap. **Timeline stays absent** (no events →
  would be permanently empty) — **backend-gated, documented.**
- **Dispute → no Timeline, no Audit.** Both are backend-gated (`disputes` not on
  the audit allowlist; no events emitted). Adding either would show a permanently
  empty rail — worse than absence. **Documented as a backend/data-model gap**
  (adding `disputes` to the audit allowlist / emitting events is backend work,
  out of scope).

### Rail-title drift (audit #14) — decision
Titles for the object's attribute/metadata rail section: **"Details"** (Invoice,
Payment, Journal Entry, Credit Note, Dispute) vs **"Metadata"** (Subscription,
Account) vs **"Scope"** (Reconciliation Run). "Details" is the reference + majority.
**Proposed:** align **Subscription** and **Account** "Metadata" → **"Details"**.
Keep **"Scope"** on Reconciliation Run (it labels the *run's* scope — invoices
checked, recorded-by, tigerbeetle-compared — a domain-meaningful heading, not
generic object details). Small, safe, label-only. _(Decision to confirm — could
also standardize Scope→Details, or keep all as-is.)_

### Shared Details+Timeline+Audit rail — NOT extracted
`ObjectTimeline` and `AuditTrail` are **already shared primitives**; each page
composes them in ~2 lines. A combined `<ObjectRail>` would have to accept each
object's entity type, data, and section config → **page-specific props**, i.e. the
"ObjectPageEverything" anti-pattern the brief forbids. The existing primitives
already deliver the reuse. **No new rail primitive.** Documented.

---

## Accessibility (verify + the Track-A hardening)
- Track A adds `th scope="col"` + an accessible table name to the 4 raw tables
  (the only real a11y gap found).
- StatusBadge (Track B, R-1) keeps status meaning as text + tone (never colour-only)
  and is keyboard/SR-neutral.
- `humanizeFailure` keeps the failure reason as real text; raw code is a `title=`
  hint (quiet, non-primary).
- No heading/label/link/button semantics change elsewhere.

## Money — audited, already canonical
Payments (`<Money>` ✓), AccountPage / WalletPage / QuotePage / PricingSimulator all
already render amounts via `<Money>` with `tabular-nums`. **No raw currency
formatting to migrate.** `<Money>` itself untouched.

---

## Proposed change set (smallest that closes the seams)

1. **Track A a11y:** `scope="col"` + accessible name on the 4 raw tables. No
   structural/visual change.
2. **Track B:** Payments `StatusPill`→`StatusBadge`; raw `failure_code`→
   `humanizeFailure` (+ raw code in `title`). Extend `StatusBadge` REGISTRY
   (`returned`, `initiated`) — and (`reconciled`, `discrepancies`) for R-1.
3. **Reconciliation Run:** raw `Badge`→`StatusBadge`; `AttentionBanner` when
   discrepancies > 0.
4. **Credit Note:** add `AuditTrail` (real data).
5. **Rail titles:** Subscription + Account "Metadata"→"Details" (pending decision).

**Deliberately unchanged (documented):** the 4 raw tables' structure/density
(specialized); Journal Entry's minimal rail (immutable posting); Account's type
Badge (not a status); Reconciliation USD amounts (ADR-010); Dispute Timeline+Audit
& Payment/CreditNote Timeline (backend-gated); `<Money>`, ObjectHeader, useObjectQuery.

## Backend / data-model gaps discovered (documented, not fixed)
- **`disputes` absent from the audit allowlist** (`middleware/audit.go`) → no audit
  trail available for disputes.
- **No domain events emitted for credit notes or disputes** → no object timeline
  available for either.
- **Reconciliation run currency** — single functional currency by ADR-010 (this is
  a documented decision, not a gap).

## STOP conditions honoured
No shared primitive needing page-specific props (none proposed); no accounting
presentation made worse (raw tables kept); no ObjectHeader redesign; no backend
change (gaps documented); no new abstraction duplicating a primitive; scope stays
out of Batch F.

---

## Decisions for approval
1. **Track A** — keep all 4 raw tables specialized; apply minimal a11y hardening
   (`th scope` + accessible name). Confirm (vs. leave entirely / vs. migrate).
2. **Rail titles** — align Subscription + Account "Metadata"→"Details", keep
   Reconciliation "Scope". Confirm (vs. standardize all / vs. leave).
3. **Credit Note AuditTrail** — add it (real data); Dispute + all Timelines stay
   absent as backend-gated. Confirm.
4. **Reconciliation Run** — StatusBadge + AttentionBanner-on-discrepancies. Confirm.

**STOP — investigation complete. Awaiting approval before any production code.**
