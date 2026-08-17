# Recurso Cloud self-billing — "Recurso runs on Recurso"

**Status:** Increment 1 shipped (customer mirror). Increments 2–4 designed, not built.

## The problem

There are two billing layers, and only one existed:

1. **Tenant → their subscribers** (built, is the product). Spotify uses Recurso to
   bill *their* customers; that money flows to Spotify.
2. **Recurso → its tenants** (this doc). Every signup — Spotify — is a customer of
   **Recurso Cloud**. They should pay the founder based on plan + usage. Before
   this work, tenants were only *labelled* with `plan_tier` / `billing_status` /
   `trial` (migration `000155`); nothing charged them. The website markets this
   layer honestly as *preview / planned*.

## The model

Point Recurso at itself. The founder's own tenant (`PLATFORM_TENANT_ID`) is the
**Recurso Cloud business account**. Every other tenant is mirrored as a
**Customer** inside it, on a **"Recurso Cloud" plan**, billed by the exact same
subscription → invoice → payment → dunning → ledger machinery every tenant uses.

The founder then manages the whole business from the **normal dashboard**:
Spotify is just a customer — click it for its subscription, usage, invoices,
payments, and ledger. No special operator UI required.

```
signup tenant (Spotify)  ──mirrored as──▶  Customer in PLATFORM_TENANT_ID
        │                                        │
        │ their own billing (layer 1)            │ subscription to "Recurso Cloud" plan
        ▼                                        ▼
   Spotify's subscribers                   metered on Spotify's usage → invoice → founder's ledger
```

## Pricing (published on recurso.dev, used as the default)

Free under **$10,000 tracked revenue / month**, then whichever is **cheaper**:
**0.4% of collected volume** or a flat **$99 / month**. Paid tiers are ordinary
Recurso plans. (Adjustable — this is config, not a hard-coded fact.)

## Increments (gated; no money moves before the increment that owns it)

- **Increment 1 — customer mirror (SHIPPED, money-free).**
  - `PLATFORM_TENANT_ID` env selects the founder tenant (unset → feature off).
  - New table `cloud_tenant_customer` maps a signup tenant → its Customer in the
    platform tenant (`UNIQUE (platform_tenant_id, tenant_id)` → idempotent).
  - `CloudBillingService.ProvisionTenant` creates the Customer via the normal
    `CustomerService.CreateCustomer` (so it gets a ledger account like any
    customer) and records the mapping. It skips the founder tenant itself and is
    a no-op if already mapped.
  - Hooked into `AuthService.Register` **best-effort / fail-open** — a
    provisioning error never fails a signup (mirrors the country-stamp pattern).
  - `Backfill` provisions existing tenants; runs once at boot (idempotent).
  - **Result:** every signup now shows up as a customer in the founder's account.
    No plan, no subscription, no charge yet.

- **Increment 2 — Recurso Cloud plan + usage metering.** Create the "Recurso
  Cloud" plan (free base + a metered component). A monthly job measures each
  tenant's own tracked revenue / collected volume (already in the DB) and pushes
  it as usage events onto that tenant's subscription. Fills
  `cloud_tenant_customer.subscription_id`. Still free under quota → no charge.

- **Increment 3 — charging (money path, invariant-harness gated).** Apply the
  quota + price, generate the invoice, post the ledger legs. This is the first
  increment that moves money and must keep reconciliation at zero.

- **Increment 4 — collection + dunning.** Charge the card, retry on failure —
  reusing the existing payment + dunning machinery.

## Operator setup

Set `PLATFORM_TENANT_ID=<founder tenant id>` on the Cloud Run API. On the next
boot the backfill mirrors existing tenants; new signups mirror automatically.
(To also see the raw cross-tenant funnel, `FOUNDER_TOKEN` still gates
`GET /platform/metrics`.)

## Non-goals / guardrails

- The founder tenant is never mirrored as a customer of itself.
- Layer-1 money (a tenant billing their own subscribers) is untouched.
- Nothing in Increment 1 posts a ledger leg or moves money; the invariant
  harness is unaffected until Increment 3, which is explicitly gated by it.
