# Tasks: MCP Server

Plan: `tasks/mcp-plan.md`. Spec: `docs/spec_mcp_server.md`. One PR per increment.

## Inc 1 — skeleton + auth + Tier-1 reads
- [ ] SDK spike: `go get github.com/modelcontextprotocol/go-sdk`; confirm real API via `go doc` (server, tool registration, Streamable HTTP handler, annotations). Adapt server.go to actual signatures.
  - Verify: a trivial "ping" tool registers + responds over Streamable HTTP.
  - Files: go.mod, cmd/mcp/main.go, internal/mcp/server.go
- [ ] `internal/mcp/client.go`: typed `/v1` client — key forwarding, `Idempotency-Key` on writes, `{error:{message}}` mapping.
  - Verify: unit tests vs httptest mock `/v1` (200 passthrough, 4xx→tool error, key forwarded).
  - Files: internal/mcp/client.go, internal/mcp/client_test.go
- [ ] `internal/mcp/tiers.go`: tier enum + annotations; Tier-3 gate scaffold (no tools yet).
  - Verify: registering a Tier-3 tool with opt-in=false hides it from tool-list (unit).
  - Files: internal/mcp/tiers.go, internal/mcp/tiers_test.go
- [ ] `internal/mcp/tools_read.go`: 14 Tier-1 read/simulate tools.
  - Verify: each tool's input schema validates; happy-path returns mock `/v1` body.
  - Files: internal/mcp/tools_read.go, internal/mcp/tools_read_test.go
- [ ] Contract test: every registered tool's path exists in `cmd/api/openapi.yaml`.
  - Files: internal/mcp/contract_test.go
- [ ] Acceptance: MCP client + test key lists customers + simulates charges; cross-tenant read fails closed.
  - Verify: `go build ./... && go test ./...` green.
- [ ] PR: "feat(mcp): agent-operable billing — Inc 1 (server + auth + read/simulate tools)".

## Inc 2 — Tier-2 writes (idempotent)
- [ ] `tools_write.go`: create/update customer, record usage (single+batch, deterministic transaction_id), create/update/send quote, create/update subscription. Idempotency-Key on all.
- [ ] stdio transport mode in cmd/mcp.
- [ ] Tests: idempotency-key present; same-key retry doesn't double-apply (mock asserts single effect).
- [ ] PR: Inc 2.

## Inc 3 — Tier-3 opt-in
- [ ] `mcp_connections` config (tenant → enabled + allowed tiers); migration; read at tool-list time.
- [ ] `tools_sensitive.go`: convert_quote_to_invoice, cancel_subscription, create_credit_note, wallet_top_up, add_subscription_charge, bill_usage_now (destructiveHint, gated).
- [ ] Dashboard toggle: enable MCP + choose tiers.
- [ ] Money-path review; NOT self-merged. PR: Inc 3.

## Inc 4 — deploy + docs
- [ ] Cloud Build target + `recurso-mcp` Cloud Run + `mcp.recurso.dev`.
- [ ] Docs page: connect Claude/ChatGPT with your key + tier explanation.
- [ ] E2E: read→simulate→create_customer→create_quote on docker compose.
- [ ] Website mention. PR: Inc 4.

## Parallel security fold-in (independent)
- [ ] Migrate `accounting_connections.access_token` plaintext → secretbox vault (own PR).
