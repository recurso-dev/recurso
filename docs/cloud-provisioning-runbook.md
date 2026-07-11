# Cloud Provisioning Runbook

How to stand up **one paying Recurso Cloud customer** on a fresh VM, by hand,
end to end. This is the plan of record for Recurso Cloud today: **manually
provisioned, single-tenant instances for design partners** — one VM, one
Docker Compose stack, one customer per box. There is no control plane yet
(ROADMAP Track 4). When manual onboarding proves demand, this runbook becomes
the spec for automating it.

Each instance is the production Docker Compose stack from
[deployment.md](deployment.md) (`docker-compose.prod.yml`): frontend +
API + PostgreSQL + TigerBeetle. One customer's data never shares a database,
a VM, or a backup bucket with another's.

Everything below is copy-pasteable. Anything you must fill in is written
`<LIKE_THIS>`. Run through it top to bottom for each new customer; it takes
about 30–45 minutes.

**Conventions**
- Placeholders: `<CUSTOMER_SLUG>` (short a-z0-9 handle, e.g. `acme`),
  `<CUSTOMER_DOMAIN>` (e.g. `acme.cloud.recurso.dev`), `<VM_IP>`,
  `<ADMIN_SSH_PUBKEY>`, `<S3_BUCKET>`, `<S3_ENDPOINT>`, `<S3_ACCESS_KEY>`,
  `<S3_SECRET_KEY>`, `<OPS_SLACK_WEBHOOK>`.
- All host paths assume the instance lives at `/opt/recurso/<CUSTOMER_SLUG>`.
- Commands prefixed `# (local)` run on your laptop; everything else runs on
  the VM as the `deploy` user unless it says `sudo`.

---

## 0. Before you touch a VM

- [ ] Signed order / design-partner agreement, including a Data Processing
      note (you are a **Data Fiduciary's processor** under India's DPDP Act —
      offboarding in §11 is written around that).
- [ ] A DNS zone you control for `<CUSTOMER_DOMAIN>` (we use subdomains of
      `cloud.recurso.dev`).
- [ ] An S3-compatible bucket **dedicated to this customer** (see §8). We use
      one bucket per customer so offboarding is a single delete and there is
      no cross-tenant blast radius.
- [ ] The customer's gateway keys, if they're bringing their own Stripe /
      Razorpay account. If not, the stack runs on the mock gateway and you
      wire real keys later — the API degrades gracefully (see `.env.example`).
- [ ] An entry in the ops inventory (a private repo or a spreadsheet):
      customer slug, VM IP, domain, bucket, provisioned date, Recurso image
      tag deployed.

---

## 1. Size and create the VM

The measured numbers set the floor. On a single M2 Pro box the full stack
does **7,814 authenticated reads/s** and **~578 subscription-creations/s
(≈34,600 invoices/min)** with p99 under 70 ms, all bugs fixed
(see [performance.md](performance.md)). One design partner's real traffic is
orders of magnitude below that. The constraint is **PostgreSQL working set +
headroom for pg_dump**, not CPU.

| Tier | Customer profile | Spec | Provider examples |
|---|---|---|---|
| **Starter** (default) | Design partner, < 5k active subs, < 100k invoices/yr | **2 vCPU / 4 GB / 80 GB SSD** | Hetzner CPX21, DO `s-2vcpu-4gb` |
| **Standard** | Real production volume, real gateways | **4 vCPU / 8 GB / 160 GB SSD** | Hetzner CPX31, DO `s-4vcpu-8gb` |
| **Notes** | | TigerBeetle wants a stable disk; give ≥ 2× the Postgres data size free so a `pg_dump` never fills the disk | |

Start every design partner on **Starter**; resize up later (both Hetzner and
DO resize CPU/RAM in place with a reboot). Pick a region in the customer's
data jurisdiction — for Indian customers under DPDP, prefer an India region
(Hetzner has none today; DO Bangalore `blr1` does — default to `blr1` for
India-resident data).

Create the VM with **Ubuntu 24.04 LTS**, your `<ADMIN_SSH_PUBKEY>` injected at
creation, and **no password login**. Note the `<VM_IP>`.

Point DNS before hardening so TLS issuance works in §6:

