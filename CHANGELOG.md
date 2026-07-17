# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2026-07-17

The India release: the full statutory lifecycle (invoice → IRN → GSTR-1/3B →
TDS) from a billing engine you self-host, plus the correctness and security
hardening from a deep multi-agent review of the money paths.

### Added

- **GST returns, return-ready** — `GET /v1/india/gstr1` and
  `GET /v1/india/gstr3b` assemble both returns from the period's finalized
  invoices and credit notes: GSTR-1 with B2B/B2CS/CDNR sections and the HSN
  rollup; GSTR-3B as the net summary (Table 3.1(a) net of credit notes,
  Table 3.2 inter-state unregistered by place of supply, purchase-side
  sections as explicit zeros for the CA). Each response carries a
  `gov_schema` object in the official GSTN JSON shape — government field
  names, amounts in rupees — ready to validate with the Returns Offline Tool
  (ENG-203).
- **Self-hosted data residency** — `RESIDENCY_MODE=self_hosted` hard-disables
  every optional third-party egress: telemetry (even when opted in),
  QuickBooks/Xero sync (the connect flow and existing connections), and the
  TaxJar API. Financial data leaves the deployment only through the payment
  gateways, GSP, SMTP, and webhook endpoints the operator configures.
  `docs/india-data-residency.md` states the guarantee for security reviews;
  every enforcement point is unit-tested.
- **TDS record-on-receipts** — offline payments accept `tds_amount` for the
  portion a B2B customer withheld at source. It counts toward settling the
  invoice, accumulates on `invoices.tds_amount`, and posts
  DR TDS Receivable / CR AR in the ledger with the cash leg net of TDS.
- **Provable-ledger auditor outputs** — trial balance, deferred-revenue
  rollforward, GL export, and the revenue-recognition waterfall, wired into
  the dashboard's Finance sidebar (ENG-192, ENG-194); deeper analytics
  endpoints (MRR waterfall, invoice aging, unit economics, revenue by plan
  and geography).
- **Real Redis infrastructure** — distributed locking and idempotency backed
  by Redis, with `REQUIRE_REDIS` to fail closed on multi-instance
  deployments; the lock's mutual exclusion is proven by test (ENG-161,
  ENG-193).
- **Inbound webhook idempotency** for Stripe and Razorpay events (ENG-162),
  and **team email invites** where teammates set their own password
  (ENG-196).
- **Official Go SDK** — hand-crafted, stdlib-only `recurso-go` covering the
  full API surface.
- **Demo-data seeder** (`cmd/demo_seed`) — additive, tenant-scoped, with
  `--reset`, covering rev-rec, referrals, gifts, and reconciling ledger
  postings.

### Changed

- **SDKs moved to standalone repositories** — `recurso-go`, `recurso-node`
  (1.2.0, responses/requests typed from the OpenAPI spec), and
  `recurso-python` (1.1.0, regenerated; unexpected API statuses now raise
  instead of returning `None`). The in-repo `sdk/` directory and its CI job
  are gone; install/publishing docs point at the new repos.
- Ledger documentation now states plainly: PostgreSQL is authoritative,
  TigerBeetle is an optional mirror.
- Internal business material (pitch deck, review reports) moved out of the
  repository.

### Fixed

- **Billing correctness family (ENG-140–154)** — revenue is deferred NET of
  GST and unwound on cancel/refund/downgrade; signed double-entry balances;
  atomic trial-conversion billing; downgrade credits reverse GST and persist
  as adjustment credit notes applied at billing time; account credit is a
  real liability; cash postings are net of applied credit; first-period
  proration; no phantom revenue on UPI-Autopay debits.
- **Month-end billing dates** — interval math clamps instead of overflowing
  (Jan 31 + 1 month = Feb 28, never Mar 3), and anchored subscriptions
  restore their day in long months (Feb 28 → Mar 31) instead of sticking at
  28.
- **Atomic-claim sweep (ENG-160–200, PHASE2)** — one-shot semantics enforced
  by conditional updates everywhere a retry or race could double-fire:
  mandate debits (with the idempotency key proven to reach the gateway),
  trial activation, dunning steps and bandit weights, e-invoice IRN retries,
  gift redemption, quote→invoice conversion, cancel-flow retention offers,
  virtual-account credits, refund over-issue, and idempotency-key claims
  themselves; downgrade credit + plan flip commit in one transaction; the
  IRN is registered only after the invoice durably commits.
- **Graceful shutdown** — background workers drain under a bounded
  WaitGroup; all schedulers stop concurrently and idempotently (a double
  Stop previously deadlocked the exiting process); in-flight webhook
  deliveries respect cancellation.
- **Tax fixes** — CGST/SGST always split into equal halves; non-EU B2C
  digital exports are zero-rated instead of falling back to domestic VAT;
  export exemption requires an actual cross-border supply (GB→GB regression);
  single-digit GST state codes resolve; GST/VAT rounds instead of
  truncating.
