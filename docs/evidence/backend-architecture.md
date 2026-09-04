# Evidence — backend-architecture

> Code-cited findings extracted by repository inspection (reverse-engineered
> from the implementation) on 2026-08-04. This is the evidence base for the
> corresponding source-of-truth doc(s). Every claim cites the file that
> proves it; implementation wins. Confidence is High unless a line is marked
> `ASSUMPTION`. Regenerate by re-running the inspection agents.

I have gathered comprehensive evidence. Here is the code-cited backend architecture report.

---

# Recurso Backend Architecture — Code-Derived Report

Module: `github.com/recurso-dev/recurso` (Go, gin + Postgres). Root: `/Users/swapnull/Documents/Workspace/reccurso/recur-so`. Hexagonal (ports-and-adapters): `internal/core/port` defines interfaces, `internal/adapter/*` implements them, `internal/service` holds domain logic, `cmd/api/main.go` is the composition root.

## 1. Package Map

**`cmd/` — entrypoints (each has `main.go`):**
- `cmd/api/main.go` (2192 lines) — the API server / composition root: wires DB, migrations, services, middleware, routes, schedulers, workers.
- `cmd/mcp/` — MCP server (separate build, `cloudbuild.mcp.yaml`, `Dockerfile.mcp`).
- `cmd/seed/`, `cmd/demo_seed/` (1506 lines), `cmd/demo_activity/` — data seeding for dev/demo.
- `cmd/import/` — offline import CLI (`store.go`, `importfile.go`).
- `cmd/seal_accounting_tokens/` — one-off migration to seal previously-plaintext accounting tokens (ADR-006 follow-up).
- `cmd/verify_data/`, `cmd/verify_revrec/` — data-integrity verifiers.

**`internal/core/` — the hexagon core:**
- `internal/core/domain/` — ~70 entity files, pure structs + constants, no infra imports (e.g. `invoice.go`, `subscription.go`, `ledger.go`).
- `internal/core/port/` — ~45 repository/gateway interfaces (`repository.go`, `ledger_repository.go`, `payment_gateway.go`, `tax_engine.go`, `card_vault.go`, …). These are the "ports."
- `internal/core/service/tax/` — a small sub-package for tax calc.

**`internal/service/` — application/business logic (~180 files).** This is where most logic lives (NOT under core/service despite the hexagonal naming — a structural inconsistency). E.g. `invoice.go`, `ledger.go`, `subscription.go`, `renewal.go`, `reconciliation.go`, `accounting.go`, `auth.go`, `portal.go`.

**`internal/adapter/` — driven/driving adapters (implementations of ports):**
- `adapter/db/` — Postgres repositories (~90 files) + `migrate.go` + `migrations/` (318 files) + `tx_manager.go`, `region_router.go`, `postgres.go`.
- `adapter/handler/` — gin HTTP handlers (~70 files, the "driving" adapter).
- `adapter/middleware/` — the middleware chain (`auth.go`, `rate_limit.go`, `request_id.go`, `secure.go`, `idempotency.go`, `audit.go`, `metrics.go`, `sentry.go`, `portal_auth.go`, `cache.go`, `demo_guard.go`, `region.go`).
- `adapter/gateway/` — payment gateways: `stripe.go`, `razorpay.go`, `gocardless.go`, `adyen.go`, `mock.go`, plus `resolver.go`, `smart_router.go`, `tenant_gateway.go` (BYO routing).
- `adapter/accounting/` — `quickbooks.go`, `xero.go`, `netsuite.go`, `tally.go`, `oauth.go`, `mock.go`.
- `adapter/taxprovider/` — `taxjar.go`, `avalara.go`, `ziptax.go`; `adapter/vatprovider/vies.go`.
- `adapter/crm/hubspot.go`, `adapter/marketing/brevo.go`, `adapter/ai/openai.go`, `adapter/sms/` (twilio), `adapter/email/` (smtp/console), `adapter/export/s3.go`.
- `adapter/gsp/` — India GST Suvidha Provider (e-invoicing): `nic.go`, `crypto.go`, `schema.go`.
- `adapter/tigerbeetle/client.go` — the TigerBeetle ledger client (dual-write target).
- `adapter/secretbox/`, `adapter/vault/` — at-rest crypto + card tokenization.
- `adapter/redis/` + `adapter/memory/` — idempotency store + distributed locker (two implementations behind the same port).
- `adapter/metrics/metrics.go`, `adapter/telemetry/`, `adapter/alerting/webhook.go`.

