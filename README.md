<p align="center">
  <img src="website/public/logo.svg" alt="Recurso" width="80" />
</p>

<h1 align="center">Recurso</h1>

<p align="center">
  Open-source billing engine for SaaS. Built with Go, PostgreSQL, and TigerBeetle.
</p>

<p align="center">
  <a href="https://github.com/swapnull-in/recur-so/actions"><img src="https://github.com/swapnull-in/recur-so/workflows/CI/badge.svg" alt="Build Status" /></a>
  <a href="https://github.com/swapnull-in/recur-so"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+" /></a>
  <a href="https://github.com/swapnull-in/recur-so/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg" alt="MIT License" /></a>
  <a href="https://github.com/swapnull-in/recur-so/stargazers"><img src="https://img.shields.io/github/stars/swapnull-in/recur-so?style=social" alt="GitHub Stars" /></a>
</p>

<p align="center">
  <a href="https://recurso.dev">Website</a> &middot;
  <a href="https://docs.recurso.dev">Docs</a> &middot;
  <a href="https://docs.recurso.dev/quickstart">Quickstart</a> &middot;
  <a href="https://github.com/swapnull-in/recur-so/discussions">Community</a>
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

- Subscription lifecycle management (trials, upgrades, cancellations, proration)
- Automatic invoice generation with tax calculation (GST/VAT)
- Credit notes and refund workflows
- Usage metering and metered billing
- Hosted checkout and customer self-service portal
- Multi-currency payment routing (Stripe, Razorpay)
- Webhook delivery and email notifications
- Real-time MRR and revenue analytics
- Multi-tenant architecture

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
git clone https://github.com/swapnull-in/recur-so.git && cd recur-so
make demo
```

Then open the dashboard at `http://localhost:5173` and log in with API key `sk_test_12345`. Emails sent by the system land in Mailhog at `http://localhost:8025`.

### Step by step

Prefer to run pieces individually?

```bash
git clone https://github.com/swapnull-in/recur-so.git && cd recur-so
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

## SDKs

- **Node.js** — a typed client for the Recurso API lives in [`sdk/node`](sdk/node).
  It is not yet published to npm; install it from the repository:

  ```bash
  git clone https://github.com/swapnull-in/recur-so.git
  npm install ./recur-so/sdk/node
  ```

## Documentation

- [Getting Started](https://docs.recurso.dev/quickstart)
- [API Reference](https://docs.recurso.dev/api-reference/introduction)
- [Core Concepts](https://docs.recurso.dev/concepts)
- [Going to Production](https://docs.recurso.dev/going-to-production)
- [Self-Hosting Runbook](docs/deployment.md)

## Contributing

We welcome contributions of all kinds. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
make build        # compile the API
make test         # run unit tests
make test-e2e     # run end-to-end tests
```

## License

Recurso is [MIT licensed](LICENSE).
