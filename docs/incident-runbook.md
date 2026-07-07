# Incident Runbook

What to do when Recurso misbehaves in production, in order of how much money
is at stake. Written for the current reality: a single operator, a Docker
Compose (or small K8s) deployment as described in
[deployment.md](deployment.md), and alerting via `ALERT_WEBHOOK_URL`
(see the Alerting section of deployment.md).

Commands below assume the Docker Compose production stack
(`docker-compose.prod.yml`) on the host, and `$API_KEY` set to a tenant API
key (`Authorization: Bearer` auth).

## Severity definitions

| Severity | Definition | Examples | Response |
|---|---|---|---|
| **SEV1** | Money moved wrongly, or ledger integrity is broken | double charge, refund sent twice, ledger disagrees with invoices (`/v1/finance/reconciliation` discrepancies), Postgres data loss/corruption | Drop everything. Follow "SEV1 immediate actions" below, in order. |
| **SEV2** | Payments degraded — money is *not* moving that should | gateway webhooks not arriving, retry worker erroring on every invoice, Postgres down (API returns 503) | Fix within hours. Gateways retry webhooks and dunning retries payments, so a short outage self-heals — a long one becomes SEV1 risk. |
| **SEV3** | Non-money functionality broken | emails not sending, dashboard/analytics errors, Redis or TigerBeetle down (both optional), e-invoice IRP failures (auto-retried) | Fix during working hours. |

When unsure, treat it as one level higher until the reconciliation report
says the ledger is clean.

## SEV1 immediate actions

Do these **in order**. Steps 1–2 are about not making it worse; only then
investigate.

### 1. Stop the bleeding

**Honest limitation:** there is currently **no env switch to pause individual
schedulers or workers** (dunning retry, mandate debit, renewals). The
`DUNNING_STRATEGY` variable changes the retry *strategy*; it does not disable
retries. The only reliable way to stop Recurso from initiating charges is to
stop the API process:

```bash
docker compose -f docker-compose.prod.yml stop api
```

Consequences of stopping the API (accept them during a SEV1):

- All outbound money movement stops: retry worker, mandate-debit scheduler,
  renewal invoicing, gateway refund calls.
- Inbound gateway webhooks get connection errors — that is safe. Stripe and
  Razorpay both retry deliveries for an extended period, and you can replay
  missed events from their dashboards afterwards (see Recovery).
- Customers cannot pay via hosted checkout while it is down.

If only *dunning retries* are the problem and you must keep the API up, there
is no supported partial switch today — note this gap, keep the outage short.

### 2. Snapshot state

Before touching anything, capture Postgres exactly as it is (same command as
the nightly backup in [deployment.md](deployment.md)):

```bash
docker compose -f /opt/recur-so/docker-compose.prod.yml exec -T postgres \
  pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB" \
  > /var/backups/recurso/incident-$(date +%F-%H%M).dump
```

Also capture the API logs from the incident window:

```bash
docker compose -f docker-compose.prod.yml logs --since 24h api \
  > /var/backups/recurso/incident-$(date +%F-%H%M).api.log
```

### 3. Verify ledger integrity

Start the API again if you stopped it (reads are safe; the snapshot protects
you). Run the reconciliation report for each affected tenant:

```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/v1/finance/reconciliation | jq
```

Read it as follows:

- `total_discrepancies: 0` — billing records and the Postgres ledger agree.
  The blast radius is likely a specific payment/refund, not systemic drift.
- `total_discrepancies > 0` — look at `discrepancies[]` (`type`,
  `invoice_id`, `expected_amount`, `found_amount`). `truncated: true` means
  there are more than listed.
- `tb_compared: false` with a `tb_skip_reason` is normal — TigerBeetle
  comparison is not implemented yet (ROADMAP Track 6); Postgres is the
  authoritative ledger.

The daily reconciliation scheduler also logs
`Ledger reconciliation found discrepancies` — grep for it to find when drift
started.

### 4. Check the gateway dashboards — they are the source of truth for money

Recurso's records say what *should* have happened; the gateways say what
money **actually moved**. Before reversing anything, confirm in:

- **Stripe Dashboard** → Payments / Refunds (and Developers → Events for the
  raw event stream).
- **Razorpay Dashboard** → Transactions → Payments / Refunds.

Cross-check the disputed invoice's `gateway_payment_id` against the gateway
record. A charge that exists in Recurso but not at the gateway means no money
moved — that is a bookkeeping fix, not a customer refund.

## Triage table

Symptom → first command. Run these before forming a theory.

