# Spec: Bring-Your-Own Gateway & Integration Credentials

> **Status: IN PROGRESS — Increments 1 (vault), 2 (resolver/charge-path),
> 2b-1 (checkout verify), 3 (per-connection webhooks), and 4 (dashboard UI)
> shipped under the D1–D5 defaults. BYO is now usable end-to-end for the
> interactive pay-link flow; 2b-2 (saved-card/autopay) + 5 (tax/CRM/storage
> keys) remain.**
>
> Motivation: today every third-party credential (Stripe, Razorpay, TaxJar,
> Avalara, HubSpot, S3) is a boot-time environment variable, **one set per API
> instance, shared across all tenants**. The entire app is `tenant_id`-scoped
> except the payment gateway — the one integration that most needs to be. For a
> multi-tenant hosted offering each tenant must connect **their own** merchant
> account from the dashboard.

## Objective

Every tenant configures their own payment keys (and, later, tax/CRM/storage
keys) from the dashboard. Credentials are encrypted at rest. The env-configured
"platform gateway" remains the fallback so self-hosted single-merchant and
existing deployments keep working unchanged.

## Current architecture (what we're changing)

- Gateways are built once at boot in `cmd/api/main.go` from env keys into a
  singleton `gateway.SmartRouter` (INR→Razorpay, else→Stripe, plus currency
  overrides), injected into ~a dozen services.
- Webhooks (`POST /webhooks/stripe`, `/webhooks/razorpay`) verify with a single
  env signing secret, then resolve the tenant *after* verification from the
  invoice id in the payload metadata.

## Increments

### Increment 1 — Encrypted credential vault ✅ (this PR)

- `internal/adapter/secretbox` — AES-256-GCM seal/open under a 32-byte master
  key (`GATEWAY_ENCRYPTION_KEY`, base64 or hex). Go stdlib only. Nonce-random,
  authenticated; wrong-key/tampered ciphertext fails closed.
- `domain.GatewayConnection` + migration `000107_gateway_connections` — per
  (tenant, provider) with a **partial unique index** on `active`, so a
  disconnected row survives for the audit trail. Secret columns hold ciphertext
  (`secret_key_enc`, `webhook_secret_enc`); `public_key` (Razorpay key_id /
  Stripe publishable key) is plaintext because it ships to the browser anyway.
  Secrets carry `json:"-"` — never serialized to API clients.
- Repository + `GatewayConnectionService` (seals on write, opens for the
  resolver/webhook router). **A nil vault (no key) makes every write fail with
  `ErrGatewayVaultUnavailable`** — plaintext secrets can never be persisted.
- Fully unit-tested (round-trip, nonce randomness, wrong-key, tamper, replace-
  active, validation, disconnect, vault-absent). **Nothing wired to charge
  flows yet — zero money-path risk.**

### Increment 2 — Resolver + charge-path rewire ✅

`gateway.GatewayResolver.For(ctx, tenantID) *SmartRouter` builds a per-tenant
`SmartRouter` from the tenant's decrypted connections, reusing the env gateway
for any un-connected provider slot; a per-tenant cache keyed by an
id+updated_at signature avoids re-decrypting on every charge. `gateway.
TenantGateway` wraps it as a drop-in `port.PaymentGateway`: it reads the tenant
from `ctx` (`domain.TenantIDKey`) and **falls back to the env gateway** when
there's no tenant / no connection / no vault (D1). Wiring in `main.go` swaps the
injected `paymentGateway` for this wrapper across all 8 consumers, so **no
service/handler call site changed** and with no vault the behavior is
byte-for-byte the env gateway (zero regression). Public charge-origination sites
(checkout `InitiatePayment`, `payment.CreateOrder`) inject the invoice's tenant
into `ctx` before charging. Unit-tested: routes to tenant gateway when
connected, env fallback for un-connected slot / no connection / no tenant / nil
resolver, and cache reuse + invalidation on re-key.

### Increment 2b-1 — Per-tenant checkout verification ✅

The interactive checkout flow (customer clicks a pay link) is now end-to-end
per-tenant. `GatewayResolver.StripeFor/RazorpayFor(ctx, tenantID)` return the
tenant's concrete gateway (env fallback), and the checkout handler resolves
against the invoice's tenant for:

- the **Stripe status inspector** (`CheckoutSuccess` verify) — a BYO order is now
  verified on the seller's own Stripe account;
- the **Razorpay verifier** (`RazorpayVerify`) — verified with the seller's own
  secret;
- the **buyer-details** setter (India export intents);
- the **browser public keys** (Stripe publishable / Razorpay key_id) returned by
  `InitiatePayment`, so the widget mounts on the account that created the order.

`SetTenantGateways(resolver, connLookup)` wires it; unset ⇒ env values unchanged
(backward compat). Unit-tested (StripeFor/RazorpayFor connected vs env fallback).

### Increment 2b-2 — Per-tenant saved-card & SetupIntent (pending)

