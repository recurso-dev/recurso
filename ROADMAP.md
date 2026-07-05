# Recurso Roadmap

Where Recurso is going, what's missing, and what order we're doing it in.
Status: living document. Items marked 🔒 need founder credentials/decisions
and can't be done by contributors.

**Shipped baseline (v0.1.x):** multi-tenant billing core (plans,
subscriptions, invoicing, coupons, usage), Stripe + Razorpay smart routing,
India GST/e-invoicing stack, dunning campaigns, customer portal, revenue
recognition on TigerBeetle, webhooks, quotes/credit notes, React dashboard,
Node SDK, subscriber migration importer, `make demo`, OpenAPI spec at
`/openapi.json`, hardened deployment (non-root containers, K8s RBAC/network
policies), CI with security scanning, tagged releases to GHCR.

---

## Track 1 — Trust & correctness (design-partner blockers)

The bar: a company can run real revenue through Recurso and an accountant
can sign off on the output.

- [x] **Real QuickBooks/Xero sync** — OAuth token refresh with rotation,
      per-connection adapter routing, invalid-grant deactivation, Xero
      tenant-ID resolution, reconnect-upsert.
- [x] **Accounting external-ID mapping** — provider IDs stored per
      connection and reused; customers sync before invoices and real
      CustomerRef/ContactID values are sent; create-once semantics prevent
      duplicates in the books.
- [ ] **Accounting sync updates** — create-once only today: local edits to
      already-synced entities aren't pushed (QBO SyncToken sparse update /
      Xero POST-with-ID), and QBO invoice lines lack ItemRef mapping.
- [x] **Razorpay mandate revocation** — real customer-scoped token
      deletion (idempotent), Razorpay customer id captured at creation and
      backfilled from the activation webhook; legacy mandates without a
      stored customer id fail loudly instead of fake-revoking.
- [x] **Jurisdiction-aware invoice tax** — TaxResolver selects GST / EU
      VAT / US sales tax by seller+buyer jurisdiction; GST uses the
      tenant's configured state (not hardcoded "TN"); Indian non-INR
      invoices are zero-rated exports with LUT-aware notes.
- [ ] **US sales tax provider** — the US engine is a 0%-rate stub; real US
      tax needs a TaxJar/Avalara integration (invoices are marked
      sales_tax_stub until then). EU VAT rates are a static table needing
      a maintenance story.
- [ ] **Proration invoice tax** — UpdateSubscription's proration invoices
      still hardcode TaxAmount 0; route them through the TaxResolver.
- [ ] **CA review of the GST/e-invoicing engine** 🔒 — external chartered
      accountant validates tax math and e-invoice output. Existential for
      an India-first billing product.
- [ ] **Gateway refunds end-to-end** — credit notes exist; verify/implement
      the actual Stripe/Razorpay refund API calls behind them.
- [ ] **Ledger reconciliation job** — scheduled PG↔TigerBeetle (and
      invoice↔ledger) drift detection with a report; dual-write failures
      are loud now, but drift needs a detector.
- [ ] **Idempotency coverage audit** — Redis idempotency store exists;
      verify every money-mutating endpoint honors idempotency keys.
- [ ] **Load test with published numbers** — invoices/minute, webhook
      throughput, p99s on a reference box; publish in docs.
- [ ] **Security posture page** — PCI scope statement (gateway tokens only,
      no PANs), key hashing, tenancy isolation model, disclosure policy.
- [ ] **Backup/restore drill** — actually restore from a pg_dump into a
      fresh stack and document the verified procedure.
- [ ] **Consistent API error envelope** — handlers currently return two
      error shapes (bare string / `{code,message}`); standardize.
- [ ] **TigerBeetle HA guidance** — multi-replica setup docs, or an honest
      "PG-only mode is the supported HA path" statement.

## Track 2 — Product depth

- [ ] Webhook delivery visibility in the dashboard (attempts, retries,
      dead-letter, manual redelivery).
- [ ] Plan-change proration UX in the dashboard (backend supports it).
- [ ] Trial flows end-to-end review (trialing status exists; verify
      conversion, expiry emails, dunning interplay).
- [ ] FX-normalized reporting (MRR across currencies uses real rates).
- [ ] Bulk operations in the importer (update mode, cancel-sync mode).
- [ ] Customer portal: payment-method update and invoice dispute flows.

## Track 3 — Developer experience & adoption

- [x] `make demo` one-command demo; seeded dataset with printed API key.
- [x] OpenAPI 3.1 served at `/openapi.json` / `/openapi.yaml`.
- [x] Dashboard getting-started checklist.
- [ ] **Publish `recurso-node` to npm** 🔒 (metadata is ready).
- [ ] **Make GHCR image public** 🔒 (one toggle in GitHub package settings).
- [ ] Wire Mintlify API playground to the served OpenAPI spec.
- [ ] Generated Python SDK from OpenAPI (then Go).
- [ ] Postman collection (export from OpenAPI, link in docs).
- [ ] One-click deploy buttons: Railway, Render, DigitalOcean.
- [ ] Devcontainer + Codespaces config.
- [ ] `examples/` — minimal Next.js SaaS starter wired to Recurso
      (checkout, webhook handler, portal link).
- [ ] `make dev` hot reload (air), pre-commit hook (gofmt + golangci-lint),
      GitHub issue/PR templates.
- [ ] Quickstart telemetry: none. Add opt-in, privacy-respecting usage ping
      (self-host) so activation is measurable.

## Track 4 — Recurso Cloud

- [ ] Real waitlist form on the website (replaces mailto) 🔒 (needs a
      form backend / inbox decision).
- [ ] Manual provisioning runbook: single-tenant instance per customer on
      the existing K8s manifests; onboard the first ~10 customers by hand.
- [ ] Recurso bills Recurso Cloud customers (dogfood the product).
- [ ] Status page + uptime monitoring.
- [ ] Control plane (instance lifecycle automation) — only after manual
      onboarding proves demand.

## Track 5 — Company & go-to-market 🔒 (founder-led)

- [ ] Incorporate (Delaware C-corp via Atlas if raising US venture; India
      Pvt Ltd if bootstrapping) + IP assignment to the company.
- [ ] Claim the `recur-so` GitHub org and transfer the repo; update module
      path and links in one sweep once done.
- [ ] Verify `recurso.dev` domain ownership; trademark search ("recurso" is
      a common Spanish/Portuguese word — check software-class collisions).
- [ ] 3–5 design partners running real billing (white-glove migration via
      `cmd/import`; free in exchange for feedback, logo, case study).
- [ ] Launch sequence: first design partner live → Show HN → Product Hunt
      → r/selfhosted → Indian SaaS communities (SaaSBoomi).
- [ ] Comparison/SEO pages: /vs/chargebee, /vs/stripe-billing, /vs/lago.
- [ ] Community: Discord, public roadmap (this file), monthly changelog post.
- [ ] Pricing finalization and the bootstrap-vs-preseed decision once live
      billing volume exists.

---

## Sequencing

**Next 30 days:** finish the two in-progress Track 1 items; non-INR tax at
invoice time; per-tenant tax config; refunds verification; npm + GHCR
publishing; waitlist form; incorporation decision + CA engaged; design-
partner outreach starts.

**Days 30–60:** first design partner migrated and live; load test +
security page published; ledger reconciliation; webhook visibility;
deploy buttons + examples; launch prep.

**Days 60–90:** public launches; manual cloud onboarding; Python SDK;
bootstrap-vs-raise decision with real usage data.