- **Tenant isolation (ENG-157–169)** — repository-level tenant scoping for
  handler-reachable mutations, invoice settlement, and mandate writes;
  offline payments settle an invoice only when covered and only for the
  matching customer/currency.

### Security

- Stripe webhook verification fails closed when the secret is unset;
  outbound webhook URLs are SSRF-guarded (ENG-175, ENG-177).
- Portal session tokens hashed at rest; auth tokens atomically single-use;
  TOTP codes can't be replayed; per-account login lockout (ENG-145,
  ENG-151, ENG-176).
- Owner role protected from admin privilege escalation; API-key creation
  gated; multiple cross-tenant IDORs closed (ENG-164, ENG-165, ENG-178,
  ENG-160).
- O(1) API-key lookup replaces the bcrypt-per-key scan; trusted proxies
  configured so X-Forwarded-For can't bypass rate limits; `/health` no
  longer leaks connection errors (ENG-174, ENG-197, ENG-198).

## [0.4.0] - 2026-07-10

### Added

- **Real hosted checkout** — server-verified card/ACH collection via the Stripe
  Payment Element and UPI/cards/netbanking via Razorpay Checkout, smart-routed by
  currency (INR → Razorpay, else Stripe) (ENG-4).
- **Customer self-service portal** — magic-link login, card update via Stripe
  SetupIntent, UPI mandate re-authorization, and invoice history; payment-recovery
  emails deep-link straight into it (ENG-5).
- **Smart dunning depth** — off-session saved-card retries, retries settled
  through the double-entry ledger, and recovery deep-links (ENG-5).
- **US economic-nexus tracking** — per-state year-to-date sales/transaction
  tracking that auto-establishes economic nexus when a threshold is crossed;
  `GET /v1/settings/tax/nexus/status` reports proximity. Dataset seeded
  uncertified pending professional review (ENG-16).
- **Dashboard redesign** — shadcn foundation with stone design tokens, a
  monospace Money signature, a ⌘K command palette, and a Test-mode chip
  (ENG-135, ENG-136).
- **Cloud waitlist** — API-backed early-access signup (`POST /waitlist`) (ENG-12).
- Jurisdiction-aware invoice PDFs wired to real invoices.

### Changed

- **API keys are now mode-gated (test vs. live).** Newly created keys are issued
  as `rsk_test_` (test) or `rsk_live_` (live), and a key's mode must match the
  server's `gateway_mode` (reported at `GET /version`): a test key is accepted on
  a `none`/`test` server, a live key on a `live` server. Mismatches return
  `401 key_mode_mismatch`, so a test key can never move real money. New accounts
  get a test key by default; mint a live key with
  `POST /v1/developer/keys {"mode":"live"}`.

  **Breaking:** existing generated `sk_live_` keys are grandfathered as live and
  keep working against a live-gateway server — but a request from such a key to a
  **non-live** server (the mock `none` gateway or gateway test mode) now returns
  `401 key_mode_mismatch`. If you develop against a non-live instance with an old
  `sk_live_` key, either mint an `rsk_test_` key (Settings → API Keys, or the
  developer-keys endpoint) or configure live gateway keys. The demo key
  `sk_test_12345` is unaffected — it is grandfathered as a test key.

### Fixed

- **Tenant-context audit** — features that never worked against a real
  multi-tenant database: trial conversion (ENG-3), plan changes that silently
  never persisted (ENG-6), and four more background paths (ENG-134), plus
  `tenant_id` propagation across 11 handlers and silently-zero consolidated MRR.
- **Background-job robustness** — NULL / timestamptz scans that aborted the
  nexus, churn, pre-charge, dunning-retry, and accounting sweeps against real
  data (ENG-143).
- **Dashboard** — page crashes (Security, Referrals/Gifts on empty data), sparse
  or broken detail panels (Customer, Quote, Credit Note, Coupon), and
  non-functional create buttons/routes.
- **Live-key payment fixes** — Razorpay UPI-Autopay registration payload, Stripe
  inactive method types, checkout failure states, and invoice PDF currency
  symbols.

### Security

- Removed unauthenticated portal routes that exposed cross-tenant PII (ENG-139).
- GenAI text-to-SQL tenant isolation is now enforced by the database (dedicated
  schema + read-only role), not the prompt (ENG-137).
- Tenant-gated the invoice PDF route; patched dependency CVEs flagged by
  govulncheck / Trivy (jwt/v4 4.5.2, rollup 4.62.2, vite 7.3.6, x/oauth2 0.27.0,
  goxmldsig 1.6.0, Go 1.25.12).

## [0.3.0] - 2026-07-07

### Added

