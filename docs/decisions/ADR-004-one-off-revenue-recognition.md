# ADR-004: One-off invoices recognize immediately, net of tax, with no ledger posting

## Status
Accepted

## Date
2026-07-20

## Context
Subscription invoices defer revenue: the invoice leg credits Deferred Revenue
and the rev-rec worker drains Deferred → Recognized month by month (net of
GST, which is reclassified to Tax Payable at invoice time — ENG-159/191).
One-off (no-subscription) invoices are earned immediately, so their invoice
leg credits **Revenue directly** (ENG-140) — nothing is ever deferred.

Bug (audit F3): `createImmediateRecognition` still queued a *pending*
recognition event at the invoice's **gross** total; the worker then posted
DR Deferred / CR Recognized — draining a Deferred balance that never existed
(the account went negative by the invoice amount; the reconciler's
`abnormal_account_balance` in production) and booking tax as revenue on top.

## Decision
For one-off invoices, `createImmediateRecognition` creates the schedule and a
single recognition event that is **pre-recognized at creation**
(`status='recognized'`, no ledger transaction), with `amount = total −
tax_amount` (clamped ≥ 0). The event exists purely so rev-rec *reporting*
(recognized-by-month, waterfall) includes one-off revenue; the ledger truth
was already written by the invoice leg.

## Alternatives Considered

### No recognition rows at all for one-offs
- Pros: nothing to get wrong
- Cons: rev-rec reports would understate recognized revenue vs the ledger's
  Revenue account; two "recognized revenue" numbers that disagree
- Rejected: reporting completeness matters

### Post a Revenue → Recognized reclassification
- Pros: one "Recognized" account holds everything
- Cons: extra ledger legs with no accounting meaning; Revenue vs Recognized
  is a reporting distinction, not a bookkeeping one here
- Rejected: don't post entries the books don't need

### Keep pending events but target the Revenue account
- Cons: the worker's posting (DR source / CR Recognized) would *reduce*
  Revenue that was correctly earned — same class of drain, different account
- Rejected

## Consequences
- Rev-rec reports count one-off revenue via the pre-recognized event; the
  worker never touches one-off invoices (`ProcessDueEvents` only claims
  `pending`).
- Recognition amounts are consistently **net of tax** on both the
  subscription path and the immediate path.
- Proven by `TestOneOffInvoice_ImmediateRecognition_NoDeferredDrain` (net
  amount, pre-recognized, zero ledger transactions after a worker run).
