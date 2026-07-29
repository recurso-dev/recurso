# Recurso — Startup Readiness Audit (Phase 1)

> **Purpose.** This is the gate before any Phase-2 code. It answers one question:
> *what stands between "impressive engineering project" and "a customer can
> discover, buy, adopt, and trust this"?* Scores and gaps below are **grounded in
> the actual repo** (not the pitch), and every "missing" item was verified by
> searching the code, not assumed.
>
> **Headline finding.** The *product* is largely built and largely good. The
> *business surface* — the path a stranger takes from landing page to paying,
> live, monitored customer — is where the holes are. Almost none of the top gaps
> are "more billing features." They are: **buy path, migration, trust, and
> operations.**
>
> Assessed 2026-07-29. Supersedes the scoring section of `PRODUCTION_READINESS.md`
> for startup (vs. pure engineering) readiness.

---

## 1. Production Readiness Score (0–10)

Scored against *"can a paying customer rely on this in production,"* not *"is the
code nice."* Anything ≤5 is a first-customer blocker.

| # | Area | Score | Grounded rationale → path to 10 |
|---|---|---:|---|
| 1 | Architecture | **9** | Clean Go hexagonal core; PG-authoritative double-entry ledger + TigerBeetle mirror + invariant harness. To 10: split the largest handlers/services. |
| 2 | Security | **8** | httpOnly sessions, TOTP MFA, SAML SSO, RBAC, row-level tenancy, rate-limit middleware, Trivy CVE gate. To 10: no **email verification** (see §2 Critical), no automated authz-matrix/fuzz tests. |
| 3 | Payments | **6** | Multi-gateway built (Stripe/Razorpay/GoCardless/Adyen) + BYO creds + webhooks + settlement/chargeback. **But no single real transaction has been proven end-to-end** (creds pending). Unproven money movement = 6, not 9. |
| 4 | Accounting | **9** | Double-entry GL, rev-rec, reconciler, invariant harness gates every money path. To 10: broaden failure-injection. |
| 5 | Billing | **9** | Subscriptions, usage/metering, 7 charge models, proration, dunning, credits, quotes, gifts. Feature-complete. |
| 6 | Developer Experience | **8** | 3 SDKs (Go/Node/Python) synced, OpenAPI 3.1 + drift gate, Postman, MCP server. To 10: single-command full-stack run + coverage gate. |
| 7 | Customer Experience (product) | **9** | Premium dashboard, empty/loading/error states, contextual docs. |
| 8 | Operations | **5** | `/health` component checks + alert webhook exist. No runbook coverage for on-call, no metrics dashboards, no incident tooling beyond a webhook. |
| 9 | Compliance | **6** | GST + e-invoice (IRP) + UPI built; EU EN-16931/UBL is **export-only (no Peppol transmission)**; no SOC 2 / PCI attestation story. |
| 10 | Testing | **7.5** | Strong backend + invariant harness; frontend net materially raised this session (340+ tests). To 10: coverage gate + remaining slide-overs. |
| 11 | Scalability | **7** | Row-level multi-tenancy, per-entity ledgers, batched claim-based workers. Several list endpoints unbounded (e.g. `ListInvoices`). |
| 12 | Deployment | **8** | Docker, docker-compose (dev/prod/demo), **k8s manifests**, `railway.json`, `render.yaml`, Cloud Build → Cloud Run, CF Workers. Excellent for self-host. |
| 13 | Documentation | **9** | Mintlify (dashboard/setup/SDK guides), ADRs, in-app doc links. |
| 14 | Monitoring | **4→7** | Health + alert webhook **plus a Prometheus `/metrics` endpoint** (HTTP request/latency + runtime gauges), a Grafana dashboard, and alert rules (`deploy/observability/`). To 10: error tracking (Sentry, guarded follow-up) + tracing + log aggregation. |
| 15 | Self Hosting | **9** | One-command compose, k8s, one-click Railway/Render. Genuinely strong. |
| 16 | Cloud SaaS | **2** | **Recurso does not bill itself.** No self-serve tenant provisioning, no trial, no paywall, no per-tenant metering/billing. A customer cannot buy the hosted product. This is the single biggest business gap. |
| 17 | Migration | **1→(building)** | Was **nothing**; the `import` at repo root is a stray (gitignored, untracked) 8 MB build artifact, not tooling. **Increment 1 shipped**: a Stripe-export **dry-run preview** engine + `POST /v1/import/stripe/preview` (maps customers/plans/subscriptions/PMs, links existing by email, flags conflicts/unsupported). Idempotent commit + Chargebee/RevenueCat still to come. |
| 18 | Import/Export | **3** | CSV *exports* exist (invoices, reports). No bulk *import*; no full-account/GDPR export. |
| 19 | API | **9** | OpenAPI 3.1, drift-gated, versioned, keys + RBAC. |
| 20 | SDKs | **8** | Go/Node/Python, synced to API. To 10: publish creds + a language-idiomatic quickstart each. |
| 21 | Admin | **8** | Rich admin dashboard, audit log, RBAC, entities. |
| 22 | Analytics | **8** | MRR waterfall, unit economics, churn, revenue-by-*; premium charts. |
| 23 | Support | **2** | No in-app support, help widget, ticketing, email queue, or SLA. A stuck trial user has nowhere to go. |
| 24 | Disaster Recovery | **3** | No documented RTO/RPO, no tested restore, no failover story. |
| 25 | Backups | **3** | PG is authoritative but no automated backup/PITR/verification in the repo. For a money system this is a red line. |
| 26 | Observability | **4→7** | Prometheus `/metrics` (request/latency/runtime) + Grafana dashboard + alert rules now shipped. To 10: error tracking (Sentry) + distributed tracing. |

