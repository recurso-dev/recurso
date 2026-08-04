# Evidence — api-contract

> Code-cited findings extracted by repository inspection (reverse-engineered
> from the implementation) on 2026-08-04. This is the evidence base for the
> corresponding source-of-truth doc(s). Every claim cites the file that
> proves it; implementation wins. Confidence is High unless a line is marked
> `ASSUMPTION`. Regenerate by re-running the inspection agents.

I have everything needed. Here is the code-cited report.

---

# API Contract Report — recur-so (Go/gin API)

Source of truth: implementation. All citations are `file:line`. Uncertain claims marked **ASSUMPTION**.

---

## 1. RESPONSE ENVELOPE

**Convention:** success payloads are wrapped as `{"data": ...}`. There is no single enforced helper — two coexist:

- `internal/adapter/handler/response.go:52-71` — `RespondSuccess`/`RespondCreated` emit `gin.H{"data": data}`; `RespondList` emits `{"data":..., "meta":{total,page,per_page}}`.
- Most handlers inline `c.JSON(status, gin.H{"data": ...})` directly rather than calling the helper. The `{data:}` shape is used in **53 of the handler files** (grep of `gin.H{"data"`), 239 occurrences of a `"data"` key across handlers.

Because the wrapper is a convention, not a middleware, there are **real deviations**:

### 1a. Bare-object success responses (no `{data:}` wrapper)
- **Auth** (`internal/adapter/handler/auth.go`): `Register` returns bare `{tenant, api_key, user}` (`auth.go:108-112`); `Login`/`LoginMFA` return bare `{user, tenant}` (`auth.go:152-155`, `186-189`); MFA gate returns bare `{mfa_required, mfa_token}` (`auth.go:144-147`). These are deliberate auth-shape objects, unwrapped.
- **Import handlers** return bare domain objects, not wrapped:
  - `stripe_import.go:48,74,93` (`plan`, `report`, `result`)
  - `chargebee_import.go:41,68,81`
  - `revenuecat_import.go:40,65,78`
- **Other bare returns:** `billing.go:113` (`view`), `dispute.go:110` (`resp`), `portal_api.go:193` (`resp`), `portal_api.go:279` (`customer`), `gift_handler.go:54` (`gift`).

### 1b. Action/status responses (not resources, not `{data:}`)
- `dunning_campaign.go:293` and `cancel_flow.go:304` → `{"status":"deleted"}`.
- `einvoice.go:110` → `{"message":"E-invoice cancelled successfully"}`; `einvoice.go:223,232,239` → `{"success":bool,"message":...}` (TestIRPConnection).
- `auth.go:217,239,268,296,306` → `{"message":...}` for password-reset/verify/logout flows.

### 1c. `{data:}` + sibling keys (data-wrapped but non-canonical shape)
- `gst.go:102-105`, `145-148` → `{"data":..., "gov_schema":...}`.
- `gst.go:285` → `{"data":config, "message":...}`; `einvoice.go:74-77`, `190-193` → `{"data":..., "message":...}`; `eu_einvoice.go:237` → `{"data":rec, "message":msg}`.

### 1d. CLAUDE.md is STALE on dunning/cancel-flow
CLAUDE.md claims *"Dunning-campaign and cancel-flow list/get/stats responses are UNWRAPPED (not `{data:}`)."* The **current code contradicts this** — they DO wrap: `dunning_campaign.go:48,86,113,168` and `cancel_flow.go:38,83,110,169,336,375,402,431` all use `gin.H{"data": ...}`. So the documented quirk has been fixed in code; the doc note is out of date. (Only the `{status:"deleted"}` delete responses remain unwrapped.)

### 1e. Raw `gin.H{"error":...}` that bypass the canonical error envelope
Three sites emit a bare-string `error` key instead of `{error:{code,message}}`:
- `internal/adapter/handler/webhook.go:153` → `{"error":"dedup store unavailable; retry"}` (503).
- `internal/adapter/handler/webhook_gocardless.go:87` → same (503).
- `cmd/api/main.go:1583` → `{"error":"failed to compute platform metrics"}` (500, founder-only endpoint).

