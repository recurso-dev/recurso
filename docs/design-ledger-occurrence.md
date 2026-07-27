# Design: Occurrence-aware ledger idempotency (settle → reverse → re-settle)

**Status:** approved for implementation (QA finding A, 2026-07-27)
**Amends:** ADR-002 posting semantics (idempotency mechanism only — account
treatment, codes, and "never fail the business write" are unchanged)

## Problem

Ledger postings are idempotent per `(reference_id, code)`: a partial unique
index (`uq_ledger_tx_reference_code`, migration 000079) plus
`ON CONFLICT DO NOTHING` in `applyLedgerTx` make a replayed settle a no-op. That
design assumed **each code posts at most once per reference for the invoice's
lifetime** — true until ACH late returns (PR #199) created the first flow that
returns an already-settled invoice to a re-collectable state:

```
settle          → code 3  (DR Cash / CR AR)      posts        ✓
bank return     → code 19 (DR AR / CR Cash)      posts        ✓
dunning re-collects → code 3 again, same invoice → SWALLOWED  ✗
```

The second settlement's cash leg (and, symmetrically, a TDS leg, and any second
reversal's code-19 leg) hits the unique index and is silently dropped. The
invoice reads `paid` while the ledger still reflects the reversed state: **Cash
permanently understated, AR permanently overstated** — and the payment
reconciliation check doesn't catch it, because `GetPaymentLedgerMismatches`
sums only codes (3, 10, 12) and is blind to code 19, so
`found = C` happens to equal `expected = C` even though both are wrong.

A second, related defect (QA finding C): `RecordPaymentReversal` recomputes the
reversal amount as `Total − CreditApplied − TDS`, but the settlement leg it
inverts was `Total − CreditApplied − TDS − walletSettled`. A wallet-part-funded
ACH invoice would over-reverse by the wallet portion. The original
`walletSettled` figure is unrecoverable at reversal time (MarkPaid overwrote
`amount_paid`), so recomputation can never be correct.

## Options considered

1. **Occurrence column** — widen the idempotency key to
   `(reference_id, code, occurrence)`; posting sites for the settle/reverse
   cycle compute the occurrence; everything else defaults 0. **Chosen.**
2. **New code per cycle** (code 20 = "re-settlement") — handles exactly one
   return cycle; a second cycle collides again. Escalating codes forever is not
   a design.
3. **Mangled reference_id for re-settlements** — breaks every reconciliation
   and read query keyed on `reference_id = invoice.ID`. Rejected.
4. **Delete the reversal pair on re-settle** — the ledger is append-only.
   Rejected outright.

## Design

### 1. Schema (migration 000146)

```sql
ALTER TABLE ledger_transactions ADD COLUMN occurrence SMALLINT NOT NULL DEFAULT 0;
DROP INDEX uq_ledger_tx_reference_code;
CREATE UNIQUE INDEX uq_ledger_tx_reference_code_occ
    ON ledger_transactions (reference_id, code, occurrence)
    WHERE reference_id <> '00000000-0000-0000-0000-000000000000';
```

Every existing row and every posting site that doesn't opt in gets
`occurrence = 0` — **byte-identical behavior for all codes except the
settle/reverse cycle.** (Down migration restores the old index; it can fail if
multi-occurrence rows exist by then, which is acceptable for a down.)

### 2. Occurrence derivation — the reversal count IS the cycle counter

The number of code-19 legs an invoice has accumulated is exactly how many
settle→reverse cycles it has completed. Both posting sites derive their
occurrence from it (new repo method `CountTransactionsByReferenceAndCode`):

- **Settlement legs** (code 3 cash + code 11 TDS, in `RecordPaymentWithSettled`):
  `occurrence = count(code-19 legs)` → first settle 0, post-return re-settle 1, …
- **Reversal leg** (code 19, in `RecordPaymentReversal`): **inherits the
  occurrence of the cash leg it inverts** (`cashLeg.Occurrence`). It must NOT
  be derived from the code-19 count — that is self-referential: a same-cycle
  duplicate reversal would count its own predecessor, land at a higher
  occurrence, and double-reverse. Inheriting the target leg's occurrence makes
  a duplicate compute the same key (dedup ✓) while a second genuine return
  finds the re-settle leg at the next occurrence (posts ✓). This was caught by
  the cycle test during implementation.

**Why this is race-safe:** cycles are serialized by the invoice-status
conditional UPDATEs (`MarkPaid`'s `status <> 'paid'`, `ReverseToUnpaid`'s
`status = 'paid'`) — only one settle and one reversal can win per cycle. Two
duplicate posts *within* one cycle (a redelivered webhook, a crashed-and-retried
worker) compute the **same** occurrence and dedup exactly as before. Posts in a
*new* cycle see one more code-19 leg, compute a new occurrence, and land. The
unique index keeps its role as the same-cycle backstop; it just stops eating
legitimate new-cycle legs.

### 3. Reversal amount — invert the actual leg, not a recomputation

`RecordPaymentReversal` now reads the **latest code-3 leg** for the invoice
(new repo method `GetLatestTransactionByReferenceAndCode`, `ORDER BY occurrence
DESC`) and reverses **that leg's amount**. This is exact by construction:

- Wallet-part-funded settle (leg = net cash) → reversal = net cash. Finding C
  fixed with no new persisted state.
- Fully credit/wallet-covered invoice (no code-3 leg) → nothing to reverse.
- The old `Total − CreditApplied − TDS` recomputation is deleted.

### 4. Reconciliation learns about reversals

`GetPaymentLedgerMismatches` nets code 19 instead of ignoring it:

```sql
-- found: reversals subtract; payment-shaped legs add
COALESCE(SUM(CASE WHEN t.code = 19 THEN -t.amount ELSE t.amount END), 0)
... AND t.code IN (3, 10, 12, 19)
```

For a paid invoice after a full cycle: `C − C + C = C = amount_paid` ✓. A paid
invoice whose re-settle leg went missing (the pre-fix corruption) now shows
`found = 0 ≠ expected` — i.e. **the reconciler can finally detect the very
corruption this design eliminates**, including any rows corrupted before the
fix ships.

### Non-goals (explicit)

- **TDS + reversal interaction.** A reversal inverts only the cash leg. If a
  TDS invoice were reversed and re-settled, the TDS leg would double-post
  (occurrence 1). This combination is unreachable: reversals exist only for US
  ACH (USD), TDS only for India (INR). `RecordPaymentReversal` warn-logs if it
  ever sees a TDS leg so the assumption is observable.
- **TigerBeetle dual-write** dedup parity — TB transfers use fresh IDs per call
  and have no (reference, code) dedup today; unchanged.
- **Backfilling** pre-fix corrupted rows — the reconciler now surfaces them;
  repair is an operator action.

## Verification

- **PG oracle test** (the heart of it): settle → assert leg/balances → reverse
  → assert → **re-settle → assert the cash leg actually posts** (net Cash = C,
  AR relieved, 3 legs) → second reverse (occurrence 1) → re-settle again.
  Plus: same-cycle duplicate settle still dedups; wallet-funded settle reverses
  the net leg amount, not the gross.
- **Reconciliation**: after a full cycle on a paid invoice,
  `GetPaymentLedgerMismatches` reports nothing; with the re-settle leg deleted
  (simulating pre-fix corruption), it reports the invoice.
- **Invariant harness + E2E zero-discrepancy gate** (CI) must stay green —
  merge only on fully green.