| Symptom | First command |
|---|---|
| Alert fired / general "is it up?" | `curl -s http://localhost:8080/health \| jq` — 200 with per-component status; 503 = Postgres down (critical); Redis/TigerBeetle down = degraded but serving |
| API not responding at all | `docker compose -f docker-compose.prod.yml ps` then `docker compose -f docker-compose.prod.yml logs --tail 200 api` |
| Customer says they were charged wrongly | Gateway dashboard first (did money move?), then `curl -s -H "Authorization: Bearer $API_KEY" "http://localhost:8080/v1/invoices?..."` for the invoice and its `gateway_payment_id` |
| Suspected ledger drift | `curl -s -H "Authorization: Bearer $API_KEY" http://localhost:8080/v1/finance/reconciliation \| jq '.total_discrepancies, .discrepancies'` |
| Payment retries failing en masse | `docker compose -f docker-compose.prod.yml logs api \| grep -E "Worker: (Payment failed\|Gateway infra error\|Max retries)"` |
| Refunds stuck | `docker compose -f docker-compose.prod.yml logs api \| grep -E "refund credit note requires manual processing\|gateway refund failed"` — then check credit notes with `refund_status` of `manual_required` / `refund_failed` |
| Gateway webhooks possibly missed | Gateway dashboard delivery logs (Stripe: Developers → Webhooks → endpoint → attempts; Razorpay: Settings → Webhooks). Recurso-side: `docker compose ... logs api \| grep -i webhook` |
| Outbound (tenant-facing) webhooks not delivered | `curl -s -H "Authorization: Bearer $API_KEY" http://localhost:8080/v1/events \| jq` — recent events; delivery worker logs: `grep -i "webhook" api logs` |
| Component flapping (repeated degraded/recovered alerts) | `docker compose -f docker-compose.prod.yml logs api \| grep "Health watcher"` — transitions with timestamps |

## Recovery

### Restore from backup

Full procedure and the verified drill are in
[deployment.md](deployment.md#backups) and
[performance.md](performance.md#backup--restore-drill-verified). Short form:

```bash
pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean incident-<ts>.dump
```

into a fresh postgres container, then start the API (migrations re-apply
automatically). After any restore, re-run `/v1/finance/reconciliation` and
compare against the gateway dashboard before re-enabling traffic — a restore
rewinds Recurso's books but **not** the money the gateways already moved.

### Replay missed gateway webhooks

If the API was down while payments happened, the gateways hold the truth and
can re-deliver it:

- **Stripe**: Dashboard → Developers → Webhooks → your endpoint → failed
  deliveries → *Resend*. Endpoint: `POST /webhooks/stripe`.
- **Razorpay**: Dashboard → Settings → Webhooks shows delivery status;
  Razorpay retries failed deliveries automatically for ~24h. For older
  events, find the payment in the dashboard and reconcile manually (mark the
  invoice paid via the normal flow, or record it as an offline payment).
  Endpoint: `POST /webhooks/razorpay`.

Webhook handlers are idempotent on payment state (a paid invoice stays paid),
so replaying an already-processed event is safe.

### The `manual_required` refund path

Credit notes of type refund are honest about what actually happened. A credit
note with `refund_status: manual_required` means **no gateway refund was
attempted** — typically because the invoice has no recorded
`gateway_payment_id` (offline payment, or paid before payment ids were
tracked). `refund_failed` means the gateway call was attempted and rejected;
the reason is in `refund_message`.

To resolve either:

1. Find the payment in the gateway dashboard (or the bank record for offline
   payments) and issue the refund there manually.
2. Record what you did in the incident notes with the gateway refund id.
3. There is currently **no API to flip the credit note to `processed`** —
   the credit note row stays `manual_required` as the honest audit record
   that the gateway refund happened out-of-band. (Refund webhook consumption
   to auto-advance states is a known gap — see ROADMAP Track 1.)

Over-refund protection is enforced at creation time (refunds cannot exceed
`amount_paid` minus prior refunds), so a manual gateway refund should always
be for exactly the credit note amount — never more.

## Post-incident

Record, in the incident log (a file in your ops repo is fine at this scale):

- Timeline: first alert / first symptom, actions taken with timestamps,
  resolution time.
- The exact commands run and their output (especially the reconciliation
  report before and after).
- Money delta: every charge/refund that moved wrongly, with gateway ids, and
  how each was corrected.
- Root cause and the code/config change that prevents recurrence.
- The pg_dump snapshot filename from step 2 (keep it beyond the normal
  14-day rotation).

**Notify affected tenants** when any of the following is true: their
customers were charged or refunded incorrectly (even if corrected), invoices
were issued with wrong amounts, or their data was restored from backup (they
may have lost recent writes). Tell them what happened, the exact customer
impact, what was corrected and how, and what they need to do (usually
nothing). Silence is how you lose a design partner; a clear same-day email is
how you keep one. SEV3s need no tenant notification unless a tenant noticed
and asked.

> ### Honest gaps (read before relying on this runbook)
>
> - **Single operator, no on-call rotation.** If the operator is asleep, the
>   webhook alert waits until morning. Mitigate with gateway-side alerts
>   (Stripe/Razorpay email on payment anomalies) and an external uptime
>   monitor on `/health`.
> - **Alerting is only `ALERT_WEBHOOK_URL`** — one POST per state
>   transition, from inside the API process. If the process or host dies,
>   nothing fires. There is no paging, no escalation, no deduplication
>   across replicas.
> - **No status page yet** (ROADMAP Track 4). Tenants find out via email
>   from the operator, not a status URL. Related: [status-page.md](status-page.md)
>   (planned status page + external uptime monitoring) and
>   [cloud-provisioning-runbook.md](cloud-provisioning-runbook.md)
>   (how a per-customer instance is stood up and monitored).
> - **No partial kill-switch**: schedulers/workers (dunning retries, mandate
>   debits, renewals) cannot be individually disabled via env — stopping
>   money movement means stopping the API.
> - **TigerBeetle is not compared during reconciliation** (`tb_compared:
>   false`); Postgres is the only ledger the report verifies.
> - **Refund states don't auto-advance** from gateway webhooks; manual
>   gateway refunds stay `manual_required` in Recurso by design, for now.
