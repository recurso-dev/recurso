# Evidence — accounting-and-product

> Code-cited findings extracted by repository inspection (reverse-engineered
> from the implementation) on 2026-08-04. This is the evidence base for the
> corresponding source-of-truth doc(s). Every claim cites the file that
> proves it; implementation wins. Confidence is High unless a line is marked
> `ASSUMPTION`. Regenerate by re-running the inspection agents.

I have thorough coverage across both parts. Here is my reverse-engineering report.

---

# PART A — ACCOUNTING (source-of-truth findings)

## 1. Double-entry ledger: posting codes 1–25

**Posting functions** live in `internal/service/ledger.go`; **code constants** in `internal/core/domain/ledger.go:124-251`. Every posting is a single `LedgerTransaction` with a `DebitAccountID`/`CreditAccountID`, `Code`, `ReferenceID`, and `Occurrence` (`domain/ledger.go:326-343`). Chart of accounts: Cash 1000, AR 1100, TDS Receivable 1200, Deferred Revenue 2100, Tax Payable 2200, Customer Credit 2300, Revenue 4000, Recognized Revenue 4100, Refunds 5000, Credits & Adjustments 5100 (`domain/ledger.go:124-135`, seeded by `TenantChartOfAccounts` at `:254-266`).

| Code | Constant / meaning | DR → CR | Posting fn (`internal/service/ledger.go`) |
|---|---|---|---|
| 1 | Invoice issuance (gross) | Customer AR → Revenue (one-off) **or** Deferred Revenue (subscription) | `RecordInvoice` :266-345 |
| 2 | Revenue recognition | Deferred Revenue → Recognized Revenue | `RecordRecognition` :1239-1286 |
| 3 | Payment (net cash) | Cash → Customer AR | `RecordPaymentWithSettled` :366-468 |
| 4 | Refund (cash out) | Refunds (Expense) → Cash | `RecordRefund` :742-784 |
| 5 | Deferred-revenue reversal on refund | Deferred Revenue → Refunds | `RecordDeferredRefundReversal` :796-837 |
| 6 | `LedgerCodeOutputTax` — reclassify GST out of revenue | Revenue/Deferred → Tax Payable | inside `RecordInvoice` :313-327 |
| 7 | Credit application (settle invoice by credit) | Customer Credit → Customer AR | `RecordCreditApplication` :1012-1026 |
| 8 | Adjustment credit issued | Credits & Adjustments (Expense) → Customer Credit | `RecordAdjustmentCreditIssued` :988-1003 |
| 9 | `LedgerCodeRefundTaxReversal` | Tax Payable → Refunds | `RecordRefundTaxReversal` :851-892 |
| 10 | `LedgerCodeTDSReceivable` (India withholding) | TDS Receivable → Customer AR | in `RecordPaymentWithSettled` :397-418 |
| 11 | `LedgerCodeWalletTopUp` | Cash → Customer Credit | `RecordWalletTopUp` :1035-1050 |
| 12 | `LedgerCodeWalletDrain` | Customer Credit → Customer AR | `RecordWalletDrain` :1058-1072 |
| 13 | `LedgerCodeWalletRefund` | Customer Credit → Cash | `RecordWalletRefund` :1081-1096 |
| 14 | `LedgerCodeWalletForfeit` (promo, on closure) | Customer Credit → Credits & Adjustments | `RecordWalletForfeit` :1105-1120 |
| 15 | `LedgerCodeWalletExpiry` (promo lapse sweep) | Customer Credit → Credits & Adjustments | `RecordWalletExpiry` :1130-1145 |
| 16 | `LedgerCodeDowngradeCredit` (NET, ex-tax) | Deferred Revenue → Customer Credit | `RecordDowngradeCredit` :932-947 |
| 17 | `LedgerCodeDowngradeTaxReversal` | Tax Payable → Customer Credit | `RecordDowngradeTaxReversal` :906-921 |
| 18 | `LedgerCodeCreditExpiry` (credit note lapse) | Customer Credit → Credits & Adjustments | `RecordCreditExpiry` :1155-1170 |
| 19 | `LedgerCodePaymentReversal` (ACH claw-back) | Customer AR → Cash | `RecordPaymentReversal` :671-714 |
| 20 | `LedgerCodeCreditVoid` (operator void) | Customer Credit → Credits & Adjustments | `RecordCreditVoid` :1177-1192 |
| 21 | `LedgerCodeDowngradeRevenueReversal` (already-recognized part) | Recognized Revenue → Customer Credit | `RecordDowngradeRevenueReversal` :962-977 |
| 22 | `LedgerCodeInvoiceWriteOff` (pre-tax) | Deferred/Revenue → Customer AR | `RecordInvoiceWriteOff` :502-586 |
| 23 | `LedgerCodeWriteOffTaxReversal` | Tax Payable → Customer AR | in `RecordInvoiceWriteOff` :563-578 |
| 24 | `LedgerCodeWriteOffRecovery` (pre-tax, mirror of 22) | Customer AR → Deferred/Revenue | `RecordWriteOffRecovery` :600-669 |
| 25 | `LedgerCodeWriteOffRecoveryTax` (mirror of 23) | Customer AR → Tax Payable | in `RecordWriteOffRecovery` :646-661 |