**`internal/scheduler/` — cron-style tickers** (billing_cycle, dunning, trial, mandate_debit, precharge, nexus, card_expiry, progressive_billing, subscription_resume, mrr_snapshot, reconciliation, health_alert).

**`internal/adapter/worker/` — queue/fan-out background workers** (webhook, retry, revrec, einvoice, eu_einvoice, accounting_sync, crm_sync, export, churn, dunning_campaign, demo_reset).

**Other internal packages:** `internal/logctx` (context-aware slog), `internal/residency` (self-hosted egress guard), `internal/httpsafe` (SSRF guard), `internal/safego` (safe goroutines), `internal/validate`, `internal/demo`, `internal/importer/{stripe,chargebee,revenuecat}`, `internal/mcp`.

## 2. Domain Model (`internal/core/domain/`)

Core money entities and relationships:
- **Invoice** (`invoice.go:35`) — belongs to Tenant, optional `EntityID` (multi-entity), optional `SubscriptionID` (nullable for one-off), `CustomerID`. Money is `int64` minor units: `Subtotal`, `TaxAmount`, `Total`, `AmountPaid`, `CreditApplied`. Status enum (`invoice.go:13`): `draft/open/paid/void/uncollectible/past_due`. `BillingReason` (`invoice.go:25`): `subscription_create/subscription_cycle/subscription_update/mandate_debit/gift_purchase/manual/progressive_usage`. India GST fields (IGST/CGST/SGST/HSN/IRN) present inline.
- **Subscription** (`subscription.go:20`) — links `CustomerID`→`PlanID`, `Status`, `CurrentPeriodStart/End`, `TrialEnd`, `BillingAnchor`, `CommitmentAmount` (per-period minimum true-up), `MandateID`, `CouponID`, `ResumeAt` (pause), `CancelAtPeriodEnd`, external `StripeSubscriptionID`/`RazorpaySubscriptionID`. Status `"canceled"` (one L) per CLAUDE.md.
- **Customer** (`customer.go:9`), **Plan** (`plan.go:18`).
- **Ledger** (`ledger.go`) — `AccountType`, `LedgerAccount`, `LedgerTransaction` (with `Code`), `TrialBalance`/`TrialBalanceLine`. Account codes and posting codes are constants here (see §6).
- Supporting entities: `CreditNote`, `credit_statement.go`, `Coupon`, `Mandate`, `Wallet`, `Dispute`, `payment_attempt.go`, `recovered_payment.go`, `Entity`/`organization.go` (multi-entity books), `Quote`, `gift.go`, `dunning.go`/`dunning_campaign.go`, `Usage`/`metering.go`, `revrec.go` (recognition schedules), `mrr_snapshot.go`, `Tenant`, `User`, `Session`, `portal.go` (MagicLink/PortalSession), `integration_connection.go`, `gateway_connection.go`, `accounting_connection.go`.
- Context keys live in `domain/context.go` (`TenantIDKey`, `UserIDKey`) — used by both middleware and logctx.

## 3. Request Lifecycle

**Global middleware chain, in registration order** (`cmd/api/main.go:1500-1548`):
1. `middleware.DemoGuard()` — only when `demo.Enabled()` (`main.go:1502`).
2. `middleware.RequestIDMiddleware()` (`main.go:1505`, `middleware/request_id.go`).
3. `middleware.SecureMiddleware()` (`main.go:1506`, `middleware/secure.go` — security headers).
4. `middleware.SentryMiddleware()` (`main.go:1508`) — panic/5xx reporting (inert without `SENTRY_DSN`); effectively the recovery layer.
5. `middleware.MetricsMiddleware(httpMetrics)` (`main.go:1511`, `middleware/metrics.go`).
6. `middleware.RateLimitMiddleware(rdb, "api", rateLimit, time.Minute)` (`main.go:1517`) — default 500/min.
7. CORS (inline closure, `main.go:1533`) — allowlist echo, credentials true.

