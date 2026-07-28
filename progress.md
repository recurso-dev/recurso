# Progress log — overnight 2026-07-27 → 28

## Morning session (after founder wake-up)

- **QuickBooks verified live on prod** (sandbox company): OAuth → Connected →
  bulk customers + invoices pushed with QBO ids. Intuit gotcha: redirect-URI
  lists are **per keys tab** (Development vs Production) — the URI had been
  saved under Production while the creds are Development keys. Zero adapter
  code changes needed. `QBO_SANDBOX=true` picks the sandbox host.
- **#249** — GoCardless connect 500 fixed (provider CHECK constraint from
  000107 predated the third provider; migration 000150) + sync rows carry
  their provider, Integration column, filters, full error text.
- **#250** — provider rate limits honored (Xero 429 storm from the forced
  re-push): `doWithRateLimitRetry` waits Retry-After (capped), replays POST
  bodies, bounded retries; Xero + QuickBooks routed through it.
- **#251** — per-integration Sync buttons, server-side pagination (25/page),
  integration/status filters, debounced record-id search.
- **This PR** — sync rows resolve a human name (customer name / invoice
  number) at read time; search matches names too.


Running log of the autonomous overnight session. Newest first. Every item
merged on fully green CI (invariant harness + E2E + OpenAPI drift + frontend
gates) unless marked open.

## Completed

- **SDK sync (all three repos, merged)** — go #8 (mandate `Currency`,
  VPA omitempty), node #10 (schema regen), python #10 (full regen, 1.7.0,
  smoke 21/21): mandate currency rails, `gocardless` provider enum, async
  accounting-sync responses.
- **#245** — E2E step 6c: GoCardless webhook chain (fail-closed signature +
  fulfilment activates mandate and swaps BRQ→MD id). Compose gains a
  deterministic `GOCARDLESS_WEBHOOK_SECRET`. First run failed on
  heredoc-mangled JSON escapes — rebuilt with `jq -n` and re-verified against
  a local mock-gateway boot before repush.
- **Dependabot hygiene** — 3 alerts cleared by #243; the remaining
  react-router RSC-only advisory dismissed with justification (mirrors
  `.trivyignore`).
- **#244** — GoCardless `charged_back`/`late_failure` now reverse settlements
  via `ReverseSettledPayment` (paid→past_due + occurrence-aware code-19 leg,
  #209 machinery). No classification guards needed — GoCardless models
  merchant refunds as a separate resource, unlike Stripe's
  returns-as-refunds (#210).
- **#243** — react-router-dom 6.30 → 7.18: clears all three Dependabot
  alerts, zero source changes. v8 (fixes an RSC-only advisory) needs React
  ≥19.2 → documented `.trivyignore` + backlog item.
- **#242** — `docs/backlog.md`: ranked backlog, founder-blocked vs
  engineering-ready.
- **#241** — Mandates page: currency select (INR/EUR/GBP), VPA only for UPI,
  authorization-link copy toast, exponent-aware amounts.
- **#240** — GoCardless `payments.confirmed`/`paid_out` settle invoices
  (lookup by stored `PM…` id; `isGatewayPaymentID` accepts GC ids;
  `GetByGatewayPaymentIDPublic` consumed via capability assertion).
- **#239** — Manual accounting sync is async with per-tenant single-flight
  (202/`sync_already_running`; `sync_status: syncing`; 15-min cap). Closes
  the last known Cloudflare-~100s synchronous sweep.
- **#238** — GoCardless webhook receivers (platform + per-BYO-connection):
  HMAC-verified, per-event dedup (GC batches deliveries),
  `billing_requests.fulfilled` activates mandates + swaps BRQ→MD.
- **#237** — BYO GoCardless from the dashboard. Also fixed a latent
  cross-tenant leak: the resolver aliased the env router's
  `Extra`/`currencyOverrides` maps into every tenant build — now copied,
  with a regression test.
- **docs #25** — GoCardless setup + advanced guides (no longer
  "experimental"); **website #29** — integrations chip soon→BYO.

## Verification evidence (not testimony)

Runtime-verified on a fresh DB + real API boot with real sandbox billing
request: signed fulfilment webhook → mandate `active`, token swapped;
`payments.confirmed` → invoice paid + code-3 leg; `charged_back` →
past_due + exactly one code-19 leg; redelivered chargeback did **not**
double-reverse; bad signature → 401.

## Key decisions

- Accounting sync went **async** (not batched like CRM #231) because a forced
  full re-push has no natural batch boundary; single-flight per tenant.
- GoCardless chargebacks reuse the ACH reversal machinery **without**
  classification guards — a property of GoCardless's API (refunds are a
  separate resource), documented in the handler.
- react-router stays on v7 with a justified Trivy ignore rather than rushing
  a React 19 upgrade overnight.

## Open / next steps

- **Founder (morning)**: QuickBooks app creds (see `docs/backlog.md` P1.3);
  GoCardless webhook endpoint + `GOCARDLESS_WEBHOOK_SECRET` on Cloud Run
  (P1.4); telemetry deploy; `TRAFFIC_TOKEN`; demo-sandbox hosting; fix
  customer `bed15f4d…`'s Xero-invalid email.
- Engineering-ready: pagination-consistency sweep (design first — changing
  unbounded defaults silently truncates existing clients), interface-embedding
  mock cleanup, React 19 + react-router 8.

## Full-product verification (2026-07-28, gym-time sweep)

All six repos' gates green; 32-area fresh-boot API journey: 27 PASS, 3
findings fixed in this PR (EU e-invoice tenant-ctx 500, mandate guard 500→400,
Quotes raw-UUID customers), 4 PARTIALs backlogged (S1–S5 in docs/backlog.md).
Prod dashboard sweep clean (zero console errors). All seven third-party
integrations live-verified, incl. BYO GoCardless end-to-end (mandate
authorized on hosted page → webhook flipped it active). Full evidence:
docs/verification-2026-07-28.md.

## Hardening continued (2026-07-28)

Security audit clean; money-path audit → 2 fixes (#260 wallet double-credit
HIGH, #261 accounting currency-exponent MED). All five smoke findings closed:
S1 incremental card sync (#256), S2 nested billing_address (#259), S3 GET
subscription by id (#258), S4 lump-sum quote line (#264), S5 entity country
alias (#263). SDKs synced (node/go/python). Scorecard 8.5/10
(production-readiness.md); bugs BUG-001..006 in bugs-found.md.

Remaining backlog: pagination-consistency sweep (docs/design-pagination.md —
start with documenting actual defaults in OpenAPI, zero behavior risk),
accessibility pass (axe-core; lowest score 6), performance baseline + charts
lazy-load, telemetry deploy (#215, founder), mock-widening cleanup, React 19.
