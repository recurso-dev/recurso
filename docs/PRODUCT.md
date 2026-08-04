# Recurso — Product

> What Recurso is and what it actually does. The feature inventory (§4) is
> **code-derived** — each capability names the package that implements it and is
> marked IMPL (implemented) or PARTIAL (real interface, mock/stub transport or
> provider-gated). Implementation wins over this doc.

## 1. Mission

Recurso is the **accounting-first financial operating system for SaaS**. Unlike
Stripe Billing, Chargebee, or RevenueCat, every financial event produces an
auditable, double-entry ledger posting (`internal/service/ledger.go`). Billing is
the surface; a reconcilable set of books is the product.

**The promise:** every number a customer sees traces to the journal entries
behind it, and the books always balance — enforced by the reconciler and
invariant harness (see `ACCOUNTING_PRINCIPLES.md`).

## 2. Who it's for / not for

**For:** SaaS / AI / developer-tools / B2B software companies billing
subscriptions and/or usage; their finance teams who close monthly and get
audited; global sellers owing GST (India), VAT (EU/UK), US sales tax. **Not
for:** consumer/mobile-only app-store billing where an entitlements tool
(RevenueCat) suffices; teams that want a dashboard-only tool with no ledger.

## 3. Principles (non-negotiable)

Every number explainable · every event reversible · no hidden state · no silent
corrections · the books always reconcile · everything auditable. A change that
violates one is wrong even if it ships green. (See `ANTI_PATTERNS.md`.)

## 4. Feature inventory (code-derived)

**Subscriptions — IMPL:** `subscription.go`, `subscription_trial.go` (trials),
`subscription_upgrade.go`, `subscription_cancel.go` + `cancel_flow.go` +
`subscription_retention.go` (retention offers), `subscription_pause.go`,
`subscription_addon.go`, `renewal.go`. Downgrades via the credit path (codes
16/17/21).

**Usage / metering — IMPL:** `metering.go`, `usage.go`, `rating.go`,
`pay_in_advance.go`, `progressive_billing.go`, `advanced_billing.go`,
`weighted_sum.go`, `custom_expr.go`, `usage_alert.go`.

**Invoicing + statutory:** core IMPL (`invoice.go`, `pdf_invoice.go`,
`invoice_branding.go`). India GST IRP e-invoice **IMPL (real NIC HTTP)**
(`einvoice.go` + `adapter/gsp/nic.go`; GSTR-1/3B filing incl. gov submission);
`CancelIRN` is a soft no-op — **PARTIAL**. EU UBL/Peppol: document generation
IMPL (`euinvoice_ubl.go`), but the Peppol Access Point transport is a **mock**
(`adapter/einvoice_eu/mock.go`) — **PARTIAL (real AP founder-blocked).**

**Revenue recognition — IMPL:** `revrec.go`, `trial_balance.go`, `close_pack.go`,
`revenue_waterfall.go`, `mrr_waterfall.go`.

**Collections / dunning — IMPL:** `smart_retry.go`, `dunning_campaign.go`,
`dunning_recovery.go` + `dunning_analytics.go` (recovery attribution),
`collections_actions.go` (worklist).

**Tax — IMPL (US rate PARTIAL):** GST/VAT/no-tax engines
(`core/service/tax/`); BYO TaxJar/Avalara/Ziptax (`adapter/taxprovider/`), VAT
VIES (`adapter/vatprovider/`); US nexus (`nexus_status.go`). US sales-tax rate is
a **0% stub without an injected provider** — PARTIAL.

**Accounting — IMPL (live OAuth PARTIAL):** ledger/trial-balance/multi-entity/
reconciliation (see `ACCOUNTING_PRINCIPLES.md`). External sync: real QuickBooks/
Xero/NetSuite/Tally adapters (`adapter/accounting/` + `oauth.go`), but live flows
depend on registered OAuth apps — PARTIAL (provider-gated).

**Payments — IMPL:** Stripe/Adyen/Razorpay gateways + smart routing
(`adapter/gateway/`); ACH/UPI/GoCardless mandates (`mandate.go` +
`adapter/gateway/gocardless.go`); wallets (`wallet.go`, codes 11–15); offline
payments (`offline_payment.go`); saved cards; **BYO gateways**
(`tenant_gateway.go` + `gateway_connection.go`); disputes (`dispute.go`).

**Credit notes / coupons / gifts / referrals / quotes — IMPL:** `credit_note.go`
+ `pdf_credit_note.go`; `handler/coupon.go`; `gift.go`; `referral.go`;
`quote.go`; `pricing_simulator.go`.

**Customer portal — IMPL:** `portal.go` + `handler/portal_api.go`; consent
(`consent.go`); entitlements (`entitlement.go`).

**MCP server — IMPL:** `cmd/mcp/` + `internal/mcp/` — standalone, tier-gated,
authenticates with `rsk_` keys forwarded to `/v1`, holds no DB. Deployed
separately (`Dockerfile.mcp`).

**Imports + Compare gate — IMPL:** Stripe/Chargebee/RevenueCat importers
(`internal/importer/`) + read-only Compare reports (`*_compare.go`,
`compare_report_html.go`) diffing export vs live data on coverage/fidelity/
continuity.

**SSO / organizations — IMPL (IdP OAuth provider-gated):** `sso.go`, `oauth.go`,
`auth_phase2.go`, `team.go`, `auth_invite.go`; `organization.go`, `entity.go`
(multi-entity books).

**Also implemented:** analytics + GenAI (`analytics.go`, `genai.go` +
`adapter/ai/`), churn modeling (`churn_model.go`), unit economics
(`unit_economics.go`), outbound webhooks (`webhook.go` +
`handler/webhook_management.go`), notifications (`notification.go`), demo mode
(`demo.go`).

## 5. Partial / founder-blocked (honest status)

- EU Peppol Access Point — document layer real, transport mock.
- US live sales-tax rates — 0% stub without a provider.
- Accounting sync — adapters real, live OAuth apps provider-gated.
- IRN cancellation — soft no-op.

These are clearly PARTIAL, not claimed as shipped. Tracked where applicable in
`../REMEDIATION.md`.

## 6. Architecture & differentiation

See `ARCHITECTURE.md` (hexagonal Go API, dual-write ledger, schedulers/workers).
See `COMPETITORS.md` for where we win (auditable ledger, reconciliation
guarantees) and lose (payments breadth, metering scale).

## Source of truth

- **Code:** `internal/service/*`, `internal/adapter/*`, `cmd/*` (per capability
  above).
- **Evidence file:** `docs/evidence/accounting-and-product.md`.
- **Related:** `ARCHITECTURE.md`, `ACCOUNTING_PRINCIPLES.md`, `COMPETITORS.md`,
  `DOCUMENTATION_RULES.md`, `../REMEDIATION.md`.