The candidate list (einvoice, gst, checkout) was **only partially right**: `checkout.go` actually wraps everything in `{data:}` (`checkout.go:183,281,370,402`) and is NOT a deviation. einvoice/gst deviate only via the sibling-key and message/success action shapes above. auth is the clearest bare-object deviation; the import handlers are additional deviations not in the candidate list.

---

## 2. ERROR FORMAT

**Canonical shape:** `{"error": {"code": "<machine_code>", "message": "<human message>"}}` — defined and enforced in `internal/adapter/httperr/httperr.go`:
- `envelope`/`APIError` structs: `httperr.go:39-48`.
- `Respond` (writes envelope) `httperr.go:51-53`; `Abort` (middleware, aborts chain) `httperr.go:57-59`.
- Stable snake_case codes: `httperr.go:22-37` (`validation_failed`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `rate_limited`, `internal_error`, `invalid_api_key`, `key_mode_mismatch`, plus domain codes `over_refund`, `invoice_not_paid`, `invoice_already_paid`).
- `CodeForStatus` maps runtime HTTP status → default code (`httperr.go:63-80`).
- Handler-side wrappers alias into this: `respond.go:35-43`, `response.go:27-49`.

**Driver detail is hidden on 500s.** `respondInternalError` (`respond.go:49-53`) logs the real `err` server-side (with method+path) and returns a fixed `"internal error"` body with code `internal_error`. The doc comment states this replaced 116 call sites that leaked SQL/driver detail. This is the *only* correct 500 path (`respond.go:45-48`). `/health` and `/version` apply the same info-hiding stance (`main.go:1594-1605`).

---

## 3. AUTHENTICATION

Three mechanisms, all in `internal/adapter/middleware/`:

1. **Bearer API key** — `AuthMiddleware` (`auth.go:143-168`). Keys formatted `rsk_live_…`/`rsk_test_…`; validated against tenants/api_keys via bcrypt (`resolveAPIKey` `auth.go:86-117`), with a SHA-256-keyed verified-key cache (5-min TTL, `auth.go:36-61`) to avoid per-request bcrypt. **Mode gate:** key `livemode` must equal server `serverLive`, else `401 key_mode_mismatch` (`auth.go:158-162`, message `auth.go:120-125`). Token extraction accepts `Bearer <t>` or bare token (`auth.go:66-76`). Dev bypass token `recurso_secret` only when `APP_ENV=development && ALLOW_DEV_BYPASS=true` (`auth.go:88-99`).

2. **Session cookie + API key (dual)** — `SessionOrAPIKeyMiddleware` (`auth.go:184-228`). Tries `recurso_session` cookie first (`domain.SessionCookieName`, `auth.go:190`), resolving via `SessionResolver` (implemented by `*service.AuthService`); on success sets `tenant_id`, `user_id`, `user_role`, `user`. Falls back to the API-key path if the cookie is missing/stale. **This is what guards the `/v1` group** (`main.go:1760`).

3. **Portal session** — `PortalAuthMiddleware` (`portal_auth.go:12-35`). Reads `portal_session` cookie or `X-Portal-Session` header; validates via `portalService.ValidateSession`; sets `portal_customer_id`. Guards the `/portal/api` group (`main.go:1735-1736`).

All three abort through the canonical envelope via `httperr.Abort`.

---

## 4. PAGINATION

Helpers in `internal/adapter/handler/pagination.go` — **three inconsistent conventions coexist**:

| Helper | Params | Default | Cap | Notes |
|---|---|---|---|---|
| `ParsePagination` (`pagination.go:19-61`) | page/per_page + limit/offset | per_page=50 | 250 | page-based; also honors raw limit/offset |
| `parsePageLimit` (`pagination.go:105-118`) | page + limit | 50 (`defaultPageLimit`, `pagination.go:95`) | 1000 (`maxPageLimit`, `pagination.go:98`) | Tier-1 lists (plans/subs/customers) |
| `parseLimitOffset`/`clampLimitOffset` (`pagination.go:68-86`) | limit + offset | caller-supplied `def` (house convention def=max=1000) | caller `max` | pure DoS-bound for previously-unbounded lists |

