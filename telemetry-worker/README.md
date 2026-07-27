# Recurso telemetry receiver

The collector behind `https://telemetry.recurso.dev/v1/events` — the endpoint
the product's **opt-in, anonymous** telemetry client
(`internal/adapter/telemetry`) POSTs to. Until this worker is deployed,
opted-in instances post into the void; with it, the project can answer the one
question it exists for: *how many self-hosted instances are alive, and how many
reach their first invoice?*

## What it stores

Exactly the client payload (see `docs/telemetry.mdx` for the privacy
contract) plus a server receive timestamp: event name, the instance's random
anonymous UUID, version, and coarse scalar props (deployment kind, bucketed
counts). **No IPs, no user agents, no headers, no PII.** Requests that don't
match the contract are rejected.

## Endpoints

- `POST /v1/events` — ingest (validated; 202 on accept)
- `GET /v1/stats` — aggregates, gated by `Authorization: Bearer $STATS_TOKEN`:
  total events/instances, instances active in 30d, instances heartbeating in
  7d, per-milestone distinct instances, version spread.
- `GET /healthz`

## Deploy (one-time, ~5 minutes)

```bash
cd telemetry-worker
npx wrangler d1 create recurso-telemetry   # copy database_id into wrangler.toml
npx wrangler d1 execute recurso-telemetry --remote --file schema.sql
npx wrangler secret put STATS_TOKEN        # any long random string
npx wrangler deploy                        # binds telemetry.recurso.dev
```

Check it:

```bash
curl -s https://telemetry.recurso.dev/healthz
curl -s -H "Authorization: Bearer $STATS_TOKEN" https://telemetry.recurso.dev/v1/stats
```

## Tests

```bash
node --test telemetry-worker/test/validate.test.mjs
```
