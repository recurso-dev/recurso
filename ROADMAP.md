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
- [x] **Accounting sync updates** — mapped entities now update in place
      (QBO SyncToken sparse updates with stale-token retry, Xero
      POST-with-ID upserts, deleted-at-provider recreate), and QBO invoice
      lines carry real ItemRefs.
- [x] **Accounting sync efficiency** — changed-since dirty tracking
      (force flag on manual sync), and Xero lines link items by Code.
- [x] **Razorpay mandate revocation** — real customer-scoped token
      deletion (idempotent), Razorpay customer id captured at creation and
      backfilled from the activation webhook; legacy mandates without a
      stored customer id fail loudly instead of fake-revoking.
- [x] **Jurisdiction-aware invoice tax** — TaxResolver selects GST / EU
      VAT / US sales tax by seller+buyer jurisdiction; GST uses the
      tenant's configured state (not hardcoded "TN"); Indian non-INR
      invoices are zero-rated exports with LUT-aware notes.
- [x] **US sales tax provider** — TaxJar integration shipped behind
      TAXJAR_API_KEY (sandbox via TAXJAR_API_URL): real jurisdiction rates
      via POST /v2/taxes, cached in-memory 24h per buyer (state, zip);
      invoices are marked sales_tax with the provider named in the note.
      Provider errors at invoice time degrade to 0% with TaxType
      sales_tax_error (warn-logged) instead of blocking the invoice.
      Without a key the engine remains the honest 0% stub
      (sales_tax_stub). Follow-up: nexus configuration (which states the
      seller has nexus in) is NOT modeled — TaxJar's from-address/account
      nexus settings imply it, so rates silently assume the TaxJar account
      is configured correctly.
- [x] **EU VAT** — complete 27-state standard-rate table (dated as-of
      2026-01) + VIES VAT-number validation gating B2B reverse charge.
- [x] **Proration invoice tax** — plan-change charges route through the
      TaxResolver like any other invoice.
- [ ] **CA review of the GST/e-invoicing engine** 🔒 — external chartered
      accountant validates tax math and e-invoice output. Existential for
      an India-first billing product.
- [x] **Gateway refunds end-to-end** — credit notes of type "refund" call
      the real Stripe/Razorpay refund APIs with over-refund guards, honest
      manual_required/refund_failed states, gateway payment ids captured
      from payment webhooks, and a Refunds-vs-Cash ledger reversal.
- [x] **Refund webhook consumption** — pending refunds don't auto-advance
      to processed (charge.refunded / refund.processed not consumed); and
      mandate-debit payments store an order id, not the pay_* id refunds
      need.
- [x] **Ledger reconciliation job** — GET /v1/finance/reconciliation and a
      daily scheduler detect missing/mismatched invoice and payment
      postings and orphaned transactions (set-based SQL, capped listings).
      TigerBeetle comparison is explicitly skipped until its client gains
      an enumeration API.
- [x] **Idempotency coverage audit** — middleware now scopes keys by
      tenant+method+path (was tenant-only: reused keys replayed the wrong
      endpoint's response), never caches 5xx (transient failures were
      stored for 24h), covers all money-mutating v1 endpoints.
- [x] **Load test with published numbers** — docs/performance.md:
      7,814 req/s authenticated reads, ~34,600 invoices/min with full
      ledger posting, p99 under 70ms. The test itself found and fixed
      three production bugs: per-request bcrypt (~126 req/s cap),
      single-registration-per-database (unique empty key_value), and
      ledger postings failing for all API-created tenants (missing
      account provisioning).
- [x] **Security posture page** — docs/security.md covers PCI scope,
      credential handling, tenancy isolation, webhook verification, and a
      disclosure channel (security@recurso.dev inbox needs creating 🔒).
- [x] **Backup/restore drill** — performed against ~58k invoices:
      volumes destroyed, restored from pg_dump, counts identical, keys
      still authenticate. Documented in docs/performance.md.
- [x] **Consistent API error envelope** — every handler returns
      `{"error": {"code", "message"}}` (369 sites, snake_case taxonomy);
      OpenAPI Error schema matches; frontend parse sites updated.
- [x] **TigerBeetle HA guidance** — documented: stateless API replicas +
      Postgres as source of truth is the supported HA path; single-node
      TigerBeetle is an optional accelerator.

## Track 2 — Product depth

- [x] Webhook delivery visibility in the dashboard (attempts, retries,
      dead-letter, manual redelivery).
- [x] Plan-change proration UX in the dashboard (preview endpoint +
      change-plan flow with proration breakdown).
