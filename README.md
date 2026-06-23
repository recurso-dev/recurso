<p align="center">
  <img src="website/public/logo.svg" alt="Recurso" width="80" />
</p>

<h1 align="center">Recurso</h1>

<p align="center">
  Open-source billing engine for SaaS. Built with Go, PostgreSQL, and TigerBeetle.
</p>

<p align="center">
  <a href="https://github.com/recur-so/recurso/actions"><img src="https://github.com/recur-so/recurso/workflows/CI/badge.svg" alt="Build Status" /></a>
  <a href="https://github.com/recur-so/recurso"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white" alt="Go 1.23+" /></a>
  <a href="https://github.com/recur-so/recurso/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg" alt="MIT License" /></a>
  <a href="https://github.com/recur-so/recurso/stargazers"><img src="https://img.shields.io/github/stars/recur-so/recurso?style=social" alt="GitHub Stars" /></a>
</p>

<p align="center">
  <a href="https://recurso.dev">Website</a> &middot;
  <a href="https://docs.recurso.dev">Docs</a> &middot;
  <a href="https://docs.recurso.dev/getting-started/quickstart">Quickstart</a> &middot;
  <a href="https://github.com/recur-so/recurso/discussions">Community</a>
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

**Stack:** Go 1.23+ &middot; PostgreSQL &middot; TigerBeetle &middot; Hexagonal Architecture (Ports & Adapters)

## Quick Start

```bash
git clone https://github.com/recur-so/recurso.git && cd recurso
make docker-up    # starts PostgreSQL + TigerBeetle
make run          # migrations apply automatically
```

The API is now running at `http://localhost:8080`. See the [Quickstart Guide](https://docs.recurso.dev/getting-started/quickstart) for a full walkthrough.

## Documentation

- [Getting Started](https://docs.recurso.dev/getting-started/quickstart)
- [API Reference](https://docs.recurso.dev/api-reference/plans)
- [Architecture Guide](https://docs.recurso.dev/architecture)
- [Self-Hosting Guide](https://docs.recurso.dev/deployment)

## Contributing

We welcome contributions of all kinds. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
make build        # compile the API
make test         # run unit tests
make test-e2e     # run end-to-end tests
```

## License

Recurso is [MIT licensed](LICENSE).
