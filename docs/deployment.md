# Self-Hosting Recurso

A concise runbook for running Recurso in production with Docker Compose, plus pointers for Kubernetes.

## Quick start (Docker Compose)

```bash
git clone https://github.com/recurso-dev/recurso.git && cd recur-so

# 1. Configure
cp .env.example .env
# Edit .env — at minimum set POSTGRES_USER / POSTGRES_PASSWORD / POSTGRES_DB
# and API_SECRET. See .env.example for the full annotated list
# (payments, SMTP, AI, etc. all degrade gracefully to mocks if unset).

# 2. Launch
docker compose -f docker-compose.prod.yml up -d --build

# 3. Verify
curl http://localhost:8080/health   # API
curl http://localhost/              # Frontend (SPA served by nginx)
```

The stack:

| Service       | Image                              | Ports (host)           | Notes                                              |
|---------------|------------------------------------|------------------------|----------------------------------------------------|
| `frontend`    | built from `frontend/Dockerfile`   | 80 → 8080              | Unprivileged nginx; serves the SPA, proxies `/v1`, `/auth`, `/portal/api`, `/portal/auth`, `/checkout` to the API |
| `api`         | built from root `Dockerfile`       | 8080 → 8080            | Runs DB migrations automatically on boot           |
| `postgres`    | `postgres:15-alpine`               | not exposed            | System of record                                   |
| `tigerbeetle` | `ghcr.io/tigerbeetle/tigerbeetle`  | not exposed            | Optional high-performance ledger                   |

The frontend container proxies API traffic internally (`API_UPSTREAM=http://api:8080`), so for a single-box deployment you can put TLS termination (Caddy, Traefik, or a cloud load balancer) in front of port 80 and optionally stop publishing port 8080. If browsers reach the API directly on 8080 instead, set `CORS_ORIGIN` in `.env`.

Upgrades:

```bash
git pull
docker compose -f docker-compose.prod.yml up -d --build
```

Migrations are applied by the API at startup; no separate migration step is needed.

## Where the data lives

- **`postgres_data` volume** — the source of truth. All business data (customers, subscriptions, invoices, ledger entries, configuration) lives in PostgreSQL. This is the volume you must protect.
- **`tigerbeetle_data` volume** — the TigerBeetle ledger data file. In this setup the ledger is dual-written: PostgreSQL always, TigerBeetle when reachable. TigerBeetle is therefore **optional and rebuildable from PostgreSQL** — losing this volume does not lose financial data (the API falls back to PG-only ledger mode if TigerBeetle is unavailable).

## Backups

**PostgreSQL (essential).** A nightly `pg_dump` cron on the host:

```cron
# /etc/cron.d/recurso-backup — 02:30 nightly, keep 14 days
30 2 * * * root docker compose -f /opt/recur-so/docker-compose.prod.yml exec -T postgres \
  pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB" > /var/backups/recurso/recurso-$(date +\%F).dump \
  && find /var/backups/recurso -name '*.dump' -mtime +14 -delete
```

Restore with `pg_restore -U <user> -d <db> --clean <file>.dump` into a fresh postgres container. Test restores periodically, and ship the dump directory off-host (e.g. `rclone`/S3).

**Volume snapshots.** If your host or cloud supports filesystem/block snapshots (ZFS, LVM, EBS), snapshotting the Docker volume directory is a good complement. For a fully consistent snapshot, stop the stack first (`docker compose ... stop`), snapshot, then start — or rely on `pg_dump` for consistency and use snapshots as a secondary layer.

**Drill status.** The full backup → destroy → restore cycle was executed
and verified on 2026-07-06 (see docs/performance.md for the procedure and
integrity checks).

**TigerBeetle.** No backup required in this deployment: it is an optional accelerator and the ledger is authoritative in PostgreSQL. If the volume is lost, remove it and restart — the container reformats a fresh data file and the API continues (worst case in PG-only mode until it reconnects).

## Alerting

The API ships with a built-in health watcher: every 60 seconds it evaluates the same component checks as `GET /health` (PostgreSQL, Redis if configured, TigerBeetle) and POSTs an alert to `ALERT_WEBHOOK_URL` on **state transitions only** — one alert when a component goes down (PostgreSQL = `critical`, Redis/TigerBeetle = `warning`), one when it recovers. No repeats while the state is steady, and no alerts at all if the variable is unset.

```bash
# .env — see .env.example (Observability section) for details
ALERT_WEBHOOK_URL=https://hooks.slack.com/services/T000/B000/XXXX
ALERT_WEBHOOK_FORMAT=slack   # json (default) | slack
```

`json` posts `{"severity","title","body","source":"recurso","timestamp"}`; `slack` posts `{"text": "..."}` so a Slack incoming webhook works directly. Deliveries are one POST with a 10s timeout, no retries — pair this with an external uptime check on `/health` (the watcher runs inside the API process, so it can't tell you the process itself died).

When an alert fires, follow **[docs/incident-runbook.md](incident-runbook.md)** — severity definitions, first commands, and the SEV1 money-movement procedure.

## Kubernetes

Manifests live in `k8s/` (namespace, deployment, service, ingress, configmap, secret, RBAC, network policy). They deploy the **API only** — bring your own managed PostgreSQL (set `DATABASE_URL` in `recurso-secrets`) and, optionally, serve the frontend image behind your ingress.

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/               # review configmap.yaml and secret.yaml first
```

Before applying:

- Put real values in `k8s/secret.yaml` (or replace it with an ExternalSecret/SealedSecret).
- Change the host in `k8s/ingress.yaml` from `api.recurso.dev` to your own domain (see the comments in that file); it assumes ingress-nginx + cert-manager.
- The deployment pulls `ghcr.io/recurso-dev/recurso:latest`; pin a tag for production.

Health probes hit `/health` on port 8080; the deployment runs 2 replicas as non-root with a read-only root filesystem.

## Migrating an existing subscriber base

`cmd/import` loads plans, customers, and subscriptions from another billing
system directly into the database — without generating invoices or calling
payment gateways, so migrated customers are never double-billed mid-cycle.
The renewal worker issues each subscription's next invoice at its imported
`current_period_end`.

```bash
# 1. Register your production tenant first (POST /auth/register) and note
#    the tenant UUID.

# 2. Prepare the data — JSON (see cmd/import/example.json) or CSVs.

# 3. Dry-run: validates everything, writes nothing.
DATABASE_URL=postgres://... go run ./cmd/import \
  -tenant <tenant-uuid> -input data.json -dry-run

# 4. Import for real. Idempotent: plans match by code, customers by email,
#    subscriptions by external_id — re-running skips what already exists.
DATABASE_URL=postgres://... go run ./cmd/import \
  -tenant <tenant-uuid> -input data.json
```

CSV mode: `-plans-csv plans.csv -customers-csv customers.csv
-subscriptions-csv subs.csv` (see the flag help for expected columns).

## High availability

The supported HA posture for v0.1.x:

- **PostgreSQL is the source of truth** for all state including the ledger.
  Run it with your standard HA tooling (managed Postgres, Patroni, or
  streaming replicas) and the backup regimen above.
- **The API is stateless** — run multiple replicas behind a load balancer
  (the K8s Deployment ships with 2). Schedulers use a distributed lock via
  Redis when configured, so replicas don't double-run jobs; set REDIS_URL
  in multi-replica deployments.
- **TigerBeetle is single-node in the provided manifests.** Clustered
  TigerBeetle (--replica-count > 1) is not yet part of the supported setup;
  until it is, treat TigerBeetle as an optional accelerator and rely on the
  PostgreSQL ledger, which the API does automatically when TigerBeetle is
  unreachable.
