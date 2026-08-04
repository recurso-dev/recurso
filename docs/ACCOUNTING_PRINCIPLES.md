# Recurso — Accounting Principles

> **Code-derived.** The ledger contract every money path obeys — the product's
> core differentiator. Posting functions in `internal/service/ledger.go`; code
> constants in `internal/core/domain/ledger.go`. Every row cites its function.
> Implementation wins over this doc.

## The one rule

Debits equal credits, always, and every customer-visible figure ties to the
postings behind it. The **invariant harness** and **reconciler** exist to make a
violation fail loudly — do not bypass them (§5).

## Chart of accounts (`domain/ledger.go:124-135`)

Cash 1000 · AR 1100 · TDS Receivable 1200 · Deferred Revenue 2100 · Tax Payable
2200 · Customer Credit 2300 · Revenue 4000 · Recognized Revenue 4100 · Refunds
5000 · Credits & Adjustments 5100. Seeded by `TenantChartOfAccounts`
(`:254-266`). Money is `int64` **minor units**; ledger amounts stored `uint64`
via `ledgerAmount()` which rejects negatives (`ledger.go:168`).

## Posting codes 1–25 (`service/ledger.go`)

| Code | Meaning | DR → CR | Fn |
|---|---|---|---|
| 1 | Invoice issuance (gross) | AR → Revenue (one-off) or Deferred (subscription) | `RecordInvoice:266` |
| 2 | Revenue recognition | Deferred → Recognized Revenue | `RecordRecognition:1239` |
| 3 | Payment (net cash) | Cash → AR | `RecordPaymentWithSettled:366` |
| 4 | Refund (cash out) | Refunds → Cash | `RecordRefund:742` |
| 5 | Deferred reversal on refund | Deferred → Refunds | `RecordDeferredRefundReversal:796` |
| 6 | Output tax (GST reclass) | Revenue/Deferred → Tax Payable | in `RecordInvoice:313` |
| 7 | Credit application | Customer Credit → AR | `RecordCreditApplication:1012` |
| 8 | Adjustment credit issued | Credits&Adj → Customer Credit | `RecordAdjustmentCreditIssued:988` |
| 9 | Refund tax reversal | Tax Payable → Refunds | `RecordRefundTaxReversal:851` |
| 10 | TDS receivable (India) | TDS Receivable → AR | in `RecordPaymentWithSettled:397` |
| 11–15 | Wallet topup/drain/refund/forfeit/expiry | (Cash/Credit permutations) | `RecordWallet*:1035-1145` |
| 16 | Downgrade credit (net) | Deferred → Customer Credit | `RecordDowngradeCredit:932` |
| 17 | Downgrade tax reversal | Tax Payable → Customer Credit | `RecordDowngradeTaxReversal:906` |
| 18 | Credit expiry | Customer Credit → Credits&Adj | `RecordCreditExpiry:1155` |
| 19 | Payment reversal (ACH claw-back) | AR → Cash | `RecordPaymentReversal:671` |
| 20 | Credit void (operator) | Customer Credit → Credits&Adj | `RecordCreditVoid:1177` |
| 21 | Downgrade revenue reversal | Recognized Revenue → Customer Credit | `RecordDowngradeRevenueReversal:962` |
| 22 | Invoice write-off (pre-tax) | Deferred/Revenue → AR | `RecordInvoiceWriteOff:502` |
| 23 | Write-off tax reversal | Tax Payable → AR | in `RecordInvoiceWriteOff:563` |
| 24 | Write-off recovery (mirror of 22) | AR → Deferred/Revenue | `RecordWriteOffRecovery:600` |
| 25 | Write-off recovery tax (mirror of 23) | AR → Tax Payable | in `RecordWriteOffRecovery:646` |

Key invariant: exactly **one Code-1 per invoice at gross**; GST is a *separate*
Code-6 reclass, not a second Code-1 (satisfies `uq_ledger_tx_reference_code`).
Subscription invoices credit **Deferred**; one-offs credit **Revenue** directly
(crediting Revenue for subscriptions was the ENG-140 double-booking bug,
`ledger.go:277`). **Dual-write:** Postgres always (authoritative), TigerBeetle
when connected (`ledger.go:16`).

## Minor units & currency exponent (`domain/currency.go`)

`CurrencyExponent` (`:27`) → JPY/KRW 0, KWD/BHD 3, default 2;
`MinorUnitsPerMajor` (`:38`) → 1/1000/100. All formatting is exponent-aware;
hardcoded `/100` was 100×/10× wrong for JPY/KWD and is banned (code comments).

## Deferred revenue & recognition (`service/revrec.go`)

