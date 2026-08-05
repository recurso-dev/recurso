# Risk Register — money-path correctness audit

A living log of correctness risks found while auditing Recurso as an adversary
("prove it's impossible to break"), accounting-first. Each item: severity,
business impact, how it was found, the regression that pins it, and status.

Discipline: **regression first, fix second, close only when the new test fails
on the old code and passes on the new.** The invariant harness
(`internal/service/ledger_invariant_pg_test.go`) is the primary vehicle — a
randomized property test over the real money-path services, reconciled after
every step.

Severity: **Critical** (silent financial corruption in prod) · **High** (latent
corruption path with no detection) · **Medium** (gap in the safety net) ·
**Low** (quality/robustness).

---

## Closed

### R-002 — Harness asserted only 5 of ~10 reconciler discrepancy classes · High · CLOSED
The invariant harness computed the full reconciliation report but only *failed*
on 5 classes. `invoice_amount_mismatch`, `missing_payment_transaction`,
`payment_amount_mismatch`, `orphaned_transaction`, and
`recognized_exceeds_invoice` were computed but ignored.
**Impact:** a regression that dropped a payment leg, posted a wrong-amount leg,
orphaned a transaction, or over-recognized revenue would not have failed the
harness.
**Regression/teeth:** neutering `RecordPayment` on new subscriptions →
`missing_payment_transaction {ExpectedAmount:100000 FoundAmount:0}`.
**Fix:** assert all applicable classes. (PR: harness-assert-all-classes)

### R-001 — Credit-application (A/R drawdown) leg completeness unchecked · High · CLOSED
When adjustment credit is applied to an invoice, the repo sets `credit_applied`
(and marks the invoice `paid` when fully covered) and books a best-effort code-7
leg (`DR Customer-Credit / CR A/R`). Nothing verified that leg. A dropped leg
leaves **A/R overstated and the Customer-Credit liability overstated** while the
books still balance, no sign goes abnormal, `amount_paid=0` (payment check
silent), and the credit-note issuance check is satisfied by the issuance leg —
so **no check caught it**.
**Impact:** silent balance-sheet drift; a settled receivable lingers as a
phantom asset and spent credit still reads as spendable in ledger-derived views.
**Found:** added an `apply_credit` harness op; neutering the drawdown leg left
the harness green → confirmed blind spot.
**Regression/teeth:** with the fix, neutering the leg →
`missing_credit_application_transaction {ExpectedAmount:… FoundAmount:0}` on the
`apply_credit` step (seeds 1, 2).
**Fix:** new reconciler check `GetCreditApplicationLedgerMismatches`
(`credit_applied` ⟺ sum of code-7 legs), wired into `ReconciliationService.Run`,
asserted by the harness. (PR: harness-credit-application-check)

### R-003 — `missing_credit_note_transaction` check was vacuous · Medium · CLOSED
The reconciler checked for credit-note issuance legs, but the harness created no
credit notes, so the check could never fire.
**Fix:** added an `issue_credit_note` op driving the real `CreditNoteService`;
teeth verified by neutering the issuance leg. (recurso#516)

---

## Open

### R-005 — Harness is USD-only; multi-currency / FX-exponent paths unguarded · Medium · OPEN
No non-USD currency runs through the property test, so exponent-aware money math
(JPY 0-decimal, KWD 3-decimal) and FX are not covered by the harness.
**Risk before doing it:** the reconciler sums minor units and may not be
per-currency — mixing currencies in one tenant could produce *false*
`ledger_unbalanced`/`abnormal_account_balance`. Needs per-currency books or
per-currency tenants first. Deferred as higher-risk; do it carefully.

### R-006 — Gateway refunds (`RecordRefund` money-out) unexercised · Medium · OPEN
The harness issues adjustment credits but not gateway refunds (needs a fake
`PaymentGateway`). The refund + tax-reversal legs and the over-refund guard are
not property-tested.

### R-004 — Ledger code 7 (credit application) is a magic number · Low · OPEN
Sibling codes are named constants (`LedgerCodeWalletDrain=12`,
`LedgerCodePaymentReversal=19`); credit application uses a bare `7` in
`RecordCreditApplication` and now the reconciler query. Add
`LedgerCodeCreditApplication uint16 = 7` and reference it in both.

### R-007 — Audit-checklist areas not yet property-tested · backlog · OPEN
Not yet driven through the harness / dedicated adversarial tests: disputes &
chargebacks, wallets/prepaid drawdown, tax (GST/VAT/US nexus) edge rounding,
importers, multi-tenant isolation under concurrency, RBAC on money-out. Pick the
highest-financial-impact next.
