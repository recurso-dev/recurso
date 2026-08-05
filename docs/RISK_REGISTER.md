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

### R-006 — Gateway refunds (`RecordRefund` money-out) unexercised · Medium · CLOSED
`opRefund` posts a fresh PAID invoice with a `gateway_payment_id` (so the refund
takes the gateway path, not the manual one) + a recognition schedule, then issues
a partial refund credit note. A `harnessGateway` fake (Refund → success) and
`SetRevRecService` wire the full production path: gateway Refund → `RecordRefund`
(DR Refunds / CR Cash) → rev-rec deferred unwind → over-refund guard. Runs green
across all seeds. Teeth: neutering `RecordRefund` → `missing_credit_note_
transaction` on the `refund` step (seed 7) — the issued refund note references
the leg, so the credit-note check (R-003) catches its absence.
(PR: harness-refund-op)

NOTE surfaced while doing this — a *manual* refund (invoice with no
`gateway_payment_id`) creates an `issued` refund note but posts **no** ledger leg
(createRefund returns early at RefundStatusManualRequired). Such a note WOULD be
flagged `missing_credit_note_transaction` by the reconciler. Is that intended (a
"post the manual refund's ledger entry" signal) or a false-positive? → **R-013,
open** below.

### R-013 — Manual refunds create issued notes with no ledger leg · Low/Medium · OPEN (surfaced by R-006)
`createRefund` for an invoice without a `gateway_payment_id` marks the note
`RefundStatusManualRequired` and returns before any ledger post — but the note is
`status=issued`, so the reconciler's credit-note completeness check would report
it as `missing_credit_note_transaction`. Decide: (a) intended tripwire that a
human must post the manual refund's journal entry, or (b) the check should exclude
`RefundStatusManualRequired` notes. Confirm against a real manual-refund tenant.

