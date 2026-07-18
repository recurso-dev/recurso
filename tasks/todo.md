# Tasks: Lago Parity Program

Spec: `docs/spec_lago_parity.md` · Plan: `tasks/plan.md`

## Track A — honest flags

- [x] A1a: Billing-cycle scheduler with at-most-once claims
  - Acceptance: due ACTIVE non-mandate subscriptions get a renewal invoice
    (flat + metered) and an anchor-preserving period advance; concurrent
    ticks cannot double-process; `cancel_at_period_end` subs get final
    rating then cancel; `BILLING_CYCLE_INTERVAL=0` disables.
  - Verify: `go test ./internal/scheduler/ -run BillingCycle` incl. a
    concurrency claim test; full suite green.
  - Files: internal/scheduler/billing_cycle.go(+_test), cmd/api/main.go

- [x] A1b: Payment attempt on scheduler-generated invoices
  - Acceptance: invoice charged via stored gateway method when present;
    failure leaves invoice open for dunning (no crash, no retry storm).
  - Verify: scheduler test with failing/succeeding fake gateway.
  - Files: internal/scheduler/billing_cycle.go, internal/service (reuse
    existing charge path)

- [x] A2: Metered lines on mandate-debit invoices
  - Acceptance: mandate debit invoice includes rated usage lines; retry of
    the same cycle produces no duplicate lines (cycle-key + rating claim).
  - Verify: `go test ./internal/service/ -run Mandate` extended cases.
  - Files: internal/service/mandate.go(+_test)

