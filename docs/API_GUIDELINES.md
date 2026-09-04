# Recurso — API Guidelines

> **Code-derived.** Every statement cites a file; implementation wins. §1–10 are
> the contract as implemented; the audit callouts flag real inconsistencies.
> Recurso is API-first; the dashboard is one client.

## 1. Response envelope

Success payloads wrap as `{"data": ...}`. It is a **convention, not enforced
middleware** — two helpers coexist (`internal/adapter/handler/response.go:52-71`
`RespondSuccess`/`RespondList`; most handlers inline `c.JSON(status,
gin.H{"data": ...})`). Used in 53 handler files.

**Deviations (audit):**
- Bare-object success responses: `auth.go` (Register `:108`, Login `:152`, MFA
  gate `:144`), the three import handlers (`stripe_import.go:48,74,93` etc.),
  `billing.go:113`, `dispute.go:110`, `portal_api.go:193,279`,
  `gift_handler.go:54`.
- Action responses (`{status:"deleted"}`, `{message:...}`,
  `{success,message}`): dunning-campaign/cancel-flow delete, einvoice, auth
  flows.
- `{data:}` + sibling keys: `gst.go`, `einvoice.go`, `eu_einvoice.go` add
  `gov_schema`/`message`.

## 2. Errors

Canonical `{"error":{"code","message"}}` — defined in
`internal/adapter/httperr/httperr.go` (`envelope`/`APIError` `:39-48`, `Respond`
`:51`, `Abort` `:57`, stable snake_case codes `:22-37`). **Driver detail is
hidden on 500s:** `respondInternalError` (`respond.go:49-53`) logs the real error
server-side and returns a fixed `internal_error` body.

**Deviations (audit):** three raw `{"error":"..."}` sites bypass the envelope —
`webhook.go:153`, `webhook_gocardless.go:87`, `routes_public.go` (founder endpoint).

## 3. Authentication

Three mechanisms (`internal/adapter/middleware/`): Bearer API key
(`AuthMiddleware`, `auth.go:143`; `rsk_live_`/`rsk_test_`, bcrypt + 5-min
SHA-256 cache; mode mismatch → `401 key_mode_mismatch`); session-or-key
(`SessionOrAPIKeyMiddleware`, `auth.go:184`, guards `/v1`); portal session
(`portal_auth.go:12`, `portal_session` cookie or `X-Portal-Session`). All abort
through the canonical envelope.

## 4. Pagination

`internal/adapter/handler/pagination.go` — **three conventions coexist:**
`ParsePagination` (page/per_page, default 50, cap 250), `parsePageLimit` (default
50, cap 1000; tier-1 lists), `parseLimitOffset`/`clampLimitOffset` (DoS-bound,
house convention def=max=1000). Bound every *display* list; document defaults in
OpenAPI.

**Never paginate a billing/processing sweep** — a paged read there silently
drops work. The precharge sweep (`GetSubscriptionsDueTomorrow`,
`subscription_repository.go:360`) is unbounded by design; renewal/resume claims
ARE batched (`ClaimDueForRenewal :499`).

## 5. Validation

gin binding tags at the edge. Money POSTs use `binding:"required,gt=0"`
(`coupon.go:27`, `mandate.go:29`, `offline_payment.go:24,95`). The new
`internal/validate` package registers `currency`/`country` tags backed by
`golang.org/x/text` (rejects `XXX`/`XTS`), wired at `main.go:1475`.

**Audit:** currency validation is **split** — the new `currency` tag on 3
handlers (`advanced_billing.go:37`, `plan.go:27`, `mandate.go:33`) vs legacy
`len==3` in 4 services (`catalog.go:50`, `euinvoice_ubl.go:198`,
`pricing_simulator.go:83`, `wallet.go:121`). The registered `country` tag has
zero adopters yet.

## 6. Idempotency

**HTTP** (`middleware/idempotency.go`): applies to mutating methods on `/v1`
(wired in `routes_v1.go`); `Idempotency-Key` is *recommended, not required*; keyed
`idem:<tenant>:<method>:<path>:<key>`; atomic `Claim` → replay (`X-Idempotency-Hit`)
or `409` on in-flight duplicate; 5xx/panic release the reservation. **Ledger**
(deeper layer): unique `(reference_id, code)`, extended to `(reference_id, code,
occurrence)` for settle→reverse cycles (`domain/ledger.go:333`,
`docs/design-ledger-occurrence.md`).

## 7. Rate limiting (ADR-001)

`middleware/rate_limit.go` — fixed-window, Redis + in-memory fallback, keyed per
**scope** by tenant-or-IP. Four scopes (`main.go`): `api` (global 500/min),
`public` (20/min, brute-forceable auth/checkout), `session` (120/min,
per-page-load), `expensive` (30/min per tenant — import commit/compare, PDF/GL
renders).

## 8. OpenAPI drift gate

`cmd/api/openapi_drift_test.go` fails CI if any registered `method path` is
absent from the embedded `openapi.yaml`. Spec has 263 paths / 323 operations /
323 operationIds → every operation carries an operationId (the drift gate
doesn't check that, so keep it so). Adding a verb to an existing path merges
under the existing key.

## 9. Versioning

Single `/v1` group (`routes_v1.go`); no `/v2`. Root-level unversioned:
`/auth/*` (`routes_auth.go`), `/portal/*` (`routes_portal.go`), `/checkout/*`,
`/webhooks/*`, `/health`, `/version` (`routes_public.go`).
**Breaking-change idiom:** add the new verb, keep the old one one release for
in-flight clients, then remove — as done for portal magic-link (`GET`→`POST
/portal/auth/verify`, both live in `routes_portal.go`).

## 10. Webhooks

**Inbound** (`handler/webhook*.go`, public, outside `/v1`): tenant-bound (load
the referenced invoice, inject its tenant; a foreign BYO-connection tenant is
ignored). Dedup **fail-closed** — a lookup error returns `503` so the gateway
retries (`webhook.go:151`). **Outbound** (`worker/webhook_worker.go`): atomic
claim (10-min lease, batch 10), SSRF-hardened (no redirects, connect-time IP
re-check), HMAC-SHA256 signature, exponential backoff `2^n·30s` capped 24h, max 5
attempts.

## Source of truth

- **Code:** `internal/adapter/handler/{httperr,respond,response,pagination}.go`,
  `internal/adapter/middleware/{auth,idempotency,rate_limit}.go`,
  `internal/validate/`, `cmd/api/{main.go,routes_public.go,routes_auth.go,routes_portal.go,routes_v1.go,openapi.yaml,openapi_drift_test.go}`.
- **ADRs:** ADR-001 (rate-limit scoping), ADR-002 (ledger posting), ADR-003
  (claim-based workers), ADR-006 (token connections).
- **Evidence file:** `docs/evidence/api-contract.md`.
- **Related:** `ARCHITECTURE.md`, `ACCOUNTING_PRINCIPLES.md`,
  `DOCUMENTATION_RULES.md`.
