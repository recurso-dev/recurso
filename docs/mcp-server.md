# Recurso MCP Server

`cmd/mcp` exposes Recurso as a [Model Context Protocol](https://modelcontextprotocol.io)
server so AI agents can operate a tenant's billing. It is a thin, curated,
tier-gated facade over the existing `/v1` API — **no database, no service
structs, no stored credentials.** Tenant isolation is the caller's own API key,
forwarded to `/v1` on every request (ADR-005 layered caching applies at `/v1`).

See `docs/spec_mcp_server.md` for the design and decisions (D1–D4).

## Transports

| Transport | Use | Auth |
|-----------|-----|------|
| **Streamable HTTP** (default) | Hosted, multi-tenant (`mcp.recurso.dev`) | Each caller sends `Authorization: Bearer rsk_...`; that key alone decides tenant + live/test mode |
| **stdio** | Local, single-tenant (e.g. Claude Desktop) | `RECURSO_API_KEY` env, used for every call |

Config (env): `MCP_TRANSPORT` (`http`|`stdio`, default `http`), `API_BASE_URL`
(default `http://localhost:8080`), `PORT` (default `8090`, set by Cloud Run),
`RECURSO_API_KEY` (required for stdio).

## Tool tiers (safety contract)

- **Tier 1 — reads & simulations** (always on, `readOnlyHint`): list/get customers,
  subscriptions, invoices, plans, quotes, metrics; `preview_subscription_change`;
  `simulate_charges`; `get_subscription_usage`.
- **Tier 2 — curated writes** (on by default, idempotent): create/update customer,
  record usage (single/batch), create/update subscription, create/update/send quote.
  Every write carries an `Idempotency-Key`.
- **Tier 3 — money-path / destructive** (OFF by default, **per-tenant opt-in**,
  `destructiveHint`): convert quote→invoice, cancel subscription, create credit
  note, wallet top-up, add charge, bill usage now. Gated at two layers — the
  server policy AND a call-time check of the tenant's opt-in (`GET /v1/settings/mcp`);
  fail-closed. A tenant enables it in **Settings → MCP server**.

## Deploy (`recurso-mcp` Cloud Run service)

The MCP server deploys independently of the API via `cloudbuild.mcp.yaml`
(image built from `Dockerfile.mcp`). One-time infra setup:

1. Create the Cloud Run service `recurso-mcp` in the deploy region.
2. Set its env: `API_BASE_URL=https://api.recurso.dev`.
3. Map the custom domain `mcp.recurso.dev`.
4. Point a Cloud Build trigger at `cloudbuild.mcp.yaml`.

The image build is gated on `go test ./internal/mcp/...`, so a broken commit
never deploys.

## Connect an agent

**Claude Desktop (stdio)** — add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "recurso": {
      "command": "/path/to/recurso-mcp",
      "env": {
        "MCP_TRANSPORT": "stdio",
        "API_BASE_URL": "https://api.recurso.dev",
        "RECURSO_API_KEY": "rsk_live_…"
      }
    }
  }
}
```

**Remote (Streamable HTTP)** — point any MCP client at `https://mcp.recurso.dev`
with header `Authorization: Bearer rsk_live_…`.

## Verify

- `go test ./internal/mcp/...` — unit + protocol E2E (client → Streamable HTTP →
  tool → mock `/v1`): tenant-key forwarding, missing-key fail-closed, Tier-3
  opt-in gating, idempotency.
- `./scripts/mcp_smoke.sh` — builds the binary and checks `/health` boots.
