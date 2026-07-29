# Observability

Recurso exposes Prometheus metrics for the API, plus a ready-to-import Grafana
dashboard and alert rules. This answers the buyer/on-call question: *"can you
see what the system is doing, and will you notice when it breaks?"*

## Metrics — `GET /metrics`

The API serves Prometheus text-format metrics at `/metrics` (no dependency on
client_golang — it's a small built-in collector).

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `http_requests_total` | counter | `method`, `route`, `status` | Requests processed. `route` is the gin route **template** (bounded cardinality), never the raw path. |
| `http_request_duration_seconds` | histogram | `method`, `route` | Request latency (standard buckets). Use `histogram_quantile` for p50/p95/p99. |
| `go_goroutines` | gauge | — | Live goroutines (leak signal). |
| `go_memstats_alloc_bytes` / `_sys_bytes` | gauge | — | Heap in use / OS memory. |
| `go_gc_cycles_total` | gauge | — | Completed GC cycles. |
| `process_uptime_seconds` | gauge | — | Seconds since start. |

### Access control

`/metrics` is **optionally bearer-gated** via the `METRICS_TOKEN` env var:

- unset → open (scrape from a trusted network / private ingress only), or
- set → callers must send `Authorization: Bearer <METRICS_TOKEN>`.

Prefer keeping `/metrics` off the public internet (private network, or set the
token). It is intentionally excluded from its own request metrics.

## Scraping

`deploy/observability/prometheus-scrape.yml` is a ready scrape config:

```yaml
scrape_configs:
  - job_name: recurso-api
    metrics_path: /metrics
    scheme: https
    static_configs:
      - targets: ["api.recurso.dev"]
    # authorization: { type: Bearer, credentials: "<METRICS_TOKEN>" }
```

## Dashboard

Import `deploy/observability/grafana-dashboard.json` into Grafana (uid
`recurso-api`). Panels: request rate by status, 5xx ratio, latency
p50/p95/p99, top routes, goroutines, heap, uptime.

## Alerts

Load `deploy/observability/prometheus-alerts.yml` (`rule_files:` in
`prometheus.yml`). Rules: **APIDown**, **HighErrorRate** (5xx > 5% / 5m),
**HighLatencyP95** (> 1s / 10m), **GoroutineSpike**, **MemoryHigh**. Wire
Alertmanager to your paging channel.

Health-check alerting (component up/down via a webhook) already exists
independently — see `docs/spec_incident_alerting.md` and the
`ALERT_WEBHOOK_URL` env.

## Error tracking (Sentry) — planned

Application error tracking (stack traces, release health) is the remaining
observability piece. It will be wired **guarded by a `SENTRY_DSN` env** — inert
when unset, active when a DSN is provided — mirroring how SMTP is gated by
`SMTP_HOST`. Tracked as a follow-up because it adds an SDK dependency and needs a
DSN (an external credential). Until then, structured request logs + the metrics
above cover the operational surface.
