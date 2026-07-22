# Spec: MCP Server (agent-operable billing)

## Objective

Expose Recurso as a **Model Context Protocol (MCP) server** so AI agents — Claude,
ChatGPT, and any MCP-capable client — can *operate* a tenant's billing, not just
read it: look up customers, meter usage, simulate pricing, draft quotes, and (opt-in)
perform curated write actions. Recurso already has a fully `tenant_id`-scoped,
OpenAPI-documented, idempotent `/v1` API; this turns "we have a complete API" into
"your billing is agent-native."

**Why this is a leapfrog:** neither Lago nor Stripe Billing ships a first-class,
tenant-scoped MCP surface. Recurso's API completeness + existing idempotency make it
uniquely cheap for us and uniquely valuable to agent-driven finance/ops workflows.

**Users:**
- A tenant's ops/finance person driving billing through an AI assistant ("what's ACME's
  MRR? draft a renewal quote at a 10% discount and show me the invoice preview").
- A tenant's own product agent recording usage or provisioning subscriptions.
- Internal/support use against test-mode keys.

**Success looks like:** an agent, authenticated with a tenant's `rsk_` key, can complete
a real read→simulate→draft workflow end-to-end through MCP tools, with every write
idempotent and every sensitive action gated by explicit opt-in — zero new money-path
risk versus the HTTP API.

## Assumptions (correct me before I build)

1. **Architecture = thin client over the HTTP `/v1` API**, not in-process service reuse.
   The backend has no service container (all wiring is inline in `main()`), so in-process
   reuse means replicating ~1900 lines. Calling `/v1` reuses auth, tenant-scoping,
   idempotency, validation, and OpenAPI contract for free. One extra in-cluster hop, negligible.
2. **Auth = the caller supplies their own `rsk_` API key**; the MCP server forwards it as
   the Bearer token to `/v1`. Tenant scope and live/test mode come entirely from that key.
   There is no admin tier to model — the flat key *is* the scope.
3. **Transport = remote Streamable HTTP**, hosted at `mcp.recurso.dev` as its own Cloud Run
   service, so any hosted tenant can connect without running a local binary. A `stdio` mode
   of the same binary is a cheap add for local/dev use.
4. **Tools are hand-authored and curated**, not a 1:1 generation of all 208 routes. Agents
   need a small, well-described, consolidated toolset; `openapi.yaml` is the payload-shape
   reference, not the tool list.
5. **Money stays minor-units `int64`** across tool inputs/outputs, exactly as the HTTP API.
6. **New code lives in this repo** under `cmd/mcp/` + `internal/mcp/`, reusing `domain` types
   and the OpenAPI contract; it deploys as a separate service, not inside the API binary.

## Tech Stack

- Go 1.25 (same module `github.com/recurso-dev/recurso`).
- MCP: the official Go SDK `github.com/modelcontextprotocol/go-sdk` (new direct dep;
  Streamable HTTP + stdio transports, tool registration, JSON-Schema tool inputs).
- Talks to `/v1` via a small typed HTTP client (net/http) — forwards the caller's key +
  an `Idempotency-Key` per write.
- Deploy: new Cloud Build target → Cloud Run service `recurso-mcp` at `mcp.recurso.dev`.

## Project Structure

```
cmd/mcp/main.go              → MCP server entrypoint (transport select: http | stdio)
internal/mcp/
  server.go                  → server construction, tool registry, auth extraction
  client.go                  → typed HTTP client over /v1 (key forwarding, idempotency, error mapping)
  tools_read.go              → Tier-1 read/simulate tools (no side effects)
  tools_write.go             → Tier-2 curated writes (idempotent)
  tools_sensitive.go         → Tier-3 money-path/destructive (opt-in per connection)
  tiers.go                   → tool-tier gating + MCP annotations (readOnly/idempotent/destructive hints)
  errors.go                  → /v1 error envelope → MCP tool error mapping
internal/mcp/*_test.go       → unit tests (tool schema, tier gating, client error mapping) + a mock /v1
docs/spec_mcp_server.md      → this spec
tasks/mcp-plan.md            → implementation plan (after spec approval)
tasks/mcp-todo.md            → task list (namespaced — do NOT touch tasks/plan.md/todo.md = Demo Mode)
```

## Tool surface (curated, tiered by side-effect risk)

Tiers map to MCP tool annotations and to a per-connection opt-in. **The tier a tool sits in
is a safety contract, not a suggestion.**

**Tier 1 — reads & simulations (always on, `readOnlyHint: true`, zero side effects):**
`list_customers`, `get_customer`, `list_subscriptions`, `get_subscription`,
`preview_subscription_change` (proration preview), `list_invoices`, `get_invoice_preview`,
`simulate_charges` (pricing simulator), `get_usage` (windowed), `list_plans`, `get_plan`,
`list_quotes`, `get_quote`, `list_billable_metrics`. Analytics reads (MRR etc.) optional.

**Tier 2 — curated writes (on by default, idempotent, `idempotentHint: true`):**
`create_customer`, `update_customer`, `record_usage_event` (sets deterministic
`transaction_id`), `record_usage_batch`, `create_subscription`, `update_subscription`,
`create_quote`, `update_quote`, `send_quote`. Every call sends an `Idempotency-Key`.