- [x] A3a: Node SDK metering methods (recurso@1.3.0)
  - Acceptance: billableMetrics CRUD, plans.setCharges/getCharges,
    subscriptions.usageAmount, usage.record properties — typed, tested.
  - Verify: `npm run typecheck && npm test` in recurso-node.
  - Files: ../recurso-node/src/*, test/*, package.json

- [x] A3b: Python SDK regen + release prep (recurso 1.2.0)
  - Acceptance: regenerated from synced OpenAPI; new endpoint modules
    importable; version bumped.
  - Verify: pytest in recurso-python.
  - Files: ../recurso-python/*

- [x] A3c: Go SDK metering methods (v1.1.0)
  - Acceptance: same surface, stdlib-only, APIError semantics kept.
  - Verify: `go test ./...` in recurso-go; tag prepared.
  - Files: ../recurso-go/*

## Track B — commerce

- [x] B1a: Wallet domain + migrations + repositories
  - Acceptance: wallets (customer+currency unique) and wallet_transactions
    (append-only, balance_after) tables + ports + pg repos.
  - Verify: build + repo tests; migration up/down clean.
  - Files: internal/core/domain/wallet.go, core/port, adapter/db(+migrations)

- [x] B1b: WalletService — top-up, expiry, ledger postings
  - Acceptance: top-up via offline payment or gateway produces receipt +
    ledger legs (DR Cash / CR Customer Credit); promotional top-ups carry
    expiry; expired balance excluded from available.
  - Verify: service tests incl. expiry-ordering table tests.
  - Files: internal/service/wallet.go(+_test), service/ledger.go

- [x] B1c: Wallet drain on invoice generation (D3 ordering)
  - Acceptance: after invoice commit: wallet → adjustment credits →
    gateway; drains recorded as transactions + ledger legs; invariants
    (`Total == Subtotal+Tax`, Σ lines) untouched; zero-balance no-op.
  - Verify: invoice tests extended; payment-order test.
  - Files: internal/service/invoice.go, wallet.go(+_test)

- [x] B1d: Auto-recharge
  - Acceptance: balance < threshold triggers top-up via stored method in
    the billing-cycle sweep; no method → notify tenant+customer, no crash.
  - Verify: scheduler test with fake gateway.
  - Files: internal/scheduler/billing_cycle.go, service/wallet.go

- [x] B1e: Wallet API + OpenAPI + docs
  - Acceptance: POST/GET /v1/wallets, POST /v1/wallets/{id}/top-up,
    GET /v1/wallets/{id}/transactions; docs page; spec synced.
  - Verify: handler tests; YAML valid.
  - Files: adapter/handler/wallet.go, cmd/api/{main.go,openapi.yaml},
    ../recurso-docs

- [x] B2a: Commitments + true-up line
  - Acceptance: per-period minimum on subscription; shortfall line fills
    gap at period close (taxed at plan HSN); exactly-at-commitment adds
    nothing; preview shows projected true-up.
  - Verify: boundary table tests; invariant tests.
  - Files: domain/commitment.go, adapter/db(+migration), service/invoice.go,
    service/metering.go, handler, openapi.yaml

- [x] B3a: Usage alerts + sweep + webhook
  - Acceptance: absolute/%-thresholds per subscription+metric; fires once
    per threshold per period via webhook `usage.alert.triggered` + email/
    in-app; dedup proven.
  - Verify: evaluation table tests + dedup test.
  - Files: domain/usage_alert.go, adapter/db(+migration), service,
    scheduler, handler, openapi.yaml, ../recurso-docs

## Track C — scale + trust

- [x] C1: Batch ingestion + transaction_id idempotency + covering index
  - Acceptance: POST /v1/usage/events/batch ≤500 items with per-item
    results; duplicate (subscription, transaction_id) collapses to the
    original; covering index on (subscription_id, dimension, timestamp).
  - Verify: handler+repo tests incl. duplicate collapse.
  - Files: adapter/handler/usage.go, service/usage.go, adapter/db
    (+migration), openapi.yaml

- [x] C2: Append-only audit log
  - Acceptance: config-grade mutations (plans, charges, metrics, coupons,
    entitlements, webhooks, team, wallets, commitments, alerts) write
    audit rows; GET /v1/audit-logs filters by entity/actor/time; table
    rejects UPDATE/DELETE.
  - Verify: sweep test over mutating routes; handler tests.
  - Files: domain/audit.go, adapter/db(+migration), service/audit.go,
    handlers, cmd/api, frontend page (follow-up), openapi.yaml

## Cross-cutting (per track close)

- [ ] Docs + OpenAPI sync + CHANGELOG Unreleased entry per track
- [ ] Dashboard UI: plan-charges editor, Wallets page, AuditLog page
      (after B1e / C2; may batch)

## Track D — integration parity with Lago (added 2026-07-18, founder directive)

Reference surface: getlago.com integrations + doc.getlago.com. Recurso
already has: Stripe, Razorpay (Lago lacks it), QuickBooks, Xero, Tally,
TaxJar, VIES, SMTP/Twilio, webhooks. Gaps vs Lago, grouped:

- [x] D1: Payments — Adyen + GoCardless adapters shipped EXPERIMENTAL
      (httptest-verified; sandbox verification founder-gated)
- [x] D2: Accounting — NetSuite SuiteTalk adapter shipped EXPERIMENTAL
- [x] D3: Tax — Avalara AvaTax provider shipped EXPERIMENTAL; Anrok deferred (no customer signal)
- [ ] D4: CRM — HubSpot + Salesforce (customer/subscription/invoice sync)
- [x] D5: S3 GL export sink shipped EXPERIMENTAL; Segment events + Airbyte/ClickHouse defer
- [x] D6: Automation — Zapier/n8n docs shipped (rides signed webhooks)
- [x] D7: Auth — Okta SAML recipe documented
  - Note: all egress adapters must respect RESIDENCY_MODE guards.
  - Sequencing after C2; each integration is its own spec + PR.

## Cross-cutting sweep (docs / SDKs / website / dashboard) — founder directive

- [x] S1: Docs — commitments, usage-alerts, batch/idempotency pages shipped
- [x] S2: SDKs — Node 1.4.0, Go v1.2.0, Python 1.3.0 (publish gated on founder)
- [x] S3: Website — features + announcement + honest Lago comparison column
      config, AuditLog page (with C2)
