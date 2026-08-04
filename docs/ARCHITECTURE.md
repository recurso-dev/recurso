# Recurso — Architecture

> **Code-derived.** Every statement cites the file/package that proves it;
> implementation wins over this doc. Assumptions are marked `ASSUMPTION`.

Module `github.com/recurso-dev/recurso` — Go (gin + Postgres), hexagonal
(ports-and-adapters). `internal/core/port` defines interfaces; `internal/adapter/*`
implements them; `internal/service` holds domain logic; `cmd/api/main.go` is the
composition root.

## Package map

**`cmd/` — entrypoints:**
- `cmd/api/main.go` (~2192 lines) — the API server / composition root (DB,
  migrations, services, middleware, routes, schedulers, workers).
- `cmd/mcp/` — the MCP server (separate build: `Dockerfile.mcp`,
  `cloudbuild.mcp.yaml`).
- `cmd/seed/`, `cmd/demo_seed/` (~1506 lines), `cmd/demo_activity/` — seeding.
- `cmd/import/`, `cmd/verify_data/`, `cmd/verify_revrec/`,
  `cmd/seal_accounting_tokens/` — CLIs.

**`internal/core/` — the hexagon core:**
- `internal/core/domain/` (~70 files) — pure entity structs + constants, no
  infra imports (`invoice.go`, `subscription.go`, `ledger.go`, `context.go`…).
- `internal/core/port/` (~45 files) — repository/gateway interfaces (the ports).
- `internal/core/service/tax/` — the tax calculation sub-package.

**`internal/service/` (~180 files)** — the application/business layer. Note:
most logic lives here, **not** under `internal/core/service/` despite the
hexagonal naming; `core/service` holds only `tax/`. `ASSUMPTION:` this is
intentional and `core/service` is vestigial — verify via imports.

**`internal/adapter/` — driven/driving adapters:**
- `adapter/db/` (~90 repos) + `migrate.go` + `migrations/` + `tx_manager.go`,
  `region_router.go`.
- `adapter/handler/` (~70 gin handlers) — the driving adapter.
- `adapter/middleware/` — the chain (`auth`, `rate_limit`, `request_id`,
  `secure`, `idempotency`, `audit`, `metrics`, `sentry`, `portal_auth`, `cache`,
  `demo_guard`).
- `adapter/gateway/` — Stripe/Razorpay/GoCardless/Adyen/Mock + BYO routing
  (`resolver.go`, `smart_router.go`, `tenant_gateway.go`).
- `adapter/accounting/` — QuickBooks/Xero/NetSuite/Tally + `oauth.go`.
- `adapter/taxprovider/` (TaxJar/Avalara/Ziptax), `adapter/vatprovider/vies.go`,
  `adapter/gsp/` (India IRP), `adapter/crm`, `adapter/marketing`, `adapter/ai`,
  `adapter/sms`, `adapter/email`, `adapter/export`.
- `adapter/tigerbeetle/client.go` — the dual-write ledger target.
- `adapter/secretbox/`, `adapter/vault/` — at-rest crypto + card tokenization.
- `adapter/redis/` + `adapter/memory/` — idempotency store + locker (two impls
  of one port).

**`internal/scheduler/`** — cron-style tickers. **`internal/adapter/worker/`** —
queue/fan-out workers. Cross-cutting: `internal/logctx`, `internal/residency`,
`internal/httpsafe`, `internal/safego`, `internal/validate`, `internal/demo`,
`internal/importer/{stripe,chargebee,revenuecat}`, `internal/mcp`.

## Domain model (`internal/core/domain/`)

- **Invoice** (`invoice.go:35`) — Tenant, optional `EntityID` (multi-entity),
  optional `SubscriptionID` (nullable = one-off), `CustomerID`. Money is `int64`
  minor units. Status `draft/open/paid/void/uncollectible/past_due`
  (`invoice.go:13`). `BillingReason` at `invoice.go:25`.
