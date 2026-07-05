# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2026-07-05

### Added

- **Subscriber migration tool** (`cmd/import`) — import plans, customers,
  and subscriptions from another billing system (Stripe Billing, Chargebee,
  spreadsheets) via JSON or CSV. Writes directly to the database without
  generating invoices or calling payment gateways, so migrated customers
  are never double-billed mid-cycle; the renewal worker issues each
  subscription's next invoice at its imported `current_period_end`.
  Idempotent (plans by code, customers by email, subscriptions by
  `external_id`) with a `-dry-run` mode. See `cmd/import/example.json`
  and the "Migrating an existing subscriber base" section of
  `docs/deployment.md`.
- **`make seed`** — one-command demo dataset (tenant, plans, customers,
  subscriptions, invoices) for first-time exploration; prints the demo
  dashboard API key when done. Destructive: wipes the target database.

## [0.1.0] - 2026-07-04

First public release of Recurso, an open-source, self-hosted subscription
billing engine built with Go, PostgreSQL, and TigerBeetle.

### Added

- **Multi-tenant billing core** — plans, subscriptions (trials, upgrades,
  downgrades, cancellations, proration), invoicing, coupons, and usage-based
  (metered) billing.
- **Multi-currency payments** — Stripe and Razorpay integrations with smart
  gateway routing per currency, plus FX rate handling.
- **India compliance stack** — GST calculation with Place of Supply rules,
  HSN codes, TDS tracking, and e-invoicing (IRN/GSP) workflows.
- **Dunning** — configurable dunning campaigns and a smart retry engine with
  exponential backoff to maximize payment recovery.
- **Customer-facing surfaces** — hosted checkout pages and a customer
  self-service portal.
- **Revenue recognition** — deferred revenue schedules backed by an
  immutable double-entry ledger on TigerBeetle (optional component).
- **Billing documents** — quotes and credit notes with refund workflows.
- **Integrations** — outbound webhook delivery and email notifications.
- **Node.js SDK** (`sdk/node`) — typed client for the Recurso API.
- **React dashboard** (`frontend/`) — admin UI for plans, customers,
  subscriptions, invoices, and analytics (MRR, revenue).
- **Operations** — Dockerfile and docker-compose stack, Kubernetes manifests,
  CI pipeline, and versioned release builds (`/version` endpoint, ldflags
  version stamping).

### Known limitations

- Accounting sync (QuickBooks/Xero) runs in mock mode only; no real
  provider API calls are made yet.
- Razorpay mandate revocation is not implemented.
- TigerBeetle runs as a single node; no replication/HA setup is provided.
- The Node.js SDK is not yet published to npm; install it from this
  repository (`sdk/node`).

[0.1.0]: https://github.com/swapnull-in/recur-so/releases/tag/v0.1.0