```
# In your DNS provider, create an A record:
<CUSTOMER_DOMAIN>   A   <VM_IP>   (TTL 300)
```

---

## 2. First login and a non-root deploy user

```bash
# (local) — first and only root login
ssh root@<VM_IP>

# On the VM:
adduser --disabled-password --gecos "" deploy
usermod -aG sudo deploy
install -d -m 700 -o deploy -g deploy /home/deploy/.ssh
cp /root/.ssh/authorized_keys /home/deploy/.ssh/authorized_keys
chown deploy:deploy /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys

# Passwordless sudo is NOT set — you'll type the (nonexistent) password never;
# use `sudo` which works via group membership + your key. Verify from a NEW
# terminal before closing this one:
```

```bash
# (local) — confirm the deploy user works, THEN stop using root
ssh deploy@<VM_IP> 'sudo whoami'   # must print: root
```

Keep the root session open until the line above succeeds, so you can't lock
yourself out.

---

## 3. Harden the host

Run as `deploy` on the VM. Every block is idempotent.

### 3a. System update + unattended security upgrades

```bash
sudo apt-get update && sudo apt-get -y upgrade
sudo apt-get -y install unattended-upgrades
sudo dpkg-reconfigure -f noninteractive unattended-upgrades   # enables the auto timer
```

Confirm auto-reboot is off by default (we reboot on our schedule, not at
04:00 mid-charge). Check `/etc/apt/apt.conf.d/50unattended-upgrades` — leave
`Automatic-Reboot "false";`.

### 3b. SSH: keys only, no root

```bash
sudo tee /etc/ssh/sshd_config.d/10-recurso-hardening.conf >/dev/null <<'EOF'
PermitRootLogin no
PasswordAuthentication no
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
MaxAuthTries 3
AllowUsers deploy
EOF
sudo sshd -t && sudo systemctl reload ssh
```

### 3c. Firewall (ufw): only SSH + HTTP/HTTPS

Everything internal (Postgres 5432, API 8080, TigerBeetle 3000) stays on the
Docker bridge and is **never** published to the host — the compose file
already keeps Postgres and TigerBeetle unpublished. We front the box with
Caddy on 80/443 (§6) and stop publishing the API's 8080 to the world.

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp     comment 'ssh'
sudo ufw allow 80/tcp     comment 'http (ACME + redirect)'
sudo ufw allow 443/tcp    comment 'https'
sudo ufw --force enable
sudo ufw status verbose
```

### 3d. fail2ban for SSH

```bash
sudo apt-get -y install fail2ban
sudo tee /etc/fail2ban/jail.d/recurso.conf >/dev/null <<'EOF'
[sshd]
enabled = true
mode    = aggressive
maxretry = 4
bantime  = 1h
findtime = 10m
EOF
sudo systemctl enable --now fail2ban
sudo fail2ban-client status sshd
```

---

## 4. Install Docker

```bash
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
sudo apt-get update
sudo apt-get -y install docker-ce docker-ce-cli containerd.io \
  docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker deploy
newgrp docker   # or log out/in so the group applies
docker compose version   # sanity check
```

---

## 5. Deploy the stack

### 5a. Fetch the source

```bash
sudo install -d -o deploy -g deploy /opt/recurso
git clone https://github.com/recurso-dev/recurso.git /opt/recurso/<CUSTOMER_SLUG>
cd /opt/recurso/<CUSTOMER_SLUG>
# Pin to a released tag — never deploy a moving branch to a paying customer:
git checkout <RECURSO_TAG>          # e.g. v0.2.3, current release
```

### 5b. Generate per-customer secrets and write `.env`

Every instance gets **freshly generated** secrets — never reuse a password or
`API_SECRET` across customers. Generate them on the box and never let them
leave it except into the backup bucket (which is per-customer and encrypted).

```bash
cd /opt/recurso/<CUSTOMER_SLUG>
umask 077

POSTGRES_PASSWORD="$(openssl rand -base64 30 | tr -d '/+=' | cut -c1-32)"
API_SECRET="$(openssl rand -base64 48 | tr -d '/+=' | cut -c1-48)"

cat > .env <<EOF
# Recurso Cloud — <CUSTOMER_SLUG> — provisioned $(date -u +%F)
APP_ENV=production

