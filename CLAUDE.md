# CLAUDE.md — working in this repo

Recurso: subscription billing platform. Go API (`cmd/api`, gin + Postgres via
golang-migrate, migrations auto-run on boot) + React dashboard (`frontend/`,
Vite + shadcn/Radix + Tremor + react-query). Deploys are automatic on merge to
main: Cloudflare Workers (dashboard, app.recurso.dev) and Google Cloud Build →
Cloud Run (API, api.recurso.dev).

## Commands

| What | Command |
|---|---|
| Backend build + tests | `go build ./... && go test ./...` |
| Postgres-backed tests (ledger/rev-rec/harness) | `TEST_DATABASE_URL=postgres://localhost:5432/<db>?sslmode=disable go test ./internal/service/ ...` (CI sets this; tests skip without it) |
| Replay one invariant-harness seed | `LEDGER_INVARIANT_SEED=<n> go test ./internal/service/ -run TestLedgerInvariants` |
| Frontend | `cd frontend && npm run lint && npm run build && npx vitest run` |
| E2E (full stack) | `docker compose up -d --build && ./scripts/e2e_test.sh` |

Never pipe lint to `tail`/`grep` when you rely on its exit code — a pipe
swallows the failure (this exact mistake shipped a broken button once).

## Hard gates (CI fails without these)

- **OpenAPI drift**: every registered route must exist in
  `cmd/api/openapi.yaml`. When a path already exists, merge the new verb under
  the existing key — a duplicate path key is invalid YAML.
- **Invariant harness**: any invoice-creating flow must post its ledger legs
  (see ADR-002) or randomized-sequence reconciliation fails CI. The E2E suite
  ends with the same zero-discrepancy gate.
- Frontend CI runs lint + build + vitest; Go pre-commit runs golangci-lint.

## Backend conventions

- Money is **minor units** (`int64`); subscription status is `"canceled"`
  (one L); coupon `discount_type` is `"percent"`/`"amount"`.
- List endpoints are inconsistent about pagination: the shared tier-1 default
  is now `limit=50` (raised from 10; `parsePageLimit` cap 1000), `ParsePagination`
  defaults 50/cap 250, some lists 50/100/200, and internal billing sweeps are
  deliberately unbounded. Always pass an explicit limit when you need the full
  set (silent truncation has bitten twice), and use `ParsePagination`/
  `clampLimitOffset` for new list endpoints. Never paginate a processing sweep.
- Nullable text columns scan through `sql.NullString`, never bare `string`.
- Optional service dependencies use nil-safe `Set*` wiring
  (`SetLedgerService`, `SetCreditApplier`, …) — follow that idiom.
- Workers over due rows use atomic claims, not locks (ADR-003).
- New migration = next sequential number in
  `internal/adapter/db/migrations/`; both `.up.sql` and `.down.sql`.
- Success responses wrap as `{data: ...}` by convention (not enforced
  middleware), so deviations exist — bare-object shapes in `auth.go` and the
  import handlers, action shapes (`{status}`/`{message}`), and a few
  `{data}`+sibling-key responses. (The old "dunning-campaign/cancel-flow are
  unwrapped" note is now STALE — those endpoints DO wrap `{data:}` in current
  code; only their delete responses stay `{status:"deleted"}`.) Three raw
  `{"error":...}` sites still bypass the canonical error envelope
  (`webhook.go`, `webhook_gocardless.go`, the founder metrics endpoint).

## Frontend conventions

- Patterns live in `src/components/patterns/` (DataTable, PageHeader,
  StatCard, EmptyState) and `src/components/ui/` (shadcn). Detail views and
  create/add forms are **right-side Sheets** (`sm:max-w-md`, header +
  description, pinned footer); confirmations use `ConfirmDialog`; row-level
  quick actions may stay dialogs.
- Never ask users for raw UUIDs — use the pickers
  (`CustomerSelect`/`CustomerName` in patterns, `usePlans`/`useSubscriptions`
  in `src/lib/useCustomers.js`). Those hooks are react-query backed and
  shared; caching contract is in ADR-005.
- When a page gains a new `endpoints.*` call, extend the corresponding test
  mock (`src/pages/__tests__/`) or the page's tests will hang on a missing
  method. Test wrappers need `QueryClientProvider` (retry: false).
- `lucide-react` is pinned at 0.294.0 — verify an icon exists before
  importing (`node -e "console.log('X' in require('lucide-react'))"`).

## Product & design source of truth

Before building UI, writing copy, designing an endpoint, or touching a money
path, read the relevant doc in `docs/` (index: `docs/README.md`): PRODUCT,
DESIGN, BRAND, UX_RULES, ANTI_PATTERNS, ACCOUNTING_PRINCIPLES, API_GUIDELINES,
WEBSITE, COMPETITORS. They are the durable philosophy; this file is the
mechanical how-to. The through-line: Recurso is accounting-first — every number
explainable, every event reversible, the books always reconcile.

## Decisions

Architectural rationale lives in `docs/decisions/` (ADR-001…006: rate-limit
scoping, ledger posting semantics, claim-based workers, one-off recognition,
layered caching, token-based accounting connections). Read the relevant ADR
before re-deciding any of those areas; supersede with a new ADR rather than
editing history.

## Autonomous overnight mode

When the founder invokes overnight/autonomous mode, follow
`docs/autonomous-mode.md` (their standing directive) — in short: act as the
whole engineering org, never stop after one task, work highest-ROI first,
self-review after every task, keep `docs/backlog.md` and `progress.md`
current, ship each increment as its own green-CI PR, and leave complete
handoff notes before stopping. Success = how much better the repo is when
the founder wakes up.
