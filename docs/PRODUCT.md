# Recurso — Product

> The durable answer to "what is this?" — read before deciding what to build.

## Mission

Recurso is the **financial operating system for SaaS companies**. Unlike Stripe
Billing, Chargebee, or RevenueCat, Recurso is **accounting-first**: every
financial event produces an auditable, double-entry ledger posting. Billing is
the surface; a reconcilable set of books is the product.

The one-sentence promise: **every number a customer sees can be traced to the
journal entries behind it, and the books always balance.**

## Who it's for

- SaaS and B2B software companies that bill subscriptions and/or usage
- AI startups with consumption pricing (tokens, calls, compute)
- Developer-tools companies that need a real API, not a dashboard-only tool
- Finance teams at those companies who close the books monthly and get audited
- Global sellers who owe GST (India), VAT (EU/UK), and US sales tax

The buyer we optimize for is a **founder or finance lead who will personally be
asked "can you prove this number?"** — by an auditor, an investor, or a board.

## Problems we solve

| Area | What Recurso does |
|---|---|
| Subscription billing | Plans, trials, upgrades/downgrades with proration, pauses, cancellations |
| Usage billing | Metered aggregation, seven charge models, progressive billing, pay-in-advance |
| Invoicing | Branded invoices + statutory documents (GST e-invoice/IRP, EU UBL/Peppol) |
| Revenue recognition | Deferred revenue, recognition schedules, month-end close pack |
| Collections & dunning | Smart retries, dunning campaigns, collections worklist, recovery attribution |
| Tax | GST, VAT, US sales tax (BYO TaxJar/Avalara/Ziptax), nexus, exemptions |
| Accounting | Double-entry ledger, trial balance, multi-entity books, QuickBooks/Xero sync |
| Reconciliation | On-demand + CI-enforced proof that invoices, payments, and the ledger agree |
| Payments | Cards, ACH, UPI/GoCardless mandates, wallets, offline payments, BYO gateways |

## Product principles (non-negotiable)

These are the load-bearing beliefs. A change that violates one is wrong even if
it ships green.

1. **Every number must be explainable.** A figure on a screen links to the
   postings that produced it. No magic totals.
2. **Every financial event must be reversible.** Refunds, write-offs, reversals,
   voids — each has a defined inverse that keeps the books balanced.
3. **No hidden state.** Status, balances, and pending work are visible, not
   inferred.
4. **No silent corrections.** The system never quietly patches a discrepancy; it
   surfaces it (see the reconciler) and records the fix as its own event.
5. **The books always reconcile.** Debits equal credits, invoices tie to the
   ledger, and CI fails if a code path can break that.
6. **Everything is auditable.** Config changes, money moves, and API actions
   leave an append-only trail.

## What "done" means here

A feature is done when it has: the happy path, the reversal, a ledger posting
(if it moves money), a test that fails on the old code, and no way to leave the
books unbalanced. See `ACCOUNTING_PRINCIPLES.md` and `ANTI_PATTERNS.md`.

## Related docs

- `ACCOUNTING_PRINCIPLES.md` — the ledger contract every money path obeys
- `DESIGN.md` / `BRAND.md` — how it looks and sounds
- `UX_RULES.md` / `ANTI_PATTERNS.md` — how it behaves, and what never to build
- `API_GUIDELINES.md` — the API contract
- `COMPETITORS.md` — where we win and where we don't
- `../REMEDIATION.md` — current engineering remediation state
