<p align="center">
  <img src="website/public/logo.svg" alt="Recurso" width="80" />
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

- **Immutable Financial Ledger** — Double-entry accounting powered by TigerBeetle. Every transaction is audit-ready from day one.
- **India-First Compliance** — Native GST, Place of Supply rules, HSN codes, TDS tracking, and e-invoicing readiness built in, not bolted on.
- **AI-Powered Dunning** — Smart retry engine analyzes failure patterns and schedules retries with exponential backoff to maximize recovery.
- **No Success Tax** — Flat infrastructure cost. You don't pay more as your revenue grows.
- **Truly Open Source** — MIT licensed. Self-host, fork, extend. Full control over your billing data.

## Features

- Subscription lifecycle — trials (with expiry reminders and auto-conversion), plan changes with proration preview, pause/resume, add-ons, cancellation
- Automatic and one-off invoicing with jurisdiction-aware tax and printable/e-invoice-ready PDFs
- Hosted checkout — real card + ACH collection via the Stripe Payment Element, and UPI/cards/netbanking via Razorpay, with server-verified settlement
- Customer self-service portal — magic-link login, card update (Stripe SetupIntent), UPI mandate re-authorization, invoice history
- Multi-currency payment routing (INR → Razorpay, others → Stripe) with saved-card off-session retries
- Smart dunning — a multi-armed-bandit retry engine plus multi-channel recovery campaigns and recovery attribution
- Tax — India GST (Place of Supply, HSN, e-invoicing via GSP), EU VAT (VIES), US sales tax (TaxJar) with economic-nexus threshold tracking
- Credit notes, refunds (Stripe/Razorpay lifecycle), coupons, gifts, referrals, quotes
- Double-entry ledger (PostgreSQL-authoritative, optional TigerBeetle mirror) with reconciliation and ASC 606 revenue recognition
- Usage metering, real-time FX-normalized MRR, churn scoring, webhook delivery tracking, QuickBooks/Xero sync
- Platform — native auth (sessions, TOTP MFA, OAuth, SAML SSO), teams/roles, full OpenAPI 3.1, Node/Python/Go SDKs, row-level multi-tenancy

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

| | **Recurso** | **Chargebee** | **Stripe Billing** |
|---|---|---|---|
| **Pricing** | Free (self-hosted) | From $599/mo | 0.5%–0.8% of revenue |
| **Source Code** | Open (MIT) | Closed | Closed |
| **India Compliance** | Native GST + e-invoicing | Partial | Limited |
| **Financial Ledger** | Immutable (TigerBeetle) | None | None |
| **Smart Dunning** | Built-in AI retries | Add-on | Basic |
| **Data Ownership** | Full (your infrastructure) | Vendor-hosted | Vendor-hosted |

## Architecture

```
Go (Gin) API  -->  PostgreSQL (state)  -->  TigerBeetle (ledger)
      |                                           |
      +--> Stripe / Razorpay (payments)           |
      +--> Email notifications                    |
      +--> Webhooks                               |
      +--> Background workers (dunning, metering) +
```

**Stack:** Go 1.25+ &middot; PostgreSQL &middot; TigerBeetle &middot; Hexagonal Architecture (Ports & Adapters)

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
[![Deploy to DigitalOcean](https://www.deploy.do/do-btn-blue.svg)](https://cloud.digitalocean.com/apps/new?repo=https://github.com/recurso-dev/recurso/tree/main)

- **Render** reads [`render.yaml`](render.yaml).
- **DigitalOcean** reads [`.do/app.yaml`](.do/app.yaml).
- **Railway** reads [`railway.json`](railway.json) to build the API; add a PostgreSQL plugin and set `DATABASE_URL=${{Postgres.DATABASE_URL}}` in the service variables.

These blueprints are provided as-is and have not been verified against live accounts — review sizes, regions, and image visibility before production use. For a self-hosted Docker Compose setup, see the [Self-Hosting Runbook](docs/deployment.md).

Building on Recurso? See [`examples/nextjs-starter`](examples/nextjs-starter) for a minimal Next.js SaaS starter.

## SDKs

Typed clients for the Recurso API live in the repo. They are packaged and
tested but **not yet published** to npm / PyPI — install from source for now:

- **Node.js** ([`sdk/node`](sdk/node)) — `npm install ./recur-so/sdk/node`
- **Python** ([`sdk/python`](sdk/python)) — `pip install ./recur-so/sdk/python`
- **Go** ([`sdk/go`](sdk/go)) — `import "github.com/recurso-dev/recurso/sdk/go"`

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