- **Subscription** (`subscription.go:20`) — `PlanID`, status (`"canceled"`, one
  L), period dates, `CommitmentAmount`, `MandateID`, `CouponID`, `ResumeAt`,
  `CancelAtPeriodEnd`.
- **LedgerTransaction / LedgerAccount / TrialBalance** (`ledger.go`) — codes and
  account constants (see `ACCOUNTING_PRINCIPLES.md`).
- Plus Customer, Plan, CreditNote, Coupon, Mandate, Wallet, Dispute, Entity,
  Quote, Gift, Usage, revenue schedules, Tenant/User/Session, portal
  MagicLink/PortalSession, integration/gateway/accounting connections.

## Request lifecycle

Global middleware, in order (`cmd/api/main.go:1500-1548`): DemoGuard (demo mode
only) → RequestID → Secure (headers) → Sentry (recovery/report) → Metrics →
RateLimit `api` 500/min → CORS. The `/v1` group then adds
`SessionOrAPIKeyMiddleware` → `IdempotencyMiddleware` → `Audit`
(`main.go:1759-1762`). A handler reads `tenant_id` from context, calls a
`service.*` method, which calls a `port.*Repository` implemented in
`adapter/db/*`. Tenant id flows on `context.Context` (`stampAuthContext`,
`middleware/auth.go:135`) so repos and the logctx slog handler both see it.

## Auth

- **Dashboard session or API key** — `SessionOrAPIKeyMiddleware`
  (`middleware/auth.go:184-228`): `recurso_session` cookie first, else Bearer
  API key (`rsk_live_`/`rsk_test_`, bcrypt-verified with a 5-min SHA-256 cache,
  `auth.go:36-61`). Key `livemode` must equal server mode or `401
  key_mode_mismatch`.
- **Sessions** — 32-byte CSPRNG token, only the SHA-256 hash persisted
  (`service/auth.go:178-185`). MFA/reset/verify/OAuth/SAML in `auth_phase2.go`,
  `oauth.go`, `sso.go`.
- **Portal** — magic-link (`service/portal.go`): single-use link → `PortalSession`;
  token now read from a POST body (`PortalAuthMiddleware`,
  `middleware/portal_auth.go`).

## Billing flow

Renewal → invoice → paid: `RenewalService.ProcessDueRenewals`
(`service/renewal.go:117`) → `InvoiceService.GenerateInvoice`
(`service/invoice.go:245`, posts the Code-1 ledger leg via `recordInvoiceLeg`
`:99`) → `SubscriptionService.MarkInvoicePaid` (`subscription_payment.go:18`,
atomic `MarkPaid` claim so one settler runs side-effects: dunning-recovery
attribution, write-off recovery if previously uncollectible, then the cash leg
net of any wallet-settled amount). Clawback: `ReverseSettledPayment`
(`subscription_payment.go:138`).

## Accounting flow

Double-entry ledger in `service/ledger.go` (see `ACCOUNTING_PRINCIPLES.md` for
the code table). Dual-write: Postgres (authoritative) always, TigerBeetle when
connected (`ledger.go:16`). The **reconciler** (`service/reconciliation.go`,
read-only) asserts invoice/payment/credit-note completeness, trial-balance
integrity, and a deferred-≥-scheduled invariant. The **invariant harness**
(`service/ledger_invariant_pg_test.go`) runs randomized billing sequences and
fails CI on any discrepancy — the gate CLAUDE.md/ADR-002 describe.

## Background jobs

**Schedulers** (`internal/scheduler/`, started `main.go:938-1137`, all select on
`ctx.Done()`): BillingCycle, Dunning, Trial, MandateDebit, PreCharge, Nexus,
CardExpiring, ProgressiveBilling, SubscriptionResume, MRRSnapshot, Reconciliation,
HealthAlert.

