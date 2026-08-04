# ADR-009: Bad-debt write-off splits by recognized-vs-deferred; tax relief is policy-driven

## Status
Accepted

## Date
2026-08-04

## Context
When an invoice is written off as uncollectible, the correct accounting depends
on whether its revenue was already recognized. Under cash recognition nothing was
recognized, so a write-off simply reverses the Deferred Revenue that funded the
unpaid invoice. Under accrual (ADR-008), revenue is recognized at issuance — so
by the time an invoice proves uncollectible, some or all of its revenue is
already on the P&L. Reversing Deferred in that case would be wrong twice: it
would drain a Deferred balance that no longer holds the recognized portion, and
it would fail to record the economic loss as an expense. GAAP records that loss
as **Bad Debt Expense**. Separately, whether the tax portion of a written-off
invoice can be reclaimed varies by jurisdiction (e.g. India GST vs US sales tax),
so tax relief must not be hardcoded.

## Decision
Split the write-off reversal by the recognized-vs-deferred boundary, and drive
tax treatment from the accounting policy (ADR-007).

- A new **Bad Debt Expense** account (code `5200`, seeded in the tenant chart of
  accounts) and two posting codes: `LedgerCodeBadDebtWriteOff = 26` (expenses the
  recognized portion) and `LedgerCodeBadDebtRecovery = 27` (its mirror on
  recovery).
- `LedgerService.RecordInvoiceWriteOff` reads how much of the invoice was
  recognized and splits the pre-tax reversal: the **recognized** portion posts
  DR Bad Debt Expense / CR AR (code 26); the still-**deferred** portion reverses
  DR Deferred / CR AR (code 22). Under the cash model recognized = 0, so the
  write-off is byte-identical to the previous behavior — the change is safe by
  construction.
- `RecordWriteOffRecovery` inverts whatever the write-off actually posted
  (22→24, 26→27, 23→25), so a recovery restores the exact prior balances rather
  than assuming a fixed shape.
- Tax relief on the written-off amount is governed by
  `AccountingPolicy.BadDebt` (`AllowTaxRelief`, `RecoverableTaxes`,
  `RecognitionDelayDays`) from ADR-007 — never a country conditional in the
  ledger.

## Alternatives Considered

### Always reverse Deferred Revenue on write-off
- Pros: one code path
- Cons: under accrual, recognized revenue is not in Deferred; reversing it drains
  Deferred past what it holds (the reconciler's `deferred_below_scheduled_revenue`
  / `abnormal_account_balance` class) and never books the loss as an expense
- Rejected: wrong under accrual

### Always expense Bad Debt (never reverse Deferred)
- Cons: under cash, nothing was recognized, so there is no expense to book — the
  correct action is to reverse the Deferral. Expensing would understate Deferred
  and overstate expense
- Rejected: the split is required; treatment depends on what was recognized

### Hardcode tax relief per country in the write-off path
- Cons: re-introduces the `if country == "IN"` entanglement ADR-007 removes
- Rejected: tax relief is a policy field

## Consequences
- Write-offs are correct under both cash and accrual, and recoveries are exact
  inverses. Proven by `internal/service/writeoff_baddebt_split_pg_test.go`
  (deferred/bad-debt split + recovery inversion + balanced books) and
  `internal/service/accrual_writeoff_pg_test.go` (real schedule: recognized →
  Bad Debt, deferred → reversal, pending events cancelled).
- A standing invariant guards the accrual side: the reconciler's
  `recognized_exceeds_invoice` discrepancy fails CI if any schedule ever
  recognizes more than its recognizable total (fabricated revenue that a
  mis-split could otherwise hide). See ADR-002 for the reconciliation/invariant
  mechanism.
- This ADR is coupled to ADR-008: they shipped together because accrual without
  the bad-debt split would over-reverse Deferred on every write-off of an
  already-recognized invoice.
- Open follow-up: a per-jurisdiction bad-debt **policy catalog** (US / India GST /
  UK VAT / EU VAT / AU GST) registered on the ADR-007 resolver; today the default
  is conservative (no automatic tax relief).