# --- Postgres container credentials (compose derives DATABASE_URL) ---
POSTGRES_USER=recurso
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=recurso

# --- API ---
API_SECRET=${API_SECRET}
BASE_URL=https://<CUSTOMER_DOMAIN>
PORTAL_URL=https://<CUSTOMER_DOMAIN>

# --- Operational alerting (see §7) ---
ALERT_WEBHOOK_URL=<OPS_SLACK_WEBHOOK>
ALERT_WEBHOOK_FORMAT=slack

# --- Payments: leave unset for mock gateway; fill in the customer's keys ---
# STRIPE_SECRET_KEY=<STRIPE_SECRET_KEY>
# STRIPE_WEBHOOK_SECRET=<STRIPE_WEBHOOK_SECRET>
# RAZORPAY_KEY_ID=<RAZORPAY_KEY_ID>
# RAZORPAY_KEY_SECRET=<RAZORPAY_KEY_SECRET>
# RAZORPAY_WEBHOOK_SECRET=<RAZORPAY_WEBHOOK_SECRET>

# --- Email (leave unset to log to console) ---
# SMTP_HOST=<SMTP_HOST>
# SMTP_PORT=587
# SMTP_USERNAME=<SMTP_USERNAME>
# SMTP_PASSWORD=<SMTP_PASSWORD>
# SMTP_FROM=billing@<CUSTOMER_DOMAIN>
# SMTP_USE_TLS=true
EOF

chmod 600 .env
```

The variable names above are exactly those consumed by `docker-compose.prod.yml`
and the API binary (cross-checked against `.env.example`): `POSTGRES_USER`,
`POSTGRES_PASSWORD`, `POSTGRES_DB`, and `API_SECRET` are read by the compose
file; `DATABASE_URL` is derived inside compose from the Postgres trio, so you
do **not** set it here. `BASE_URL`/`PORTAL_URL`, `ALERT_WEBHOOK_URL`,
`ALERT_WEBHOOK_FORMAT`, and the payment/SMTP keys are read by the API.

> Record `POSTGRES_PASSWORD` and `API_SECRET` in your secrets manager
> (1Password/Bitwarden vault item named `recurso-cloud/<CUSTOMER_SLUG>`)
> immediately — the disk is the only other copy, and you'll need them for
> restore.

### 5c. Bring it up

```bash
cd /opt/recurso/<CUSTOMER_SLUG>
docker compose -f docker-compose.prod.yml up -d --build
# Migrations run automatically on API boot (see deployment.md).

# Wait for health (the frontend publishes :80, API is internal):
until curl -fsS http://localhost:8080/health >/dev/null 2>&1; do sleep 2; done
curl -s http://localhost:8080/health | jq   # expect 200, components green
```

Note: the compose file publishes API `8080` and frontend `80` to the host.
For a single-tenant box behind Caddy you only need the frontend on 80 (Caddy
reaches it internally). ufw already blocks 8080 from the internet in §3c, so
leaving it published to `localhost`/host is acceptable; if you want it gone
entirely, remove the `api` service's `ports:` mapping in a local override —
but do **not** edit the committed compose file (other tooling depends on it).

### 5d. Register the customer's tenant

```bash
curl -s -X POST http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"<CUSTOMER_ADMIN_EMAIL>","password":"<TEMP_STRONG_PASSWORD>","organization_name":"<CUSTOMER_ORG_NAME>"}' | jq
```

Hand the API key and the temporary password to the customer over a secure
channel; tell them to rotate the password on first login.

---

## 6. TLS — Caddy

**Decision: Caddy, not Traefik.** For a single-tenant, single-box instance
Caddy is the right tool: automatic HTTPS with ACME (issuance + renewal, zero
cron), HTTP→HTTPS redirect, and OCSP stapling from a two-line Caddyfile, as a
single static binary with no dependency on Docker labels or a dynamic config
provider. Traefik earns its complexity when you're routing many services or
doing dynamic service discovery in a cluster — neither is true here, and its
label-driven config is more moving parts to get wrong per box. We run Caddy on
the host (not in compose) so TLS survives `docker compose down` and so the
committed compose file stays untouched.

```bash
sudo apt-get -y install debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | \
  sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | \
  sudo tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