**Workers** (`internal/adapter/worker/`, `startWorker` at `main.go:873-888`):
Retry, Webhook (outbound delivery), Churn, RevRec, EInvoiceRetry (India),
EUEInvoiceRetry, DunningCampaign, AccountingSync, CRMSync, Export (GL→S3),
DemoReset. Concurrency uses atomic DB claims, not locks (ADR-003).

## Database

~165 numbered migration pairs in `internal/adapter/db/migrations/` (`000001` …
`000165`), **auto-run on boot** via golang-migrate with `//go:embed`
(`db.RunMigrations`, `main.go:144`; `migrate.go`). New migration = next
sequential number, both `.up.sql` and `.down.sql`.

## External integrations

Payment gateways (`adapter/gateway/`) with per-tenant BYO routing
(`GatewayResolver` → `SmartRouter`, reads acting tenant from
`domain.TenantIDKey`, falls back to env gateway). Tax (`adapter/taxprovider/` +
`adapter/gsp/` for India IRP, behind `port.TaxEngine`). Accounting
(`adapter/accounting/`, OAuth token refresh). CRM/marketing/AI/storage/SMS.
Non-gateway operator integrations via `integration_connection` (categories
tax/crm/storage), SSRF-guarded.

## Deployment & observability

- **Deploy:** API → Google Cloud Build → Cloud Run (`api.recurso.dev`); dashboard
  → Cloudflare Workers (`app.recurso.dev`), automatic on merge to main.
  `cloudbuild.yaml` gates on the full test suite incl. Postgres-backed tests and
  runs migrations before deploy. MCP has its own `cloudbuild.mcp.yaml`.
- **Logging:** `slog` + `internal/logctx.ContextHandler` stamps
  `request_id`/`tenant_id`/`user_id` on every `*Context` log.
- **Metrics:** Prometheus at `GET /metrics` (`main.go:1556`, optionally
  `METRICS_TOKEN`-gated). **Health:** `GET /health` pings postgres/redis/
  tigerbeetle without leaking DSN detail. `GET /version` reports gateway_mode.

## Security boundaries

- **Tenant scoping** on every `/v1` request; the only cross-tenant surface is
  `/platform/metrics`, gated by `FOUNDER_TOKEN` (404 when unset,
  `main.go:1565-1587`). Many `*_tenant_pg_test.go` enforce isolation.
- **Residency guard** (`internal/residency`): `RESIDENCY_MODE=self_hosted`
  disables third-party SaaS egress at construction sites.
- **Credential vault** (`adapter/secretbox`): AES-256-GCM under
  `GATEWAY_ENCRYPTION_KEY`. Sessions/keys stored only hashed.
- **SSRF guard** (`internal/httpsafe`): rejects private/loopback/link-local +
  cloud-metadata IPs, re-checks the connect-time IP (DNS-rebind defense).

## Tech-debt map

Largest files (complexity hotspots): `cmd/api/main.go` (~2192), `cmd/demo_seed`
(~1506), `service/ledger.go` (~1325), `db/invoice_repository.go` (~1303),
`service/invoice.go` (~993), `service/credit_note.go` (~992). Zero
TODO/FIXME/HACK markers in non-test `internal/*.go`. Known self-labeled gaps:
accounting-connection tokens stored plaintext (`secretbox.go` comment; ADR-006
accepts it, `cmd/seal_accounting_tokens/` retrofits). See `../REMEDIATION.md`
for the active remediation list.

**Doc-drift found during derivation** (now corrected in CLAUDE.md): the
dunning-campaign / cancel-flow responses ARE `{data:}`-wrapped in current code
(the old "unwrapped quirk" is fixed); the default list limit is 50, not 10.

## Related

`ACCOUNTING_PRINCIPLES.md` · `API_GUIDELINES.md` · `PRODUCT.md` ·
`docs/decisions/ADR-00{1..6}` · `DOCUMENTATION_RULES.md`