**Tier 3 — sensitive money-path / destructive (OFF by default, opt-in per connection,
`destructiveHint: true`):** `convert_quote_to_invoice`, `cancel_subscription`,
`create_credit_note`, `wallet_top_up`, `add_subscription_charge`, `bill_usage_now`.
Enabling Tier 3 is a deliberate per-tenant choice surfaced in the dashboard.

Prohibited from MCP entirely (do via dashboard/human): API-key/gateway-credential
management, team/role changes, tenant/org settings, raw ledger posting.

## Code Style

Match existing backend idiom. One representative tool:

```go
// tools_read.go
func (s *Server) registerSimulateCharges(reg *mcp.ToolRegistry) {
	reg.Add(mcp.Tool{
		Name:        "simulate_charges",
		Description: "Preview line items and GL impact for a plan's usage charges at a given quantity. Read-only; no invoice is created.",
		Annotations: mcp.Annotations{ReadOnlyHint: true},
		InputSchema: schemaFor[SimulateChargesInput](),
	}, func(ctx context.Context, in SimulateChargesInput) (*mcp.ToolResult, error) {
		// Forward the caller's key; /v1 does the tenant-scoping and math.
		resp, err := s.client.Post(ctx, callerKey(ctx),
			fmt.Sprintf("/v1/plans/%s/simulate-charges", in.PlanID), in)
		if err != nil {
			return nil, mapAPIError(err) // /v1 {error:{message}} → MCP tool error
		}
		return jsonResult(resp), nil
	})
}
```

Conventions: money in minor-units `int64`; every Tier-2/3 tool sends `Idempotency-Key`;
tool descriptions are written for an agent (say what it does AND its side effect); errors
surface the `/v1` message, never a raw stack.

## Testing Strategy

- **Unit** (`internal/mcp/*_test.go`, table-driven, no network): tool input-schema validity,
  tier-gating (Tier-3 tool rejected when connection hasn't opted in), idempotency-key
  presence on writes, `/v1` error-envelope → MCP error mapping. A **mock `/v1` server**
  (httptest) backs client tests.
- **Contract**: assert every tool's target path exists in `cmd/api/openapi.yaml` (a small
  drift guard analogous to the API's openapi drift test) so a renamed route can't silently
  break a tool.
- **E2E** (opt-in, real stack): against `docker compose up` + a test-mode key, run a
  read→simulate→create_customer→create_quote flow through the MCP server and assert results.
- Gate: `go build ./... && go test ./...` green; MCP package included in CI.

## Boundaries

- **Always:** forward the caller's key (never a service-wide key); send `Idempotency-Key` on
  writes; keep Tier-3 off unless the connection opted in; money in minor units; run
  `go build ./... && go test ./...` before commit.
- **Ask first:** adding any Tier-3 tool; exposing analytics/`ask` (LLM-in-the-loop) via MCP;
  adding a new external dependency beyond the MCP SDK; any change that touches `/v1` handlers
  or the ledger.
- **Never:** expose key/gateway/credential management, tenant/org/team settings, or raw ledger
  posting as tools; bypass `/v1` to write the DB directly; self-merge money-path changes;
  weaken idempotency; publish/tag SDKs (founder-gated).

## Success Criteria

1. An MCP client authenticated with a tenant `rsk_test_` key lists that tenant's customers,
   simulates charges, and drafts a quote — and cannot see any other tenant's data.
2. Every Tier-2/3 write is idempotent (a retried tool call with the same key does not
   double-apply), verified in tests.
3. Tier-3 tools are unavailable until the connection opts in; verified in tests.
4. A cross-tenant attempt (key A, resource B) fails closed.
5. `go build ./... && go test ./...` green; MCP tools' paths verified against `openapi.yaml`.
6. Deployable as its own Cloud Run service without touching the API binary.

## Proposed increments (plan detail after spec approval)

- **Inc 1 — skeleton + auth + Tier 1:** `cmd/mcp` server (Streamable HTTP), key extraction &
  forwarding, typed `/v1` client with error mapping, all Tier-1 read/simulate tools, unit +
  contract tests. Proves the surface and tenant isolation with zero write risk.
- **Inc 2 — Tier 2 writes:** curated idempotent writes (customer/subscription/quote/usage),
  idempotency-key wiring, tests. `stdio` transport mode added here for local dev.
- **Inc 3 — Tier 3 opt-in:** sensitive money-path tools behind a per-connection flag +
  dashboard toggle to enable MCP and choose tiers; annotations surfaced to clients.
- **Inc 4 — deploy + docs:** Cloud Run service + `mcp.recurso.dev`, docs page (how to connect
  Claude/ChatGPT), E2E flow. Website mention.

## Decisions (LOCKED 2026-07-22)

- **D1** Architecture: **thin HTTP client over `/v1`** (no in-process service reuse). ✅
- **D2** Transport: **remote Streamable HTTP** primary; stdio mode added in Inc 2 for local dev. ✅
- **D3** Tool safety model: **three tiers, Tier-3 opt-in per tenant** — reads/sims always on,
  curated idempotent writes on by default, money-path/destructive off until the tenant opts in. ✅
- **D4** Auth ergonomics: **raw `rsk_` bearer key** now; OAuth flow deferred. ✅

## Parallel (folded in regardless of this epic)

Security debt from [[byo-gateway-series]]: accounting OAuth tokens are stored **plaintext** in
`accounting_connections.access_token`. Migrate onto the existing secretbox vault. Independent
of MCP; tracked so it doesn't get lost.