- **Opt-in anonymous telemetry** — measures self-hosted activation (how
  many instances reach their first real invoice) so the project can see
  adoption without a hosted service. Strictly opt-in (TELEMETRY_OPTIN=
  true); default OFF means zero network calls and zero data written.
  When enabled: a random instance ID, once-ever milestone events, and a
  24h heartbeat with range-bucketed counts — never amounts, names,
  emails, keys, IDs, or exact numbers. docs/telemetry.md documents every
  payload and the one-line opt-out.
- **One-click deploy** — Render, Railway, and DigitalOcean templates
  with README deploy buttons; a devcontainer for Codespaces; and a real
  Next.js starter in examples/ (pricing, signup, usage with entitlement
  headroom, feature gating) that builds and runs against `make demo`.
- **Competitor comparison pages** — honest /vs/ pages for Lago,
  Flexprice, Kill Bill, Chargebee, and Stripe Billing.
- **Cloud operations groundwork** — a keystroke-level provisioning
  runbook for the first manually-provisioned cloud customers, and a
  status-page plan.

## [0.2.3] - 2026-07-07

### Added

- **US sales tax via TaxJar** — set TAXJAR_API_KEY and US invoices get
  live jurisdiction rates (24h per-location cache, retry-once, invoices
  never blocked by lookup failures: they ship at 0% flagged
  sales_tax_error for review). Unconfigured stays the honest
  sales_tax_stub. Nexus-state configuration is not yet modeled.
- **Usage platform depth** — GET /v1/usage (time-bucketed windows),
  GET /v1/subscriptions/{id}/usage (current-period and lifetime usage
  per dimension with entitlement limit and remaining headroom),
  GET /v1/usage/dimensions catalog.
- **Accounting sync efficiency** — changed-since dirty tracking (daily
  sync pushes only what changed; manual sync forces a full re-push);
  Xero invoice lines now reference synced Items by code.
- SDKs: Node usage.query/usage.dimensions/subscriptions.usage (78
  tests); Python regenerated (12-check smoke). New docs: US sales tax
  compliance page, usage windowing and entitlement-headroom patterns,
  incremental-sync semantics.

### Fixed

- **Security**: POST /v1/usage/events accepted events against any
  tenant's subscription; it now enforces subscription ownership and
  customer match.
- Stale usage docs described fictional request fields; corrected to the
  real API.

## [0.2.2] - 2026-07-06

### Added

- **Webhook delivery tracking** — GET /v1/events/{id}/deliveries and
  GET /v1/webhooks/{id}/deliveries expose per-delivery status, attempts,
  response codes and errors; POST /v1/events/{id}/redeliver re-queues
  delivery idempotently.
- **FX-normalized MRR** — tenant and org MRR convert across currencies
  to a reporting currency (tenant BaseCurrency, else REPORTING_CURRENCY)
  with the rates, source (live / static-fallback), and timestamp in the
  response; unconvertible currencies are flagged, never silently mixed.
- **Refund lifecycle completion** — Stripe and Razorpay refund webhooks
  advance pending refunds to processed or refund_failed (with the
  gateway's reason); mandate-collected invoices now capture a
  refundable payment id.
- SDKs: Node gains delivery/redeliver/MRR methods (17 resources, 64
  methods, 75 tests); Python regenerated with the new endpoints
  (10-check smoke). Docs updated across API reference and guides.

### Fixed

- Payment-success webhook processing (Stripe and Razorpay) failed with
  a tenant-context error and returned 500 before recording anything —
  handlers now resolve the invoice's tenant themselves.
- The dashboard MRR tile always showed $0 (read a field the API never
  returned).
- Delivery worker retries no longer erase the failing HTTP status code.
- The new-customer form's oversized styling, stale state on country
  switch, and literal \n placeholder.

## [0.2.1] - 2026-07-06

### Added

- **Python SDK** (`sdk/python`) — generated from the served OpenAPI spec:
  typed models, sync and async clients, 32 API modules, quickstart README,
  no-network smoke test.
- **Node SDK test suite** — 71 tests with 100% client coverage, including
  a reflection guard that fails when a method ships untested; SDK builds
  and tests now gate CI.
- **Dashboard**: entitlements editing on plan detail (PUT-replace
  semantics with validation), a Finance > Reconciliation page (summary
  cards, TigerBeetle comparison badge, discrepancy table, "Books
  balanced" zero state), and event payload/type visibility on the
  Developers page.
- Docs: entitlements guide + API reference, recovered-revenue and
  reconciliation pages, performance numbers, error-envelope taxonomy,
  and an interactive API playground wired to the full spec.

### Fixed

- The create-API-key dialog displayed an empty key (read the wrong
  response field); key listings now show real prefix/type/status/date.
- Removed remaining mock content from the dashboard (fake usage tiers,
  dead buttons, mock pagination).
- OpenAPI corrections: PUT /v1/plans/{id}/entitlements takes a bare
  JSON array; reconciliation documents the TigerBeetle comparison.

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

[0.1.0]: https://github.com/recurso-dev/recurso/releases/tag/v0.1.0
