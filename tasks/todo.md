# Tasks: Demo Mode & Hosted Sandbox

Spec: `docs/spec_demo_mode.md` (APPROVED, D1–D6 accepted) · Plan: `tasks/plan.md`

## Phase 1 — Safety core

- [x] T1: internal/demo package + forced-safe wiring (M)
  - Acceptance: `demo.Enabled()` reads DEMO_MODE; when true, main.go
    constructs mock gateways, console notifier, nil GSP/IRP, disables
    webhook deliveries and CRM/S3/accounting egress regardless of env;
    boot log announces demo mode. Flag off = byte-identical behavior.
  - Verify: unit test on wiring decisions; full suite green.
  - Files: internal/demo/demo.go(+_test), cmd/api/main.go,
    internal/adapter/worker/webhook_worker.go

- [x] T2: demo guard middleware (S)
  - Acceptance: POST/PUT/DELETE on team, SSO settings, live-key rotation,
    data-region, account-delete routes 403 `{code: demo_mode}`; everything
    else untouched; wired on /v1 + /auth subsets only when enabled.
  - Verify: httptest table over blocked + allowed routes.
  - Files: internal/adapter/middleware/demo_guard.go(+_test), cmd/api/main.go

### Checkpoint 1: safety proven (suite green, guards tested, off = identical)

## Phase 2 — Experience

- [x] T3: seed extension for v0.6.0 showcase (M)
  - Acceptance: demo seed also creates 2 billable metrics, graduated
    charges on one plan, usage events, a funded wallet (+promo credit),
    one usage alert, a commitment; seed is idempotent (re-run = no dupes).
  - Verify: seed test against pg (skips without TEST_DATABASE_URL) or
    fake-repo unit path; make demo output lists new objects.
  - Files: cmd/demo_seed/*, seed helpers

- [x] T4: /auth/demo + dashboard auto-login + banner (M)
  - Acceptance: POST /auth/demo (only when enabled) creates a session for
    the seeded demo user; dashboard ?demo=1 auto-calls it and lands
    logged in; banner shows reset cadence + sk_test_12345; endpoint 404s
    when disabled.
  - Verify: handler test (enabled/disabled); frontend build + vitest.
  - Files: internal/adapter/handler/auth demo bits, cmd/api/main.go,
    frontend (App/banner)

- [x] T5: reset worker (M)
  - Acceptance: every DEMO_RESET_INTERVAL (default 1h) tenant data is
    truncated and reseeded via the same boot path; logs each reset; only
    runs when enabled.
  - Verify: worker test with injected clock/spy seeder; pg reset test
    optional.
  - Files: internal/adapter/worker/demo_reset_worker.go(+_test), main.go

### Checkpoint 2: fresh boot → seeded → auto-login → reset restores pristine

## Phase 3 — Delivery

- [x] T6: docker-compose.demo.yml + docs (S)
  - Acceptance: one command serves Postgres+API(DEMO_MODE)+dashboard on
    :80 from a clean checkout; docs page documents self-hosting the demo;
    CHANGELOG entry.
  - Verify: compose config validates; local smoke if Docker available.
  - Files: docker-compose.demo.yml, recurso-docs page, CHANGELOG.md

- [x] T7: website "Open live demo" CTA behind flag (S)
  - Acceptance: VITE_DEMO_URL set → primary hero CTA appears linking
    ?demo=1; unset → current hero unchanged.
  - Verify: website builds both ways.
  - Files: recurso-website Hero.jsx (+ env plumbing)

### Checkpoint 3: compose end-to-end; CTA dark until DNS (founder: point demo.recurso.dev)