Usage: `parsePageLimit` on plans (`plan.go:153`), subscriptions list (`subscription.go:105`), customers (`customer.go:353`); `ParsePagination` on subscription events (`subscription.go:158`, capped 250). ~21 handler files use one of these helpers.

**Note:** CLAUDE.md still says a few lists "default to limit=10" — code has since raised the shared default to 50 (`pagination.go:92-95` comment documents the 10→50 change). Doc is stale.

**Deliberately unbounded billing sweeps:** internal worker/scheduler queries are intentionally not page-bounded. `GetSubscriptionsDueTomorrow` (precharge sweep) has **no LIMIT** (`subscription_repository.go:360-377`). Contrast the claim-based renewal/resume sweeps which ARE bounded via a `limit` arg: `ClaimDueForRenewal` `subscription_repository.go:499,511` (`LIMIT $2`), `ClaimDueForResume:555,562`, and other fixed `LIMIT 100` batches (`:457,:619`). So "billing sweeps unbounded" is true for the precharge notification sweep but the renewal/resume claims are batched — an inconsistency worth noting. **ASSUMPTION:** these unbounded internal sweeps are acceptable because they run server-side over one tenant-day's due rows, not on public list endpoints.

**Remaining unbounded public lists:** the bare-return import endpoints and any handler not in the helper-usage list still risk silent truncation; CLAUDE.md's standing guidance is to always pass an explicit limit.

---

## 5. VALIDATION

- **gin binding tags** are the primary edge validation. Money POSTs use `binding:"required,gt=0"`: `coupon.go:27` (discount_value), `mandate.go:29` (max_amount), `offline_payment.go:24,95` (amount).
- **New `internal/validate` package** (`validate.go`): registers gin binding tags `currency` and `country` (`validate.go:73-82`) backed by `golang.org/x/text` (CLDR/ISO), not a hand table (`validate.go:11-20`). `Currency` rejects `XXX`/`XTS` (`validate.go:31-33`); `Country` constrains to canonical 2-letter ISO-3166-1, filtering groupings like EU (`validate.go:39-48`). Also exposes `AmountNonNegative`/`AmountPositive` (`validate.go:53-56`), `Percentage`, `Email`. Registered once at startup: `main.go:1475`.
- **New `currency` tag adopters:** `advanced_billing.go:37`, `plan.go:27` (`required,currency`), `mandate.go:33` (`omitempty,currency`). Only 3 handler structs so far.
- **Ad-hoc `len==3` currency checks still live** in the service layer, not migrated to the tag: `service/catalog.go:50`, `service/euinvoice_ubl.go:198`, `service/pricing_simulator.go:83`, `service/wallet.go:121`. So currency validation is split between the new binding tag (handlers) and legacy `len()!=3` (services) — an inconsistency. No handler currently uses a `country` binding tag (grep returned none) despite the tag being registered.

---

## 6. IDEMPOTENCY

**HTTP idempotency middleware** — `internal/adapter/middleware/idempotency.go`:
- Applies to mutating methods only (POST/PUT/PATCH/DELETE, `idempotency.go:35-40,68-71`); GETs pass through.
- `Idempotency-Key` header is **recommended, not required** (`idempotency.go:47-48,74-78`) — requests without it are never replayed.
- Storage key scoped per tenant+method+path: `idem:<tenant>:<method>:<path>:<key>` (`idempotency.go:83-87`).
- Concurrency gate via atomic `Claim` (`idempotency.go:90`, port `internal/core/port/idempotency.go:19-27`): completed response → replay with `X-Idempotency-Hit: true` (`idempotency.go:99-108`); in-flight duplicate → `409 conflict` (`idempotency.go:110-113`).
- 5xx/panic release the reservation so the key stays retryable (`idempotency.go:126-136`).
- **Wired group-wide on `/v1`:** `main.go:1761`. So **every** money-mutating POST under `/v1` (subscriptions, charges, advance invoices, credit notes, usage events, e-invoice retries, gift purchases, offline payments, quote conversion) *accepts* a key but none *require* one (`idempotency.go:30-34` comment). Public checkout/webhook routes are outside `/v1` and thus outside this middleware.
- Backing stores: Redis (`internal/adapter/redis/idempotency_store.go`) and in-memory fallback (`internal/adapter/memory/idempotency_store.go`).

