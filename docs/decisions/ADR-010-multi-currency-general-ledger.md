# ADR-010: The general ledger is single-functional-currency; multi-currency GL is out of scope

## Status
Accepted

## Date
2026-08-06

## Context
Recurso stores all money as integer minor units (`int64`) and posts every
financial event to a double-entry ledger. Individual documents already carry a
currency: `invoices.currency`, `credit_notes.currency`, `wallets.currency`, and
`ledger_accounts.currency` (since migration 000013) all exist, and a tenant can
transact with customers in more than one currency.

The money-path audit (R-005) asked whether the ledger and reconciler are correct
for a tenant that mixes currencies. The findings, established by code review and
by driving a JPY wallet through the invariant harness alongside USD activity
(`opWalletTopUpJPY`), are:

1. **Ledger accounts are keyed by `(tenant/entity, code)`, NOT by currency.**
   `getOrCreate*Account` resolves accounts via `GetAccountByTenantAndCode` /
   `GetAccountByEntityAndCode`, neither of which takes a currency argument. So a
   USD posting and a JPY posting for the same code (e.g. Customer-Credit 2300)
   land in the **same** account row and mix minor units. The `currency` column on
   `ledger_accounts` is descriptive metadata (defaulting to `USD`), not part of
   the account's identity.

2. **The reconciler's equality invariants are unaffected by the mixing.** Every
   transaction posts an equal debit and credit in the *same* currency and equal
   amount, so `totalDebits == totalCredits` survives cross-currency summation —
   there is no false `ledger_unbalanced`. `abnormal_account_balance` is evaluated
   per account (each single-currency in practice) and asserts only a sign. The
   Customer-Credit invariant sums the *same* mixed postings on both sides. The
   JPY-mixed harness run reconciles clean across all seeds. So mixing currencies
   does **not** corrupt the reconciler.

3. **But a mixed account balance is not a meaningful monetary figure.** A
   Customer-Credit balance of `1100` that is `$1.00 + ¥1000` has no defined
   value. For a genuinely multi-currency tenant, the trial balance and any GL
   account balance become mixed-minor-unit totals — arithmetically consistent for
   reconciliation, but semantically meaningless as reported numbers.

4. **Currency normalization already lives in a separate layer.** Reporting
   (MRR, ARR, revenue analytics, consolidated multi-entity reporting) converts to
   a single reporting currency through the FX layer (`fx.go` / `FXNormalizer`,
   hardened in #413–#426 for exponent-aware conversion), and `LedgerService`
   carries a `reportingCurrency`. The ledger itself is not where currency
   translation happens.

The open question this ADR resolves: **should Recurso support a true
multi-currency general ledger** — per-currency sub-accounts, an FX-revaluation
close step, and per-currency trial balances — or does it assume a single
functional currency per set of books?

## Decision
**The general ledger is single-functional-currency. Recurso does not support a
multi-currency GL, and this is a deliberate boundary, not a bug.**

- Each set of books (a tenant, or an entity under Multi-Entity Books) has one
  **functional currency** — the currency its GL account balances, trial balance,
  and close pack are denominated in. This is the reporting currency.
- Foreign-currency *documents* (a EUR invoice on a USD-functional tenant) are
  supported at the document layer: they invoice, collect, and reconcile in their
  own currency, and the reconciler's per-document, single-currency checks remain
  correct.
- Foreign-currency *amounts are translated for reporting*, in the FX/reporting
  layer, not re-booked into per-currency GL accounts.
- We do **not** book, revalue, or report GL account balances per currency. A
  tenant whose GL genuinely spans multiple functional currencies is out of scope
  today; the recommended pattern for that need is **one legal entity per
  functional currency** under Multi-Entity Books, each entity keeping its own
  single-currency books.

### Why
- **It matches how the ledger is already built** (integer minor units, code-keyed
  accounts) and how every existing tenant operates (single functional currency).
  No current tenant needs multi-currency GL.
- **The reconciler stays sound** without change — its guarantees were verified,
  not assumed.
- **A true multi-currency GL is a large, standards-bound feature** (ASC 830 /
  IAS 21): per-currency sub-ledgers, spot/average/closing rate policies, a
  cumulative translation adjustment (CTA) equity account, and an FX-revaluation
  posting at each close. Building it speculatively would add material complexity
  to the core money path for demand that does not yet exist.
- **The entity-per-currency workaround already exists** and gives correct,
  per-currency books today for the rare tenant that needs separated foreign
  operations.

## Consequences
- **Guardrail to add (follow-up):** the ledger has no enforcement that a set of
  books stays single-currency — nothing stops a foreign-currency document from
  posting into a functional-currency account and silently mixing minor units in a
  *reported* balance. Because account identity is code-based, the safe home for a
  foreign-currency document's own-currency legs is a distinct entity (its own
  ledger). A lightweight assertion — "all postings to a given ledger share one
  currency, or are translated first" — would turn today's documented assumption
  into an invariant. Tracked as a future reconciler check; not required for
  correctness of single-functional-currency tenants (all tenants today).
- Reporting remains the single place currency translation happens; the ledger is
  never asked to translate.
- If multi-currency GL is ever prioritized, it supersedes this ADR and must add:
  per-(account, currency) balances, an FX rate-policy engine (extending the
  ADR-007 policy engine), a CTA equity account, and a revaluation step in the
  close pack — with the invariant harness extended to a multi-functional-currency
  tenant before shipping.

## References
- Risk register R-005 (`docs/RISK_REGISTER.md`) — the audit finding and the
  JPY-mixed harness evidence.
- ADR-002 (ledger posting semantics), ADR-007 (accounting policy engine),
  ADR-008 (accrual recognition) — the surfaces a future multi-currency GL touches.
- #413–#426 — FX exponent-aware conversion in the reporting layer.
