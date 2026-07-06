# Performance

Measured numbers, not aspirations. Methodology and raw results below so
they can be reproduced and challenged.

**Reference box:** Apple M2 Pro (10 cores, 16 GB), macOS, full
`docker-compose` stack (API + PostgreSQL 15 + TigerBeetle connected +
Mailhog), load generated with [`hey`](https://github.com/rakyll/hey) from
the same machine. Payment gateways in mock mode (no external network), so
these numbers isolate Recurso + PostgreSQL. Date: 2026-07-06, commit range
around v0.1.1+.

## Results

| Scenario | Concurrency | Duration | Throughput | p50 | p95 | p99 | Errors |
|---|---|---|---|---|---|---|---|
| `GET /version` (no auth, baseline) | 50 | 15s | 28,039 req/s | 1.6 ms | — | 4.9 ms | 0 |
| `GET /v1/customers` (authenticated) | 50 | 30s | 7,814 req/s | 4.8 ms | 17.7 ms | 27.2 ms | 0 |
| `POST /v1/subscriptions` (creates subscription + invoice + tax + double-entry ledger posting) | 20 | 30s | **578 req/s ≈ 34,600 invoices/min** | 32.2 ms | 55.3 ms | 67.0 ms | 0 (17,352 × 201) |

Ledger integrity under load: the 17,352 subscription creations produced
exactly 17,352 ledger transactions (verified by count and by the
`/v1/finance/reconciliation` report showing zero discrepancies for the
post-fix window).

## What the load test found (and fixed)

Running this the first time surfaced three production bugs, all fixed in
the same change series:

1. **Per-request bcrypt capped the API at ~126 req/s** (p50 373 ms). Every
   request re-verified the API key with a full bcrypt compare. A TTL'd
   verified-key cache (SHA-256 keyed, 5-minute expiry, so revocation still
   takes effect) took the authenticated read path from 126 → 7,814 req/s
   and p99 from 773 ms → 27 ms.
2. **Only one tenant could ever register per database.** Hashed API keys
   store an empty `key_value`, which still carried a UNIQUE constraint —
   the second registration always failed. Migration 000054 drops it.
3. **Ledger postings failed for every API-created tenant/customer.**
   Customer AR accounts and the tenant chart of accounts were never
   provisioned outside the seeder, and the fallback account UUIDs pointed
   at rows that cannot exist (FK-guaranteed failure). Postings now
   provision missing accounts on first use, self-healing existing
   deployments. The reconciliation report caught all 40,859 historical
   gaps from before the fix.

## Backup / restore drill (verified)

Performed 2026-07-06 against the stack above with ~58k invoices and ~17k
ledger transactions:

1. `docker compose exec -T postgres pg_dump -U user recurso > backup.sql`
   (43 MB)
2. `docker compose down -v` — **all volumes destroyed**
3. Fresh stack up, `psql < backup.sql` into a clean database
4. Row counts identical across tenants/customers/subscriptions/invoices/
   ledger_transactions; API keys still authenticate; `/health` green.

## Caveats

- Single-node everything, loopback networking, generator on the same box —
  treat as an upper bound for one instance, not a cluster benchmark.
- Mock gateways: real Stripe/Razorpay calls add their own latency to
  subscription creation (the API's own work is what's measured here).
- Rate limiting was raised via `RATE_LIMIT_PER_MINUTE` for the test; the
  default remains 500/min per key.
