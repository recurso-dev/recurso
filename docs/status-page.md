# Status Page & Uptime Monitoring

A decision doc and setup guide for Recurso's public status page and external
uptime monitoring. This closes two honest gaps called out in
[incident-runbook.md](incident-runbook.md): "**No status page yet**" and
"**Alerting is only `ALERT_WEBHOOK_URL`** … from inside the API process — if
the process or host dies, nothing fires." An external monitor is the thing
that notices when the whole box is dark.

## Decision: hosted Better Stack free tier (now) → self-hosted Uptime Kuma (later)

**Recommendation: start on Better Stack's free tier** (hosted status page +
uptime monitoring), and revisit self-hosting only if cost or data-control
pressure appears.

**Why hosted, and why this one, for a pre-revenue startup:**

- **A monitor must not share fate with what it monitors.** The whole point is
  to catch "the host/process died and the in-API health watcher couldn't
  fire." A self-hosted monitor on our own infra reintroduces exactly that
  shared-fate risk — if our provider account or region has a bad day, the
  monitor goes down *with* the thing it's watching, and the status page that's
  supposed to reassure customers is itself offline. Hosted, off our infra, is
  the correct default for the outermost watchdog.
- **Zero ops cost at zero revenue.** A self-hosted Uptime Kuma needs its own
  tiny VM (~€4/mo), its own patching, its own TLS, its own backup — one more
  thing for a single operator to keep alive. Better Stack's free tier covers
  a handful of monitors + a hosted status page with SMS/email/Slack alerting
  and no VM to run. The right time to spend operator-hours self-hosting is
  when we have enough instances that per-check cost matters, not now.
- **Paging we don't otherwise have.** The incident runbook notes there is
  "no paging, no escalation." A hosted monitor gives phone/SMS/email + Slack
  alerting out of the box, which is a real upgrade over one Slack POST from
  inside the API.

**When to switch to self-hosted Uptime Kuma:** once there are enough customer
instances that the free-tier monitor cap bites, or a customer contract
requires that monitoring data stay on infra we control. At that point run
Uptime Kuma in Docker on a **separate, minimal VM in a different provider/
region** from any customer instance — never co-located with a thing it
monitors. Setup is a one-liner (`docker run -d -p 3001:3001 -v uptime-kuma:/app/data louislam/uptime-kuma:1`), and the check list below transfers unchanged.

Both options are honest, small-team choices; the deciding factor is
shared-fate + operator time, and hosted wins today.

## What to monitor (initially)

Add checks in this order; the first two exist today, the third grows with
Recurso Cloud.

| # | Check | Target | Type | Alert |
|---|---|---|---|---|
| 1 | **Docs site** | `https://docs.recurso.dev` | HTTPS 200, keyword match | Slack + email |
| 2 | **Marketing/API site** | `https://recurso.dev` (and GitHub repo reachability) | HTTPS 200 | Slack |
| 3 | **Per-customer instances** | `https://<CUSTOMER_DOMAIN>/health` | HTTPS 200, JSON body check | Slack + SMS (see below) |

Notes:

- **GitHub** as a dependency: monitor `https://github.com/swapnull-in/recur-so`
  reachability (that's the deploy source for every provisioning run — see
  [cloud-provisioning-runbook.md](cloud-provisioning-runbook.md)). GitHub's own
  status is at `githubstatus.com`; we only need to know if *our* repo/releases
  are reachable, not run a monitor for GitHub itself.
- **`/health` is the right probe** for instances: it returns 200 with
  per-component status, or 503 when Postgres is down (see the health watcher /
  deployment.md). Configure the monitor to alert on non-200 **and**, if the
  tool supports body assertions, on a component showing unhealthy in the JSON.
- **Escalation:** route instance (`/health`) failures to SMS/phone, since
  those are money-movement-adjacent (Postgres down = SEV2 heading toward
  SEV1). Route docs/site failures to Slack only — they're SEV3.
- **Check interval:** 60 s for `/health` (matches the in-API watcher cadence),
  180 s for docs/marketing.
- Add each new instance to the monitor **as a provisioning step** (it's on the
  §10 sign-off checklist in the provisioning runbook), and remove it during
  offboarding (§11).