**Ledger idempotency** (independent, deeper layer) — `internal/service/ledger.go` + `internal/core/domain/ledger.go`:
- Uniqueness key is `(reference_id, code)` — unique index `uq_ledger_tx_reference_code` (`ledger.go:300`), so a replayed posting never double-posts (`domain/ledger.go:147,160,166`; `ledger.go:1236-1266`).
- For settle→reverse cycles the key extends to `(reference_id, code, occurrence)` — `Occurrence` is a `uint16` cycle counter (`domain/ledger.go:333-340`, design doc `docs/design-ledger-occurrence.md`). Computed as completed cycles: `ledger.go:379-390` (settle), reversal inherits the cash leg's occurrence (`ledger.go:480-487,702-703`), write-off/recovery cycle-aware (`ledger.go:498-535,604-643`). This is what makes re-collection after a return post fresh legs while a same-cycle duplicate dedups.

---

## 7. RATE LIMITING (ADR-001)

`internal/adapter/middleware/rate_limit.go` — fixed-window limiter, Redis-backed with in-memory fallback (`rate_limit.go:23-70`). Key namespaced per **scope**, keyed by tenant if present else IP (`rate_limit.go:30-33`). Emits `X-RateLimit-Limit`/`-Remaining` headers; 429 via `httperr.Abort` code `rate_limited` (`rate_limit.go:61-67`). The scope namespacing exists precisely because a shared key caused login lockouts (`rate_limit.go:16-22`).

Four scopes wired in `main.go`:
- **`api`** — global, `RATE_LIMIT_PER_MINUTE` (default 500), applied to *every* request (`main.go:1513-1517`).
- **`public`** — 20/min per IP, for brute-forceable endpoints (`main.go:1662`): checkout pay/verify, payments/order, waitlist, `auth/register|login|login/mfa|forgot-password|reset-password|verify-email`, oauth start/callback, saml, portal magic-link request/verify, accounting callback (`main.go:1673-1732`).
- **`session`** — 120/min, for per-page-load session endpoints (`main.go:1666`): `auth/logout`, `auth/me`, `verify-email/resend`, `oauth/providers` (`main.go:1698-1713`).
- **`expensive`** — 30/min per tenant, for CPU/IO-heavy paths (`main.go:1671`): all import preview/commit/compare routes (`main.go:1779-1789`), plus PDF/HTML renders and GL export per the comment (`main.go:1667-1671`).

---

## 8. OPENAPI DRIFT GATE

`cmd/api/openapi_drift_test.go` — `TestOpenAPISpecCoversRegisteredRoutes`:
- Statically scans `main.go` for `r|v1|portal|analytics` gin registrations (`openapi_drift_test.go:33-44`), maps group→prefix, converts `:param`→`{param}` (`:54-58`), and fails if any registered `method path` is absent from the embedded `openapi.yaml` (`:63-76`). Sanity floor: fails if <100 routes scanned (`:42-44`).
- Allowlist for intentionally-undocumented routes is currently **empty** (`:23-25`).