Key invariant enforced in `RecordInvoice`: exactly **one** Code-1 per invoice at the gross total; GST is a *separate* Code-6 reclassification (not a second Code-1) to satisfy the `uq_ledger_tx_reference_code` unique index (`ledger.go:298-327`). Subscription invoices credit **Deferred** (liability); one-off invoices credit **Revenue** directly — crediting Revenue for subscriptions was the ENG-140 double-booking bug (`:277-288`). The service **dual-writes**: always PG (source of truth), and TigerBeetle when connected (`LedgerService` struct `:17-31`).

## 2. Minor units + currency exponent — CONFIRMED, no hardcoded /100

`internal/core/domain/currency.go`. All money is `int64` minor units. `CurrencyExponent` (`:27-32`) maps zero-decimal (JPY/KRW/…) → 0 and three-decimal (KWD/BHD/…) → 3, default 2. `MinorUnitsPerMajor` (`:38-47`) returns 1/1000/100. Ledger amounts stored as `uint64` via `ledgerAmount()` which rejects negatives (`ledger.go:168-173`). `MinorToMajor`/`FormatMoneyPlain`/`FormatMoney` (`:53-106`) are all exponent-aware; the comments explicitly call out that hardcoded `/100` was 100×/10× wrong for JPY/KWD and is banned. The trial balance carries a single `ReportingCurrency` and notes it is a base-currency approximation for multi-currency tenants (`domain/ledger.go:116-120`).

## 3. Deferred revenue + recognition

`internal/service/revrec.go`. `CreateScheduleForInvoice` (`:315-409`):
- **Idempotent per invoice** via `GetActiveScheduleByInvoice` guard (`:322-328`) — prevents a second schedule when an ACH-returned invoice is re-collected.
- Schedules the **NET** (`Total − TaxAmount`), because GST was reclassified to Tax Payable at invoice time; scheduling gross drove Deferred negative by the tax (ENG-191, `:355-364`).
- **Schedule-debt mechanism** (ENG-191f): `ConsumeScheduleDebt` shrinks the new schedule by any deferral a prior downgrade credit already clawed back from *unscheduled* deferral (`:366-386`); recorded via `RecordScheduleDebt`/`AddScheduleDebt` (`:261-263`, interface `:46-47`).
- One-off invoices → `createImmediateRecognition` (`:419-453`): creates a **pre-recognized** event with **no ledger post** (Revenue was already credited at Code-1) — a pending event here drove Deferred negative (F3).

**CONFIRMED — called at BOTH payment and generation:**
- At payment: `subscription_payment.go:99` and via `MarkInvoicePaid` (interface `invoice.go:64-70`).
- At generation for wallet/credit-covered invoices: `invoice.go:533-538` — when a wallet drain or account credit fully pays the invoice at generation (so it never flows through `MarkInvoicePaid`), the schedule is created inline.

Recognition worker: `ProcessDueEvents` (`:68-92`) **claims** due events (pending→processing) before posting Code-2, so concurrent workers are disjoint (F2). Unwind paths: `UnwindOnCancel` (breakage, `:104-139`), `UnwindOnRefund` (tail reduction + Code-5, `:153-215`), `ReverseRecognizedForDowngrade` (caps Code-21 at genuinely-recognized events, `:227-256`), `ReduceScheduleForDowngrade` (`:272-311`).

## 4. Reconciliation + invariant harness

