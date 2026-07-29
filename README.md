<p align="center">
  <img src="docs/logo.svg" alt="Recurso" width="80" />
</p>

<h1 align="center">Recurso</h1>

<p align="center">
  Open-source billing engine for SaaS. Built with Go, PostgreSQL, and TigerBeetle.
</p>

<p align="center">
  <a href="https://github.com/recurso-dev/recurso/actions"><img src="https://github.com/recurso-dev/recurso/workflows/CI/badge.svg" alt="Build Status" /></a>
  <a href="https://github.com/recurso-dev/recurso"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+" /></a>
  <a href="https://github.com/recurso-dev/recurso/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg" alt="MIT License" /></a>
  <a href="https://github.com/recurso-dev/recurso/stargazers"><img src="https://img.shields.io/github/stars/recurso-dev/recurso?style=social" alt="GitHub Stars" /></a>
</p>

<p align="center">
  <a href="https://recurso.dev">Website</a> &middot;
  <a href="https://docs.recurso.dev">Docs</a> &middot;
  <a href="https://docs.recurso.dev/quickstart">Quickstart</a> &middot;
  <a href="https://github.com/recurso-dev/recurso/discussions">Community</a>
</p>

---

## Why Recurso?

Most billing platforms charge a percentage of your revenue and lock you into their ecosystem. Recurso is different.

- **Immutable Financial Ledger** — Double-entry accounting in PostgreSQL (the authoritative ledger), with an optional TigerBeetle mirror for high-throughput deployments. Every transaction is audit-ready from day one.
- **A real usage-based billing engine** — 8 aggregations (including a time-weighted average and a sandboxed custom expression), 7 charge models, and billing in arrears, in advance, or progressively past a threshold — all priced with exact rational math (no float drift), reconciled onto the ledger. Built for AI and API companies that bill by the call.
- **Multi-region tax compliance** — India GST (Place of Supply, HSN, TDS, e-invoicing via GSP) built in, not bolted on, plus EU VAT (reverse charge + VIES) and US sales tax with economic-nexus tracking.
- **AI-Powered Dunning** — Smart retry engine analyzes failure patterns and schedules retries with exponential backoff to maximize recovery.
- **No Success Tax** — Flat infrastructure cost. You don't pay more as your revenue grows.
- **Truly Open Source** — MIT licensed. Self-host, fork, extend. Full control over your billing data.

## Features

- Subscription lifecycle — trials (with expiry reminders and auto-conversion), plan changes with proration preview, pause/resume, add-ons, cancellation
- **Usage-based billing** — idempotent event ingestion; **8 aggregations** (count, sum, max, unique, latest, percentile, time-weighted average, and a sandboxed custom expression); **7 charge models** (per-unit, graduated, volume, package, percentage, graduated-percentage, dynamic); billed in arrears, in advance per event, or **progressively** past a threshold; dimensional (per-property) pricing; a pricing **simulator**; exact `big.Rat` math rounded once per line
- Automatic and one-off invoicing with jurisdiction-aware tax and printable/e-invoice-ready PDFs
- Hosted checkout — real card + ACH collection via the Stripe Payment Element, and UPI/cards/netbanking via Razorpay, with server-verified settlement
- Customer self-service portal — magic-link login (httpOnly-cookie session), card update (Stripe SetupIntent), UPI mandate re-authorization, invoice history
- Payments — multi-currency routing (INR → Razorpay, EUR/GBP mandates → GoCardless bank debit, US ACH → Stripe, others → Stripe or Adyen), prepaid wallets with auto-recharge, and **bring-your-own-gateway**: connect your own Stripe/Razorpay/GoCardless so recurring autopay (renewal, dunning, wallet) settles in *your* account
- Smart dunning — a multi-armed-bandit retry engine plus multi-channel recovery campaigns and recovery attribution, with a **Collections Intelligence** operator layer (worklist, analytics, manual controls, timing) on top
- Disputes & chargebacks — customer-raised invoice disputes with an accept/reject review workflow (accepting can issue a resolution credit note in one step), plus provider-webhook-driven gateway chargebacks with automatic ledger reversal
- Tax & e-invoicing — India GST (Place of Supply, HSN, TDS, IRN e-invoicing via GSP), EU VAT (reverse charge + VIES) with **EN 16931 / UBL e-invoice export**, US sales tax (TaxJar/Avalara) with economic-nexus threshold tracking
- Credit notes, refunds (Stripe/Razorpay lifecycle), coupons, gifts, referrals, quotes (CPQ)
- Double-entry ledger (PostgreSQL-authoritative, optional TigerBeetle mirror) with reconciliation, ASC 606 revenue recognition, and a month-end close pack
- **Multi-entity books** — multiple legal entities under one tenant, each with its own gapless invoice series, tax identity, per-entity ledger, and consolidated reporting
- Real-time FX-normalized MRR, churn scoring, entitlements, commitments, webhook delivery tracking, QuickBooks/Xero/NetSuite/Tally accounting sync, HubSpot CRM sync
- **Migration** — import customers, plans, and subscriptions from **Stripe** and **Chargebee**: a dry-run preview shows exactly what will import (create / link-existing-by-email / skip / conflict), then an idempotent commit brings subscriptions over in their *current* billing state — no re-billing — via a guided dashboard wizard (or the API)
- **Operations** — automated backups + a tested-restore runbook (with a double-entry ledger-balance integrity check), a Prometheus `/metrics` endpoint with a Grafana dashboard and alert rules, health checks with webhook alerting, and a public status page
- **MCP server** — agent-operable billing: drive the API from an LLM/agent over the Model Context Protocol, with RBAC-scoped tools
- **Ask AI** — natural-language analytics: ask billing questions in plain English, answered as read-only tenant-scoped queries, with auto-charts, CSV export, and a saved query history
- Platform — native auth (sessions, email verification, TOTP MFA, OAuth, SAML SSO), teams/roles, full OpenAPI 3.1, Node/Python/Go SDKs, row-level multi-tenancy, and in-dashboard contextual documentation links