**Per-route rate buckets** (`main.go:1662-1671`): `public` (20/min), `session` (120/min), `expensive` (30/min per-tenant) — ADR-001 scoped rate limiting.

**Auth groups:**
- `/v1` group (`main.go:1759-1762`): `SessionOrAPIKeyMiddleware(tenantRepo, authService, serverLive)` → `IdempotencyMiddleware(idempotencyStore)` → `Audit(auditLogRepo)`. Tenant scoping is set here (see §4/§11).
- `/portal/api` group (`main.go:1735-1736`): `PortalAuthMiddleware(portalService)`.

**Flow:** handler (`adapter/handler/*`) reads `tenant_id` from context via `middleware.GetTenantID(c)` (`middleware/auth.go:231`), calls a `service.*` method, which calls a `port.*Repository` implemented in `adapter/db/*`. Tenant id flows on `context.Context` (stamped by `stampAuthContext`, `middleware/auth.go:135`) so both repositories and the logctx slog handler see it.

## 4. Auth Flow

**Dashboard session vs API key** — unified in `SessionOrAPIKeyMiddleware` (`middleware/auth.go:184-228`):
1. First tries the `recurso_session` cookie (`domain.SessionCookieName`) via `resolver.ResolveSession` → sets `tenant_id`, `user_id`, `user_role`, `user`.
2. Falls through to Bearer API key (`extractBearerToken`, `auth.go:66`) resolved by `resolveAPIKey` (`auth.go:86`): dev bypass (`recurso_secret`, only `APP_ENV=development`+`ALLOW_DEV_BYPASS=true`), then a SHA-256-keyed `verifiedKeyCache` (5-min TTL, avoids per-request bcrypt — `auth.go:25-61`), then `repo.GetTenantByKey` (bcrypt compare).

**Live/test mode gating** (`auth.go:143-168`, `184-228`): a key carries `livemode`; it must equal the server's `serverLive` (derived from gateway key prefixes, `main.go:1638-1650/1758`), else `CodeKeyModeMismatch` (test keys `rsk_test_`, live keys `rsk_live_` per the message at `auth.go:120-125`). `AuthMiddleware` (`auth.go:143`) is the API-key-only variant sharing `resolveAPIKey`.

**Session creation** (`service/auth.go`): `newSessionToken` (`auth.go:185`) = 32-byte CSPRNG; only the SHA-256 `hashToken` is persisted (`auth.go:178`). `openSession`/`OpenSessionForUser` (`auth.go:196/221`). `Register` (`auth.go:252`) creates tenant + owner user + API key + session. Phase-2 auth (MFA, password reset, email verification, OAuth, SAML SSO) in `auth_phase2.go`, `oauth.go`, `sso.go`, wired at `main.go:1691-1722`.

**Portal magic-link** (`service/portal.go`): `RequestMagicLink` (`portal.go:51`) creates a `domain.MagicLink` (expiry `db.MagicLinkExpiry`) and emails a login URL; `VerifyMagicLink` (`portal.go:99`) validates + single-use `MarkUsed` + creates a `domain.PortalSession` (`db.PortalSessionExpiry`). Routes public at `main.go:1730-1732` (POST-body token preferred over deprecated query-string). Portal routes guarded by `PortalAuthMiddleware` (`middleware/portal_auth.go`).

## 5. Billing Flow