**Reconciler:** `internal/service/reconciliation.go`, `ReconciliationService.Run` (`:146-257`). Read-only; "fixing drift is a human decision" (`:117-118`). Checks:
- `GetInvoiceLedgerMismatches` — every non-draft invoice has Code-1 summing to total.
- `GetPaymentLedgerMismatches` (`:169-176`) — every paid invoice has Code-3 summing to amount_paid.
- `GetCreditNoteLedgerMismatches` (`:178-188`).
- `GetOrphanLedgerTransactions` — Code-1/3 referencing a missing invoice.
- **Trial-balance integrity** (`trialBalanceDiscrepancies` `:263-282`): `ledger_unbalanced` (debits≠credits) + `abnormal_account_balance` (wrong-sign, e.g. Deferred going net-debit).
- **Deferred-vs-scheduled invariant** (`:218-240`): `deferred_below_scheduled_revenue` — Deferred balance must be ≥ `SumPendingRecognitionEvents`; survives cross-subscription aggregation where the abnormal-sign check does not.
- Optional TigerBeetle cross-check (`compareTigerBeetle` `:299-412`): diffs PG (authoritative) vs TB by shared 128-bit txID → `missing_in_tigerbeetle` / `missing_in_postgres` / `tb_amount_mismatch`; degrades honestly (`TBCompared`/`TBSkipReason`) above the 100k in-memory guard (`:22-27`).

**Invariant harness:** `internal/service/ledger_invariant_pg_test.go`, `TestLedgerInvariants_RandomizedBillingSequences` (`:36-82`). Drives randomized real billing sequences (new subs, up/downgrades, one-offs, recognition, cancel-with-unwind) through the real services; after **every** step `assertAuditGrade` (found at file end) runs the reconciler and **fails on**: `DiscrepancyMissingInvoiceTx`, `DiscrepancyMissingCreditNoteTx`, `DiscrepancyLedgerUnbalanced`, `DiscrepancyAbnormalBalance`, `DiscrepancyDeferredBelowScheduled`. Fixed seeds `{1..8,23,39}` (23/39 are ENG-191e regressions), overridable via `LEDGER_INVARIANT_SEED`. This is the CI gate CLAUDE.md calls the "invariant harness."

## 5. Trial balance + month-end close

Trial balance computed in `LedgerService.GetTrialBalance` (called from close pack `close_pack.go:116`); `Balanced` = total debits == total credits; `IsDebitNormal` drives the abnormal-sign flag (`domain/ledger.go:83-85`, `TrialBalanceLine`/`TrialBalance` `:87-121`). `DeferredRollforward` (`:268-284`): Opening + Added − Released == Closing.

**Close pack:** `internal/service/close_pack.go`, `Generate` (`:112-165`). The **deferred tie-out identity** is `ledger closing == schedule deferred + awaiting_payment`, computed as `UnexplainedDelta = rollforward.Closing − recognition.DeferredBalance − AwaitingPayment` with `Ties = delta == 0` (`:146-147`). `AwaitingPayment` = pre-tax deferral of unpaid subscription invoices (schedules are created on payment), via `SumUnscheduledDeferral` (`:80-91`, `:138-144`) — added so open invoices don't make the tie-out structurally amber (recurso#466). `ReadyToClose` requires a balanced TB **and** zero reconciliation discrepancies (`closeBlockers` `:171-182`); the tie-out is a soft signal, not a blocker (`:30-31`).

## 6. Reversibility + occurrence-aware idempotency

Idempotency key is **(reference_id, code, occurrence)** — `Occurrence` documented at `domain/ledger.go:334-343`, design in `docs/design-ledger-occurrence.md`.
- **Refund:** Code-4 cash + Code-5 deferred reversal + Code-9 tax reversal, each a distinct code so the three legs for one credit note never dedupe against each other (`ledger.go:786-892`).
- **Write-off (22/23) + recovery (24/25):** cycle-aware — a fresh write-off posts only when every prior one is recovered (`nWO == nRec`, occurrence = completed cycles, `RecordInvoiceWriteOff` :516-527); recovery posts only when an unrecovered write-off exists (`nWO > nRec`, `RecordWriteOffRecovery` :608-619), and must run **before** the Code-3 cash leg (`:596-599`).
- **Payment reversal (19):** inverts the *actual latest* Code-3 leg (same amount, accounts swapped) and **inherits** its occurrence so a redelivered same-cycle return dedups but a genuine second return posts (`:671-714`). Warns (never fails) if a TDS invoice is reversed (`:690-693`).
- **Downgrade credit (16/17/21):** split across Deferred (16), Tax Payable (17), and already-recognized Recognized Revenue (21), so Deferred is never driven wrong-sign; each distinct code keeps them idempotent (`:906-977`).
- **Credit-note void (20):** DR Customer Credit / CR Credits & Adjustments, distinct from expiry (18) so a manual void is auditable apart from an automatic lapse; only the unspent balance reversed (`:1177-1192`).

