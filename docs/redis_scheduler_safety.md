# Redis & scheduler safety (ENG-161)

All background schedulers and workers run **inside the `cmd/api` process**. To
ensure a scheduler tick runs on exactly one instance, each tick takes a
distributed lock (`locker.Obtain`). That lock — and the API idempotency-key
store — are only real when Redis is configured.

**Without `REDIS_URL`** the app falls back to a **no-op locker** and a
**per-instance in-memory idempotency store**. That is safe on a *single*
instance only. On a multi-instance deployment every instance runs every
scheduler concurrently, and the API idempotency-key dedup stops working across
instances.

The money-moving schedulers now defend themselves regardless of the lock —
`mandate_debit` (atomic `ClaimDueForDebit`) and trial conversion (atomic
`ActivateTrialWithTx`) — but Redis is still required so the remaining
schedulers don't run redundantly and so the Idempotency-Key middleware works.

## Configuration

| Var | Meaning |
| --- | --- |
| `REDIS_URL` | e.g. `redis://host:6379` (or `rediss://…` for TLS). Enables the real locker + idempotency store. The app PINGs it at startup and logs `Using Redis for Locker and Idempotency`. |
| `REQUIRE_REDIS` | `true` → the app refuses to start (fatal) if `REDIS_URL` is missing/invalid/unreachable, instead of silently degrading. Set this on any multi-instance deployment. |

**Ordering matters:** only set `REQUIRE_REDIS=true` *after* `REDIS_URL` points at a
reachable Redis — otherwise the next deploy crash-loops.

## Local dev

Nothing to do — `docker-compose.yml` runs a `redis` service and sets
`REDIS_URL=redis://redis:6379` on the `api` service. Startup logs should show
`Using Redis for Locker and Idempotency`.

## Self-hosted prod (docker-compose.prod.yml)

Already wired: a `redis` service (append-only persistence) with
`REDIS_URL=redis://redis:6379` and `REQUIRE_REDIS=true` on `api`.

## Cloud Run (`recurso-api`, asia-south1)

Cloud Run env vars live on the service (the CD deploy step inherits them), so
they are set with `gcloud`, not in `cloudbuild.yaml`. Cloud Run reaches a
private Memorystore instance through a Serverless VPC Access connector.

```bash
# 1. Memorystore (Redis) in the same region.
gcloud redis instances create recurso-redis \
  --region=asia-south1 --size=1 --redis-version=redis_7_0

# note the host IP:
REDIS_HOST=$(gcloud redis instances describe recurso-redis \
  --region=asia-south1 --format='value(host)')

# 2. Serverless VPC connector so Cloud Run can reach the private IP.
gcloud compute networks vpc-access connectors create recurso-connector \
  --region=asia-south1 --range=10.8.0.0/28

# 3. Point the service at Redis (do NOT set REQUIRE_REDIS yet).
gcloud run services update recurso-api --region=asia-south1 \
  --vpc-connector=recurso-connector \
  --set-env-vars=REDIS_URL=redis://$REDIS_HOST:6379

# 4. Verify the new revision logs "Using Redis for Locker and Idempotency",
#    then enforce it so a future misconfig fails loudly instead of silently:
gcloud run services update recurso-api --region=asia-south1 \
  --update-env-vars=REQUIRE_REDIS=true
```

Until this is done, keep the service at a single instance
(`--min-instances=1 --max-instances=1`) so the no-op locker is safe.