**Spec counts** (`cmd/api/openapi.yaml`): 246 path keys, **304 method-level operations**, **288 `operationId`s**. → **~16 operations lack an `operationId`.** The drift gate only checks path+method presence, **not** operationId, so those 16 pass CI silently — a documented-surface gap worth an audit note. **ASSUMPTION:** the 16 missing operationIds are on otherwise-documented paths (they don't fail drift), likely on secondary verbs.

---

## 9. VERSIONING

- **Single version prefix `/v1`** — one group only: `r.Group("/v1")` (`main.go:1759`); `analytics` is a sub-group `v1.Group("/analytics")` (`main.go:1857`). No `/v2` exists. Non-versioned routes (`/auth/*`, `/portal/*`, `/checkout/*`, `/webhooks/*`, `/health`, `/version`) sit at root.
- **Breaking-change pattern (GET→POST, one release overlap):** the portal magic-link verify shipped both verbs simultaneously — deprecated `GET /portal/auth/verify` (query-string token, "kept one release for links in flight") and preferred `POST /portal/auth/verify` (token in body, not logged) (`main.go:1731-1732`). This is the codified deprecation idiom: add the new verb, keep the old one one release for in-flight clients, then remove.

---

## 10. WEBHOOKS

### Inbound (gateway → Recurso), tenant-bound, dedup fail-closed
`internal/adapter/handler/webhook.go` + `webhook_stripe.go`/`webhook_razorpay.go`/`webhook_gocardless.go`. Routes are public, outside `/v1` (`main.go:1680-1688`), including per-connection (BYO) variants `/webhooks/{gw}/:connID`.
- **Tenant binding:** webhooks carry no tenant; the handler loads the referenced invoice/subscription and injects *its* tenant into context (`webhook_stripe.go:126-144,609-611`). BYO safety: if the webhook's connection tenant ≠ the invoice's tenant, it's ignored (`webhook_stripe.go:139-140,227-228,327-328`).
- **Dedup, fail-closed:** `InboundWebhookDedup` (`webhook.go:124-127`). `alreadyProcessed` (`webhook.go:146-162`): a lookup **error** fails CLOSED with `503` so the gateway retries (`webhook.go:151-154`) — deliberately preferring deferral over re-running non-idempotent side effects; only nil-store/empty-id fail open (`webhook.go:147-149`). Duplicate → `200 {"status":"duplicate ignored"}` (`webhook.go:158`). `markProcessed` is best-effort post-processing (`webhook.go:166-173`). (Note: the 503 body here is one of the raw `{"error":...}` envelope deviations from §1e.)

### Outbound (Recurso → customer endpoints), delivery worker with retries
`internal/adapter/worker/webhook_worker.go`:
- Polls every 10s (`webhook_worker.go:70-81`), **atomically claims** due deliveries with a 10-min lease, batch of 10 (`ClaimPending`, `:90`) so multiple Cloud Run instances can't double-POST (ADR-003, `:84-90`).
- **SSRF hardening:** no redirect following (`:48-51`), connect-time IP re-validation via `httpsafe.DialControl` to block rebind-to-private (`:52-59`), 10s timeout, 1KB response cap (`:140`).
- **Signing:** HMAC-SHA256 over payload with the endpoint secret → `X-Recurso-Signature`, plus `X-Recurso-Event-ID` (`:124-128,193-197`).
- **Retries:** exponential backoff `2^attempt * 30s` capped at 24h (`:172-177`), max 5 attempts then marked delivered-failed so it stops being picked up (`:162-169`). Transport failures retry with status 0 (`:132-134`).
- Demo mode parks the worker without draining the queue (`:33-35,64-68`).

---

## Cross-cutting audit summary (inconsistencies)
1. Success envelope is a convention, not enforced — bare-object deviations in auth + 3 import handlers + several action responses (§1a–c).
2. CLAUDE.md is stale on two points: dunning/cancel-flow are now `{data:}`-wrapped (§1d); default list limit is 50 not 10 (§4).
3. Three raw `{"error":"..."}` sites bypass the canonical envelope (§1e).
4. Three pagination conventions coexist; precharge sweep unbounded while renewal/resume sweeps are batched (§4).
5. Currency validation split: new `currency` binding tag (3 handlers) vs legacy `len()!=3` (4 services); registered `country` tag has zero adopters (§5).
6. ~16 OpenAPI operations lack operationIds and the drift gate doesn't catch them (§8).