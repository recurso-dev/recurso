# Recurso Dashboard Principles

## Gold-standard pages

The following pages establish the reference UX:

1. Customer
2. Subscription
3. Invoice
4. Payment
5. Ledger

Do not independently redesign every page.

First solve reusable patterns on these pages.

Then propagate those patterns.

---

## Object detail pages

Where supported, object pages should expose:

Identity
State
Financial summary
Relationships
Lifecycle
Activity
History
Actions
Technical information
Audit trail

---

## Deep linking

Important objects must have stable URLs.

Examples:

/customers/:id
/subscriptions/:id
/invoices/:id
/payments/:id
/journal-entries/:id
/reconciliation/:id

---

## Exception-first dashboard

The dashboard should prioritize:

"What needs my attention?"

Examples where actual data exists:

- failed payments
- past-due invoices
- unreconciled transactions
- failed webhooks
- billing errors

Do not add metrics merely because dashboards normally have metrics.

---

## Financial actions

For consequential actions such as:

- refund
- void
- cancellation
- adjustment
- write-off
- plan changes

communicate:

- what will happen
- amount affected
- resulting state
- accounting impact
- reversibility

---

## Relationships

Users should be able to move naturally:

Invoice
→ Customer
→ Subscription
→ Usage
→ Payment
→ Journal Entry
→ Reconciliation