Per ADR-002, posting failures are surfaced for retry/reconciliation, not silently swallowed (`RecordInvoice` `:329-338`).

## 7. Multi-entity

Per-entity ledgers via `entityLedgerReader` wired by `SetEntityReader` (nil-safe; unset → primary `LedgerID 1`, byte-identical to single-entity) (`ledger.go:30-45`, `resolveEntity` :63-77). Non-primary entities get isolated AR sub-ledgers via a SHA-1 namespace (`arAccountID` :82-87, `arNamespace` :58); GL accounts resolved/created per entity by code (`getOrCreateEntityAccount` :114-138). `LedgerAccount.EntityID`/`TrialBalanceLine.EntityID`/`GeneralLedgerRow.EntityID` tag the issuing entity; backfilled to primary by migration 000129 (`domain/ledger.go:98-104,300-316`). Recognition drains the **same** entity's Deferred (`ledger.go:1246-1249`). Gapless invoice series is per-entity (see `spec_multi_entity_books.md`; IRP submits per-entity credentials, `gsp/nic.go:163-181`). *ASSUMPTION: the exact gapless-numbering SQL lives in the invoice repo/migrations, not read here.*

## 8. Tax

`internal/core/service/tax/`. `NewTaxEngineWithSalesTaxProvider` (`factory.go:20-48`) routes: `IN`→`NewGSTEngine(state)` (`gst.go`, intra/inter-state CGST+SGST vs IGST), `US`→`NewUSSalesTaxEngineWithProvider` (`sales_tax.go`; 0%-rate stub unless a `SalesTaxProvider` is injected), EU-27 + `GB`→`NewEUVATEngine` (`vat.go`), everything else→`NewNoTaxEngine` (`notax.go`, 0% with audit note — deliberately avoids misapplying GST to unsupported jurisdictions, ENG-152). Interface is `port.TaxEngine`. **Tax posting:** engines compute `TaxAmount` on the invoice; the ledger reclassifies it via **Code-6** DR Revenue/Deferred → CR Tax Payable inside `RecordInvoice` (`ledger.go:313-327`) — net effect: Revenue = taxable value, Tax Payable = tax, AR = gross. BYO tax providers exist under `internal/adapter/taxprovider/` and `internal/adapter/vatprovider/`; US sales-tax resolver at `salestax_resolver.go`/`tax_resolver.go`.

## 9. FX

`fxNormalizer` in `internal/service/fx.go` (`:27-115`): primary + static-fallback `port.ExchangeRateProvider`, tracks every rate used and whether fallback was consulted (`snapshot()` → `FXSnapshot{Source: "live"|"static-fallback"}`). `convert` (`:79-85`) delegates minor-unit math to `domain.ConvertMinorUnits` (`currency.go:66-69`), which normalizes minor→major (÷10^exp_from), ×rate, major→minor (×10^exp_to), rounding half-away-from-zero — never multiplies minor units by the raw rate. Providers: `internal/adapter/fx/openexchangerates.go` (live) + `static_rates.go` (fallback). Consumed by analytics, MRR waterfall, dunning recovery, revenue segments, invoice aging, organization reporting.

---

# PART B — PRODUCT FEATURE INVENTORY (implemented capabilities)

Legend: **IMPL** = implemented in code; **PARTIAL** = real interface but mock/stub transport or provider-gated.

### Subscriptions — IMPL
`subscription.go` + `subscription_trial.go` (trials), `subscription_upgrade.go` (upgrades), `subscription_cancel.go`/`cancel_flow.go`/`subscription_retention.go` (cancel + retention offers), `subscription_pause.go` (pauses), `subscription_addon.go` (add-ons), `subscription_payment.go`, `renewal.go`, `subscription_retention.go`. Downgrades handled through the credit path (codes 16/17/21). Handlers: `handler/subscription.go`, `cancellation.go`, `cancel_flow.go`.

