# Recurso Cloud self-billing — "Recurso runs on Recurso"

**Status:** Increments 1 (customer mirror) + 2 (usage meter) shipped — both money-free. Increments 3 (plan + subscription + charging) and 4 (collection) designed, not built.

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

- **Increment 2 — usage meter (SHIPPED, money-free).** A daily scheduler
  (`CloudUsageScheduler`, distributed-locked, gated behind `PLATFORM_TENANT_ID`)
  measures every tenant's current-month activity into `cloud_tenant_usage`, one
  reading per `(tenant, period, currency)`:
  - `tracked_revenue_minor` = `SUM(invoices.total)` in the window (paid or not —
    matches the published definition; the free-tier threshold compares here).
  - `collected_volume_minor` = `SUM(payment_attempts.amount)` for succeeded
    attempts (currency from the invoice) — the base a usage fee applies to.
  Idempotent upsert, so re-measuring the accruing month just refreshes rows.
  **Why no plan/subscription here:** in Recurso, creating a (non-trial)
  subscription immediately raises an invoice (`CreateSubscription` builds the
  first invoice, even at $0), and a trial auto-converts on schedule — both move
  money. To keep money strictly out until it's gated, the plan + subscription
  are created *together with* charging in Increment 3, not here. This increment
  only reads billing data and writes readings; the invariant harness is
  unaffected.

- **Increment 3 — Recurso Cloud plan + subscription + charging (money path,
  invariant-harness gated).** Create the "Recurso Cloud" plan (free base + a
  metered component), subscribe each cloud customer (filling
  `cloud_tenant_customer.subscription_id`), and turn the `cloud_tenant_usage`
  readings into a charge: apply the quota + price, generate the invoice, post
  the ledger legs — all under a controlled billing run that keeps reconciliation
  at zero. This is the first increment that moves money.

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