`CreateScheduleForInvoice` (`:315`) is **idempotent per invoice** (`:322`),
schedules the **NET** (Total − Tax, `:355`), and shrinks by any prior
downgrade's `ConsumeScheduleDebt` (`:366`). **Called at BOTH** payment
(`subscription_payment.go:99`) **and** generation for wallet/credit-covered
invoices (`invoice.go:533`). One-offs recognize immediately with no extra ledger
post (`createImmediateRecognition:419`). The worker `ProcessDueEvents` (`:68`)
**claims** events before posting Code-2. Unwinds: `UnwindOnCancel:104`,
`UnwindOnRefund:153`, `ReverseRecognizedForDowngrade:227`.

> **Policy in motion (#466, `../REMEDIATION.md`):** moving schedule creation to
> *issuance* (full accrual) so the tie-out is structurally zero. Coupled to
> bad-debt (#477): recognizing on an unpaid invoice means a later write-off hits
> recognized revenue and must expense **Bad Debt**, not reverse Deferred.
> Founder-approved direction: model bad-debt as an `AccountingPolicy` with
> jurisdiction adapters, keeping accounting/tax/jurisdiction engines separate.

## Trial balance & month-end close (`close_pack.go`)

`Balanced` = debits == credits; `IsDebitNormal` flags abnormal-sign accounts.
Deferred rollforward: Opening + Added − Released == Closing. **Deferred tie-out
identity** (`:146`): `UnexplainedDelta = Closing − scheduled − AwaitingPayment`,
`Ties = delta==0`. `AwaitingPayment` = pre-tax deferral of unpaid subscription
invoices (`SumUnscheduledDeferral:80`) — exists because schedules are created on
payment (#466). `ReadyToClose` requires balanced TB **and** zero reconciliation
discrepancies (`:171`); the tie-out is a soft signal, not a blocker.

## Reversibility & occurrence-aware idempotency

Every money event has a defined inverse keyed by `(reference_id, code,
occurrence)` (`domain/ledger.go:334`): refund (4/5/9 distinct legs), write-off
(22/23) + recovery (24/25) cycle-aware (fresh write-off only when `nWO==nRec`,
`ledger.go:516`), payment reversal (19) inherits the cash leg's occurrence
(`:671`), downgrade (16/17/21 split so Deferred never goes wrong-sign), credit
void (20 vs expiry 18). Posting failures are surfaced for retry/reconciliation,
not swallowed (ADR-002, `RecordInvoice:329`).

## Reconciler & invariant harness (§5 — the safety nets)

**Reconciler** (`service/reconciliation.go`, `Run:146`, read-only): invoice
Code-1 completeness, payment Code-3 completeness, credit-note completeness,
orphan postings, trial-balance integrity (`ledger_unbalanced` +
`abnormal_account_balance`), and `deferred_below_scheduled_revenue` (Deferred ≥
`SumPendingRecognitionEvents`). Optional TigerBeetle cross-check degrades
honestly above 100k rows.

**Invariant harness** (`service/ledger_invariant_pg_test.go:36`): randomized real
billing sequences through the real services; after every step `assertAuditGrade`
runs the reconciler and fails CI on missing invoice/credit-note tx, unbalanced
ledger, abnormal balance, or deferred-below-scheduled. Fixed seeds `{1..8,23,39}`;
`LEDGER_INVARIANT_SEED` to explore. **Do not weaken these to pass CI.**

## Multi-entity

Per-entity ledgers (`resolveEntity:63`, unset → primary `LedgerID 1`,
byte-identical to single-entity); isolated AR sub-ledgers via SHA-1 namespace
(`arAccountID:82`); GL accounts per entity by code; recognition drains the same
entity's Deferred. Gapless invoice series is per-entity.

## Tax (`internal/core/service/tax/`)

`factory.go:20` routes: IN → GST engine (intra/inter-state CGST+SGST vs IGST),
US → sales-tax engine (**0% stub unless a provider is injected** — `factory.go:24`,
PARTIAL for live US rates), EU-27+GB → VAT engine, else → no-tax (0% with audit
note, ENG-152). Tax posts via Code-6 (Revenue/Deferred → Tax Payable), so Revenue
= taxable value, Tax Payable = tax, AR = gross.

## FX (`service/fx.go`)

`fxNormalizer` (`:27`): live provider + static fallback, tracks every rate and
whether fallback was used. `domain.ConvertMinorUnits` (`currency.go:66`)
normalizes minor→major, ×rate, major→minor, half-away-from-zero — never
multiplies minor units by the raw rate.

## Source of truth

- **Code:** `internal/service/{ledger,revrec,reconciliation,close_pack,fx}.go`,
  `internal/core/domain/{ledger,currency}.go`, `internal/core/service/tax/`,
  `internal/service/ledger_invariant_pg_test.go`.
- **ADRs:** ADR-002 (posting semantics), ADR-004 (one-off recognition);
  `docs/design-ledger-occurrence.md`.
- **Evidence file:** `docs/evidence/accounting-and-product.md`.
- **Related:** `PRODUCT.md`, `ARCHITECTURE.md`, `ANTI_PATTERNS.md`,
  `../REMEDIATION.md` (#466/#477).