- [x] Trial flows end-to-end review (trialing status exists; verify
      conversion, expiry emails, dunning interplay).
- [x] FX-normalized reporting (MRR across currencies uses real rates).
- [x] Bulk operations in the importer (update mode, cancel-sync mode).
- [x] Customer portal: payment-method update and invoice dispute flows
      (admin dashboard dispute UI is a follow-up).
- [x] **Per-product HSN codes & itemized invoice tax** — plans/charges carry an
      HSN/SAC; invoices are itemized and each line is taxed at its own rate, so
      mixed-rate catalogs (SaaS 18% + e-books 5%) are correct and the e-invoice
      reports real per-line HSN + assessable value. **All 3 phases SHIPPED**:
      P1 line items + atomic persistence (000070), P2 catalog HSN + per-line
      rates (000071), P3 discount distribution / charges-as-lines (000072) /
      PDF per-line. Reconciliation-invariant tested; dashboard renders line
      items. Remaining is external-only: IRP-sandbox certification of the
      itemized e-invoice payload (founder/ops). `docs/design-per-product-hsn.md`.
- [ ] **US sales-tax nexus depth** — collect US tax only where the seller has
      nexus (declared physical/voluntary states; Phase 2 adds economic-nexus
      threshold tracking + alerts). Native, works with or without TaxJar. Phase 1
      (config + gating) is safe (no state-rule data); Phase 2's threshold dataset
      needs a US sales-tax pro to certify (compliance boundary). Design + Phase 1
      tasks: `docs/design-us-nexus.md`.

### Authentication & Access (dashboard)

Real per-user dashboard accounts, shipped in three phases (July 2026) — all
**native Go on Postgres, no external auth dependency** (Supabase/Clerk were
evaluated and rejected to preserve the self-host / data-sovereignty story).
Coexists with the existing API-key auth via a dual middleware, so machine
access and the demo key never broke.

- [x] **Phase 1** — user accounts, sessions, roles (owner/admin/member), team
      management (`/v1/users`), email/password login. Dual-auth middleware
      (session cookie OR Bearer API key) preserves the `tenant_id` contract for
      all ~100 existing endpoints; regression-tested.
- [x] **Phase 2** — password reset (SMTP, single-use token, kills sessions),
      TOTP two-factor auth (two-step login, backup codes), active-session
      management (list / revoke one / log out everywhere else).
- [x] **Phase 3** — OAuth social login (Google + GitHub, CSRF state + PKCE,
      `email_verified` enforced, feature-flagged by env); SAML SSO foundation
      (per-tenant IdP config, SP metadata/login/ACS via `crewjam/saml`,
      assertion email → existing user).
- [ ] **Certify SAML against a live IdP** — the ACS signature round-trip
      (Okta / Azure AD / Google Workspace) can't be proven without a real IdP.
      Config, role-gating, tenant-scoping, SP metadata, SP-initiated redirect,
      and email→user mapping are unit-tested; certify before an enterprise sale.
- [x] Account-security docs (password reset, 2FA, sessions) published.

## Track 3 — Developer experience & adoption

- [x] `make demo` one-command demo; seeded dataset with printed API key.
- [x] OpenAPI 3.1 served at `/openapi.json` / `/openapi.yaml`.
- [x] Dashboard getting-started checklist.
- [ ] **Publish `recurso-node` to npm** 🔒 (metadata is ready).
- [ ] **Make GHCR image public** 🔒 (one toggle in GitHub package settings).
- [x] OpenAPI spec covers the full surface: 109 paths / 137 operations,
      zero missing registered routes, redocly-clean, minimum-path test.
- [x] Wire Mintlify API playground to the served OpenAPI spec.
- [x] Generated Python SDK from OpenAPI. Hand-crafted Go SDK
      (`github.com/swapnull-in/recurso-go`, `sdk/go/`) — 18 resources / 68
      methods, stdlib-only, httptest-tested. (Node, Python, Go all shipped.)
- [x] Postman collection (generated from OpenAPI into `postman/` — 33
      tag-grouped folders + an environment file; regen command + import steps
      in `postman/README.md`).
- [x] One-click deploy buttons: Railway, Render, DigitalOcean.
- [x] Devcontainer + Codespaces config.
- [x] `examples/` — minimal Next.js SaaS starter wired to Recurso
      (checkout, webhook handler, portal link).
- [x] `make dev` hot reload (air, auto-installed), pre-commit hook (gofmt +
      golangci-lint on staged files, via `make hooks`), GitHub issue/PR
      templates.
- [x] Quickstart telemetry: none. Add opt-in, privacy-respecting usage ping
      (self-host) so activation is measurable.