**Renewal → invoice → paid:**
1. `RenewalService.ProcessDueRenewals` (`service/renewal.go:117`) claims due locally-billed subscriptions; `RenewSubscription` (`renewal.go:142`) advances the period and calls the invoicer; `attemptPayment` (`renewal.go:197`) charges a saved method if configured. Driven by `BillingCycleScheduler` (see §7).
2. `InvoiceService.GenerateInvoice` (`service/invoice.go:245`) builds the invoice (base + metered lines via `meteredLines`/`filteredMeteredLines`, `invoice.go:705/823`; commitment true-up; coupons/credit). On commit it posts the ledger leg (`recordInvoiceLeg`, `invoice.go:99`) and fires side-effects (`notifyInvoiceCreated`, `generateEInvoiceAfterCommit`/`generateEUEInvoiceAfterCommit`). Variants: `GenerateFinalUsageInvoice` (`invoice.go:578`), `GenerateAdvanceInvoice` (`invoice.go:895`).
3. `SubscriptionService.MarkInvoicePaid` (`service/subscription_payment.go:18`): atomic claim via `invoiceRepo.MarkPaid` conditional UPDATE (`subscription_payment.go:54`) so only one settler runs side-effects; then dunning-recovery attribution (`:70`), write-off recovery if previously uncollectible (`:78`), and the ledger cash leg net of any wallet-settled amount (`walletSettled`, `:38`, `:85+`). `ReverseSettledPayment` (`subscription_payment.go:138`) handles clawbacks. Upgrades/downgrades: `subscription_upgrade.go`, `subscription_cancel.go`, `subscription_pause.go`, `subscription_trial.go`.

## 6. Accounting Flow (double-entry ledger)

**Account codes** (`domain/ledger.go:125-134`): AR 1100, Cash 1000, Revenue 4000, Deferred Revenue 2100, Recognized Revenue 4100, Tax Payable 2200, Customer Credit 2300, Refunds 5000, Credits Issued 5100, TDS Receivable 1200.

**Posting codes** (`LedgerTransaction.Code`, `domain/ledger.go:137-249`): 1=invoice, 2=revenue recognition, 3=payment, 6=OutputTax, 9=RefundTaxReversal, 10=TDSReceivable, 11-15=wallet (topup/drain/refund/forfeit/expiry), 16/17=downgrade credit/tax reversal, 18=CreditExpiry, 19=PaymentReversal, 20=CreditVoid, 21=DowngradeRevenueReversal, 22/23=InvoiceWriteOff/tax, 24/25=WriteOffRecovery/tax.

**Posting logic** (`service/ledger.go`, 1325 lines): `RecordInvoice` (`ledger.go:266`) DR AR / CR Revenue, plus DR Revenue / CR Tax Payable (code 6) for collected GST (`ledger.go:310-324`). `RecordPaymentWithSettled` (`ledger.go:366`) DR Cash + DR TDS Receivable / CR AR, excluding already-settled wallet amount. `RecordRefund` (`ledger.go:742`), `RecordDeferredRefundReversal` (`ledger.go:796`), `RecordPaymentReversal` (`ledger.go:671`), `RecordInvoiceWriteOff`/`RecordWriteOffRecovery` (`ledger.go:502/600`). Idempotency via `CountTransactionsByReferenceAndCode` before posting. Multi-entity: postings resolve a `ledgerEntity` (`resolveEntity`, `ledger.go:63`) with its own TigerBeetle `LedgerID`.

**Dual-write** (`ledger.go:16`): always writes Postgres (`pgRepo`, authoritative), and TigerBeetle (`tbClient`) when connected (`ledger.go:186/213`). PG-only mode passes nil TB client.

**Reconciler** (`service/reconciliation.go`): `ReconciliationService.Run` (`reconciliation.go:146`) — read-only; asserts every non-draft invoice has Code-1 postings summing to total, every paid invoice has Code-3 summing to amount_paid, every Code-1/3 posting references an existing invoice; `trialBalanceDiscrepancies` (`reconciliation.go:259`) checks debits==credits and abnormal balances; a second pass diffs PG vs TigerBeetle by transaction id (`compareTigerBeetle`, bounded by `MaxTBComparedRows`, `reconciliation.go:22`). It never fails the report on TB errors — reports `TBCompared=false`/`TBSkipReason`.

**Invariant harness** (`service/ledger_invariant_pg_test.go`): `TestLedgerInvariants_RandomizedBillingSequences` (`:36`) — property test running randomized billing sequences (trial conversion, quote, gift, new sub, plan change, backdate, recognize — `randomOp`, `:185`), `opsPerSeed=25`, fixed seeds for CI, `LEDGER_INVARIANT_SEED` to explore. This is the CI gate CLAUDE.md and ADR-002 describe. Revenue recognition: `service/revrec.go` `ProcessDueEvents` (`revrec.go:64`) claims events pending→processing; `UnwindOnCancel` (`revrec.go:94`).

