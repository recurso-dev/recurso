# Progress log — overnight 2026-07-27 → 28

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
