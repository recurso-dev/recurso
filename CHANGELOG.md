# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-07-06

### Added

- **Entitlement engine v1** — plan-level feature grants (booleans and
  limits), effective-entitlement resolution per customer (union across
  active and trialing subscriptions: any-true booleans, max limits), a
  single-query `GET /v1/entitlements/check` fast path for feature gating,
  Node SDK support, and plan-detail UI.
- **Recovery attribution** — invoices that collect after failed attempts
  are recorded with amount, attempts, strategy, and days-to-recover;
  `GET /v1/analytics/dunning/recovered` serves totals and a 12-month
  series; the dunning dashboard shows recovered revenue.
- **TigerBeetle reconciliation** — the ledger reconciler now enumerates
  TigerBeetle transfers (paginated) and reports missing/mismatched
  entries against PostgreSQL instead of skipping the comparison.
- **Health alerting** — `ALERT_WEBHOOK_URL` (JSON or Slack format) fires
  on component state transitions; SEV1 incident runbook in
  `docs/incident-runbook.md`.
- **Jurisdiction-aware tax** — GST computed from each tenant's own
  registration state, zero-rated exports (LUT-aware), EU VAT with B2B
  reverse charge, US sales-tax engine (0% stub, explicitly marked).
- **Real accounting sync** — QuickBooks/Xero OAuth token refresh with
  rotation, provider external-ID mapping (no more duplicate books
  entries), true update pushes (QBO SyncToken sparse updates, Xero
  upserts), QBO invoice line ItemRefs.
- **Gateway refunds** — refund credit notes call the real Stripe/Razorpay
  refund APIs with over-refund guards, honest manual-required states,
  and a Refunds-vs-Cash ledger reversal.
- **Ledger reconciliation endpoint** — `GET /v1/finance/reconciliation`
  plus a daily scheduled drift check.
- Consistent API error envelope (`{"error": {"code", "message"}}`),
  hardened idempotency (scoped keys, no 5xx caching), OpenAPI spec
  covering the full 113-path surface, `RATE_LIMIT_PER_MINUTE` knob,
  published performance numbers (docs/performance.md), verified
  backup/restore drill.

### Fixed

- Per-request bcrypt capped the API at ~126 req/s — verified-key cache
  takes authenticated reads to ~7,800 req/s (p99 27ms).
- Only one tenant could register per database (unique constraint on the
  always-empty hashed key column).
- Ledger postings failed for all API-created tenants/customers — AR and
  chart-of-accounts provisioning is now self-healing on first posting.
- Portal magic-link login could never match a customer; links are now
  actually emailed.
- Razorpay mandate revocation really revokes (customer-scoped token
  deletion) instead of silently succeeding.

### Known limitations

- US sales tax remains a 0%-rate stub pending a TaxJar/Avalara
  integration; EU VAT rates are a static table.
- Accounting sync re-pushes all mapped entities daily (no dirty
  tracking); Xero invoice lines don't yet reference synced items.
- Refund webhooks (charge.refunded / refund.processed) are not consumed,
  so pending refunds don't auto-advance.

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