### R-004 — Ledger code 7 (credit application) is a magic number · Low · CLOSED
Added `LedgerCodeCreditApplication uint16 = 7` in `domain/ledger.go` (it was the
only unnamed ledger code); `RecordCreditApplication` now passes the named
constant, and the reconciler query's `t.code = 7` carries a `-- domain.Ledger
CodeCreditApplication` comment (SQL literals match the existing payment-query
pattern). Pure refactor — harness green. (PR: ledger-code-7-constant)

### Systemic root cause behind R-001/008/009/010/011 — reversal/drawdown legs are unguarded

The reconciler's completeness checks are **count-based** ("does *a* tx reference
X?") or **threshold-based** ("Deferred ≥ pending"). Neither catches a missing
**reversal / drawdown** leg that leaves a balance *overstated*: the books still
balance (the omitted leg is itself balanced), no sign goes abnormal, and any
count/threshold check is already satisfied by the *original* leg. R-001 was the
first instance found & fixed; the ones below share its exact shape. There are
~25 best-effort ledger posts (`grep "reconciliation needed" internal/service`);
these are the ones whose absence is currently invisible.

**Systemic fix — IMPLEMENTED (R-012 below).** The balance-vs-source invariant
**Customer-Credit ledger balance == Σ outstanding adjustment-type
`credit_notes.balance`** now runs in `ReconciliationService.Run`. The two caveats
were investigated and resolved: (1) the downgrade path debits Deferred but
**credits** Customer-Credit (`RecordDowngradeCredit` + `RecordDowngradeRevenueReversal`
both `CR Customer-Credit`), so downgrade credits reconcile; (2) refund-type notes
also carry a balance but post to Cash, so the query filters `type='adjustment'`.
Empirically verified across all seeds, teeth-proven by neutering the drawdown leg.

### R-012 — Customer-Credit liability invariant (systemic fix for the drawdown-leg class) · High · CLOSED
One reconciler check that catches the whole reversal/drawdown-leg class:
**Customer-Credit account balance must equal Σ outstanding adjustment-type
credit-note balances** (`SumSpendableCreditNoteBalance`, compared against the
Customer-Credit trial-balance line in `Run`). Every spendable credit (manual or
downgrade) credits Customer-Credit for its balance; every drawdown
(application/expiry/void) debits it and lowers the note balance in lockstep, so a
dropped leg leaves the ledger balance above the notes.
**Regression/teeth:** neutering `RecordCreditApplication` →
`customer_credit_liability_mismatch {Expected:0 Found:11221}` on the
`apply_credit` step (seeds 1, 2). Clean run green (invariant holds across
issuance, application, and downgrade).
**Impact:** closes the *detection* gap for R-008 (expiry) and R-009 (void) — a
missing expiry/void reversal leg now diverges Customer-Credit from the notes and
is flagged — and adds a second line of defense over R-001.
(PR: harness-customer-credit-invariant)

### R-014 — Customer-Credit invariant omitted wallets → false-positive on every wallet tenant · High · CLOSED
The R-012 invariant compared the Customer-Credit ledger balance against *only*
`Σ adjustment-type credit-note balances`. But **prepaid wallets post to the SAME
`AccountCodeCustomerCredit` (2300) account** — `RecordWalletTopUp` credits it
(code 11), `RecordWalletDrain` debits it (code 12). So a tenant whose
Customer-Credit balance was funded by a wallet (with no credit notes) would show
`ledger balance = wallet balance` but `expected = 0`, tripping a false
`customer_credit_liability_mismatch` on **every production tenant with a wallet.**
A safety check that cries wolf on healthy books is worse than none — it trains
operators to ignore the reconciler.
**Fix:** the invariant now sums both funding sources —
`expected = SumSpendableCreditNoteBalance + SumWalletBalance` (new
`SumWalletBalance` = `SELECT COALESCE(SUM(balance),0) FROM wallets WHERE
tenant_id=$1`). A dropped drawdown leg on *either* a credit note or a wallet
still diverges the ledger from the expected sum and is flagged.
**Regression/teeth:** `TestReconciliationWalletFundsCustomerCredit` (ledger 100 =
credit-notes 60 + wallet 40 → clean) fails on the pre-fix code with
`customer_credit_liability_mismatch {Expected:60 Found:100}` and passes on the
new code; `TestReconciliationCustomerCreditMismatchIncludesWallet` proves a real
gap (ledger 100 ≠ 60+30) still fires with `Expected:90 Found:100`.
**Impact:** the R-012 invariant is now correct for the wallet subsystem — no
false positives, and wallet drawdown-leg drops are now in scope of detection.
(PR: reconciler-wallet-in-customer-credit)

### R-008 — Credit-expiry reversal leg unguarded · Medium · CLOSED
Detection by R-012 + now exercised end-to-end: `opExpireCredit` issues a credit
past its `expires_at` and runs `ExpireDueCredits`. Teeth: neutering the expiry
post → `customer_credit_liability_mismatch {Expected:5059 Found:9221}` on the
`expire_credit` step (seeds 1, 4). (PR: harness-expire-void-ops)

Mechanism (why it was a gap): `RecordCreditExpiry` reverses the Customer-Credit
liability best-effort; the count-based credit-note check is satisfied by the
*issuance* leg, so a missing expiry leg was uncaught. R-012 catches it and
`opExpireCredit` exercises it.

### R-009 — Credit-void reversal leg unguarded · Medium · CLOSED
Detection by R-012 + exercised end-to-end: `opVoidCredit` issues a credit and
voids it (`CreditNoteService.Void` → `RecordCreditVoid`, DR Customer-Credit). A
dropped void-reversal leg diverges Customer-Credit from the zeroed note balance —
caught by R-012 (identical mechanism to the R-008 teeth-test). `Void` sets status
`void` (not in the count-check set), which is exactly why the old count-based
check missed it. (PR: harness-expire-void-ops)

### TWO-LAYER SAFETY NET (correction to R-010/R-011 hypotheses)
The safety net has **two** layers, and the harness only asserts the first:
1. `ReconciliationService.Run` — invoice/payment/credit-note/credit-application
   completeness, trial-balance integrity, deferred≥scheduled, recognized≤invoice.
   **This is what the harness asserts after every op.**
2. `ClosePackService` (month-end close, recurso#473) — the **Deferred identity**
   `rollforward.Closing - recognition.DeferredBalance - AwaitingPayment ==
   UnexplainedDelta` (must be 0).

**DONE (PR: harness-closepack-tieout):** the harness now asserts layer 2 —
`assertClosePackTies` runs `ClosePackService.Generate` once per seed and requires
`Deferred.Ties` (UnexplainedDelta == 0). Wired with revrec + the unscheduled-
deferral reader so the identity is exact. Holds across all seeds on clean books.

**IMPORTANT ASYMMETRY (corrected — my earlier "layer-2 catches write-off too" was
WRONG):** the identity catches a dropped **forfeit** leg but NOT a dropped
**write-off** leg. Why: `SumUnscheduledDeferral` (AwaitingPayment) *re-includes*
an `uncollectible` invoice when it has **no** code-22 write-off leg
(`i.status='uncollectible' AND NOT EXISTS (… t.code=22)`). So a missing write-off
leg is **absorbed** by AwaitingPayment → Deferred-not-reduced == schedule +
AwaitingPayment-that-still-counts-it → `UnexplainedDelta` stays 0. The amount is
report-*visible* (it lingers in AwaitingPayment — the code comment's "keeps
un-reversed write-offs visible") but there is **no hard failure**. Forfeit is on
*paid+scheduled* invoices, which AwaitingPayment does NOT cover, so a dropped
forfeit leg *does* break the identity. **Teeth (forfeit):** neutering
`UnwindOnCancel`'s `RecordRecognition` → `UnexplainedDelta=100000` on `seed=7
final`. **This closes R-011.** R-010 (write-off) needs its own hard check —
see below.

**Recommended next (highest-value):** wire `ClosePackService` into the harness
and assert `UnexplainedDelta == 0` (and trial-balance `Balanced`) after each op —
this makes the property test cover BOTH safety-net layers and would exercise+pin
write-off (R-010) and forfeit (R-011) end-to-end. Requires wiring the close-pack
deps (rollforward reader, recognition summary, `SumUnscheduledDeferral`, tb).

### R-010 — Write-off (bad-debt) reversal leg · Medium · CLOSED
`MarkUncollectible` flips an open/past_due invoice to `uncollectible` and posts a
best-effort `RecordInvoiceWriteOff`; a dropped leg leaves A/R + Deferred
overstated, missed by the reconciler (A/R positive is normal-sign) AND by the
close-pack identity (AwaitingPayment absorbs it). **Fix (hard, per-invoice,
R-001-style):** new reconciler check `GetWriteOffLedgerMismatches` — every
`status='uncollectible'` invoice must carry write-off legs (codes 22 deferred + 26
bad-debt + 23 tax, all CR A/R) summing to its **total**. Key insight that made it
clean: the three codes together always sum to `preTax + tax = total`, regardless
of the deferred/recognized split, so `expected = i.total`. Exercised by a new
`opWriteOff` (open subscription invoice → `uncollectible` → `RecordInvoiceWriteOff`).
Teeth: neutering `RecordInvoiceWriteOff` → `missing_write_off_transaction
{ExpectedAmount:12281 FoundAmount:0}` on the `write_off` step (seed 7).
(PR: harness-writeoff-check)

### R-011 — Cancel forfeit / deferred-reversal leg · Medium · CLOSED
On cancel-with-unwind the forfeit leg drains still-deferred revenue; a missing leg
leaves Deferred too high, which `DeferredBelowScheduled` (too-LOW only) misses.
Exercised by `opCancelWithUnwind` + now caught by the layer-2 close-pack
assertion. Teeth: neutering the forfeit `RecordRecognition` →
`close-pack Deferred tie-out broken: UnexplainedDelta=100000` (seed 7), reconciler
silent. (PR: harness-closepack-tieout)

### R-007 — Audit-checklist areas not yet property-tested · backlog · OPEN
Not yet driven through the harness / dedicated adversarial tests: disputes &
chargebacks, ~~the entire wallet/prepaid subsystem~~ (**now fully covered** —
top-up 11, drain 12, refund 13, forfeit 14, expiry 15 — see below), tax
(GST/VAT/US nexus) edge rounding, importers, multi-tenant isolation under
concurrency, RBAC on money-out. Pick the highest-financial-impact next.

**Wallet top-up now exercised (`opWalletTopUp`):** the harness drives a real
`WalletService.CreateWallet` + `TopUp` through Postgres (entity reader resolves
the primary entity, so the `wallets.entity_id` FK is satisfied). Top-up posts
DR Cash / CR Customer-Credit (code 11) and denormalizes `wallets.balance`, so the
R-014 Customer-Credit invariant (which now counts wallet balances) is exercised
end-to-end. **Teeth:** neutering the top-up leg → `customer_credit_liability_
mismatch {Expected:32887 Found:0}` on the `wallet_topup` step (seeds 1-4).
(PR: harness-wallet-topup-op)

**Wallet drain now exercised (`opWalletDrain`):** drives real
`WalletService.DrainForInvoice` — a funded wallet drains against a one-off
invoice (CR Revenue, not Deferred, so the close-pack identity is untouched). The
drain posts DR Customer-Credit / CR AR (code 12) and decrements `wallets.balance`
in lockstep; the wallet fully covers the invoice, which is then marked paid with
`amount_paid` = the drained amount (code 12 is a payment-shaped leg). So BOTH the
R-014 invariant (Customer-Credit == wallet balance) and the payment-leg check
(Σ code {3,10,12} == amount_paid) guard it. **Teeth:** neutering the drain leg →
`customer_credit_liability_mismatch {Expected:2847 Found:12734}` on the
`wallet_drain` step (seeds 1-4) — the ledger holds the full top-up while
`wallets.balance` dropped. (PR: harness-wallet-drain-op)

**Wallet promotional expiry now exercised (`opWalletExpire`):** drives real
`WalletService.ExpireOverdueCredits`. A promotional top-up posts DR Credits &
Adjustments (expense) / CR Customer-Credit (free credit granted); when its
residue lapses the sweep zeroes `wallets.balance` and posts the discharging leg
(code 15, DR Customer-Credit / CR Credits & Adjustments), reclaiming the
liability so lapsed promo credit doesn't linger in the GL. Net once fully
expired: Customer-Credit and `wallets.balance` both return to zero, so the R-014
invariant ties. The op tops up with a valid future expiry (TopUp rejects past
ones), backdates the residue to overdue, then runs the sweep. **Teeth:** neutering
the expiry leg → `customer_credit_liability_mismatch {Expected:0 Found:10887}` on
the `wallet_expire` step (seeds 1-4) — the liability stays funded while
`wallets.balance` zeroed. (PR: harness-wallet-expiry-op)

**Wallet closure now exercised (`opWalletClose`) — closes out the subsystem:**
drives real `WalletService.CloseWallet`, the last two wallet legs in one op. A
wallet funded with BOTH a manual (paid, refundable) and a promotional
(non-refundable) top-up is closed: the paid residue refunds (code 13,
DR Customer-Credit / CR Cash) and the promo residue forfeits (code 14,
DR Customer-Credit / CR Credits & Adjustments). Net once closed: Customer-Credit,
the wallet's Cash delta, Credits & Adjustments, and `wallets.balance` all return
to zero — R-014 ties, trial balance balanced. **Teeth:** neutering the forfeit
leg → `customer_credit_liability_mismatch {Expected:0 Found:3847}` on the
`wallet_close` step (seeds 1-4). **The wallet subsystem is now fully property-
tested** end-to-end (codes 11-15), every leg teeth-proven against the R-014
invariant. (PR: harness-wallet-close-op)
