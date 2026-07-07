# Anonymous instance telemetry

Recurso can report a tiny, anonymous signal so the maintainers can answer one
question they otherwise cannot: **how many self-hosted instances reach their
first real invoice.** That single activation metric decides where development
effort goes.

It is **off by default**. With the default configuration Recurso makes **zero
network calls** for telemetry and writes **zero telemetry rows**. Nothing is
sent unless you set `TELEMETRY_OPTIN=true`.

## One-line disable

Don't set `TELEMETRY_OPTIN` (or set it to anything other than `true`). That's
the default — telemetry is already off. To turn it off after enabling it,
remove `TELEMETRY_OPTIN=true` and restart.

## What is sent

When (and only when) telemetry is enabled, Recurso POSTs small JSON events to
`TELEMETRY_ENDPOINT` (default `https://telemetry.recurso.dev/v1/events`). Each
event is a single fire-and-forget POST with a 5-second timeout and **no
retries**; if it fails, Recurso logs one line at debug level and moves on.
Telemetry never touches a request path.

Every event carries an anonymous `instance_id` — a random UUID generated once
and stored in your database (`telemetry_instance` table). It is **not** derived
from your hostname, MAC address, license key, domain, or anything else that
could identify you or your customers.

### `instance_started` — sent once per process start

```json
{
  "event": "instance_started",
  "instance_id": "b1f4c2a0-0e2d-4c5a-9f7e-2d3a4b5c6d7e",
  "version": "v0.1.0",
  "timestamp": "2026-07-07T09:15:00Z",
  "os": "linux",
  "arch": "amd64",
  "deployment": "docker"
}
```

`deployment` is a coarse hint only: `docker` (a container marker file exists)
or `binary`.

### Milestones — each sent at most once, ever, per instance

The milestone fires the first time your instance creates a plan, a customer,
generates an invoice, or collects a payment. A flag is persisted on the
`telemetry_instance` row, so a milestone never fires twice — not across
restarts, not across replicas sharing one database.

```json
{
  "event": "milestone_first_plan",
  "instance_id": "b1f4c2a0-0e2d-4c5a-9f7e-2d3a4b5c6d7e",
  "version": "v0.1.0",
  "timestamp": "2026-07-07T09:20:00Z"
}
```

The four milestone events are:

- `milestone_first_plan`
- `milestone_first_customer`
- `milestone_first_invoice` — the activation metric
- `milestone_first_payment`

### `heartbeat` — sent at startup, then every 24 hours

```json
{
  "event": "heartbeat",
  "instance_id": "b1f4c2a0-0e2d-4c5a-9f7e-2d3a4b5c6d7e",
  "version": "v0.1.0",
  "timestamp": "2026-07-07T09:15:01Z",
  "os": "linux",
  "arch": "amd64",
  "deployment": "docker",
  "tenants": "1-9",
  "subscriptions": "10-99"
}
```

`tenants` and `subscriptions` are **bucketed ranges, never exact numbers**. The
only possible values are `"0"`, `"1-9"`, `"10-99"`, and `"100+"`. The exact
counts are read locally and collapsed into a range before anything leaves the
process.

## What is never sent

- No amounts, prices, currencies, revenue, or MRR.
- No names, emails, phone numbers, addresses, or tax IDs.
- No API keys, secrets, tokens, or connection strings.
- No customer, plan, invoice, subscription, or tenant IDs.
- No hostnames, IP addresses, domains, or MAC addresses.
- No exact counts of anything — only the four bucket ranges above.

The complete list of fields ever transmitted is exactly the keys shown in the
JSON examples above. There is no free-text field and no catch-all payload.

## How to verify

Don't take our word for it — point the endpoint at a server you control and
watch every byte:

```bash
# Terminal 1: a throwaway collector that prints whatever it receives
python3 -m http.server 9999

# Terminal 2: run Recurso against it
TELEMETRY_OPTIN=true \
TELEMETRY_ENDPOINT=http://localhost:9999/v1/events \
  ./recurso
```

Any request-logging endpoint works (`nc -l 9999`, a webhook.site URL, an
`ngrok` tunnel, etc.). Create a plan, a customer, a subscription, and mark an
invoice paid, and you'll see exactly the events documented here — and nothing
else.

You can also read the source directly:
[`internal/adapter/telemetry/telemetry.go`](../internal/adapter/telemetry/telemetry.go)
is the only code that builds and sends payloads; every field is assembled in
`Client.send`, `sendHeartbeat`, and `milestone`.

## The collector

`https://telemetry.recurso.dev` is where hosted events are aggregated. The
collector is being stood up separately; until it exists, an enabled instance
simply POSTs into the void and logs the failure at debug level — which changes
nothing about your instance's behavior.