**Weighted read:** the *engineering median* is ~8. The *business-surface* items
(16 Cloud SaaS, 17 Migration, 23 Support, 24 DR, 25 Backups, 14/26 Monitoring/Obs)
sit at 1–4 and are exactly the things a buyer checks before wiring you their
revenue. **The gap is not quality; it's the last mile to "buyable and trustworthy."**

---

## 2. Missing Features (customer-*required* only — no nice-to-haves)

### 🔴 Critical — blocks first paying customer
1. **Email verification on signup.** No verification-token flow exists. Table-stakes for trust + abuse prevention.
2. **Managed-cloud buy path.** Self-serve provisioning: signup → org → **trial** → usage → **paywall/billing of the tenant**. Recurso must bill itself. Without this there is no product to *purchase*.
3. **Import from Stripe** (customers, plans, subscriptions, payment methods). The #1 switching-cost killer; nobody migrates billing by hand.
4. **One proven live payment**, end-to-end, on a real gateway account (external-cred dependency — flag, don't fake).
5. **Automated backups + one tested restore** of the money database.
6. **Public status page** — buyers check it before signing.

### 🟠 High
7. **Import from Chargebee and RevenueCat.**
8. **Metrics + error tracking** (`/metrics` Prometheus + Sentry on API and dashboard).
9. **Legal/trust docs**: Privacy Policy, Terms of Service, DPA, subprocessor list.
10. **Server-side pagination** on unbounded list endpoints (`ListInvoices` et al.) — correctness + scale.
11. **Webhook delivery reliability surface** — verify retries/backoff + a delivery-log UI a customer can self-debug.

### 🟡 Medium
12. **In-app support/contact** (email queue or widget) + response-time promise.
13. **SOC 2 readiness roadmap** (controls doc + evidence plan; the report comes later).
14. **Full account data export** (GDPR/portability).
15. **EU Peppol Access Point transmission** (currently export-only).

### 🟢 Low (real, but not first-customer)
16. Intercompany eliminations, per-entity GSTR-1/3B, audit-log actor filter, ledger-backed-credit expiry, consent-record UI.

---

## 3. Startup Readiness Audit

*Can a stranger, unaided, do each of these today?*

| Step | Status | Evidence / gap |
|---|:--:|---|
| Discover us | ✅ | Website + pricing page + CTA + docs + SDKs. |
| Sign up | ✅ | `Register.jsx` + `auth.go`. |
| Start a trial | ❌ | No trial concept for Recurso itself; product doesn't bill tenants. |
| Verify email | ❌ | No verification flow in auth handlers. |
| Create an organization | ✅ | Tenant creation on signup. |
| Invite teammates | ✅ | Team handler + invites + RBAC. |
| Connect payment gateway | 🟡 | BYO onboarding built; **not live-verified**. |
| Import data | ❌ | No importer exists. |
| Generate invoices | ✅ | Full invoicing + PDF + e-invoice. |
| Collect money | 🟡 | Built; not proven with a real charge. |
| Pay taxes (GST) | ✅ | GST + IRP e-invoice; EU export-only. |
| Receive webhooks | ✅ | Developers page + webhook infra. |
| Integrate via API | ✅ | OpenAPI + keys + 3 SDKs. |
| Deploy (self-host) | ✅ | Compose / k8s / Railway / Render. |
| Monitor | 🟡 | Health + alert webhook; no metrics/tracing. |
| Recover from failures | ❌ | No backups/DR/restore. |
| Contact support | ❌ | No channel. |

**Verdict:** the funnel breaks in four places — **trial, email verification,
import, support** — plus two unproven links (live payment, monitoring).

---

## 4. Trust Audit

| Control | Status | Note |
|---|:--:|---|
| Security page (public) | 🟡 | `SECURITY.md` in repo; not a customer-facing page. |
| Status page | ❌ | None. |
| SOC 2 readiness | ❌ | No controls doc / evidence plan. |
| PCI story | 🟡 | Gateways tokenize (no PAN storage) — but no written SAQ-A scope statement. |
| GDPR | 🟡 | Audit logs + data model support it; no DPA/export/erasure flow documented. |
| DPA | ❌ | None. |
| Privacy Policy | ❌ | None. |
| Terms of Service | ❌ | None. |
| Incident response | 🟡 | `incident-runbook.md` + alert scheduler exist; not customer-facing. |
| Backups | ❌ | Not automated/tested. |
| Audit logs | ✅ | Present + filterable + paginated. |
| Encryption (in transit / at rest) | 🟡 | TLS at edge; at-rest depends on deploy — undocumented. |
| Key management | 🟡 | Secrets via env/k8s secret; no rotation policy. |
| Secret rotation | ❌ | No documented rotation. |
| Rate limiting | ✅ | Middleware present. |
| Abuse protection | 🟡 | Rate limit yes; no email verification, no signup abuse controls. |
| Fraud detection | 🟡 | Dunning/chargeback handling; no proactive fraud scoring. |

---

## 5. DevOps Audit

| Capability | Status | Note |
|---|:--:|---|
| CI/CD | ✅ | Lint + Frontend + Test + E2E + Trivy + Workers gates. |
| Docker | ✅ | API + frontend + compose (dev/prod/demo). |
| Kubernetes | ✅ | Full manifest set (deploy/svc/ingress/rbac/netpol/secret/cm/ns). |
| Terraform | ❌ | No IaC for the managed environment. |
| Monitoring | 🟡 | Health checks + alert webhook only. |
| Alerts | 🟡 | `ALERT_WEBHOOK_URL` health alerts; no metric-threshold alerting. |
| Health checks | ✅ | `/health` with PG/Redis/TigerBeetle component checks. |
| Logging | 🟡 | App logging; no centralized aggregation/retention. |
| Metrics | ❌ | No `/metrics` / Prometheus. |
| Tracing | ❌ | No OTel. |
| Backups | ❌ | Not automated. |
| Disaster Recovery | ❌ | No RTO/RPO/tested restore. |
| Rollback | 🟡 | Health-gated Cloud Run revision hold; no documented DB-migration rollback drill. |
| Blue/Green | 🟡 | Cloud Run revision routing approximates it. |
| Canary | ❌ | Not configured. |
| Feature flags | ❌ | No flag system (env toggles only). |

---

## 6. Customer Onboarding Audit — can they self-serve end-to-end?

```
Landing Page      ✅  website + pricing
   ↓
Signup            ✅  Register + auth
   ↓
Trial             ❌  no trial / no self-serve provisioning   ← BREAK
   ↓
Create Org        ✅  tenant on signup
   ↓
Configure Billing ✅  plans/pricing built
   ↓
Connect Gateway   🟡  built, not live-verified
   ↓
Import Data       ❌  no importer                             ← BREAK
   ↓
Generate Invoice  ✅
   ↓
Collect Payment   🟡  built, not proven live
   ↓
Receive Webhook   ✅
   ↓
Use Dashboard     ✅
```

**Answer: No — not without talking to us.** Two hard breaks (self-serve
trial/provisioning, import) and two unproven links (live gateway, live payment).
Fix those four and the funnel closes.

---

## Phase 2 — execution order (Founder Mode filter: *"does this get us the first paying customer?"*)

Ordered by **(customer-unblock × trust) ÷ effort**, with external dependencies flagged.

| # | Work | Why it's the buy path | Blocked? |
|---|---|---|:--:|
| 0 | **Repo hygiene**: the stray `import` binary + `" 2"/" 3"` dupes are already gitignored/untracked; only two lowercase legacy report files (`bugs-found.md`, `production-readiness.md`) are tracked — dedupe in a careful pass (case-insensitive FS). | Credibility; a buyer *will* look at the repo. | no |
| 1 | **Email verification** (token issue + verify endpoint + UI + tests). | Closes signup trust/abuse hole; table-stakes. | no |
| 2 | **Import from Stripe** (customers→plans→subscriptions→PMs, dry-run + report + tests). | The switching-cost killer. Highest single "can I actually adopt this" unlock. | no (build against fixtures) |
| 3 | **Backups + tested restore runbook** + **public status page** + **legal pages** (Privacy/Terms/DPA). | Trust table-stakes before anyone wires revenue. | partly (legal review) |
| 4 | **Observability**: `/metrics` + Sentry + basic dashboards/alerts. | "Can I monitor / will you notice an outage" — a buyer question. | no |
| 5 | **Managed-cloud buy path**: self-serve provisioning + trial + Recurso-bills-itself paywall. | Turns the product into something *purchasable*. Largest build. | partly (pricing = business decision) |
| 6 | **Import from Chargebee + RevenueCat.** | Widens the migration funnel. | no |
| 7 | **One proven live payment** + live gateway verification. | Converts "built" → "works." | **yes — live creds** |

**Founder-Mode honesty:** items 5 and 7 depend on a **business decision**
(managed-cloud pricing/packaging) and **external credentials** (live gateway/GSP/IRP)
respectively — I'll build everything up to the credential/decision boundary and
flag exactly what's needed from you. Items 0–4 and 6 are **fully buildable now**
and are where I'll start, top-down.

> Standing caveat from prior analysis, unchanged: closing these makes the product
> *buyable*. It does not by itself prove *demand* — the 10–15 discovery calls on
> the sharpened wedge still run in parallel. Buyable + validated = a startup.