## 7. Background Jobs

**Schedulers** (`internal/scheduler/`, started in `main.go:938-1137`, each `.Start()`; all select on `ctx.Done()`):
- `BillingCycleScheduler` (`billing_cycle.go:12`, `main.go:1040`) — claims/processes due locally-billed renewals (ADR-003 claims).
- `DunningScheduler` (`dunning.go`, `main.go:948`) — retry/reminder schedule for failed payments.
- `TrialScheduler` (`trial.go`, `main.go:962`) — trial-ending reminders + conversion.
- `MandateDebitScheduler` (`mandate_debit.go:13`, `main.go:983`) — UPI/mandate auto-debit with a claim lease shorter than the tick.
- `PreChargeScheduler` (`precharge.go:17`, `main.go:938`) — 24h pre-charge notifications.
- `NexusScheduler` (`nexus.go:14`, `main.go:1053`) — daily US economic-nexus threshold evaluation.
- `CardExpiringScheduler` (`card_expiry.go:23`, `main.go:973`).
- `ProgressiveBillingScheduler` (`progressive_billing.go:14`, `main.go:1078`) — interim usage bills when accrued usage crosses a threshold.
- `SubscriptionResumeScheduler` (`subscription_resume.go`, `main.go:1093`) — auto-resume paused subs.
- `MRRSnapshotScheduler` (`mrr_snapshot_scheduler.go:24`, `main.go:1086`) — daily per-sub MRR snapshots.
- `ReconciliationScheduler` (`reconciliation_scheduler.go:25`, `main.go:1063`) — daily ledger-vs-billing under a distributed lock; runs once on start (`:50`).
- `HealthAlertScheduler` (`health_alert_scheduler.go`, `main.go:1136`) — probes postgres/redis/tigerbeetle every 60s, fires webhook alerts.

**Workers** (`internal/adapter/worker/`, started via `startWorker` at `main.go:873-888`):
- `RetryWorker` (`retry_worker.go`, `main.go:766`) — smart payment retries; records recovery attribution.
- `WebhookWorker` (`webhook_worker.go`, `main.go:779`) — outbound event delivery (parked in demo mode).
- `ChurnWorker` (`churn_worker.go`, `main.go:783`), `RevRecWorker` (`revrec_worker.go`, `main.go:784`).
- `EInvoiceRetryWorker` (`einvoice_worker.go:13`) — India IRP retries w/ backoff 5m/15m/1h/6h/24h; `EUEInvoiceRetryWorker` (`eu_einvoice_worker.go:12`) — EU Access Point redrive.
- `DunningCampaignWorker` (`dunning_campaign_worker.go:11`, `main.go:858`).
- `AccountingSyncWorker` (`accounting_sync_worker.go`, `main.go:862`) — parked in demo mode (no SaaS egress).
- `CRMSyncWorker` (`crm_sync_worker.go:15`, `main.go:801`), `ExportWorker` (`export_worker.go:18`, GL→S3 daily), `DemoResetWorker` (`demo_reset_worker.go:10`, `main.go:1236`).

Worker concurrency uses atomic DB claims, not locks (ADR-003; e.g. `MarkPaid`, `Claim*` methods).

## 8. DB Schema

- **318 files** in `internal/adapter/db/migrations/` (~165 numbered pairs, `000001_init_schema` … `000165_import_compare_reports`, sequential `.up.sql`/`.down.sql`).
- **Auto-run on boot**: `db.RunMigrations(dbURL)` at `main.go:144`, implemented in `internal/adapter/db/migrate.go` — golang-migrate with `//go:embed migrations/*.sql`, `m.Up()` treating `ErrNoChange` as success. CLAUDE.md notes a mis-migration holds the previous healthy Cloud Run revision.
- Repositories (`adapter/db/*_repository.go`, ~90) map tables to domain. Notable: `invoice_repository.go` (1303 lines), `ledger_repository.go` (661), `subscription_repository.go` (669), `usage_repository.go`. Cross-cutting infra: `tx_manager.go`, `region_router.go`, `postgres.go`. Key tables (from repo/domain): tenants, api_keys, users, sessions, customers, plans, subscriptions, invoices/invoice_items, ledger_accounts/ledger_transactions, credit_notes, coupons, mandates, wallets, disputes, payment_attempts, dunning_campaigns, usage, revrec schedules, gateway_connections, integration_connections, accounting_connections, audit_log.

