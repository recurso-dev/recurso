# Dashboard Polish — Batch E Report
## Object-Page Parity & Payments Cleanup

Status: **complete, awaiting review.** The remaining "special implementation"
seams between financial objects are closed, while genuinely specialized
accounting presentations are preserved. Implements the approved design
(`DASHBOARD_POLISH_BATCH_E_DESIGN.md`, PR #706). Shipped as **#707**. No new
page-specific primitives, no DataTable migration of specialized tables, no
backend changes, no motion/token changes; `<Money>` / `ObjectHeader` /
`useObjectQuery` untouched. Lint, build, and the full **720**-test suite are green.

---

### 1. Raw-table inventory + migration decisions

| Table | Nature | Decision | Rationale |
|---|---|---|---|
| **AccountPage** — journal activity (Date · Posting · Against · Debit · Credit) | specialized ledger (debit/credit columns) | **Keep specialized; a11y-harden** | Not a worklist; DataTable can't model debit-or-blank / credit-or-blank columns; migration would change intentional density |
| **WalletPage** — movements | specialized wallet ledger | **Keep specialized; a11y-harden** | same |
| **QuotePage** — line items (Description · Qty · Unit price · Amount) | specialized document line-items | **Keep specialized; a11y-harden** | richer than Invoice's breakdown; a tabular Qty/unit layout is the correct presentation |
| **PricingSimulator** — computed pricing + GL preview | specialized preview (in a slide-over) | **Keep specialized; a11y-harden + money fix** | transient preview; also had raw `formatCurrency` (money-gap, fixed) |

**None migrated to DataTable** (none are operator worklists). The **a11y
hardening** (the only real seam) added `th scope="col"` to every column header
and an accessible name to every table — `"Journal activity"` / `"Wallet
movements"` / `"Quote line items"` / `"Simulated charges"` + `"GL preview"` — with
**zero visual or density change**. **No** `AccountTable`/`WalletTable`/`QuoteTable`/
`PricingTable` primitive.

### 2. Payments cleanup

- **Status → canonical `StatusBadge`.** Deleted the bespoke `StatusPill` and its
  local `STATUS_TONE` map (raw lowercase status, mono pill). The status column now
  renders `<StatusBadge status={p.status} />` — identical rendering to the payment's
  own detail page and every other object.
- **`<Money>` unchanged** — Payments already rendered amounts via `<Money>` (no
  money migration needed here).

### 3. Failure-code coverage

Every user-visible raw gateway code on the Payments log is now humanized. The
"Reason" column was `<span class="font-mono text-destructive">{p.failure_code}</span>`
(raw `R01` / `card_declined` as the primary explanation) → now
`humanizeFailure(p.failure_code)` (the shared helper, e.g. "Card declined") with
the **raw code preserved as a quiet `title=` technical detail**, mirroring
`PaymentAttempts` exactly. No SQL / exception / gateway string is ever the primary
explanation. **`humanizeFailure`** is now used by PaymentAttempts, Collections,
PaymentPage, **and Payments** — full coverage of the app's failure-code surfaces.

### 4. StatusBadge migration + registry

`StatusPill` → `StatusBadge` required two additions to the canonical REGISTRY (per
its own docstring: *"extend REGISTRY here instead"* of per-page maps), and **only**
these two, as scoped:
- `initiated: "neutral"` — an in-progress start.
- `returned: "destructive"` — an ACH/card return, failure-adjacent.

`processing`/`succeeded`/`failed` were already mapped. No `PaymentStatusBadge`, no
new status abstraction. Labels humanize correctly ("Returned", "Initiated", …).

### 5. Object-page parity findings + fixes

- **Reconciliation Run** — raw `<Badge variant=…>` → `<StatusBadge>` (reusing the
  existing `success`/`failed` statuses with a `label` override for the count, so
  **no recon-specific registry entries**). Added an **`AttentionBanner`** when
  `total_discrepancies > 0` — the exception state now banners like Invoice/
  Subscription/Payment, instead of being a quiet section. The most audit-critical
  page now shares the object status + exception language.
- **Credit Note** — added a real **`AuditTrail`** rail section (`entityType=
  "credit-notes"`), since `credit-notes` is on the backend audit allowlist.
- **Rail titles** — Subscription + Account "Metadata" → **"Details"** (align to
  the Invoice reference / majority). **Reconciliation keeps "Scope"** (labels the
  run's scope — invoices checked, recorded-by, tigerbeetle — a domain-meaningful
  heading, not generic object details).
- **Journal Entry / Account** — intentionally minimal rails (an immutable posting
  and a chart-of-accounts entry have no lifecycle/status) — **unchanged**. Account's
  `<Badge>` is the account *type* (not a status) → correctly a neutral Badge, **not**
  forced onto StatusBadge.

### 6. Shared-primitive decisions

- **No shared `ObjectRail` (Details+Timeline+Audit) primitive.** `ObjectTimeline`
  and `AuditTrail` are *already* shared primitives, composed per page in ~2 lines;
  a combined rail would need each object's entity type + data + section config →
  page-specific props (the "ObjectPageEverything" anti-pattern). Existing
  primitives already deliver the reuse.
- **No new status / table / money primitives.** Everything reused: `StatusBadge`
  (extended), `AttentionBanner`, `AuditTrail`, `humanizeFailure`, `<Money>`.

### 7. Pages changed (9 + 5 tests)

`AccountPage`, `WalletPage`, `QuotePage`, `PricingSimulator` (a11y + PricingSim
money), `Payments` (StatusBadge + humanizeFailure), `status-badge.jsx` (registry
+2), `ReconciliationRunPage` (StatusBadge + AttentionBanner), `CreditNotePage`
(AuditTrail), `SubscriptionPage` (rail title). Tests: `Payments`, `status-badge`,
`ReconciliationRunPage`, `CreditNotePage`, `QuotePage`.

### 8. Pages deliberately unchanged (and why)

- **Journal Entry** — immutable posting; minimal rail is intentional.
- **Account status** — a ledger account has no lifecycle status; its type Badge is
  correct (not a StatusBadge candidate).
- **The 4 raw tables' structure/density** — specialized presentations preserved.
- **Reconciliation discrepancy amounts** — kept on their consistent signed-delta
  formatters (`formatMinorUnits` / `formatDifference`); migrating only the absolute
  columns to `<Money>` would break within-row consistency, and the Difference
  column is a signed delta, not a plain amount. Specialized-and-consistent.
- **Dispute** — no Timeline/Audit added (backend-gated, §11).
- **`<Money>`, `ObjectHeader`, `useObjectQuery`, DataTable behaviour** — untouched.

### 9. Accessibility verification

- 4 raw tables: `th scope="col"` on every header + an accessible table name.
- `StatusBadge` keeps status meaning as text + tone (never colour-only), keyboard/
  SR-neutral.
- `humanizeFailure` keeps the failure reason as real text; the raw code is a `title`
  hint only.
- No heading/label/link/button semantics changed elsewhere.

### 10. Visual QA

Verified on the deployed Batch E build (app.recurso.dev, post-#707). Invoice is
the reference; the side-by-side shows Payment / Credit Note / Reconciliation now
speak the same status / exception / rail language.

| Page | Confirmed live |
|---|---|
| **Payments** | Status renders as canonical **StatusBadge** ("Processing" info, "Returned" destructive — matching Invoice's "Past due"); the failure reason is **humanized** ("ACH: insufficient funds"), not a raw code (raw code moved to `title`) |
| **Reconciliation Run** | Verdict is a **StatusBadge** ("12 discrepancies", destructive) + a new **AttentionBanner** ("⚠ 12 discrepancies — … Review each below.") banners the exception at top; "Scope" rail kept; discrepancy table (specialized formatters) intact |
| **Credit Note** | New **"Audit trail"** rail section renders **real** data ("POST /v1/credit-notes/:id/reject · Aug 15" + "View full audit log") |
| **Quote** | Line-items table: accessible name **"Quote line items"**, all `th scope=col`, Money signature present — specialized layout unchanged |
| **Account** | Rail title now **"Details"** (was "Metadata"); journal-activity table accessible name **"Journal activity"**, all `th scope=col`, Money present |
| **Subscription** | Rail title now **"Details"**; rail = Details → Timeline → Audit trail (reference-tier) |
| **Invoice** (reference) | Unchanged — the bar the others were brought to |
| **Journal Entry / Dispute** | Unchanged (intentionally minimal / backend-gated — §8, §11) |

**Side-by-side result:** Payment, Credit Note, and Reconciliation Run now read as
members of the same system as Invoice — same StatusBadge status language, same
AttentionBanner exception treatment, same Details/rail structure — while the
specialized accounting tables (recon discrepancies, account/wallet/quote ledgers)
keep their intentional presentations. No visual regression; money right-aligned/
tabular; loading/error/not-found states (Batch B) intact. (Wallet's table a11y is
the byte-identical edit verified live on Quote + Account; PricingSimulator's
`formatCurrency`→`<Money>` is a slide-over preview, verified by build + unit
suite.)

### 11. Backend / data-model gaps discovered (documented, not fixed)

- **`disputes` absent from the audit allowlist** (`internal/adapter/middleware/
  audit.go`) → no audit trail available for disputes.
- **No domain events emitted for credit notes or disputes** → no object timeline
  available for either (Credit Note gets Audit, not Timeline; Dispute gets neither).
- **Reconciliation run currency** — single functional currency by **ADR-010** (a
  documented decision, not a gap); recon amounts are correctly in the functional
  currency.

No frontend data was invented for any of these.

### 12. Tests / lint / build / CI

- `npm run lint` → clean · `npm run build` → clean.
- `npx vitest run` → **720 passed (720)**, 0 skipped.
- New/updated coverage: StatusBadge `returned`/`initiated`; Payments humanized
  reason + `title` + StatusBadge labels; Reconciliation AttentionBanner + count
  badge; Credit Note "Audit trail" section; Quote line-items a11y (name + `th
  scope`). No test weakened; no unrelated test touched.

### 13. Bugs found / fixed

- **PricingSimulator used `formatCurrency`, not `<Money>`** (the design doc had
  assumed all 4 raw tables were on `<Money>`) — a genuine money-display gap, fixed
  (4 cells → `<Money>`).
- No functional/product bug (presentation + a11y only; no payment/accounting
  behaviour change).

### 14. Deferred Batch F items

- Trailing-chevron / loose-row de-noise; contextual back-navigation; misused
  `title=` definitions → real Tooltips; link the remaining dead-end IDs — all
  Batch F interaction work, untouched here.

---

**Stop.** Batch E is complete and self-contained. Not starting Batch F. No new
full-dashboard audit.