The remaining concrete sub-flows are card-on-file / autopay, not the interactive
pay-link path, so deferred: retry + renewal + wallet **saved-card off-session
charger** (`ChargeSavedPaymentMethod` — saved PMs live on the tenant's Stripe
customer) and portal **SetupIntent** (`EnsureStripeCustomer`/`CreateSetupIntent`/
`FinalizeSetupIntent`). Needed before a BYO tenant can store cards / run autopay
on their own account.

### Increment 3 — Per-connection webhooks ✅

Each tenant's gateway account has its **own** signing secret, and the signature
must be verified *before* the payload is trusted. New per-connection routes
`POST /webhooks/stripe/:connID` and `/webhooks/razorpay/:connID` share the
existing handlers: `webhookSecretFor` resolves the `:connID` connection →
decrypts its webhook secret → the same signature check + processing runs with
that secret; the legacy env routes (no `:connID`) are unchanged. **Fail closed**
throughout — unknown/mismatched-provider/inactive connection → 404 (no id
enumeration), a connection with no webhook secret → 503, bad signature → 401.
OpenAPI spec + drift test cover both new paths. Unit-tested: no-resolver,
invalid id, not-found, wrong-provider, no-secret-fail-closed, and
resolved-secret-reaches-verification. The dashboard (increment 4) will surface
each connection's webhook URL + secret to paste into the gateway console.

### Increment 4 — Dashboard: Integrations → Payments ✅

`GatewayConnectionHandler` exposes `GET/POST /v1/gateway-connections`,
`DELETE /v1/gateway-connections/:provider`, and
`PUT /v1/gateway-connections/:provider/webhook-secret` (writes owner/admin-
gated). Secrets are **write-only** — the API returns only a secret-free
`GatewayConnectionView` (public key, mode, `has_webhook_secret`,
`webhook_path`). A `PaymentGateways` component on the Integrations page shows
Stripe/Razorpay cards with a connect sheet (mode + keys), a mode badge, the
per-connection **webhook URL** to copy into the gateway console, and a two-step
webhook-secret field (create the webhook at the URL, paste the secret back).
`vault_ready:false` (no `GATEWAY_ENCRYPTION_KEY`) disables connecting with an
explanatory banner. OpenAPI + drift test cover all four routes; handler and
service unit-tested; frontend lint/build/vitest green.

### Increment 5 — Extend the vault to tax/CRM/storage keys

Same `secretbox` + connection pattern for the operator-only integrations, so
they move from env-only to per-tenant dashboard config. Env stays the fallback.
Founder decision (2026-07-20): do all three — tax, CRM, storage.

**5a — backend + API + tax providers ✅.** A generic `integration_connections`
table (migration 000108) stores a sealed JSON config blob per (tenant, category,
provider); `IntegrationConnectionService` seals on write and `Resolve`s a
decrypted config per tenant (env fallback on miss). `SalesTaxProviderResolver`
builds a tenant's TaxJar/Avalara from their connection (injected factory, cached
per tenant, invalidated on re-key) and `TaxResolver.WithPerTenantSalesTax` uses
it at invoice time — same `RESIDENCY_MODE=self_hosted` guard as the env provider.
`IntegrationConnectionHandler` exposes `GET/POST /v1/integration-connections` and
`DELETE /v1/integration-connections/:category/:provider` (secret-free views,
owner/admin writes). OpenAPI + drift green; service/resolver/handler unit-tested.

**5b — dashboard UI (pending):** a Tax/CRM/Storage section on the Integrations
page (cards + connect sheet driven by each provider's required fields).

**5c — CRM (HubSpot) + storage (S3) resolvers (pending):** wire `Resolve` into
the HubSpot sync and the S3 GL-export paths (the vault + API already support
them; only the per-tenant resolution at those call sites remains).

## Founder decisions

| # | Question | Proposed default |
| --- | --- | --- |
| D1 | Fallback when a tenant has no connection | **Use the env/platform gateway** (keeps self-hosted + existing tenants working). Alternative: hard-fail with "connect a gateway first". |
| D2 | `GATEWAY_ENCRYPTION_KEY` absent on a multi-tenant deploy | **BYO disabled, dashboard shows "unavailable", env gateway still works.** Never store plaintext. |
| D3 | Key rotation | v1: single key, document rotation as decrypt-all/reseal migration. Envelope keys (per-connection DEK) deferred. |
| D4 | Mandate/UPI (Razorpay-only) under BYO | Mandate debit resolves the tenant's Razorpay connection; tenants without one fall back to env (D1). |
| D5 | Live-key validation at connect time | v1: **format-check only** (no test API call), to avoid egress under `RESIDENCY_MODE=self_hosted`. Optional "Test connection" button in increment 4. |

## Boundaries

- **Never** persist a gateway secret in plaintext; **never** serialize `*Enc`
  fields or opened secrets to API clients; **never** log decrypted secrets.
- **Always** keep the env gateway path working (fallback); route every charge
  through the resolver once increment 2 lands.
- **Ask first:** anything that changes charge routing semantics or webhook
  verification (increments 2–3) before merge.
