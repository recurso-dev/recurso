# Polish Batch A Report — Money Signature + Object Hero

> Implemented, merged (PR #695, squash `20bf6264`), deployed to app.recurso.dev,
> and live-QA'd. Scope held to exactly the two approved items — **make `<Money>`
> the canonical renderer** and **establish the canonical object-page hero**. No
> Batch B–F work, no backend, no motion changes, no unrelated cleanup.

## 1. Files Changed (13)

**Primitives**
- `frontend/src/components/ui/money.jsx` — `size` vocabulary.
- `frontend/src/components/patterns/ObjectPage.jsx` — `ObjectHeader` `amount` +
  `amountLabel` hero slot.

**Canonical object pages (migrated to hero + `<Money>`)**
- `frontend/src/pages/InvoicePage.jsx`
- `frontend/src/pages/PaymentPage.jsx`
- `frontend/src/pages/SubscriptionPage.jsx`
- `frontend/src/pages/JournalEntryPage.jsx`
- `frontend/src/pages/OfflinePayments.jsx` (the one sized-className `<Money>` caller)

**Tests**
- `frontend/src/components/ui/__tests__/money.test.jsx` (new)
- `frontend/src/components/patterns/__tests__/ObjectHeader.test.jsx` (new)
- `frontend/src/pages/__tests__/{InvoicePage,PaymentPage,SubscriptionPage,JournalEntryPage}.test.jsx`
  (assertions updated for the hero)

## 2. `<Money>` Changes

Added a **deliberately-small size vocabulary** — the exact set from the audit:

| size | class | role |
|---|---|---|
| `sm` | `text-xs` | inline / secondary metadata |
| `md` | *(none — inherits)* | **default**: table cells, standard amounts |
| `lg` | `text-lg font-semibold` | object metric strips |
| `hero` | `text-2xl font-semibold` | the object's single dominant amount |

- `md` is the default and adds **no** size class, so every existing unsized caller
  is visually unchanged (verified in test).
- **All formatting semantics preserved** — exponent-aware minor units (JPY 0,
  KWD 3), currency, negatives (leading minus), zero, `null`/`undefined` → the
  currency's zero, locale. The `.money` mono-tabular-muted-symbol signature is
  unchanged; `size` only scales the visual.
- Accessibility preserved: the full amount is the element's text content (the
  symbol/number split is presentational), so screen readers announce
  "$24,000.00" verbatim regardless of size (asserted in test).

No underlying formatting logic was redesigned (no correctness problem existed).

## 3. Object Hero Changes

`ObjectHeader` gained an optional **`amount`** (a `<Money size="hero">`) and
**`amountLabel`** (small muted context) slot, rendered directly under the title.
This establishes the canonical top-down hierarchy on every object page:

```
kicker (object type)
title  [StatusBadge]                 [actions]
HERO AMOUNT  · amountLabel            ← the ONE primary financial fact
id · date  (secondary metadata)
```

- Compact and authoritative — **not** a KPI card (no card chrome, no gradient, no
  giant hollow number). One large amount + a small label.
- Objects with no meaningful single amount **omit** it (verified in test): the
  Customer page and the Reconciliation-run page pass no `amount` — per the
  batch's "do not force a hero" rule (a customer is multi-currency; a run's
  headline is its discrepancy verdict, already the badge).
- No new framework, no duplicate header — the existing `ObjectHeader` was
  extended.

## 4. Consumers Migrated

**Money-hero adopted (4 object types):**
| Page | Hero amount | Label |
|---|---|---|
| Invoice | `total` | `$X due` (red) / "paid in full" |
| Payment | `amount` | the plain-language outcome |
| Subscription | `mrr` | "MRR" |
| Journal Entry | `amount` | "posted to the ledger" |

**`formatCurrency`/sized-className → `<Money>` migrations:**
- **InvoicePage** — the entire amount block: the `text-3xl font-bold`
  `formatCurrency` total (retired — off-convention weight, no money signature)
  now the hero; line items and the totals box (subtotal/tax/TDS/total/credit/
  paid/due) all `<Money size="sm">`; section renamed **"Breakdown"**; unused
  `formatCurrency` import removed.
- **PaymentPage** — amount → hero; the redundant "Amount" section removed; raw
  gateway code moved to the Details rail as quiet technical detail.
- **OfflinePayments** — the lone `<Money className="text-xs">` → `size="sm"`.

**Count:** 4 object pages now lead with a money-hero; ~14 individual amount
renders on the invoice/payment pages migrated from `formatCurrency`/ad-hoc sizing
to sized `<Money>`.

## 5. Before / After Examples

