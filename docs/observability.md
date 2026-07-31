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

## Error tracking (Sentry)

Application error tracking is wired on both the API and the dashboard, **gated by
a DSN** — inert when unset, active when provided (mirroring how SMTP is gated by
`SMTP_HOST`):

```bash
# API (server): captures panics + 5xx handler errors
SENTRY_DSN=https://...ingest.sentry.io/...
# Dashboard (build-time): captures unhandled client errors
VITE_SENTRY_DSN=https://...ingest.sentry.io/...
```

- **API** — `middleware.SentryMiddleware` sits inside gin's Recovery: a panic is
  reported to Sentry and then re-raised so the request still returns a 500;
  handler errors that produce a 5xx are captured too. Errors are tagged with the
  route template + method. Events flush on shutdown.
- **Dashboard** — `Sentry.init` in `main.jsx` runs only when `VITE_SENTRY_DSN`
  is set at build time.

Both default to **errors only** (no tracing/replay); turn those on later if
wanted. Combined with the metrics above, this covers the operational surface.

## Product analytics (PostHog)

The dashboard can send product analytics to PostHog — pageviews, interactions,
and a signup → activation → retention picture. **Inert unless `VITE_POSTHOG_KEY`
is set at build time**, mirroring Sentry:

```bash
# Dashboard (build-time)
VITE_POSTHOG_KEY=phc_...                       # enables PostHog; unset = off
VITE_POSTHOG_HOST=https://us.i.posthog.com     # or your self-hosted instance
```

`posthog.init` runs in `main.jsx` only when the key is set; `AuthProvider`
`identify()`s the signed-in tenant/user (`tenant_id` as a property) once auth
resolves, so events tie back to a tenant. Self-host PostHog (set
`VITE_POSTHOG_HOST`) if you want the same own-your-data posture as the rest of
the stack.
