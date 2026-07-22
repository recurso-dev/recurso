#!/usr/bin/env bash
# Boot smoke for the deployable MCP server: builds the binary, starts it, and
# checks the /health endpoint Cloud Run probes. The full protocol flow (client →
# Streamable HTTP → tool → /v1, tenant isolation, Tier-3 gating) is covered by
# the Go E2E in internal/mcp/e2e_test.go; this verifies the shipped artifact boots.
#
# Usage: ./scripts/mcp_smoke.sh
set -euo pipefail

cd "$(dirname "$0")/.."

PORT="${PORT:-8091}"
BIN="$(mktemp -d)/recurso-mcp"

echo "==> building cmd/mcp"
CGO_ENABLED=0 go build -o "$BIN" ./cmd/mcp

echo "==> starting server on :$PORT"
API_BASE_URL="http://127.0.0.1:9" PORT="$PORT" "$BIN" &
SRV_PID=$!
trap 'kill "$SRV_PID" 2>/dev/null || true' EXIT

# Wait for the port to accept connections.
for i in $(seq 1 30); do
  if (echo > "/dev/tcp/127.0.0.1/$PORT") >/dev/null 2>&1; then break; fi
  sleep 0.2
done

echo "==> GET /health"
BODY="$(curl -fsS "http://127.0.0.1:$PORT/health")"
if [ "$BODY" != "ok" ]; then
  echo "FAIL: /health returned '$BODY', expected 'ok'" >&2
  exit 1
fi

echo "PASS: MCP server boots and serves /health"