## Track 4 — Recurso Cloud

- [ ] Real waitlist form on the website (replaces mailto) 🔒 (needs a
      form backend / inbox decision).
- [x] Manual provisioning runbook: single-tenant instance per customer;
      onboard the first ~10 by hand. Runbook written
      (docs/cloud-provisioning-runbook.md — Docker Compose per VM); the
      by-hand onboarding is the remaining work.
- [x] Recurso bills Recurso Cloud customers (dogfood the product) — the Cloud
      catalog maps to Recurso primitives ($299 base plan + add-on plans + usage
      dimensions for overage); `scripts/cloud-billing-setup.sh` provisions it and
      `docs/cloud-dogfooding-runbook.md` documents the onboarding + monthly
      usage→overage→invoice flow. Verified end-to-end against a live instance
      (fixed a latent bug: the add-charge/advance-invoice endpoints never
      injected `tenant_id` into the request context). 🔒 remaining: live payment
      keys, final add-on prices, and the production Cloud tenant.
- [ ] Status page + uptime monitoring. Approach chosen and documented
      (docs/status-page.md — hosted Better Stack now, self-host Uptime Kuma
      later); setup still to execute.
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
- [x] Comparison/SEO pages: /vs/chargebee, /vs/stripe-billing, /vs/lago.
- [ ] Community: Discord, public roadmap (this file), monthly changelog post.
- [ ] Pricing finalization and the bootstrap-vs-preseed decision once live
      billing volume exists.



### From the strategic market evaluation (July 2026)

- [ ] **Licensing/monetization decision** 🔒 — the MIT license maximizes
      enterprise adoption (vs Lago/Flexprice's AGPLv3) but permits cloud
      providers to offer managed Recurso without contributing. Options on
      the table: stay MIT + compete on the cloud-trained dunning moat and
      operations; open-core with proprietary enterprise features
      (SAML/SSO, multi-region HA, premium tax); or SSPL-style relicense
      (community cost). Founder decision before significant cloud revenue.
- [x] **EU VAT provider** — 27-state rate table + VIES validation shipped.
- [x] **Comparison pages vs open-source peers** — /vs/lago, /vs/flexprice,
      /vs/killbill (MIT-vs-AGPLv3 and the TigerBeetle ledger are the
      differentiators the evaluation highlights).
- Validated by the evaluation and already true in code: RBI e-mandate
  24h pre-debit notifications (internal/scheduler/precharge.go), DPDP
  consent tracking, GST/IRP automation, TaxJar (shipped v0.2.3 after
  the evaluation was written).

## Track 6 — Toward the Revenue Operating System

Synthesis of external strategic analysis (July 2026). The long-term vision:
billing engine → revenue operating system. Triaged by tense — build-now
items feed Tracks 1–3; the rest are recorded so the vision doesn't get
re-derived every quarter.

**Build now (prerequisites for the monetization wedge):**
- [x] **Recovery attribution** — recovered_payments records amount,
      attempts, strategy, and days-to-recover whenever a previously-failed
      invoice collects; GET /v1/analytics/dunning/recovered serves totals
      and a 12-month series; DunningDashboard shows recovered revenue.
- [x] **Entitlement engine v1** — plan-level boolean/limit grants,
      union resolution across active+trialing subscriptions (any-true /
      max-limit), single-query check endpoint, Node SDK resource, plan
      detail UI.
- [x] **TigerBeetle reconciliation** — paginated transfer enumeration;
      the reconciler now reports missing-in-TB / missing-in-PG / amount
      mismatches with tb_compared=true when connected.
- [x] **Incident runbook + alerting** — docs/incident-runbook.md (SEV
      definitions, stop-the-bleeding, triage table, honest gaps) and
      ALERT_WEBHOOK_URL transition-based health alerting.

**Next (Phase 2, after design partners):**
- Usage platform depth (seats, storage, active-user dimensions; credits)
- CRM integrations (HubSpot first), workflow automation on billing events
- Multi-product catalog: add-ons, bundles, one-time services
- CPQ: ramp pricing and custom contract schedules for enterprise quotes
- Localized payment methods via Stripe rails (SEPA, iDEAL) and UPI depth
- Developer platform: CLI, MCP server, Terraform provider, generated
  Python/Go SDKs (now unblocked by the complete OpenAPI spec)

**Later (Phase 3–4, needs scale/data):**
- Revenue intelligence (leak detection, forecasting, anomalies)
- AI pricing assistant, price A/B testing
- Identity platform (SAML/SCIM/RBAC) as the enterprise tier
- Marketplace/plugins, embedded finance

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
