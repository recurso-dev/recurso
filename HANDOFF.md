# HANDOFF

Single entry point for picking up work on Recurso — from any machine or account.
Everything here lives in the repo, so it travels with a `git clone`.

## Current state (2026-08-03)

- **Released:** `v0.9.0 — The paper-trail release` (Latest). See the
  [GitHub release](https://github.com/recurso-dev/recurso/releases/tag/v0.9.0)
  and `CHANGELOG.md`. (v0.8.0 "correctness" and v0.7.0 "bank-debit" preceded
  it — v0.8.0 and v0.9.0 both shipped 2026-08-03.)
- **Production:** healthy on the released code (verified post-deploy: /health
  green, migrations 000156–000160 applied). Deploys are automatic on merge to
  `main` (Cloud Run = API `api.recurso.dev`, Cloudflare Workers = dashboard
  `app.recurso.dev`), *not* on the tag — the tag is the versioned milestone.
- **No open correctness bugs.** Two full audit waves (#413–#432): the revrec
  downgrade family, a coupon-proration money-out arbitrage, FX exponent
  corruption, PIA aggregation mis-billing, portal per-currency totals — all
  fixed with failing-first oracles; the safety net now also covers couponed
  subscriptions and 128 randomized seeds.
- **Statutory credit-note documents** (#428): credit notes store their tax
  breakdown (migration 000160) and the downloadable document renders a
  GST-grade CDN; the detail sheet mirrors it (#431).

## Where to look first

| File | What it is |
|---|---|
| `docs/backlog.md` | **The source of truth for "what's left."** Ranked by ROI; done items struck through with their PR #, remaining items spelled out with impact/effort/notes. |
| `progress.md` | Session-by-session narrative log (newest first). The 2026-07-31 entry summarizes the last big push. |
| `CHANGELOG.md` | Keep-a-Changelog; `[Unreleased]` accumulates the next release. |
| `CLAUDE.md` | Repo conventions, commands, and gotchas — **read before making changes.** |
| `docs/decisions/` | ADRs (ledger posting, claim-based workers, one-off recognition, layered caching, …). Read the relevant one before re-deciding. |

## What's remaining (none are open bugs)

All tracked in `docs/backlog.md`. In short:

- **Needs a product/policy decision:** L2 (should a pause freeze usage
  accrual?), S4-remaining (require idempotency keys on money POSTs — the
  fail-open webhook-dedup half is DONE, #425).
- **Latent / unreachable:** W2, W3 (documented, by-design or currently
  unreachable). B2 is DONE (#428 — its old "no consumer" note was stale).
- **Engineering hygiene (small):** backlog #14 (interface-embedding test
  mocks), #15 (unwrapped dunning/cancel responses — breaking, batch with v2),
  P2 #11/#12 (gift-cancel & wallet-close UI edges, dunning alert edit UI).
- **Founder-credential-blocked:** QuickBooks OAuth, GoCardless webhook
  registration, telemetry deploy, `TRAFFIC_TOKEN`, real Peppol AP creds, demo
  sandbox hosting, and fixing one customer's Xero-invalid email.

## The ledger safety net (how correctness is protected)

Any invoice- or credit-note-creating flow **must post its double-entry ledger
leg**, or reconciliation fails. This is enforced two ways:

- **Invariant harness** — `internal/service/ledger_invariant_pg_test.go`
  (`TestLedgerInvariants_RandomizedBillingSequences`) drives randomized
  sequences of the **real services** (new-sub, up/downgrade, one-off, cancel,
  quote conversion, gift purchase, trial conversion) through the reconciler and
  asserts zero discrepancy after every step. Adding a new invoice-creating
  service path? Add an op here.
- **Reconciler** — `ReconciliationService.Run` checks invoice (Code-1) and
  payment (Code-3) completeness, credit-note leg completeness, orphans, trial
  balance, and abnormal account signs.

Run it locally (needs Postgres):
```
createdb recurso_dev_test
TEST_DATABASE_URL=postgres://localhost:5432/recurso_dev_test?sslmode=disable \
  go test ./internal/service/ -run TestLedgerInvariants
```

## Commands

| What | Command |
|---|---|
| Backend build + tests | `go build ./... && go test ./...` |
| PG-backed tests | `TEST_DATABASE_URL=postgres://localhost:5432/<db>?sslmode=disable go test ./internal/service/ …` |
| Frontend | `cd frontend && npm run lint && npm run build && npx vitest run` |
| E2E (full stack) | `docker compose up -d --build && ./scripts/e2e_test.sh` |

## Cutting a release

Tag-triggered (`.github/workflows/release.yml` runs on `v*`):

1. Promote `CHANGELOG.md` `[Unreleased]` → `[x.y.z] - <date>` (via a PR; merge it).
2. `git tag -a vX.Y.Z origin/main -m "…"` && `git push origin vX.Y.Z`.
3. The workflow runs Build & Test → pushes `ghcr.io/recurso-dev/recurso:vX.Y.Z`
   → creates the GitHub Release (auto notes + container-pull body). Then
   `gh release edit vX.Y.Z --title "…" --notes-file …` for the themed title +
   curated highlights (matching prior releases).

## Money conventions (bite people who forget)

- Money is **minor units** (`int64`); currency **exponent matters** (JPY=0,
  KWD/BHD=3, default=2). Backend: `domain.CurrencyExponent`/`FormatMoney`/
  `MinorToMajor`. Frontend: `toMinorUnits`/`fromMinorUnits`/`formatCurrency` in
  `lib/utils.js` (Intl-derived). Never hardcode `/100`.
- Subscription status is `"canceled"` (one L); coupon `discount_type` is
  `"percent"`/`"amount"`.
- New DB migration = next sequential number in
  `internal/adapter/db/migrations/` (both `.up.sql` and `.down.sql`).

---

*This repo is the durable, account-portable record. Local Claude Code memory and
transcripts (`~/.claude/projects/…`) stay on the machine that created them.*
