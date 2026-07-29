# Backups & Disaster Recovery

Recurso's Postgres database is the **authoritative store** — it holds tenants,
customers, billing state, and the double-entry ledger (ADR-002). TigerBeetle,
when enabled, is a mirror; Postgres is the source of truth. Protecting Postgres
*is* protecting the business's money data. This runbook is the answer to the two
questions every buyer (and every on-call engineer) asks: **"where are the
backups, and have you ever restored one?"**

## Targets

| Metric | Target | Meaning |
|---|---|---|
| **RPO** (recovery point) | ≤ 5 min | Max data loss in a disaster. Met by managed PITR (WAL). |
| **RTO** (recovery time) | ≤ 1 hour | Time to a working, verified database. |
| **Backup retention** | 30 days PITR + 90 days snapshots | Rollback window. |
| **Restore drill cadence** | Monthly | A backup is not real until restored (see below). |

## Two layers of backup

### 1. Primary — managed automated backups + PITR (recommended for production)

On managed Postgres (Cloud SQL, RDS, Neon, Supabase) enable:

- **Automated daily snapshots** with ≥ 30-day retention.
- **Point-in-time recovery (PITR)** via continuous WAL archiving → RPO of minutes.

This is the primary path: no cron to babysit, and PITR lets you recover to any
second before an incident (e.g. just before a bad migration or a bad delete).

> Cloud SQL: `--backup-start-time` + `--enable-point-in-time-recovery`.
> RDS: automated backups + `BackupRetentionPeriod` ≥ 30.

### 2. Portable — logical dumps (self-host, and the export/migration path)

`scripts/backup.sh` takes a consistent, compressed `pg_dump -Fc` dump. Use it:

- as the **self-host** backup path (a nightly cron), and
- as a **portable export** (moving providers, taking a copy off-platform).

```bash
DATABASE_URL=postgres://…/recurso ./scripts/backup.sh /var/backups/recurso
# → /var/backups/recurso/recurso-<UTC-timestamp>.dump  (prunes >14d by default)
```

Run it on a schedule and **ship the dumps off-box** (object storage in another
region/account). A backup on the same disk as the database is not a backup.

```cron
# nightly at 02:15 UTC, then sync to object storage
15 2 * * *  DATABASE_URL=… /opt/recurso/scripts/backup.sh /var/backups/recurso && aws s3 sync /var/backups/recurso s3://recurso-backups/
```

## Restore

`scripts/restore.sh` restores a dump into a target database. It is deliberately
awkward to point at production: it requires `TARGET_DATABASE_URL` (it never falls
back to `DATABASE_URL`) and a typed confirmation.

```bash
TARGET_DATABASE_URL=postgres://…/recurso_restore ./scripts/restore.sh recurso-<ts>.dump
```

For **PITR** (managed), restore to a timestamp through the provider console/CLI,
which creates a new instance; then repoint `DATABASE_URL` once verified.

### Always verify the ledger after a production restore

Postgres is the authoritative ledger, so after any restore, **prove integrity
before trusting the database**:

```bash
# 1. The double-entry invariant harness (randomized reconciliation):
TEST_DATABASE_URL="$TARGET_DATABASE_URL" go test ./internal/service/ -run TestLedgerInvariants

# 2. And/or the live reconciler endpoint — expect zero discrepancy:
curl -s "$API/finance/reconciliation" | jq .
```

## Restore drills — the part everyone skips

**A backup you have never restored is not a backup.** `scripts/verify_backup.sh`
restores a dump into a throwaway scratch database and asserts:

- core tables (tenants, customers, subscriptions, invoices, ledger) exist and
  are populated, and
- the **double-entry ledger balances** (Σ debits_posted = Σ credits_posted).

```bash
SCRATCH_DATABASE_URL=postgres://…/recurso_verify ./scripts/verify_backup.sh recurso-<ts>.dump
# → "VERIFY OK — backup is restorable and internally consistent."
```

Run it **monthly** against the latest production dump (a scheduled job is ideal),
and record the result. A failed verify is a page-worthy event — it means today's
backups would not have saved you.

## Incident quick-reference

| Scenario | Action |
|---|---|
| Bad migration / bad bulk write | PITR to the second before it; verify ledger. |
| Instance lost | Restore latest snapshot (managed) or latest dump (`restore.sh`); verify ledger. |
| Suspected corruption | `verify_backup.sh` on the last known-good dump; restore that. |
| Provider outage | Restore the off-region dump into a new instance; repoint `DATABASE_URL`. |

See also: `docs/incident-runbook.md` (on-call playbook) and
`docs/spec_incident_alerting.md` (health alerting).
