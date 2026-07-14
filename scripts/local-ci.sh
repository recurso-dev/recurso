#!/usr/bin/env bash
# local-ci.sh — run the CI gates on this machine, mirroring
# .github/workflows/ci.yml, for when GitHub-hosted Actions is unavailable
# (e.g. an account spending-limit block). Spins throwaway Postgres + Redis,
# runs build / vet / lint / race-tests / vuln-scan, and cleans up.
#
# Usage:  scripts/local-ci.sh            # full run on the current worktree
#         SKIP_VULN=1 scripts/local-ci.sh
set -uo pipefail
cd "$(dirname "$0")/.."

# Go-installed tools (golangci-lint, govulncheck) live in GOPATH/bin, which a
# non-interactive shell may not have on PATH.
export PATH="$(go env GOPATH)/bin:${PATH}"

PG_PORT="${LOCAL_CI_PG_PORT:-55471}"
RD_PORT="${LOCAL_CI_RD_PORT:-63781}"
HEAD=/usr/bin/head; TAIL=/usr/bin/tail  # avoid shells that alias head/tail
fail=0
log="$(mktemp)"
step() { printf '\n\033[1m=== %s ===\033[0m\n' "$1"; }
ok()   { printf '\033[32mPASS\033[0m %s\n' "$1"; }
bad()  { printf '\033[31mFAIL\033[0m %s\n' "$1"; fail=1; }

if ! docker info >/dev/null 2>&1; then
  echo "docker is required (for throwaway Postgres + Redis)"; exit 2
fi

step "starting throwaway Postgres + Redis"
# Named + force-removed first so a leftover from an interrupted run can't hold
# the port (idempotent).
docker rm -f recurso-local-ci-pg recurso-local-ci-redis >/dev/null 2>&1
PG=$(docker run -d --rm --name recurso-local-ci-pg -e POSTGRES_PASSWORD=pw -e POSTGRES_DB=recurso_ci -p "${PG_PORT}:5432" postgres:16-alpine)
RD=$(docker run -d --rm --name recurso-local-ci-redis -p "${RD_PORT}:6379" redis:7-alpine)
cleanup() { docker stop "$PG" "$RD" >/dev/null 2>&1; rm -f "$log"; }
trap cleanup EXIT
for _ in $(seq 1 30); do docker exec "$PG" pg_isready -U postgres >/dev/null 2>&1 && break; sleep 1; done
for _ in $(seq 1 20); do docker exec "$RD" redis-cli ping 2>/dev/null | grep -q PONG && break; sleep 0.5; done
export TEST_DATABASE_URL="postgres://postgres:pw@localhost:${PG_PORT}/recurso_ci?sslmode=disable"
export TEST_REDIS_URL="redis://localhost:${RD_PORT}/0"
export APP_ENV=test
echo "postgres @ ${PG_PORT}, redis @ ${RD_PORT}"

step "go build ./..."
if go build ./... >"$log" 2>&1; then ok "build"; else bad "build"; "$TAIL" -20 "$log"; fi

step "go vet ./..."
if go vet ./... >"$log" 2>&1; then ok "vet"; else bad "vet"; "$TAIL" -20 "$log"; fi

step "golangci-lint run"
if command -v golangci-lint >/dev/null 2>&1; then
  if golangci-lint run >"$log" 2>&1; then ok "lint"; else bad "lint"; "$TAIL" -30 "$log"; fi
else
  echo "(golangci-lint not installed — skipping)"
fi

step "go test -race ./... (with Postgres + Redis)"
if go test -race ./... >"$log" 2>&1; then
  ok "tests ($(grep -c '^ok' "$log") packages)"
else
  bad "tests"; grep -E "^(FAIL|--- FAIL|panic:)" "$log" | "$HEAD" -40
fi

step "govulncheck ./..."
if [ "${SKIP_VULN:-0}" = "1" ]; then
  echo "(SKIP_VULN=1 — skipping)"
elif command -v govulncheck >/dev/null 2>&1; then
  if govulncheck ./... >"$log" 2>&1; then ok "govulncheck"; else bad "govulncheck"; "$TAIL" -30 "$log"; fi
else
  echo "(govulncheck not installed — 'go install golang.org/x/vuln/cmd/govulncheck@latest' — skipping)"
fi

printf '\n'
if [ "$fail" -eq 0 ]; then printf '\033[32m\033[1mLOCAL CI: PASS\033[0m\n'; else printf '\033[31m\033[1mLOCAL CI: FAIL\033[0m\n'; fi
exit "$fail"
