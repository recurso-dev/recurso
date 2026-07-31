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
| S2 | Cross-tenant BYO webhook binding — **invoice paths DONE (#382)**, **refund path DONE (#384)**, VA reconcile is the last piece | **HIGH** (security) — the high-impact invoice settle/reverse + refund vectors are closed; the residual is the lowest-impact vector | LOW | **#382** binds every invoice-settle/failure/reversal load across all 3 gateways to the connID's tenant (`webhookSecretFor` stashes the conn tenant on the request context; `invoiceBelongsToWebhookConn` guards the 7 load sites). **#384** binds the gateway-refund-event path: `ProcessGatewayRefundEvent(…, expectedTenant)` ignores a resolved refund credit note owned by a different tenant (Stripe/Razorpay BYO handlers pass `webhookConnTenant`; `uuid.Nil` env route skips). **Last piece (open, LOW):** Razorpay virtual-account reconcile (`OfflinePaymentService.ReconcileVirtualAccount` by `razorpay_va_id`) still needs the same binding — but the check must land BEFORE `IncrementAmountReceived` (the increment itself, on another tenant's VA, is the harm: it can push their VA over `AmountExpected` and settle their invoice). Fix: thread `expectedTenant` and scope the atomic increment to it (`WHERE razorpay_va_id=$1 AND (tenant_id=$expected OR $expected IS NULL)`) so a foreign `va_id` matches 0 rows → no increment, no settle. Lowest-impact vector (needs a BYO Razorpay VA + knowledge of another tenant's `razorpay_va_id`). From security audit 2026-07-31. |
| S4 | Idempotency is opt-in on money-moving POSTs; webhook dedup fails open | LOW/MED — strong secondary guards blunt the money impact | LOW-MED | `idempotency.go:74` processes normally when `Idempotency-Key` is absent; `webhook.go:104` `alreadyProcessed` returns false (processes) on a dedup-store lookup error. Refund/offline/top-up don't *require* a key. But `CreateRefundWithinLimit` caps at `amount_paid` (blocks full double-refund), `MarkInvoicePaid`/ledger reversals are separately idempotent, and the dunning reward is `transitioned`-guarded — so exposure is partial-refund races + fail-open re-triggering non-idempotent side effects (email/dunning). Consider requiring a key on refund/offline + failing dedup closed. From security audit 2026-07-31. |
| B2 | Downgrade credit note lacks a CGST/SGST/IGST breakdown | LOW — the credit note **amount** is already correct (audit B1, #357); only the printed itemized GST split is missing, a compliance-doc nicety for India | MED | Needs a schema migration: `domain.CreditNote` has **no** tax columns (subtotal/tax/IGST/CGST/SGST). Add columns + populate from the proration `taxRes` in `persistPlanChange` + render on the credit-note PDF. Deferred from the 2026-07-31 exponent audit as a feature, not a fix. |
| W2 | TDS portion is dropped from a payment-reversal's retained amount and the TDS leg (code 10) is never reversed | LOW — **unreachable today** (TDS is INR-only; chargeback/return paths are USD-ACH + EUR/GBP GoCardless), but the code's "reversals are USD-only" invariant is already stale now that GC SEPA/Bacs chargebacks are wired | MED | `subscription_payment.go:143` excludes `TDSAmount` from `retainPaid`; `ledger.go` `RecordPaymentReversal` only inverts the code-3 cash leg, `slog.Warn`s on TDS. If any INR mandate-chargeback path is ever added this becomes a silent over-collect + AR imbalance. Fix when an INR reversal path lands: retain+reverse the TDS leg too, with a PG oracle. From wallet audit 2026-07-31. |
| W3 | GoCardless chargeback path is thinner than the ACH path | LOW — mostly by-design | LOW | `webhook_gocardless.go:168` calls `ReverseSettledPayment` without the ACH path's payment-attempt state update; reopened GC invoices are mandates → scheduler routes them email-only (not auto-re-debited), so a GC chargeback isn't actively re-collected (documented mandate choice). Narrow double-reverse edge if the invoice is re-settled between `charged_back` and `late_failure` (distinct event ids), guarded in practice by the paid-guard + `gateway_payment_id` keying. From wallet audit 2026-07-31. |
| ~~C1~~ | ~~Recurring coupon dropped on renewal + trial~~ | — | — | **DONE (#369, #370, #373)** — the root cause was that the subscriptions repo **never persisted or loaded `coupon_id`** (making #369/#370 inert until #373). #373 persists+loads `coupon_id`, adds `coupon_periods_applied` (migration 000157) threaded through Create/Update/ActivateTrialWithTx, and applies the coupon per `couponAppliesThisPeriod` (forever=every period, once=first only, repeating=first N). PG round-trip + repeating-N tests. Recurring coupons now work end-to-end. |
| ~~T1~~ | ~~Economic-nexus auto-establishment poisons the US collection gate~~ | — | — | **DONE (#371)** — added `tenant_tax_nexus.auto_established` (migration 000156, backfills existing economic→true); `NexusFor.declaredAny` now counts manual declarations only, `inState` counts any. Auto economic nexus is collected but no longer flips a provider-deferring tenant into the restrictive gate. PG oracle in `tax_nexus_gate_pg_test.go`. |
| ~~T3~~ | ~~`COMPANY_STATE` defaults to "TN" for a US env-seller~~ | — | — | **DONE (#375)** — `NewTaxResolver` applies the India "TN" state default only when the seller country is India; `main.go` no longer hardcodes "TN" for an unset `COMPANY_STATE`. A US/EU seller with no state keeps it empty (providers resolve from the from-zip). India behaviour preserved. |
| T4 | Seller misclassified as Indian on a GST row with StateCode but no GSTIN — **WON'T FIX (documented)** | LOW — requires a malformed config row | — | Requiring a GSTIN to classify as an Indian seller would also change India seller **state** resolution: a real India tenant with a StateCode-only config (e.g. KA) would fall through to the env state default (TN), flipping intra/inter-state GST. That's a primary-market behaviour change for a LOW, malformed-config edge — not worth the risk. Left as-is intentionally. From tax audit 2026-07-31. |

## P1 — verification & parity (mostly founder-blocked)

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
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