### Usage / metering billing — IMPL
`metering.go`, `usage.go`, `rating.go`, `pay_in_advance.go`, `progressive_billing.go`, `advanced_billing.go`, `weighted_sum.go`, `custom_expr.go` (custom rating expressions), `usage_alert.go`. Charge-model variety (progressive tiers, pay-in-advance) present. Spec: `docs/spec_usage_billing.md`. Handlers: `handler/metering.go`, `usage.go`, `usage_alert.go`.

### Invoicing + statutory — IMPL (India live HTTP) / PARTIAL (EU transport mock)
Core: `invoice.go`, `invoice_lines.go`, `invoice_aging.go`, `pdf_invoice.go`, `handler/invoice_pdf.go`, `invoice_branding.go`.
- **India GST IRP e-invoice — IMPL (real):** `einvoice.go` + `internal/adapter/gsp/nic.go` calls real NIC endpoints (`einv-api.nic.in`), full RSA/SEK auth + payload encryption + IRN/QR parsing (`gsp/nic.go:20-305`). `GSTR1`/`GSTR3B` filing incl. government submission: `gstr1.go`/`gstr1_gov.go`/`gstr3b.go`/`gstr3b_gov.go`. Note: `CancelIRN` is a soft no-op (`nic.go:308-316`) — **PARTIAL**.
- **EU UBL / Peppol — PARTIAL:** UBL 2.1 document generation is real (`euinvoice_ubl.go`, `euinvoice_service.go`, `domain/euinvoice.go`), but the Peppol Access Point transport is a **mock** (`adapter/einvoice_eu/mock.go` — returns `mock-<hash>` message ids; real AP "plugs in behind this interface without touching the document layer", `port/euinvoice_transport.go:9-20`). Handler: `handler/eu_einvoice.go`. **Flag: real Peppol AP not wired (founder-blocked).**

### Revenue recognition — IMPL
`revrec.go`, `trial_balance.go`, `close_pack.go`, `revenue_waterfall.go`, `revenue_segments.go`, `mrr_waterfall.go`. Covered fully in Part A §3–5.

### Collections / dunning — IMPL
`smart_retry.go` (smart retry), `dunning_campaign.go` (campaigns), `dunning_recovery.go` + `dunning_analytics.go` (recovery attribution, `docs/spec_recovery_attribution.md`), `collections_actions.go` (collections worklist), `nexus_alert.go`. Handlers: `handler/dunning.go`, `dunning_campaign.go`, `collections.go`. Spec: `docs/spec_smart_dunning.md`, `design-collections-intelligence.md`.

### Tax (GST/VAT/US + BYO) — IMPL
See Part A §8. BYO providers: `adapter/taxprovider/`, `adapter/vatprovider/`; US nexus tracking: `nexus_status.go`, `handler/tax_nexus.go`, `us_tax.go`, `docs/design-us-nexus.md`, `providers-us-sales-tax.md`. US sales-tax rate is a **0% stub unless a provider is injected** (`factory.go:24-26`) — **PARTIAL for live US rates**.

### Accounting — IMPL (sync PARTIAL on live OAuth)
Ledger/trial-balance/multi-entity: see Part A. `accounting.go`, `entity.go`, `reconciliation.go`, `handler/accounting.go`, `close_pack.go`, `ledger.go`, `reconciliation.go`. **External sync:** `adapter/accounting/` has real adapters — `quickbooks.go`, `xero.go`, `netsuite.go`, `tally.go` with `oauth.go` (real OAuth2 token exchange, `RealmID` handling) and `ratelimit.go`. Spec `docs/spec_accounting_sync.md`, ADR-006 (token-based connections). **Flag: live OAuth app credentials are provider-gated — code is real but some flows depend on registered OAuth apps (founder-blocked).** `handler/crm_sync.go` + `adapter/crm/hubspot.go` (CRM sync).

### Payments — IMPL (gateways real; some provider-gated)
- Cards/gateways: `adapter/gateway/stripe.go`, `adyen.go`, `razorpay.go`, plus `smart_router.go` (routing), `resolver.go`. Webhooks: `handler/webhook_stripe.go`, `webhook_razorpay.go`, `webhook_gocardless.go`.
- **ACH / UPI / GoCardless mandates:** `mandate.go` + `adapter/gateway/gocardless.go` (`docs/design-us-ach.md`, `spec_integrations_payments.md`).
- **Wallets:** `wallet.go` (top-up/drain/refund/forfeit/expiry, ledger codes 11–15).
- **Offline payments:** `offline_payment.go`, `handler/offline_payment.go`.
- **Saved cards:** `saved_card_gateway.go`.
- **BYO gateways — IMPL:** `adapter/gateway/tenant_gateway.go` + `gateway_connection.go`/`handler/gateway_connection.go` route to the tenant's connected gateway, falling back to env gateway (`docs/spec_byo_gateway.md`). Disputes: `dispute.go`.

