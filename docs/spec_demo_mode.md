# Spec: Demo Mode & Hosted Sandbox (demo.recurso.dev)

> **Status: DRAFT — awaiting founder decisions (D1–D6)**

## Objective

Let anyone play with Recurso — create plans, customers, subscriptions,
invoices, meter usage — **within 10 seconds of clicking "Open live demo"
on recurso.dev**, with zero signup and zero risk. The same mode makes
`make demo` self-hosters safer and gives screenshots/demos a stable,
attractive data set.

Success looks like: a visitor clicks the hero button, lands in a seeded
dashboard, creates an invoice, sees it in the ledger — and nothing they do
can email a human, charge a card, reach a SaaS, or persist past the next
reset.

## Design

### DEMO_MODE=true (server)

One env flag that composes existing pieces:

- **Auto-seed on boot** when the database is empty — reuses `cmd/demo_seed`
  (12 customers, 9 subscriptions, 14 invoices…) plus a metering extension:
  2 billable metrics, charges on one plan, a wallet with balance, one
  usage alert — so v0.6.0 features demo well.
- **Forced-safe adapters** regardless of other env: mock payment gateways,
  console notifier (no SMTP/Twilio), no GSP/IRP, `RESIDENCY_MODE`
  irrelevant because every SaaS adapter is skipped. Outbound webhook
  deliveries are **disabled** (endpoints can be created and inspected;
  nothing egresses).
- **Blocked edges** (403 `demo_mode`): team invites/role changes, SSO
  config, password/email changes on the demo user, API-key rotation for
  the live mode, data-region changes, account deletion.
- **Demo session endpoint** — `POST /auth/demo` (only in DEMO_MODE):
  issues a dashboard session for the pre-seeded demo user, so the website
  button can deep-link straight into the dashboard
  (`https://demo.recurso.dev/?demo=1` → auto-login). The demo API key
  (`sk_test_12345`) is shown in a banner for curl users.
- **Dashboard banner**: "Demo environment — resets every hour. Data is
  public; don't paste anything real."

### Reset worker

In DEMO_MODE only: every `DEMO_RESET_INTERVAL` (default 1h) the worker
truncates tenant data and reseeds, logging the reset. Runs the same seed
path as boot, so drift is impossible. Active requests during reset get
clean errors (single transaction per table group; the demo is allowed a
seconds-long blip).

### docker-compose.demo.yml

`docker compose -f docker-compose.demo.yml up -d` on any box =
Postgres + API (DEMO_MODE=true) + dashboard behind the existing nginx
image, listening on :80. Founder points `demo.recurso.dev` at it.

### Website

Hero gains **"Open live demo"** as the primary CTA (self-hosting CTA
stays), pointing at `https://demo.recurso.dev/?demo=1`; the button ships
behind a config flag until the box is live so we never link to a 404.

## Tech stack / commands / structure

Unchanged (Go/Gin, React, compose). New files:
`internal/demo/demo.go` (mode + guards), `internal/adapter/worker/demo_reset_worker.go`,
`cmd/demo_seed` extension, `internal/adapter/middleware/demo_guard.go`,
`docker-compose.demo.yml`, dashboard banner component, website CTA.
Tests follow house style: table tests + httptest for the guard middleware,
worker test with fake clock, seed idempotency test.

## Boundaries

- **Always:** mock gateways + console notifier enforced in code (not by
  env convention) when DEMO_MODE=true; full suite green; guard tests for
  every blocked edge.
- **Ask first:** anything that would let the demo egress; per-visitor
  tenancy (scope change).
- **Never:** live gateway keys, SMTP creds, or GSP config honored in
  DEMO_MODE; demo endpoints compiled into non-demo behavior paths.

## Success criteria

1. `DEMO_MODE=true` boot on an empty DB → seeded dashboard reachable,
   `POST /auth/demo` returns a working session; with the flag off, that
   endpoint 404s and behavior is byte-identical to today.
2. In DEMO_MODE, attempts to invite a teammate / change SSO / rotate live
   keys 403 with code `demo_mode` (guard-tested).
3. No outbound network side effects: gateway calls hit mocks, emails go
   to console, webhook deliveries stay queued/disabled (asserted).
4. Reset worker wipes + reseeds on the interval; a post-reset login shows
   the pristine data set.
5. `docker compose -f docker-compose.demo.yml up` from a clean checkout
   serves the whole demo on :80.
6. Website CTA appears when enabled and deep-links into a logged-in
   dashboard.

## Founder decisions

| # | Question | Recommendation |
| --- | --- | --- |
| D1 | Tenancy | **One shared demo tenant** (per-visitor tenants later if abuse warrants) |
| D2 | Entry | **Auto-login via `/auth/demo`** + visible `sk_test_12345` for curl |
| D3 | Reset cadence | **Hourly** (`DEMO_RESET_INTERVAL=1h`) |
| D4 | Write access | **Full create/edit within the tenant** (that's the point); only the blocked-edges list is off-limits |
| D5 | Webhooks | **Creatable + inspectable, deliveries disabled** (no egress) |
| D6 | Website CTA | **Primary hero button, behind a flag until DNS is live** |

## Open questions

1. Hosting target — your VPS, Fly.io free tier, or Railway? (Affects
   nothing in code; compose file works on all.)