## 9. External Integrations

- **Payment gateways** (`adapter/gateway/`): Stripe (`stripe.go`, 520 lines), Razorpay, GoCardless (ACH/bank mandate), Adyen, Mock. Per-tenant BYO routing: `GatewayResolver` (`resolver.go:22`) builds a per-tenant `SmartRouter` from encrypted `gateway_connections`, caching per tenant+updated_at; `TenantGateway` (`tenant_gateway.go:11`) reads acting tenant from `domain.TenantIDKey` and falls back to env gateway. `SmartRouter` supports currency overrides via `GATEWAY_CURRENCY_OVERRIDES` (`smart_router.go:38`). Webhooks: `/webhooks/{stripe,razorpay,gocardless}` + per-connection `/:connID` variants (`main.go:1680-1688`).
- **Tax** (`adapter/taxprovider/`): TaxJar, Avalara, Ziptax (US sales tax), selected at `main.go:388-403`; VAT via `adapter/vatprovider/vies.go`. India GST via `adapter/gsp/` (NIC IRP). Behind `port.TaxEngine` (`port/tax_engine.go:61`).
- **Accounting** (`adapter/accounting/`): QuickBooks + Xero (OAuth), NetSuite + Tally (token-based, ADR-006). `service/accounting.go` (646 lines) manages token refresh (`tokenRefreshWindow`, `accounting.go:22`) and entity-id mapping.
- **CRM/Marketing/AI/Storage/SMS**: HubSpot (`adapter/crm/hubspot.go`), Brevo (`adapter/marketing/brevo.go`), OpenAI (`adapter/ai/openai.go`), S3 (`adapter/export/s3.go`), Twilio (`adapter/sms/twilio_sms.go`).
- **BYO integration_connection** (`domain/integration_connection.go`): non-gateway operator integrations — categories `tax` (taxjar/avalara/ziptax), `crm` (hubspot), `storage` (s3), validated by `ValidIntegration` (`:21`). Managed by `IntegrationConnectionService` with SSRF egress control (`main.go:417`).

## 10. Deployment & Observability

- **Deploy** (CLAUDE.md + `cloudbuild.yaml`): API → Google Cloud Build → Cloud Run at `api.recurso.dev`; dashboard → Cloudflare Workers at `app.recurso.dev`, automatic on merge to main. `cloudbuild.yaml` gates build/push/deploy on the full test suite incl. Postgres-backed tests (`cloudbuild.yaml:1-5`), runs migrations before deploy (`:45`), deploys with `gcloud run` to `${_DEPLOY_REGION}` (`:82-91`). MCP has its own `cloudbuild.mcp.yaml`. Alt manifests present but not primary: `render.yaml`, `railway.json`, `k8s/`, `docker-compose.*`.
- **Logging**: `slog` with `internal/logctx.ContextHandler` (`logctx.go:16`) — stamps `request_id`/`tenant_id`/`user_id` from context onto every `*Context` log record; IDs put on context by middleware (`middleware/auth.go:135`).
- **Metrics**: `adapter/metrics/metrics.go` Prometheus counters via `MetricsMiddleware`; scrape at `GET /metrics` (`main.go:1556`), optionally `METRICS_TOKEN`-gated. Errors → Sentry (`middleware/sentry.go`).
- **Health**: `GET /health` (`main.go:1589`) pings postgres/redis/tigerbeetle, returns `degraded`/503 without leaking DSN errors; `GET /version` (`main.go:1651`) reports version + `gateway_mode`; `GET /platform/metrics` (`main.go:1571`) is a founder-token-gated cross-tenant funnel.

## 11. Security Boundaries

