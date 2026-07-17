# Data Residency: the Self-Hosted Guarantee

Recurso is self-hostable, and when you run it with

```bash
RESIDENCY_MODE=self_hosted
```

the deployment enforces this guarantee:

> **Your financial data never leaves your deployment, except through the
> statutory and collection channels you yourself configure.**

This document is written to be handed to a security reviewer. It states
exactly what the flag disables, what egress remains (and why), and how to
verify the guarantee against the source.

## What `RESIDENCY_MODE=self_hosted` hard-disables

| Channel | Normal behavior | Under `self_hosted` |
|---|---|---|
| **Anonymous telemetry** | Opt-in via `TELEMETRY_OPTIN=true` (coarse, anonymous milestones — see `docs/telemetry.md`) | **Disabled even if opted in.** The residency guarantee outranks the opt-in. (`internal/adapter/telemetry/telemetry.go`, `NewFromEnv`) |
| **Accounting-SaaS sync** (QuickBooks, Xero) | Two-way sync of invoices/payments/credit notes when connected | **Disabled.** The OAuth connect flow cannot start (client configs are blanked), and existing connection rows in the database are refused at sync time — they fall back to a no-op adapter and log a warning. (`cmd/api/main.go`; `internal/service/accounting.go`, `getAdapterForConnection`) |
| **External sales-tax API** (TaxJar) | US sales-tax rates fetched when `TAXJAR_API_KEY` is set | **Disabled.** The US tax engine runs as its honest 0% stub; no invoice or address data is sent to a third-party tax API. (`cmd/api/main.go`) |

**Tally export remains available** — it is a one-way local file export
(JSONL on your disk), not network egress.

## What egress remains — always operator-configured

These channels exist because billing cannot function without them. Each one
is configured by you, points only at endpoints you choose, and sees only the
data its function requires:

1. **Payment gateways** (Stripe and/or Razorpay, per your API keys) — payment
   collection requires sending charge/mandate data to the gateway you chose.
2. **GSP / IRP** (your NIC credentials) — Indian e-invoicing requires
   registering invoices for IRN with the government via your GSP session.
3. **SMTP** (your relay) — invoice and dunning emails go through the mail
   server you configure.
4. **Outbound webhooks** — signed events are delivered only to endpoint URLs
   you register.

No other network destination receives financial data. There is no license
server, no phone-home, no usage reporting, no error-tracking SaaS in the
data path.

## Storage

All financial data lives in **your PostgreSQL** (and, if you enable the
optional mirror, **your TigerBeetle cluster**). There is no Recurso-hosted
storage tier; backups are whatever you configure for your own database.

## Verifying the guarantee

- **From source:** the enforcement points are listed in
  `internal/residency/egress_guard_test.go`; each carries a unit test
  (`internal/adapter/telemetry/residency_test.go`,
  `internal/service/accounting_residency_test.go`).
- **At startup:** the API logs each disabled channel, e.g.
  `Accounting sync (QuickBooks/Xero) disabled by RESIDENCY_MODE=self_hosted`.
- **On the network:** run the stack with an egress-filtering proxy or
  `tcpdump` and observe that outbound connections go only to your configured
  gateway, GSP, SMTP, and webhook hosts.

## Scope notes

- The flag governs the **API and its workers**. The dashboard is a static
  React app served by you; it talks only to your API.
- The guarantee is about **financial data**. Pulling container images or OS
  packages during deployment is your infrastructure's concern, outside the
  application's data path.
