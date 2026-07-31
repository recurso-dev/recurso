# Engineering backlog

Ranked by ROI (impact ÷ effort). Updated 2026-07-28 (overnight session).
Struck as already-shipped on re-audit: per-entity GSTR-1/3B (`?entity_id=` +
primary-aware GSTIN, #205/#208) and per-entity MRR UI (ExecutiveSummary +
Entities pages).
Items marked **founder** are blocked on credentials/infrastructure only the
founder can provide; everything else is engineering-ready.

## P0 — money-path correctness

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| 2 | **Xero-invalid customer email** (`bed15f4d…`) | MED — one customer's invoices never sync (QuickBooks rejects it too) | — | **founder** fixes the email in the dashboard; sync rows now show the customer name + id. |
| ~~S2~~ | ~~Cross-tenant BYO webhook binding~~ | — | — | **DONE (#382, #384, #386)** — every BYO-webhook money-move is now bound to the connection's own tenant. **#382**: all invoice settle/failure/reversal loads across the 3 gateways (`invoiceBelongsToWebhookConn`). **#384**: gateway-refund-event status advance (`ProcessGatewayRefundEvent(…, expectedTenant)` ignores a foreign refund credit note). **#386**: Razorpay virtual-account reconcile — `IncrementAmountReceived(…, expectedTenant)` scopes the atomic UPDATE (`WHERE razorpay_va_id=$1 AND ($3::uuid IS NULL OR tenant_id=$3)`) so a foreign `va_id` matches 0 rows (the increment itself is the harm, so the check is in the UPDATE, not after); `ReconcileVirtualAccount` swallows the no-rows as an ignore. `uuid.Nil` (env route) disables the filter throughout. Each vector has a failing-first oracle; WHERE-clause semantics verified against Postgres. From security audit 2026-07-31. |
| ~~L1~~ | ~~Pay-in-advance usage bills paused/canceled subscriptions~~ | — | — | **DONE (#388)** — usage ingestion captured a pay-in-advance charge the instant an event landed with **no status gate**, so a **paused** sub kept billing per-event during the pause and a **canceled** (terminal) sub accrued phantom `UnbilledCharge`s no future invoice would sweep up. Gated `PayInAdvanceBiller.BillEvent`: paused/canceled accrue nothing (event still recorded; only the charge is suppressed); active/trialing/past-due bill normally. Oracle: PIA event on paused + canceled → 0 charges (was 1 each), active still bills. From lifecycle×metering audit 2026-07-31. |
| ~~L3~~ | ~~Gift subscription auto-renews and bills the recipient~~ | — | — | **DONE (#390, HIGH)** — a redeemed gift minted the recipient's subscription `Active` with a future period end but `CancelAtPeriodEnd=false` (unresolved dev comments flagged it). A gift is prepaid by the buyer for a fixed duration and the recipient has no payment method, but the renewal worker bills any active sub whose period ended unless `CancelAtPeriodEnd` (renewal.go:145 vs :167) → the recipient was invoiced for a renewal they never agreed to and sent to dunning. Fix: set `CancelAtPeriodEnd=true` at redemption so the gift cleanly expires at period end. Oracle: redeemed gift sub is cancel-at-period-end (+ existing TestRenewSubscription_CancelAtPeriodEnd proves it cancels not invoices). From lifecycle×metering audit 2026-07-31. |
| L2 | Arrears (non-PIA) usage recorded during a pause is billed at the next period close | LOW — product-policy question, not a correctness break | LOW | Companion to L1 (#388): L1 gated the *immediate* pay-in-advance charge, but *arrears* usage events ingested while a subscription is `paused` still aggregate into the next period-close invoice. Whether a pause should suppress usage accrual entirely (freeze), bill it on resume, or bill it normally is a genuine product decision — not obviously a bug. If "pause = no charges" is the intended semantics, exclude paused-window events from the arrears aggregation (touches the usage aggregation query + needs a pause-window record). Deferred pending a product call. From lifecycle×metering audit 2026-07-31. |
| ~~L4~~ | ~~Accepted cancel-flow discount offer was never applied~~ | — | — | **DONE (#392, MED-HIGH)** — the retention/cancel flow's `OfferTypeDiscount` case only **logged** ("coupon application pending") and applied nothing, so a customer who stayed for a promised discount was billed full price (broken promise + silent retention failure). Added `SubscriptionService.ApplyRetentionDiscount`: mints a percentage coupon (repeating for the offer's duration, or forever when 0), attaches it to the subscription with the applied-periods counter reset → the renewal path (`GenerateInvoice`→`couponAppliesThisPeriod`) discounts from the next renewal for the full duration. No new wiring (SubscriptionService already holds the coupon repo). Oracle: "20% for 3 months" → repeating/percent/20 coupon attached, `couponDiscountFor(10000)=2000`, applies periods 0–2 stops at 3; duration 0 → forever; out-of-range percent rejected. Follow-up (optional): model retention discounts via a template instead of one coupon per acceptance. From lifecycle/dev-comment audit 2026-07-31. |
| S4 | Idempotency is opt-in on money-moving POSTs; webhook dedup fails open | LOW/MED — strong secondary guards blunt the money impact | LOW-MED | `idempotency.go:74` processes normally when `Idempotency-Key` is absent; `webhook.go:104` `alreadyProcessed` returns false (processes) on a dedup-store lookup error. Refund/offline/top-up don't *require* a key. But `CreateRefundWithinLimit` caps at `amount_paid` (blocks full double-refund), `MarkInvoicePaid`/ledger reversals are separately idempotent, and the dunning reward is `transitioned`-guarded — so exposure is partial-refund races + fail-open re-triggering non-idempotent side effects (email/dunning). Consider requiring a key on refund/offline + failing dedup closed. From security audit 2026-07-31. |
| ~~Q1~~ | ~~Quote→invoice conversion posts no ledger leg~~ | — | — | **DONE (#394, HIGH)** — `QuoteService.ConvertToInvoice` created the invoice via a raw `invoiceRepo.Create` and had **no ledger dependency**, so unlike every other invoice-creating flow (all call `RecordInvoice`) a quote-converted invoice carried no Code-1 leg (AR→Revenue). When paid, the cash leg (CR AR) had no originating debit → AR negative, Cash overstated, revenue never recognized. The CI invariant harness doesn't exercise the quote path, so it slipped through (same blind spot as revrec R1/R2). Fix: nil-safe `QuoteService.SetLedgerPoster` (wired in main.go) posts the leg after Create; a converted quote has no subscription so `RecordInvoice` books DR AR / CR Revenue immediately + GST→Tax Payable. Oracle: conversion posts exactly one RecordInvoice (was zero). From bug-smell/untrodden-path audit 2026-07-31. |
| ~~Q2~~ | ~~Gift-purchase (buyer) invoice posts no ledger leg~~ | — | — | **DONE (#396, HIGH)** — sibling of Q1, found by auditing all 10 `invoiceRepo.Create` call sites. `GiftService.PurchaseGift` created the buyer's invoice via a raw `invoiceService.InvoiceRepo.Create` with no leg → the buyer's payment posted a cash leg with no originating AR debit → AR negative, gift-sale revenue never recognized. Fix: `s.invoiceService.recordInvoiceLeg(ctx, inv)` after create (one-off, DR AR/CR Revenue immediately; LedgerPoster already wired). Oracle: 3-mo gift posts one RecordInvoice (Total 3000, no sub); zero before. Call-site sweep otherwise CLEAN — the 8 other create sites all pair with a ledger post. From invoice-create-site audit 2026-07-31. |
| B2 | Downgrade credit note lacks a CGST/SGST/IGST breakdown — **investigated 2026-07-31: no consumer, do NOT add the columns yet** | LOW | — | Re-investigated end-to-end before implementing. Adding `subtotal/tax/igst/cgst/sgst/tax_type/hsn` columns to `credit_notes` (migration + domain + repo + populate at downgrade) was **built and reverted** because **nothing reads them**: (1) there is NO credit-note PDF/document renderer; (2) the only statutory consumer, GSTR-1 CDNR (`invoice_repository.go:1173` `GetGSTR1CreditNotes`), already derives the split via `proportionalTax(amount, invIGST, invTotal)` and is scoped to `type='refund' AND invoice_id IS NOT NULL` — so refund credits are already correct (proportional = exact for a same-invoice refund) and downgrade credits (adjustment-type, nil `invoice_id`) aren't in the report at all; (3) no credit-note e-invoicing/IRN path. Stored columns would be dead data. **Real prerequisite work** before B2 has value: a credit-note **document/PDF** OR credit-note **e-invoicing (IRP CDN)** — either would then legitimately need the stored split (esp. for a downgrade whose new plan has a different GST rate than the original, where proportional-from-invoice is wrong). Until one of those exists, B2 is a no-op. |
| W2 | TDS portion is dropped from a payment-reversal's retained amount and the TDS leg (code 10) is never reversed | LOW — **unreachable today** (TDS is INR-only; chargeback/return paths are USD-ACH + EUR/GBP GoCardless), but the code's "reversals are USD-only" invariant is already stale now that GC SEPA/Bacs chargebacks are wired | MED | `subscription_payment.go:143` excludes `TDSAmount` from `retainPaid`; `ledger.go` `RecordPaymentReversal` only inverts the code-3 cash leg, `slog.Warn`s on TDS. If any INR mandate-chargeback path is ever added this becomes a silent over-collect + AR imbalance. Fix when an INR reversal path lands: retain+reverse the TDS leg too, with a PG oracle. From wallet audit 2026-07-31. |
| W3 | GoCardless chargeback path is thinner than the ACH path | LOW — mostly by-design | LOW | `webhook_gocardless.go:168` calls `ReverseSettledPayment` without the ACH path's payment-attempt state update; reopened GC invoices are mandates → scheduler routes them email-only (not auto-re-debited), so a GC chargeback isn't actively re-collected (documented mandate choice). Narrow double-reverse edge if the invoice is re-settled between `charged_back` and `late_failure` (distinct event ids), guarded in practice by the paid-guard + `gateway_payment_id` keying. From wallet audit 2026-07-31. |
| ~~C1~~ | ~~Recurring coupon dropped on renewal + trial~~ | — | — | **DONE (#369, #370, #373)** — the root cause was that the subscriptions repo **never persisted or loaded `coupon_id`** (making #369/#370 inert until #373). #373 persists+loads `coupon_id`, adds `coupon_periods_applied` (migration 000157) threaded through Create/Update/ActivateTrialWithTx, and applies the coupon per `couponAppliesThisPeriod` (forever=every period, once=first only, repeating=first N). PG round-trip + repeating-N tests. Recurring coupons now work end-to-end. |
| ~~T1~~ | ~~Economic-nexus auto-establishment poisons the US collection gate~~ | — | — | **DONE (#371)** — added `tenant_tax_nexus.auto_established` (migration 000156, backfills existing economic→true); `NexusFor.declaredAny` now counts manual declarations only, `inState` counts any. Auto economic nexus is collected but no longer flips a provider-deferring tenant into the restrictive gate. PG oracle in `tax_nexus_gate_pg_test.go`. |
| ~~T3~~ | ~~`COMPANY_STATE` defaults to "TN" for a US env-seller~~ | — | — | **DONE (#375)** — `NewTaxResolver` applies the India "TN" state default only when the seller country is India; `main.go` no longer hardcodes "TN" for an unset `COMPANY_STATE`. A US/EU seller with no state keeps it empty (providers resolve from the from-zip). India behaviour preserved. |
| T4 | Seller misclassified as Indian on a GST row with StateCode but no GSTIN — **WON'T FIX (documented)** | LOW — requires a malformed config row | — | Requiring a GSTIN to classify as an Indian seller would also change India seller **state** resolution: a real India tenant with a StateCode-only config (e.g. KA) would fall through to the env state default (TN), flipping intra/inter-state GST. That's a primary-market behaviour change for a LOW, malformed-config edge — not worth the risk. Left as-is intentionally. From tax audit 2026-07-31. |

## P1 — verification & parity (mostly founder-blocked)

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| ~~H1~~ | ~~Invariant harness doesn't drive service-layer invoice-create paths~~ | — | — | **DONE (#399)** — the harness called `ledger.RecordInvoice` **directly**, so it never exercised the service create-paths (the blind spot that hid Q1/Q2). Added ops driving the real `QuoteService.ConvertToInvoice` + `GiftService.PurchaseGift` through the reconciler. Proven: neutering the quote leg fails the harness with `missing_invoice_transaction` at `op=quote_conversion`. On its FIRST run it caught **Q3** (a real HIGH bug — quote conversion broken against Postgres). All 8 seeds green. From H1 hardening 2026-07-31. |
| ~~Q3~~ | ~~Quote→invoice conversion FK-violates against Postgres (feature broken)~~ | — | — | **DONE (#399, HIGH)** — `ConvertToInvoice` ran `ClaimForConversion` (UPDATE quotes SET invoice_id) BEFORE inserting the invoice; `quotes.invoice_id` has a **non-deferrable** FK to `invoices`, so against real Postgres the claim violated `quotes_invoice_id_fkey` and conversion **failed entirely**. Mock-based unit tests never caught it (mock sets an in-memory field). Fix: insert the invoice and claim the quote in ONE transaction (invoice first → FK satisfied; lost race rolls back the invoice + its gapless number → no orphan/gap); legacy claim-then-create kept for mocks. Added `QuoteRepository.ClaimForConversionWithTx`, `QuoteService.SetTxManager` (wired main.go). Caught by the H1 harness on first run. From H1 hardening 2026-07-31. |
| 3 | **QuickBooks live OAuth verification** | HIGH — parity claim untested; Xero verification found 3 real bugs, QBO likely has rot too | LOW once creds exist | **founder**: developer.intuit.com → create app → redirect URIs `http://localhost:8199/v1/accounting/callback/quickbooks` + `https://api.recurso.dev/v1/accounting/callback/quickbooks` → `QBO_CLIENT_ID`/`QBO_CLIENT_SECRET` into `recur-so/.env`. |
| 4 | **GoCardless webhook registration** | HIGH — mandate activation + settlement (#238/#240) are dead until GC can reach us | LOW | **founder**: GC dashboard → webhook endpoint `https://api.recurso.dev/webhooks/gocardless` + secret → `GOCARDLESS_WEBHOOK_SECRET` on Cloud Run (BYO tenants use the per-connection URL on the Payment Gateways card). |
| 5 | Telemetry receiver deploy (#215) | MED — adoption visibility | LOW | **founder**: 4 wrangler commands in `telemetry-worker/README.md`. |
| 6 | `TRAFFIC_TOKEN` org secret | MED — traffic history beyond GitHub's rolling 14 days | LOW | **founder**: classic PAT with repo scope, org-level secret, then re-dispatch `traffic-snapshot.yml` in all 6 repos. |
| 7 | Real Peppol AP creds (EU e-invoicing inc 2) | MED | MED | **founder** account; retry worker (#89) is merged and waiting. |
| 8 | Demo sandbox hosting + `VITE_DEMO_URL` | MED — website CTA is dark | LOW | **founder**: hosting + DNS; code 100% ready (#214, website #24). |

## P2 — product completeness

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| 11 | Gift-subscription cancel + wallet-close UI edge cases | LOW | LOW | Deferred from roadmap run 2026-07-20. |
| 12 | Dunning alert edit UI | LOW | LOW | Deferred from roadmap run 2026-07-20. |

## P2b — smoke-sweep findings (2026-07-28, see docs/verification-2026-07-28.md)

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| S1 | Card-level accounting Sync should be incremental (force=false) | HIGH — forced full re-push exceeds the 15-min budget on real tenants | LOW | Keep force=true only on the header Sync-now; dirty-tracking already exists. |

## P3 — engineering hygiene

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| ~~12b~~ | ~~React 19 + react-router 8 upgrade~~ | — | — | **DONE** — React 18.2 → 19.2.8, react-router-dom 7 → react-router 8.3.0 (v8 dropped the -dom package; 51 imports renamed), Tremor/lucide React-18 peers overridden to a single React 19, `.trivyignore` (GHSA-qwww-vcr4-c8h2) removed. lint/build/161 tests green. |
| 13 | Pagination consistency on list endpoints | MED — silent truncation has bitten twice (CLAUDE.md) | MED | A few endpoints default `limit=10`, some 50/100/200, many unbounded. Normalize on `ParsePagination` + document defaults in OpenAPI. |
| 14 | Interface-embedding test mocks | LOW-MED — every port widening breaks/panics mocks (`mockLedgerRepoFor*`, `stubCollectionsAgg`, …) | MED | Either generate mocks or convert to narrow per-test interfaces (capability-assertion pattern used by webhook/CRM paths is the house style now). |
| 15 | Dunning-campaign + cancel-flow responses are unwrapped (no `{data:}`) | LOW — known API quirk, clients must stay tolerant | LOW | Breaking change; batch with a future v2 or additive alias. |
| 16 | `head` HTTP-tool alias footgun, iCloud `" 2"` duplicate files | — | — | Environment quirks, documented in memory; no code change. |

## Recently closed (context for the ranking)

- **Revrec + money-endpoint-security audit (2026-07-31, sweep 6)** — two adversarial
  agents. Revrec (both HIGH, both slipped past the invariant harness): a
  wallet/credit-covered invoice funded Deferred but never created a recognition
  schedule → never recognized (#380); a reversal→re-collection created a SECOND
  schedule → Deferred over-drained + revenue double-recognized (#377, idempotent
  per invoice). Security: a low-privilege `member` could self-issue gateway refunds
  (money out, no approval) — now admin/owner-gated (#378); wallet-close +
  offline-payment gated to admin/owner (#379). Each with a failing-first test
  (PG for the revrec ledger). S2 (cross-tenant BYO webhook, HIGH) + S4 (idempotency,
  LOW) above are the open follow-ups. Verified clean: signature verification is
  timing-safe + fail-closed on every gateway; replay dedup exists; secrets are
  write-only; revrec rounding + UnwindOnRefund + multi-entity posting correct.

- **Recurring-coupon + nexus-gate implementation (2026-07-31, sweep 5)** — the two
  HIGH deferrals from sweep 4, built out: coupon now applies on trial conversion
  (#369) and re-applies a `forever` coupon on renewals (#370, discounts the flat
  plan fee, tax on the post-discount base); the economic-nexus auto-establishment
  no longer poisons the US collection gate (#371, `auto_established` flag +
  migration 000156). Each with an oracle test failing on old code. Remaining:
  C1-`repeating` (needs a period counter).
- **Tax-resolution + coupon/credit audit (2026-07-31, sweep 4)** — two adversarial
  agents. Coupon: clamp discount to subtotal + reject >100% percent coupon so no
  negative taxable base reaches the e-invoice/GST (#366). Tax: seller origin added
  to the sales-tax cache key so origin-sourced rates don't leak across tenants
  (#367). Deferred (above): C1 (recurring coupons — feature), T1 (economic-nexus
  gate — needs a migration/decision), T3/T4 (LOW config edges). Verified CLEAN by
  agents: EU VAT B2B reverse-charge + B2C OSS + rate table; credit clamping/ledger
  legs/currency/idempotency; discount order-of-operations.
- **Metering + wallet/dunning audit (2026-07-31, sweep 3)** — two adversarial
  agents. Metering: progressive sweep double-billed pay-in-advance charges (#362,
  HIGH), pay-in-advance rejected with period-cumulative clamps + excluded from
  the usage preview (#363). Wallet re-collect: the settle→return→re-collect cycle
  is correct post-#348/#349; fixed the reversal error-fallback to fail closed
  instead of reopening with retainPaid=0 (#364). W2/W3 above are the LOW residuals
  (latent/by-design). Each fix has an oracle test failing on old code.
- **Currency-exponent + proration-tax audit (2026-07-31)** — two adversarial
  audit agents → 6 fixes shipped: proration tax now uses each plan's own HSN
  rate, not one rate on the net (#357, hits India); exponent-aware
  customer-facing money display (#358); exponent-correct amounts to accounting
  adapters + Avalara (#359); exponent-aware EU e-invoice UBL + import hints
  (#360). Each fix ships with an oracle test that fails on the old code. B2
  (above) is the one deferred item. Verified clean: ledger balance, refund tax
  double-reversal, truncation.

- BYO GoCardless from the dashboard (#237), mandate activation webhooks
  (#238), payment settlement webhooks (#240), currency-aware mandate UI
  (#241), chargeback/late-failure settlement reversal (ACH-parity, reusing
  the #209 occurrence-aware ledger machinery) — overnight 2026-07-27/28.
- Manual accounting sync async + single-flight (#239) — last known
  Cloudflare-timeout landmine.
- QA sweep of #199–#207: all findings closed (#208–#212).
- Live-verified integrations: Stripe (test), Razorpay (test), TaxJar
  (sandbox), Xero (production, at scale), HubSpot (production), GoCardless
  (sandbox, mandate + debit).
