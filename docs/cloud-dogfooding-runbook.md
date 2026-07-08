# Recurso Cloud dogfooding runbook

**Recurso Cloud bills its own customers with Recurso.** A dedicated Recurso
tenant (the "Recurso Cloud" business) runs the Cloud pricing as a catalog and
invoices Cloud subscribers on the same engine we sell. This is the strongest
proof the product works — our own revenue runs through it.

This runbook is the operational playbook. It's engineering-complete: the catalog
and flow below work today against any Recurso instance. The 🔒 items (live
payment credentials, final add-on prices, the production tenant) are founder
decisions and are called out at the end.

## Pricing → Recurso primitives

From the public [pricing page](https://recurso.dev/pricing):

| Cloud pricing element | Recurso primitive |
|---|---|
| $299/mo base (incl. 10k invoices + 5k active subs) | **Plan** `cloud_base`, recurring, $299/mo |
| $0.02 / invoice over 10,000 | **Usage** dimension `invoices` → monthly overage **charge** |
| $0.05 / active sub over 5,000 | **Usage** dimension `active_subscriptions` → monthly overage **charge** |
| Add-ons (churn, analytics, accounting, FX) | **Add-on plans**, attached per subscription |
| Enterprise | Custom quote (out of scope for self-serve billing) |

The base fee and add-ons ride the normal recurring invoice. Overage is metered
via the usage platform and added to the next invoice as an itemized charge.

## One-time: create the catalog

```bash
API_URL=https://billing.recurso.dev/v1 API_KEY=<cloud-tenant-key> \
  ./scripts/cloud-billing-setup.sh
```

Idempotent — creates `cloud_base` plus the four add-on plans, skipping any that
already exist. Add-on prices are placeholders; override with `CLOUD_ADDON_*`
env vars once finalized.

## Onboard a Cloud customer

1. **Create the customer** (the company signing up for Cloud):

   ```bash
   curl -X POST "$API_URL/customers" -H "Authorization: Bearer $API_KEY" \
     -H 'Content-Type: application/json' \
     -d '{"name":"Acme Inc","email":"billing@acme.com"}'
   ```

2. **Subscribe them to the base plan** (`plan_id` = the `cloud_base` id):

   ```bash
   curl -X POST "$API_URL/subscriptions" -H "Authorization: Bearer $API_KEY" \
     -H 'Content-Type: application/json' \
     -d '{"customer_id":"<cust_id>","plan_id":"<cloud_base_id>","payment_terms":"net15"}'
   ```

3. **Attach any add-ons** they opted into (per add-on plan id):

   ```bash
   curl -X POST "$API_URL/subscriptions/<sub_id>/addons" \
     -H "Authorization: Bearer $API_KEY" -H 'Content-Type: application/json' \
     -d '{"plan_id":"<cloud_addon_analytics_id>","quantity":1}'
   ```

The customer now recurs at $299/mo + add-ons automatically.

## Monthly cycle: meter usage → bill overage

Once per billing period, before the invoice generates:

1. **Report the customer's usage** for the period (their Recurso instance's
   invoice and active-subscription counts):

   ```bash
   curl -X POST "$API_URL/usage/events" -H "Authorization: Bearer $API_KEY" \
     -H 'Content-Type: application/json' \
     -d '{"subscription_id":"<sub_id>","customer_id":"<cust_id>","dimension":"invoices","quantity":14000}'
   curl -X POST "$API_URL/usage/events" -H "Authorization: Bearer $API_KEY" \
     -H 'Content-Type: application/json' \
     -d '{"subscription_id":"<sub_id>","customer_id":"<cust_id>","dimension":"active_subscriptions","quantity":7000}'
   ```

2. **Compute overage** against the included allowances:

   ```
   invoice_overage = max(0, invoices − 10000) × $0.02
   sub_overage     = max(0, active_subscriptions − 5000) × $0.05
   ```

   For the example above: `(14000−10000)×0.02 = $80` + `(7000−5000)×0.05 = $100`
   = **$180** overage.

3. **Add the overage as a charge** on the next invoice (amount in cents):

   ```bash
   curl -X POST "$API_URL/subscriptions/<sub_id>/charges" \
     -H "Authorization: Bearer $API_KEY" -H 'Content-Type: application/json' \
     -d '{"amount":18000,"currency":"USD","description":"Usage overage (Jul): 4,000 invoices + 2,000 subscriptions"}'
   ```

4. The recurring invoice then generates as **base $299 + add-ons + overage**,
   and **dunning** (already built) handles any failed payment automatically.

> Tip: `GET /subscriptions/<sub_id>/usage` returns the current-period and
> lifetime usage per dimension, so step 2 can be automated with a small cron
> that reads usage and posts the charge.

## What runs automatically vs. by hand

- **Automatic:** recurring $299 + add-ons, invoice generation, payment capture,
  dunning/retries, receipts.
- **Monthly (scriptable):** report usage, compute overage, post the overage
  charge. A cron reading `GET /subscriptions/:id/usage` closes this loop.

## 🔒 Founder / production items

- **Live payment gateway keys** (Stripe/Razorpay) on the Cloud tenant — required
  to actually capture money. Local/testing uses the mock gateway.
- **Final add-on prices** — the placeholders in the setup script are a pricing
  decision.
- **The production "Recurso Cloud" tenant** — provision it (see
  `cloud-provisioning-runbook.md`), run `cloud-billing-setup.sh` against it, and
  point the signup flow at it.
- **Metering source** — wire the cron that pulls each Cloud instance's real
  invoice/subscription counts into the usage events above.
