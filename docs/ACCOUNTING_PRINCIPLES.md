# Recurso — Accounting Principles

> The contract every money path obeys. This is the product's core differentiator
> and the hardest thing to get right. Read `docs/decisions/ADR-002` (ledger
> posting semantics) alongside this. When in doubt, the books balancing wins.

## The one rule

**Debits equal credits, always, and every customer-visible financial figure ties
to the ledger postings behind it.** A code path that can violate this is wrong,
even if every test passes — which is why the invariant harness and reconciler
exist to make it fail loudly.

## Double-entry, minor units, int64

- Money is `int64` **minor units** throughout (cents/paise). Never floats for
  money. Currency exponent is derived (`domain.CurrencyExponent`) — JPY 0, KWD
  3 — never a hardcoded `/100`.
- Every financial event posts a balanced transfer (debit account, credit
  account, amount). Postings are append-only.

## Posting codes (the vocabulary)

Each movement carries a numeric code naming what it is. The full glossary lives
in the dashboard's Ledger page and the docs; the load-bearing ones:

- **1** Invoice raised — AR → Deferred (subscriptions) or Revenue (one-offs),
  gross.
- **2** Revenue recognized — Deferred → Recognized, per recognition event.
- **3** Payment — Cash settles AR.
- **6** Output tax — reclassified from Deferred to Tax Payable at issue.
- **8** Credit note issued; **19** payment reversal (bank return);
  **22/23** write-off + its tax reversal; **24/25** write-off recovery.

Codes can repeat across a settle → reverse → re-settle cycle; the idempotency
key is `(reference_id, code, occurrence)` so a re-collected invoice posts fresh
legs instead of being swallowed.

## Deferred revenue & recognition

- Subscription revenue is **deferred at invoice issuance** (Code-1 to Deferred).
- Recognition schedules drive Code-2 events that move Deferred → Recognized over
  the service period.
- The **month-end close** proves the tie-out: `ledger deferred == scheduled +
  awaiting-payment`. A residual is a reconciling item, surfaced and explained —
  never labeled "unexplained."
- **Policy in motion (#466 / REMEDIATION.md):** moving schedule creation to
  *issuance* (full accrual) so the tie-out is structurally zero. Accrual couples
  to bad-debt: recognizing on an unpaid invoice means a later write-off hits
  recognized revenue and must expense **Bad Debt**, not reverse Deferred.

## Reversibility

Every money-out or money-in has a defined inverse that keeps the books balanced:
refund (4/5/9), payment reversal (19), write-off (22/23) and its recovery
(24/25), credit-note void (20), downgrade credit (16/17/21). Building a new
money path means building its reversal too.

## Policy-driven accounting (direction)

Keep **three engines separate** (founder direction, 2026-08-04):

1. **Accounting engine** — double-entry postings. Knows debits/credits, not tax
   law.
2. **Tax engine** — GST/VAT/US sales tax rate + treatment. Knows jurisdictions.
3. **Jurisdiction rules / policy** — an `AccountingPolicy` (revenue-recognition
   method, bad-debt treatment: tax relief, recognition-delay days, recoverable
   taxes) resolved by per-jurisdiction adapters (US / IndiaGST / UKVAT / EUVAT /
   AustraliaGST).

The accounting engine must **not** hardcode GST/VAT rules. Jurisdiction-specific
behavior lives behind a policy interface so international expansion is adding an
adapter, not editing the ledger.

## Safety nets (do not bypass)

- **Invariant harness** — randomized-sequence reconciliation in CI; any
  invoice-creating flow must post its legs or it fails.
- **Reconciler** — on-demand + E2E check that every invoice and payment agrees
  with the ledger (0 discrepancies is the gate).
- **Trial balance** — debits == credits, abnormal-sign flags.
- **Restore/DR drill** — proves row parity + double-entry conservation on a
  restored dump.

Making CI pass by weakening these is the one unforgivable move.

## Multi-entity

Per-entity ledgers, gapless invoice series per entity, per-entity tax identity,
consolidated reporting. An entity's books stand alone and roll up.

## Related

- `docs/decisions/ADR-002` (posting semantics), `ADR-004` (one-off recognition),
  `docs/design-ledger-occurrence.md` (settle→reverse cycles)
- `PRODUCT.md` — why this matters; `ANTI_PATTERNS.md` — the money don'ts
- `../REMEDIATION.md` — #466 accrual epic + #477 bad-debt coupling
