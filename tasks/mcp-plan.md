# Plan: MCP Server (agent-operable billing)

Spec: `docs/spec_mcp_server.md` (decisions D1–D4 locked 2026-07-22).
Task list: `tasks/mcp-todo.md`. **Namespaced `mcp-*` — do NOT touch `tasks/plan.md`/`todo.md` (Demo Mode).**

## Architecture (locked)

```
MCP client (Claude/ChatGPT)
   │  Streamable HTTP, Authorization: Bearer rsk_...
   ▼
cmd/mcp  (recurso-mcp Cloud Run svc, mcp.recurso.dev)
   │  forwards caller key + Idempotency-Key
   ▼
/v1 HTTP API  (existing, tenant-scoped, idempotent)
```

The MCP server holds **no DB connection and no service structs** — it is a curated,
tier-gated, agent-ergonomic facade over `/v1`. Tenant isolation is the caller's key; we
never hold a service-wide credential.

## Dependency graph & order

1. **SDK spike (blocking, do first):** `go get github.com/modelcontextprotocol/go-sdk`,
   confirm the *actual* API (server construction, tool registration signature, Streamable
   HTTP handler, annotations struct) via `go doc` before writing any tool code. The spec's
   code sketch is illustrative — real signatures come from this step. If the official SDK's
   API differs materially, adapt `internal/mcp/server.go` to it (thin adapter, tools unaffected).
2. **Typed `/v1` client** (`internal/mcp/client.go`) — depends on nothing but net/http.
   Key forwarding, `Idempotency-Key` generation for writes, `{error:{message}}` → error mapping.
   Testable against an httptest mock immediately.
3. **Server skeleton + auth extraction** (`server.go`, `cmd/mcp/main.go`) — depends on 1.
   Extract bearer key from the MCP request context; reject missing/blank.
4. **Tier framework** (`tiers.go`) — tool registration carries a tier + MCP annotations;
   Tier-3 gated on a per-connection opt-in flag. Depends on 3.
5. **Tier-1 read tools** (`tools_read.go`) — depend on 2+4. No side effects. **← Inc 1 ends here.**
6. **Tier-2 write tools** (`tools_write.go`) + stdio transport — depend on 5. Idempotent. **← Inc 2.**
7. **Tier-3 tools + opt-in + dashboard toggle** (`tools_sensitive.go`) — depend on 6. **← Inc 3.**
8. **Deploy + docs + E2E** — Cloud Build/Run, `mcp.recurso.dev`, docs page. **← Inc 4.**

Parallelizable: (2) and the SDK spike (1) can proceed together; tool files (5/6/7) are
independent of each other once (4) exists.

## Increments (each a reviewable PR; money-path never self-merged)

### Inc 1 — skeleton + auth + Tier-1 (zero write risk)
- `cmd/mcp/main.go`: Streamable HTTP transport, config (`API_BASE_URL`, port), graceful shutdown.
- `internal/mcp/client.go` + tests (mock `/v1`): GET/POST, key forwarding, error mapping.
- `internal/mcp/server.go`: server, per-request key extraction, tool registry.
- `internal/mcp/tiers.go`: tier + annotations plumbing (Tier-3 present but no tools yet).
- `internal/mcp/tools_read.go`: `list_customers`, `get_customer`, `list_subscriptions`,
  `get_subscription`, `preview_subscription_change`, `list_invoices`, `get_invoice_preview`,
  `simulate_charges`, `get_usage`, `list_plans`, `get_plan`, `list_quotes`, `get_quote`,
  `list_billable_metrics`.
- **Contract test:** each tool's target path exists in `cmd/api/openapi.yaml`.
- Verify: `go build ./... && go test ./...` green. **Acceptance:** an MCP client with a
  test key lists customers + simulates charges; cross-tenant read fails closed (test).

### Inc 2 — Tier-2 curated writes (idempotent)
- `tools_write.go`: `create_customer`, `update_customer`, `record_usage_event` (deterministic
  `transaction_id`), `record_usage_batch`, `create_subscription`, `update_subscription`,
  `create_quote`, `update_quote`, `send_quote`. Every call sends `Idempotency-Key`.
- `stdio` transport mode in `cmd/mcp` for local dev.
- Tests: idempotency-key presence; retry with same key does not double-apply (mock asserts).

### Inc 3 — Tier-3 opt-in
- `tools_sensitive.go`: `convert_quote_to_invoice`, `cancel_subscription`, `create_credit_note`,
  `wallet_top_up`, `add_subscription_charge`, `bill_usage_now` — all `destructiveHint`, gated
  on a per-connection opt-in.
- Persist the opt-in: small `mcp_connections` config (tenant → enabled + allowed tiers), read
  at tool-list time so disabled tiers don't even appear. Dashboard toggle (enable MCP + choose
  tiers). Backend money-path review; NOT self-merged.

### Inc 4 — deploy + docs
- Cloud Build target + `recurso-mcp` Cloud Run service + `mcp.recurso.dev` DNS.
- Docs page: connect Claude/ChatGPT to Recurso MCP (auth with your key, tier explanation).
- E2E: read→simulate→create_customer→create_quote through the server on `docker compose`.
- Website mention.

## Risks & mitigations

- **SDK API uncertainty** → the blocking spike (step 1) resolves it before tool code; the
  `/v1` client and tool *logic* are SDK-agnostic, so churn is confined to `server.go`.
- **Cross-tenant leakage** → we never hold a service key; every call forwards the caller's
  key and `/v1` enforces scope. Explicit fail-closed test in Inc 1.
- **Money-path exposure** → Tier-3 off by default + per-tenant opt-in + `destructiveHint`;
  idempotency on all writes; Inc 3 reviewed, not self-merged.
- **Route drift breaking a tool** → openapi contract test fails CI if a tool's path vanishes.
- **iCloud mid-merge reverts** (known repo hazard) → verify `origin/main` via `git ls-tree`
  after any stacked merge (per prior lessons).

## Verification checkpoints

- After Inc 1: tenant isolation + read/simulate proven; no `/v1` or ledger change touched.
- After each inc: `go build ./... && go test ./...` green; tools' paths ⊆ `openapi.yaml`.
- Before Inc 4 deploy: E2E flow green on a real stack.
