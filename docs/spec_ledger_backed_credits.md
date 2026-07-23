# Spec: Ledger-Backed Credits

## Objective

Make a customer's account credit a first-class, auditable, time-bounded balance —
built on the credit infrastructure that already posts to the general ledger.

Today account credits already exist and are GL-backed: each `credit_notes` row is
issued (DR Credits & Adjustments 5100 / CR Customer Credit 2300) and drawn down on
invoice settlement (DR Customer Credit 2300 / CR AR, code 7), with
`credit_note_applications` as the draw-down audit trail. What's missing is (1) a
consolidated **statement** that surfaces the running balance + history, and (2) a
credit **lifecycle** — expiry dates and the accounting for lapse.

Founder-approved scope (2026-07-23): **both**, as two increments.

## Increment 1 — Customer credit statement (read-only)

A per-customer statement: running spendable balance, the grants that make it up,
and the invoice applications that drew it down. No schema or money-path change —
it reads existing tables.

- `GET /v1/customers/:id/credit-statement` → `{ balances[], grants[], applications[], summary }`
  - **balances**: net spendable balance per currency (spendable = `type='adjustment'`,
    `status IN ('issued','used')`, `balance > 0` — the same eligibility the credit
    applier uses). Credits are per-currency and per-entity, so balance is grouped,
    never summed across currencies.
  - **grants**: every credit note for the customer (id, amount, balance, currency,
    reason, type, status, entity_id, created_at, expires_at once Inc 2 lands) — the
    full history, newest first.
  - **applications**: `credit_note_applications` joined to invoices (credit_note_id,
    invoice_id, invoice_number, amount, created_at).
  - **summary**: total issued / applied / current spendable, per currency.
- Dashboard: a **Credits** section on the customer detail — balance chip(s), grants
  table, application history, and a client-side **CSV export** of the statement.
  (PDF export is a follow-up, not Inc 1.)

## Increment 2 — Credit lifecycle & expiry (money-path)

- Migration: `credit_notes.expires_at TIMESTAMPTZ NULL`; widen the status CHECK to
  add `'expired'`. Optional `expires_at` on `CreateCreditNoteRequest`.
- **Expiry sweep worker** (mirrors the wallet promo-expiry worker): claims credit
  notes with `balance > 0`, `status IN ('issued','used')`, `expires_at <= now` via
  an atomic lease (`FOR UPDATE SKIP LOCKED`, all-UTC), zeroes the balance, sets
  status `expired`, and posts the GL write-off.
- **GL write-off**: `DR Customer Credit 2300 / CR Credits & Adjustments 5100` — the
  reversal of the original issuance, discharging the liability and unwinding the
  grant expense. New `LedgerCodeCreditExpiry` (next free code; `TestLedgerCodesAreUnique`
  enforces uniqueness). Consistent with the wallet-forfeit treatment we shipped.
  (Decision: reverse the issuance expense rather than book a separate
  "expiry revenue" account — keeps the credit's lifetime P&L net-zero and matches
  the wallet pattern. Revisit if a dedicated forfeiture-income line is required.)
- Verified on the invariant harness: expiry must leave the books balanced and the
  Customer Credit liability exactly discharged.

## Commands

- Backend: `go build ./... && go test ./...` (PG-backed via `TEST_DATABASE_URL`)
- Frontend: `cd frontend && npm run lint && npm run build && npx vitest run`

## Boundaries

- **Always**: reads tenant-scoped by `customer.tenant_id`; money-path (Inc 2)
  verified on the invariant harness; not self-merged.
- **Ask first**: a dedicated expiry-income account instead of reversing 5100.
- **Never**: mutate credit balances outside a ledger-posting path; expire a credit
  without a balanced GL write-off.

## Success criteria

- Inc 1: the statement's spendable balance equals the credit applier's view and
  reconciles to the GL Customer Credit (2300) account for the customer.
- Inc 2: an expired credit is unspendable, its balance is 0, and the sweep posts a
  balanced `2300→5100` write-off; the invariant harness stays green.

## Open questions

- Grant "source" taxonomy (goodwill / promo / SLA / refund): Inc 1 breaks down by
  the existing `type` + free-text `reason`. A structured `source` enum, if wanted,
  folds into Inc 2 (it's about how credits are granted). Confirm if a formal
  taxonomy is required.
