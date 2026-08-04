# Recurso — API Guidelines

> The contract the public API keeps. Recurso is API-first; the dashboard is one
> client. Grounded in the conventions in `CLAUDE.md` and `cmd/api/openapi.yaml`.

## Shape

- **REST + JSON.** Resources are nouns, verbs are HTTP methods. No destructive
  GETs, ever.
- **Envelope:** successful responses wrap data as `{ "data": ... }`. (A few
  legacy endpoints are unwrapped — dunning-campaign and cancel-flow — documented
  quirks; new endpoints wrap. Reducing the drift is tracked in REMEDIATION.md.)
- **Errors:** canonical `{ "error": { "code": "...", "message": "..." } }` via
  the shared `httperr` helpers. The message is human-actionable; driver detail
  is never leaked. No raw `gin.H{"error": ...}`.
- **Every route is in `openapi.yaml`** — a hard CI gate. Adding a verb to an
  existing path merges under the existing key (a duplicate path key is invalid
  YAML). Every operation has an `operationId`.

## Money & correctness

- Amounts are `int64` **minor units**; currency is an ISO-4217 code (validated —
  `internal/validate`). Reject a non-ISO or reserved (XXX/XTS) code at bind.
- Money-moving POSTs validate amounts (`> 0` / `>= 0` as appropriate) at the
  binding layer **and** the service layer (defense in depth).
- **Idempotency:** money-moving POSTs accept an idempotency key; settlement and
  ledger posting are idempotent so a redelivered webhook can't double-post.

## Pagination

- Use `ParsePagination` / `clampLimitOffset`. Bound every *display* list; clamp
  abusive values. Document the default and cap in OpenAPI.
- **Never paginate a processing/billing sweep** — that silently drops work. Bound
  what's displayed, not what's computed.

## Auth & tenancy

- Bearer API key or session cookie; live vs test keys (`rsk_live_`/`rsk_test_`)
  must match server mode.
- **Tenant scoping is mandatory** on every query — a handler resolves the tenant
  from auth and every repository call is scoped by it. IDOR is a release blocker.
- BYO credentials (gateways, tax, CRM, storage) are sealed in the vault, never
  returned, and SSRF-guarded (no tenant-controlled internal URLs).

## Rate limiting (ADR-001)

- Scoped buckets: `api` (global), `public` (auth/brute-force), `session`,
  `expensive` (import commit/compare, PDF/GL renders). Different limits use
  different scopes — a shared key makes the strictest limiter judge the total.

## Observability

- Every request carries a `request_id`; it flows on the context so service logs
  (`slog.*Context`) are stamped with `request_id` / `tenant_id` / `user_id`. A
  production incident must be reconstructable from a trace.

## Webhooks

- Delivery is retried with backoff; consumers must be idempotent. Inbound
  gateway webhooks are tenant-bound (a foreign connection's event is ignored)
  and dedup-guarded fail-closed.

## Versioning & compatibility

- Breaking changes are additive-first; when a shape must change, keep the old
  path one release (see the portal magic-link GET→POST migration). SDKs
  (Go/Node/Python) are generated from the OpenAPI spec — keep it accurate.

## Related

- `CLAUDE.md` (repo conventions), `cmd/api/openapi.yaml` (the spec)
- `ACCOUNTING_PRINCIPLES.md` — the money invariants the API must not break
- `ANTI_PATTERNS.md` — never silently retry money; never leak state