## Project status

Recurso is **feature-complete and self-hostable today**, and the core money
paths — checkout, settlement, dunning, the ledger, trials, and proration — are
covered by end-to-end tests and have been verified against live Stripe and
Razorpay **test** keys. It is **pre-1.0 and pre-incorporation**: APIs may still
change, and a few things are deliberately gated on outside sign-off before you
rely on them in production:

- **India GST / e-invoicing** and the **US economic-nexus thresholds** are
  implemented but await review by a tax professional before being relied on for
  filing (the nexus dataset self-reports `dataset_certified: false` until then).
- **Payment webhooks** are wired but want verification on a deployed environment
  before real-money use.
- **SAML SSO** needs certification against a live identity provider.

Self-host it, fork it, build on it. If you're evaluating for production, start
with [Going to Production](https://docs.recurso.dev/going-to-production).

## Recurso vs. Alternatives

These are all capable, mature products — Chargebee in particular ships
usage-based billing, ML-powered dunning, connected gateways, and its own MCP
server. The table below sticks to the differences that are actually *structural*,
not feature checkboxes both sides can tick.

| | **Recurso** | **Chargebee** | **Stripe Billing** |
|---|---|---|---|
| **Pricing** | Free (self-hosted); usage-based on managed cloud | From ~$599/mo | 0.5%–0.8% of revenue |
| **Source code** | Open (MIT) — self-host, fork, extend | Closed | Closed |
| **Data ownership** | Full (runs on your infrastructure) | Vendor-hosted | Vendor-hosted |
| **Financial ledger** | Built-in double-entry (Postgres; optional TigerBeetle mirror), reconciled + ASC 606 rev-rec | Not a ledger | Not a ledger |
| **Usage-based billing** | 8 aggregations · 7 charge models · exact rational math · posted to the ledger | Yes (usage + hybrid) | Metered (basic) |
| **Dunning** | Bandit-retry engine + Collections operator layer | ML-powered retries | Basic retries |
| **India GST depth** | Place of Supply, HSN, TDS, IRN e-invoicing via GSP — built in | Partial | Limited |
| **Migrate in** | Self-serve importers from Stripe + Chargebee (preview → commit, no re-billing) | — | — |

## Architecture

```
Go (Gin) API  -->  PostgreSQL (state + authoritative double-entry ledger)
      |
      +--> Stripe / Razorpay / GoCardless / Adyen (payments)
      +--> Accounting sync (QuickBooks / Xero / NetSuite / Tally) + HubSpot CRM
      +--> Email notifications
      +--> Webhooks (inbound provider events + outbound delivery tracking)
      +--> Background workers (dunning, metering, e-invoice, settlement)
      +--> MCP server (agent-operable billing)
      +--> TigerBeetle (optional ledger mirror; PG stays authoritative)
```

Postgres is the source of truth for the ledger. TigerBeetle is an optional,
best-effort mirror that's written non-fatally when connected and only read as a
fallback — enable it (via `TIGERBEETLE_ADDRESS`) when ledger throughput warrants
it; the engine runs fully without it.

**Stack:** Go 1.25+ &middot; PostgreSQL &middot; TigerBeetle (optional) &middot; Hexagonal Architecture (Ports & Adapters)

## Quick Start

One command from clone to a populated dashboard — builds and starts the full stack (API, dashboard, PostgreSQL, TigerBeetle, Mailhog) and loads demo data:

```bash
git clone https://github.com/recurso-dev/recurso.git && cd recur-so
make demo
```

Then open the dashboard at `http://localhost:5173` and log in with API key `sk_test_12345`. Emails sent by the system land in Mailhog at `http://localhost:8025`.

### Step by step

Prefer to run pieces individually?

```bash
git clone https://github.com/recurso-dev/recurso.git && cd recur-so
make docker-up    # starts PostgreSQL + TigerBeetle
make run          # migrations apply automatically
```

The API is now running at `http://localhost:8080`. To start the React dashboard:

```bash
cd frontend && npm install && npm run dev
```

Want something to look at right away? Load demo data (a sample tenant with
plans, customers, subscriptions, and invoices), then log in to the dashboard
with API key `sk_test_12345`:

```bash
make seed   # WARNING: wipes existing data in the dev database
```

See the [Quickstart Guide](https://docs.recurso.dev/quickstart) for a full walkthrough.

## Deploy

Spin up a hosted instance (API + managed PostgreSQL) with one click:

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/recurso-dev/recurso)
[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/new)
[![Deploy to DigitalOcean](https://www.deploytodo.com/do-btn-blue.svg)](https://cloud.digitalocean.com/apps/new?repo=https://github.com/recurso-dev/recurso/tree/main)

- **Render** reads [`render.yaml`](render.yaml).
- **DigitalOcean** reads [`.do/app.yaml`](.do/app.yaml) (API service + a managed PostgreSQL database). After the first deploy, set `CORS_ORIGIN` to your dashboard's origin — without it the API rejects browser calls from the dashboard. For more than one instance, also attach a managed Redis/Valkey and set `REDIS_URL` (+ `REQUIRE_REDIS=true`) so the scheduler lock is shared; a single instance runs safely without it. Or, on a Droplet, run [`docker-compose.prod.yml`](docker-compose.prod.yml) (bundles Redis).
- **Railway** reads [`railway.json`](railway.json) to build the API; add a PostgreSQL plugin and set `DATABASE_URL=${{Postgres.DATABASE_URL}}` in the service variables.

These blueprints are provided as-is and have not been verified against live accounts — review sizes, regions, and image visibility before production use. For a self-hosted Docker Compose setup, see the [Self-Hosting Runbook](docs/deployment.md).

Building on Recurso? See [`examples/nextjs-starter`](examples/nextjs-starter) for a minimal Next.js SaaS starter.

## SDKs

Typed clients for the Recurso API live in their own repositories:

- **Node.js** ([recurso-node](https://github.com/recurso-dev/recurso-node)) — `npm install recurso`
- **Python** ([recurso-python](https://github.com/recurso-dev/recurso-python)) — packaged and tested, not yet on PyPI; `pip install ./recurso-python` from a checkout
- **Go** ([recurso-go](https://github.com/recurso-dev/recurso-go)) — `go get github.com/recurso-dev/recurso-go`

A [Postman collection](postman/) generated from the OpenAPI spec is also
included.

## Documentation

- [Getting Started](https://docs.recurso.dev/quickstart)
- [API Reference](https://docs.recurso.dev/api-reference/introduction)
- [Core Concepts](https://docs.recurso.dev/concepts)
- [Going to Production](https://docs.recurso.dev/going-to-production)
- [Self-Hosting Runbook](docs/deployment.md)

## Telemetry

Recurso can report anonymous, opt-in usage signals so we can measure how many
self-hosted instances reach their first real invoice. It is **off by default** —
with the default config there are zero network calls and nothing is written.

To opt in, set `TELEMETRY_OPTIN=true`. Only a random instance ID, version,
OS/arch, milestone events, and coarse bucketed counts (e.g. `1-9`, `10-99`,
`100+`) are ever sent — never amounts, names, emails, keys, or exact numbers.
See [docs/telemetry.md](docs/telemetry.md) for the full payloads, the never-sent
list, and how to point it at your own server to verify.

## Contributing

We welcome contributions of all kinds. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
make build        # compile the API
make test         # run unit tests
make test-e2e     # run end-to-end tests
```

## License

Recurso is [MIT licensed](LICENSE).