### Credit notes / coupons / gifts / referrals / quotes — IMPL
`credit_note.go` + `pdf_credit_note.go` + `handler/credit_note.go` (ledger codes 5/8/16-21). Coupons: `handler/coupon.go` (percent/amount, per CLAUDE.md). Gifts: `gift.go`/`handler/gift_handler.go`. Referrals: `referral.go`/`handler/referral_handler.go`. Quotes: `quote.go`/`handler/quote.go`. Pricing tools: `pricing_simulator.go`, `catalog.go`, `handler/plan.go`.

### Customer portal — IMPL
`portal.go` + `handler/portal_api.go` (`docs/spec_customer_portal.md`). Consent: `consent.go`/`handler/consent.go`. Entitlements: `entitlement.go`/`handler/entitlement.go` (`spec_entitlement_engine.md`).

### MCP server — IMPL
`cmd/mcp/main.go` + `internal/mcp/` + `handler/mcp_settings.go`. Standalone tier-gated MCP server (Streamable HTTP multi-tenant, or stdio single-tenant); authenticates with `rsk_` API keys forwarded to `/v1`, holds no DB of its own (`cmd/mcp/main.go:1-40`). Docs: `docs/mcp-server.md`. Deployed separately (`Dockerfile.mcp`, `cloudbuild.mcp.yaml`).

### Imports (Stripe / Chargebee / RevenueCat + Compare gate) — IMPL
`stripe_import.go`, `chargebee_import.go`, `revenuecat_import.go` (concrete importers under `internal/importer/`). **Compare gate** — IMPL: `stripe_compare.go`, `chargebee_compare.go`, `revenuecat_compare.go` produce a read-only `CompareReport`/`CompareIssue` diffing the export against live Recurso data on coverage, money-critical fidelity, and billing continuity (`chargebee_compare.go:13-24`), rendered by `compare_report_html.go` + `handler/compare_report.go`. Spec: `docs/spec_bulk_importer.md`. `cmd/import/`.

### SSO / organizations — IMPL
SSO: `sso.go`/`handler/sso.go` + `oauth.go`/`handler/oauth.go` (auth also in `auth.go`, `auth_phase2.go`, `auth_invite.go`, `handler/account_security.go`, `team.go`). *ASSUMPTION: some IdP OAuth flows depend on registered app credentials (provider-gated).* Organizations/multi-tenant: `organization.go`/`handler/organization.go`, `tenant.go`/`handler/tenant.go`, `entity.go` (multi-entity books).

### Additional implemented capabilities (found, not in the ask)
Analytics/GenAI: `analytics.go`, `genai.go` + `adapter/ai/` (`docs/spec_genai_analytics.md`). Churn modeling: `churn.go`/`churn_model.go`. Unit economics: `unit_economics.go`. Webhooks (outbound + visibility): `webhook.go`/`handler/webhook_management.go` (`spec_webhook_visibility.md`). Notifications: `notification.go` + `adapter/email`, `adapter/sms`. Demo mode: `demo.go`/`handler/demo.go` + `cmd/demo_seed`. Waitlist: `handler/waitlist.go`.

### Partial / founder-blocked summary
- **EU Peppol Access Point** — document layer real, transport is a mock (`adapter/einvoice_eu/mock.go`). **PARTIAL.**
- **US live sales-tax rates** — engine is a 0% stub without an injected provider (`tax/factory.go:24-26`). **PARTIAL.**
- **Accounting sync (QuickBooks/Xero/NetSuite/Tally)** — adapters + OAuth code real, but live flows depend on registered OAuth app credentials. **PARTIAL (provider-gated).**
- **IRN cancellation** (`gsp/nic.go:308-316`) and `GetIRNByDocDetails` (`:319-327`) are soft no-ops/unsupported without tenant context. **PARTIAL.**
- TigerBeetle comparison in the reconciler is skipped above 100k rows and when TB is disconnected (honest degradation, not a defect).

Note: `docs/ACCOUNTING_PRINCIPLES.md` and `docs/PRODUCT.md` already exist; the above is derived independently from the implementation and can be used to validate/refresh them.