sudo apt-get update && sudo apt-get -y install caddy

sudo tee /etc/caddy/Caddyfile >/dev/null <<'EOF'
<CUSTOMER_DOMAIN> {
    encode gzip
    reverse_proxy 127.0.0.1:80
}
EOF
# ^ replace <CUSTOMER_DOMAIN> with the real host before reloading.

sudo systemctl restart caddy
sudo systemctl enable caddy

# Verify TLS end to end (DNS from §1 must already resolve to this box):
curl -sSI https://<CUSTOMER_DOMAIN>/health
```

Caddy proxies to the frontend container on `127.0.0.1:80`, which in turn
proxies `/v1`, `/auth`, `/portal/*`, `/checkout` to the API internally (see
`frontend/nginx.conf`). ACME certs auto-renew; no further action.

---

## 7. Monitoring hookup

Two layers, matching the honest limits called out in
[incident-runbook.md](incident-runbook.md):

1. **Built-in health watcher → shared ops Slack.** Already wired via the
   `.env` from §5b: `ALERT_WEBHOOK_URL=<OPS_SLACK_WEBHOOK>` and
   `ALERT_WEBHOOK_FORMAT=slack`. The API checks Postgres/Redis/TigerBeetle
   every 60 s and POSTs Slack messages on **state transitions only** (down +
   recovered), one per transition, no repeats. Postgres down = `critical`;
   Redis/TigerBeetle = `warning`. Use **one shared `#recurso-ops` channel**
   for all customers and prefix nothing — the payload already reads
   `[CRITICAL] <title> — <body>`; add the customer via the Slack incoming
   webhook's channel or a per-customer webhook if you want them separable.

   Verify a real alert fires end to end once, right after provisioning:

   ```bash
   docker compose -f docker-compose.prod.yml stop postgres   # trigger a transition
   # expect a [CRITICAL] message in #recurso-ops within ~60s
   docker compose -f docker-compose.prod.yml start postgres   # expect a recovery message
   ```

2. **External uptime check on `/health`.** The watcher runs *inside* the API
   process, so it cannot tell you the process or host died. Cover that with an
   external monitor hitting `https://<CUSTOMER_DOMAIN>/health` — this is
   exactly what the status page provides. Add every new instance to it as part
   of provisioning (see [status-page.md](status-page.md)).

---

## 8. Backups — per-customer S3, with retention

Adapt the nightly `pg_dump` from [deployment.md](deployment.md) to ship each
customer's dump to **their own** S3-compatible bucket. Postgres is the system
of record; TigerBeetle needs no backup (it rebuilds from Postgres — see
deployment.md). The drill in [performance.md](performance.md#backup--restore-drill-verified)
(destroy volumes, restore from `pg_dump`, counts identical) is the procedure
this section automates and keeps honest.

### 8a. Install and configure rclone (per-customer remote)

```bash
sudo -v ; curl https://rclone.org/install.sh | sudo bash
mkdir -p /home/deploy/.config/rclone
cat > /home/deploy/.config/rclone/rclone.conf <<EOF
[cust-s3]
type = s3
provider = Other
env_auth = false
access_key_id = <S3_ACCESS_KEY>
secret_access_key = <S3_SECRET_KEY>
endpoint = <S3_ENDPOINT>
acl = private
EOF
chmod 600 /home/deploy/.config/rclone/rclone.conf
# Enable server-side encryption at the bucket level in the provider console,
# or use a bucket with default SSE. Verify write access:
echo ok | rclone rcat cust-s3:<S3_BUCKET>/healthcheck.txt && \
  rclone delete cust-s3:<S3_BUCKET>/healthcheck.txt
```

### 8b. Backup script

```bash
sudo tee /opt/recurso/<CUSTOMER_SLUG>/backup.sh >/dev/null <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
DIR=/opt/recurso/<CUSTOMER_SLUG>
STAMP=$(date -u +%F-%H%M)
OUT=/var/backups/recurso/<CUSTOMER_SLUG>-${STAMP}.dump
mkdir -p /var/backups/recurso
cd "$DIR"
# shellcheck disable=SC1091
set -a; . "$DIR/.env"; set +a
docker compose -f docker-compose.prod.yml exec -T postgres \
  pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB" > "$OUT"
# Ship off-host to the customer's own bucket:
rclone copyto "$OUT" "cust-s3:<S3_BUCKET>/db/$(basename "$OUT")"
# Local retention: keep 7 days on the box (bucket holds the long tail).
find /var/backups/recurso -name '<CUSTOMER_SLUG>-*.dump' -mtime +7 -delete
EOF
sudo chmod 750 /opt/recurso/<CUSTOMER_SLUG>/backup.sh
sudo chown deploy:deploy /opt/recurso/<CUSTOMER_SLUG>/backup.sh
```

### 8c. Schedule it + retention in the bucket

```bash
# Nightly at 02:30 UTC:
( crontab -l 2>/dev/null; \
  echo "30 2 * * * /opt/recurso/<CUSTOMER_SLUG>/backup.sh >> /var/log/recurso-backup.log 2>&1" ) | crontab -

# Bucket-side retention: 30 daily, then let a lifecycle rule expire objects.
# Set an S3 lifecycle rule on <S3_BUCKET> prefix db/ to expire after 35 days.
# (Do this in the provider console; keeps a ~1-month window without unbounded
# storage growth. For a customer needing longer retention, raise per contract.)
```

**Retention policy (default):** 7 days on-box, 30–35 days in the bucket via
lifecycle. Bump per contract for customers with a regulatory retention
requirement; note it in the ops inventory.

### 8d. Restore-test cadence — **monthly, non-negotiable**

A backup you have never restored is a hope, not a backup. Once a month, on a
throwaway VM or locally, prove the newest dump restores clean:

```bash
# On a scratch box with the repo checked out and a fresh empty stack up:
rclone copyto cust-s3:<S3_BUCKET>/db/<LATEST>.dump ./restore.dump
docker compose -f docker-compose.prod.yml up -d postgres
set -a; . .env; set +a
docker compose -f docker-compose.prod.yml exec -T postgres \
  pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists < ./restore.dump
# Verify: row counts sane, API boots, /health green, keys authenticate.
```

Log the date + result in the ops inventory. This mirrors the verified drill in
[performance.md](performance.md#backup--restore-drill-verified). Tear the
scratch box down afterward.

---

## 9. Upgrades

Bump the image, migrate safely, be ready to roll back. Migrations apply
automatically on API boot (deployment.md), so upgrade = deploy a new tag.

```bash
cd /opt/recurso/<CUSTOMER_SLUG>

# 1. ALWAYS back up first — this is your rollback point for data.
./backup.sh

# 2. Note the current tag for rollback:
git describe --tags   # record <PREVIOUS_TAG>

# 3. Fetch and check out the new release tag:
git fetch --tags
git checkout <NEW_RECURSO_TAG>

# 4. Rebuild and restart. Migrations run on API boot.
docker compose -f docker-compose.prod.yml up -d --build

# 5. Verify:
until curl -fsS http://localhost:8080/health >/dev/null 2>&1; do sleep 2; done
curl -s http://localhost:8080/health | jq
# Smoke-test the reconciliation report — a clean ledger post-upgrade is the
# strongest signal migrations didn't corrupt anything:
curl -s -H "Authorization: Bearer <TENANT_API_KEY>" \
  http://localhost:8080/v1/finance/reconciliation | jq '.total_discrepancies'
```

**Migration safety.** Recurso migrations are forward-only and run at boot;
there is no automated down-migration. Treat the pre-upgrade `pg_dump` as the
only reliable rollback for schema changes. Read the CHANGELOG for the target
tag before upgrading a paying customer, and stage the upgrade on a scratch
restore of *their* latest dump when the release notes mention schema changes.

**Rollback.**

```bash
# Code-only regression (no migration ran, or migration is backward-compatible):
git checkout <PREVIOUS_TAG>
docker compose -f docker-compose.prod.yml up -d --build

# Schema-breaking regression — restore the pre-upgrade dump:
docker compose -f docker-compose.prod.yml stop api
set -a; . .env; set +a
docker compose -f docker-compose.prod.yml exec -T postgres \
  pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists \
  < /var/backups/recurso/<CUSTOMER_SLUG>-<PRE_UPGRADE_STAMP>.dump
git checkout <PREVIOUS_TAG>
docker compose -f docker-compose.prod.yml up -d --build
```

After any rollback, re-run `/v1/finance/reconciliation` and compare against
the gateway dashboards before re-enabling traffic — a restore rewinds
Recurso's books but not money the gateways already moved
(see [incident-runbook.md](incident-runbook.md)).

Notify the customer of the maintenance window in advance; keep upgrades short
and outside their peak billing hours.

---

## 10. Health checklist (post-provisioning sign-off)

- [ ] `https://<CUSTOMER_DOMAIN>/health` returns 200, all components green
- [ ] Valid TLS cert (Caddy issued), HTTP redirects to HTTPS
- [ ] ufw active: only 22/80/443 open; 8080/5432/3000 not reachable externally
- [ ] SSH: root + password login refused, fail2ban `sshd` jail active
- [ ] unattended-upgrades enabled, auto-reboot off
- [ ] `.env` is `chmod 600`; `POSTGRES_PASSWORD` + `API_SECRET` saved to vault
- [ ] First `backup.sh` run succeeded and the dump landed in `<S3_BUCKET>`
- [ ] Test `[CRITICAL]` + recovery alert seen in `#recurso-ops`
- [ ] Instance added to the external uptime monitor / status page
- [ ] Tenant registered; API key + temp password handed over securely
- [ ] Ops inventory row created (slug, IP, domain, bucket, tag, date)

---

## 11. Offboarding (DPDP-conscious)

When a customer leaves, you are winding down processing of personal data on
behalf of a Data Fiduciary. Under India's DPDP Act, the default after the
purpose ends is **erasure** — you delete unless a law requires retention.
Do it deliberately and leave a paper trail.

### 11a. Final backup + data handover

```bash
cd /opt/recurso/<CUSTOMER_SLUG>
./backup.sh   # final Fc dump to the customer's bucket

# Handover format the customer can actually use:
#  1. The pg_custom dump (full fidelity, for a Recurso re-import or their DBA).
#  2. Portable CSVs of their core business records, so they aren't locked in.
set -a; . .env; set +a
for T in tenants customers plans subscriptions invoices invoice_line_items \
         payments credit_notes ledger_transactions; do
  docker compose -f docker-compose.prod.yml exec -T postgres \
    psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    -c "\copy (SELECT * FROM ${T}) TO STDOUT WITH CSV HEADER" \
    > /var/backups/recurso/<CUSTOMER_SLUG>-handover-${T}.csv
done
```

Deliver the dump + CSVs over a secure, expiring link. Get written
confirmation of receipt. (Verify the table list against the current schema in
`migrations/` before running — table names above reflect the core billing
tables; skip any that don't exist in the deployed tag.)

### 11b. Teardown checklist

- [ ] Final backup shipped and **receipt confirmed in writing** by the customer
- [ ] Handover CSVs + dump delivered; agreed retention window for the final
      backup recorded (default: delete after `<RETENTION_DAYS>`, e.g. 30)
- [ ] Remove the instance from the uptime monitor / status page
- [ ] Remove the customer's gateway webhook endpoints (Stripe/Razorpay
      dashboards) so no further events fire at a dead host
- [ ] `docker compose -f docker-compose.prod.yml down -v` — **destroys the
      Postgres and TigerBeetle volumes** (all personal data on the box)
- [ ] Destroy the VM at the provider (deletes disk + any provider snapshots —
      check the provider doesn't retain a snapshot)
- [ ] Delete the DNS record for `<CUSTOMER_DOMAIN>`
- [ ] After `<RETENTION_DAYS>`, empty and delete `<S3_BUCKET>`
      (`rclone purge cust-s3:<S3_BUCKET>`) and revoke `<S3_ACCESS_KEY>`
- [ ] Delete the vault item `recurso-cloud/<CUSTOMER_SLUG>` (secrets)
- [ ] Mark the customer offboarded in the ops inventory with dates for each
      deletion step above (this is your DPDP erasure record)

The dated inventory row is the evidence that erasure happened — keep the row
(metadata only, no personal data) even after everything else is gone.