## Setup (Better Stack free tier)

1. Create a Better Stack account; create a **Status Page** named `Recurso
   Status` at `<STATUS_PAGE_URL>` (e.g. `status.recurso.dev` via a CNAME to
   the Better Stack-provided host — set this in DNS).
2. Create **Monitors** for each row in the table above. For `/health` monitors,
   set: request URL `https://<CUSTOMER_DOMAIN>/health`, expected status `200`,
   check frequency `60s`, and (if enabled) a response-body assertion that the
   JSON does not contain an unhealthy component.
3. Attach monitors to the status page as **public resources** with friendly
   names ("Recurso Cloud — <CUSTOMER_DISPLAY_NAME>", "Documentation",
   "Website"). Keep per-customer instance names generic on the *public* page
   if customers shouldn't see each other — or give each design partner a
   private/unlisted page. For a handful of design partners, one page with
   neutral labels is fine.
4. Configure **on-call / alerting:** Slack integration into `#recurso-ops`
   (same channel the in-API alerts post to — see the provisioning runbook §7),
   plus email, plus SMS/phone for `/health` monitors.
5. Note the status page URL and monitor IDs in the ops inventory.

The in-API `ALERT_WEBHOOK_URL` alerting and this external monitor are
**complementary**: the webhook tells you *which component* degraded (it can
see inside the process); the external monitor tells you the process/host is
*reachable at all* (it can't). You need both.

## Incident communication template

Post to the status page for any **SEV1 or SEV2** (see severity definitions in
[incident-runbook.md](incident-runbook.md)). SEV3s generally don't warrant a
public post unless a customer noticed and asked. The status-page post is the
public complement to the direct tenant email the incident runbook already
requires for money-impacting incidents — post publicly *and* email affected
tenants; don't rely on one.

Lifecycle: **Investigating → Identified → Monitoring → Resolved.** Post an
update at each transition; don't go silent.

```
[INVESTIGATING] <short title>
Posted: <YYYY-MM-DD HH:MM UTC>

We are investigating <symptom, in plain language — e.g. "elevated errors on
the Recurso API for some customers">. <Impact: who/what is affected, e.g.
"Hosted checkout and dashboard access may be intermittent. No incorrect
charges have been observed.">. We'll post an update by <HH:MM UTC>.
```

```
[IDENTIFIED] <short title>
Posted: <YYYY-MM-DD HH:MM UTC>

We've identified the cause: <one line — e.g. "the database on one instance
became unreachable">. <What we're doing>. Money movement <is paused / is
unaffected>. Next update by <HH:MM UTC>.
```

```
[MONITORING] <short title>
Posted: <YYYY-MM-DD HH:MM UTC>

A fix is in place and we're monitoring recovery. <Any residual impact>.
We are reconciling the ledger against the payment gateways to confirm no
charge or refund moved incorrectly. Next update by <HH:MM UTC>.
```

```
[RESOLVED] <short title>
Posted: <YYYY-MM-DD HH:MM UTC>

This incident is resolved as of <HH:MM UTC>. Duration: <N> minutes.
<One-line cause + fix.> <If money was impacted:> Affected customers have been
contacted directly with the specifics. A brief post-incident note will follow.
```

**Severity → channel mapping**

| Severity | Status page | Direct tenant email | Public detail level |
|---|---|---|---|
| **SEV1** (money moved wrongly / ledger integrity) | Yes — post immediately, update through Resolved | **Yes**, per incident-runbook (exact customer impact, what was corrected) | Public: honest but high-level; the money specifics go in the direct email, not the public page |
| **SEV2** (payments degraded, not moving) | Yes | Only if a tenant's billing was actually delayed/affected | State the functional impact and that no incorrect charges occurred (if true) |
| **SEV3** (non-money: emails, dashboard, optional components) | Usually no | No, unless asked | — |

Wording rules: name the impact in the customer's terms, never speculate on
cause before **Identified**, always give a "next update by" time, and for
anything money-adjacent explicitly say whether charges/refunds were affected —
"no incorrect charges" is the sentence customers most want to read, so say it
as soon as the reconciliation report (see incident-runbook §3) supports it,
and not a moment before.