- **Invoice total (the flagship number).**
  Before: `<p className="text-3xl font-bold tabular-nums">{formatCurrency(total)}</p>`
  — sans-serif, bold (off-vocabulary), no money signature, buried in an "Amount"
  card of equal weight to Timeline/Audit.
  After: hero `<Money amountMinor={total} size="hero">` in the header with
  `$X due` beside it — mono-tabular, muted symbol, dominant. *(Live: "$24,000.00
  $24,000.00 due".)*
- **Payment.** Before: amount in a mid-page "Amount" section (`text-2xl
  font-semibold`), raw `gateway code: R01` beneath it. After: `$24,000.00` hero +
  "Returned by the bank after settling — ACH: insufficient funds"; `R01` → Details
  rail.
- **Subscription.** Before: header ended at the period meta; MRR only in the
  summary strip. After: `$299.00 MRR` hero leads. *(Live confirmed.)*

## 6. Tests

- **New `<Money>` suite (8):** currency-symbol split, exponent (JPY/KWD),
  negative, zero + `null` + `undefined`, USD default, each size in the vocabulary
  (incl. `md` adding no size class), extra `className` alongside a size,
  accessible text content.
- **New `ObjectHeader` hero suite (3):** renders identity+status+amount; omits the
  amount block when none is given; orders amount **before** the secondary
  metadata in the DOM.
- **Updated page assertions:** Invoice/Payment/Journal-Entry now assert the hero
  amount renders as `.money`; Payment asserts the raw code is in the rail;
  Subscription asserts MRR appears (hero + strip).
- **Full suite green: 679.** No tests weakened or skipped.

## 7. Lint / Build

`npm run lint` clean · `npm run build` clean · CI green (E2E, Frontend, Lint,
Test, Security, Workers Builds).

## 8. Live Visual QA (app.recurso.dev, real test-mode data)

Verified with screenshots against the Invoice reference:
- **Invoice** ✓ — `$24,000.00` hero + `$24,000.00 due`; "Breakdown" box all
  `<Money>` (mono, muted symbol, tabular); old bold sans total gone.
- **Payment** ✓ — `$24,000.00` hero + humanized outcome; redundant Amount section
  gone; raw `R01` in Details rail.
- **Subscription** ✓ — `$299.00 MRR` hero; Financial-summary strip intact.
- **Journal Entry** ✓ — amount hero (verified via test + prior-session live).
- **Consistency** ✓ — all four heroes share one treatment
  (kicker→title+badge→hero-amount→meta); money typography is uniformly
  mono-tabular with a muted symbol across pages.
- **Reconciliation / Customer** ✓ — correctly show **no** money-hero.

## 9. Accessibility QA

- `<Money>` full amount remains the element text content → announced verbatim at
  every size (test-asserted).
- Heading hierarchy unchanged (`ObjectHeader` still owns the single `<h1>`).
- No meaning by color alone (the "$X due" red still carries the word "due"; status
  stays a text `StatusBadge`).
- Focus/keyboard unaffected (no interactive changes in this batch).

## 10. Responsive QA

- Hero amount block is `flex flex-wrap items-baseline` → wraps cleanly at narrow
  widths (**code-verified**); it lives in the header's `min-w-0` column.
- `ObjectPageLayout` single-column collapse at ~560px was **live-verified earlier
  this session** (unchanged by this batch).
- **Note (deferred):** `ObjectHeader` *actions* still lack `flex-wrap` (audit R1)
  — multi-button headers can push horizontally on a phone. This is a **Batch F**
  item and was intentionally not touched here.

## 11. Bugs Found / Fixed

- **Fixed (indirect):** the invoice total no longer renders as bold sans-serif
  without the money signature — an audit finding (`font-bold` drift + no
  `<Money>`) closed as part of the hero migration.
- **No new bugs** introduced; no defects surfaced during migration.

## 12. Remaining Money Inconsistencies (deliberately not migrated)

- **Report / analytics pages** (`RevenueRecognition`, `RevenueWaterfall`,
  `TrialBalance`, `MonthEndClose`, `InvoiceAging`, `UnitEconomics`, etc.) still use
  `formatCurrency` and per-page local `money()` closures. The audit notes the
  string form is **partly by-design** for dense tabular cells and
  string-interpolation contexts (chart labels, `title=`), where a component can't
  render. Migrating these is broad and lower-trust; deferred.
- **ReconciliationRunPage** discrepancy table uses `formatMinorUnits`/
  `formatDifference` (still exponent-aware) in a dense table — left as tabular
  string-form money; converting is optional polish, deferred.
- **List pages** already render money via `MoneyCell`/`<Money>` where it's a
  standalone value; the remaining `formatCurrency` there is in string contexts.

These are consistency (P3) items, not regressions — a future increment could add
a `useCurrencyFormatter` and/or route the report tables through `<Money>`.

## 13. Remaining Hero Inconsistencies

- **Subscription** shows MRR in **both** the hero and the Financial-summary strip
  — an intentional glance-vs-detail redundancy; acceptable, but a future pass
  could drop MRR from the strip now that it's the hero.
- The hero treatment is otherwise uniform across the four money-bearing object
  pages. Non-money objects (Customer, Reconciliation, Plan, Coupon, etc.)
  correctly have no money-hero.

## 14. Findings Discovered But NOT Fixed (recorded for later batches)

- `ObjectHeader` actions `flex-wrap` (R1) — **Batch F**.
- Report-page money migration + `useCurrencyFormatter` — a future money increment.
- The `md`-default `<Money>` is now available everywhere; a follow-up could sweep
  remaining standalone `formatCurrency` renders in non-report pages to `<Money>`
  for full uniformity (low priority; not required for Batch A's trust goal).

---

**Acceptance:** monetary values on the canonical object pages now use `<Money>`;
no non-money values were migrated; the size vocabulary is small and deliberate;
financial typography is consistent (mono-tabular, muted symbol); the object pages
share a coherent hero hierarchy; Invoice remains the reference (and improved);
Payment/Subscription/Journal-Entry feel like the same product; no new design
system or duplicate framework; no unrelated features; no backend or motion
changes; accessibility intact; lint/build/tests pass; live QA complete. **Batch A
is complete — stopping for Batch B approval.**