- **Tenant scoping**: every `/v1` request resolves exactly one `tenant_id` onto context (`middleware/auth.go`), and repositories filter by it. The ONLY cross-tenant surface is `/platform/metrics`, deliberately outside tenant auth and gated by `FOUNDER_TOKEN` (404 when unset) (`main.go:1565-1587`). Many `*_tenant_pg_test.go` / `*_entity_isolation_pg_test.go` tests enforce isolation.
- **Residency / egress guard** (`internal/residency/residency.go`): `RESIDENCY_MODE=self_hosted` hard-disables optional third-party SaaS egress at construction sites. Callsites: telemetry (`telemetry.go:98`), accounting SaaS sync (`service/accounting.go:599`), tax providers (`main.go:388-417`), CRM/export (`main.go:794`), and `SetAllowPrivateEgress` (`main.go:417`).
- **Credential vault**: `adapter/secretbox/secretbox.go` — AES-256-GCM (`nonce||ciphertext`, base64) under `GATEWAY_ENCRYPTION_KEY` for tenant gateway/BYO credentials; wrong-length key fails at construction. Card tokenization via `adapter/vault/stripe_vault.go` (Stripe PaymentMethods, `port.CardVault`). Sessions/API keys stored only as SHA-256/bcrypt hashes.
- **SSRF guard** (`internal/httpsafe/httpsafe.go`): `ValidateExternalURL` (create-time, `:31`) rejects non-http(s) and private/loopback/link-local hosts incl. `169.254.169.254` cloud-metadata; `DialControl` (`:63`) re-checks the connect-time IP to defeat DNS rebinding/redirects.

## 12. Tech-Debt Map

- **Largest files** (accidental-complexity hotspots): `cmd/api/main.go` (2192 lines — the composition root does env parsing, wiring, routing, scheduler/worker startup all inline), `cmd/demo_seed/main.go` (1506), `internal/service/ledger.go` (1325), `internal/adapter/db/invoice_repository.go` (1303), `service/invoice.go` (993), `service/credit_note.go` (992), `service/tax_resolver.go` (757), `adapter/handler/portal_api.go` (695).
- **Structural inconsistency**: business logic lives in `internal/service/` (~180 files) rather than the hexagonal-implied `internal/core/service/` (which holds only `tax/`). Worth verifying whether this is intentional. **ASSUMPTION**: the `internal/service` package is the real application layer and `core/service` is vestigial — verify by checking imports of `core/service/tax`.
- **Known self-labeled gaps** (prose, not TODO/FIXME — the codebase has zero `TODO/FIXME/HACK` markers in non-test `internal/*.go`): accounting-connection tokens stored **plaintext** (`secretbox.go:3` "a known gap"), which ADR-006 accepts and `cmd/seal_accounting_tokens/` retrofits. Ledger code comment flags `LedgerCodeDowngradeCredit` historically hardcoded `6`, silently colliding with `LedgerCodeOutputTax` (`domain/ledger.go:163-168`).
- **ADR-005** is "layered caching" (`docs/decisions/ADR-005-layered-caching.md`) — governs the react-query + `CacheMiddleware` (`middleware/cache.go`, applied to analytics/report routes at `main.go:1858/1940`) caching contract. (The task's "ADR-005 nature" is the caching decision, not a code hack.) Full ADR set: ADR-001 scoped rate-limiting, 002 ledger posting semantics, 003 claim-based workers, 004 one-off revenue recognition, 005 layered caching, 006 token-based accounting connections (`docs/decisions/`).
- **API quirks** (per CLAUDE.md, confirmable in handlers): inconsistent list pagination defaults; dunning-campaign and cancel-flow list/get/stats responses are UNWRAPPED (not `{data:...}`). Root also carries many overlapping status docs (`BUGS_FOUND.md`, `PRODUCTION_READINESS.md`, `REMEDIATION.md`, `STARTUP_AUDIT.md`) indicating in-flight remediation.

**Verification notes / assumptions to double-check:**
- `serverLive` is derived only from Stripe/Razorpay key prefixes (`main.go:1644-1646`); GoCardless/Adyen live keys don't flip it — **worth verifying** this is intended.
- I did not open every handler; the handler→service→repository claim is verified structurally (context tenant id + port interfaces) and via `MarkInvoicePaid`/`GenerateInvoice`, not for all ~70 handlers.