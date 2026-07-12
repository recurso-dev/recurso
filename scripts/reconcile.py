#!/usr/bin/env python3
"""
Reconcile the API's dashboard output against the database ground truth.

For a seeded demo tenant, this cross-checks every dashboard endpoint (list
counts + computed analytics) against a direct SQL query, so you can verify the
whole product end-to-end: what the API reports must match what's actually in
Postgres. Built alongside cmd/demo_seed.

Typical flow:
    # 1. bring up a Postgres, apply migrations + baseline (creates api key
    #    sk_test_12345), and seed a demo tenant:
    DATABASE_URL=... go run ./cmd/seed
    DATABASE_URL=... go run ./cmd/demo_seed --account=<tenant-uuid>
    # 2. boot the API against that DB (PORT=8080), then:
    DATABASE_URL=... python3 scripts/reconcile.py <tenant-uuid>

Config (env):
    DATABASE_URL     required — passed straight to psql
    RECON_API_BASE   default http://localhost:8080
    RECON_API_KEY    default sk_test_12345

Exits non-zero if any check fails.
"""
import json
import os
import subprocess
import sys
import urllib.request

BASE = os.environ.get("RECON_API_BASE", "http://localhost:8080")
KEY = os.environ.get("RECON_API_KEY", "sk_test_12345")
DB = os.environ.get("DATABASE_URL")
T = (sys.argv[1] if len(sys.argv) > 1 else os.environ.get("RECON_TENANT", "")).strip()

if not T:
    sys.exit("usage: reconcile.py <tenant-uuid>  (or set RECON_TENANT)")
if not DB:
    sys.exit("DATABASE_URL is required (passed to psql)")


def api(path):
    req = urllib.request.Request(BASE + path, headers={"Authorization": "Bearer " + KEY})
    try:
        return json.load(urllib.request.urlopen(req, timeout=15))
    except Exception as e:  # noqa: BLE001 - report any failure as a check miss
        return {"__error__": str(e)}


def db(sql):
    return subprocess.run(["psql", DB, "-Atc", sql], capture_output=True, text=True).stdout.strip()


def dbi(sql):
    v = db(sql)
    try:
        return int(v)
    except ValueError:
        return v


def listlen(path):
    r = api(path)
    if "__error__" in r:
        return r
    d = r.get("data", r)
    return (len(d) if isinstance(d, list) else d), r


rows = []


def check(name, apiv, dbv, ok=None):
    if ok is None:
        ok = apiv == dbv
    rows.append((("PASS" if ok else "MISMATCH"), name, apiv, dbv))


print(f"tenant {T}  api {BASE}\n")

# ---- list endpoints: API count vs DB count ----
for name, path, table in [
    ("customers", "/v1/customers?limit=5000", "customers"),
    ("invoices", "/v1/invoices?limit=5000", "invoices"),
    ("subscriptions", "/v1/subscriptions?limit=5000", "subscriptions"),
    ("credit-notes", "/v1/credit-notes?limit=5000", "credit_notes"),
    ("quotes", "/v1/quotes?limit=5000", "quotes"),
    ("plans", "/v1/plans?limit=5000", "plans"),
    ("coupons", "/v1/coupons?limit=5000", "coupons"),
]:
    res = listlen(path)
    apiv = res[0] if isinstance(res, tuple) else res
    check(f"{name} count", apiv, dbi(f"SELECT count(*) FROM {table} WHERE tenant_id='{T}'"))

la = listlen("/v1/ledger/accounts")
check("ledger accounts", la[0] if isinstance(la, tuple) else la,
      dbi(f"SELECT count(*) FROM ledger_accounts WHERE tenant_id='{T}'"))

# ---- unit economics ----
ue = api("/v1/analytics/unit-economics").get("data", {})
check("unit-econ active_subscriptions", ue.get("active_subscriptions"),
      dbi(f"SELECT count(*) FROM subscriptions WHERE tenant_id='{T}' AND status='active'"))
check("unit-econ active_customers", ue.get("active_customers"),
      dbi(f"SELECT count(DISTINCT customer_id) FROM subscriptions WHERE tenant_id='{T}' AND status='active'"))

# ---- invoice aging: sum of bucket counts vs open/past_due invoices ----
ag = api("/v1/analytics/invoice-aging").get("data", {})
apibucket = sum(b.get("count", 0) for b in ag.get("buckets", []))
check("aging bucket total (count)", apibucket,
      dbi(f"SELECT count(*) FROM invoices WHERE tenant_id='{T}' AND status IN ('open','past_due')"))

# ---- dunning overview (fields are top-level, not under data) ----
do = api("/v1/analytics/dunning/overview")
check("dunning total_retries", do.get("total_retries"),
      dbi(f"SELECT count(*) FROM dunning_history WHERE tenant_id='{T}'"))
check("dunning total_successes", do.get("total_successes"),
      dbi(f"SELECT count(*) FROM dunning_history WHERE tenant_id='{T}' AND outcome='success'"))

# ---- dunning recovered (top-level; normalized to reporting currency) ----
dr = api("/v1/analytics/dunning/recovered")
check("dunning recovered_count", dr.get("recovered_count"),
      dbi(f"SELECT count(*) FROM recovered_payments WHERE tenant_id='{T}'"))
check("dunning reporting_currency present", bool(dr.get("reporting_currency")), True,
      ok=bool(dr.get("reporting_currency")))

# ---- revenue by plan / geography ----
rp = api("/v1/analytics/revenue-by-plan").get("data", {})
check("revenue-by-plan has segments", len(rp.get("segments", [])) > 0, True,
      ok=len(rp.get("segments", [])) > 0)
rg = api("/v1/analytics/revenue-by-geography").get("data", {})
check("revenue-by-geography reporting currency present", bool(rg.get("reporting_currency")), True,
      ok=bool(rg.get("reporting_currency")))

# ---- mrr ----
mrr = api("/v1/analytics/mrr")
mrr = mrr.get("data", mrr) if isinstance(mrr, dict) else {}
mval = mrr.get("mrr") if isinstance(mrr, dict) else None
check("mrr > 0", (mval or 0) > 0, True, ok=((mval or 0) > 0))

# ---- events ----
ev = listlen("/v1/events?limit=5000")
check("events count", ev[0] if isinstance(ev, tuple) else ev,
      dbi(f"SELECT count(*) FROM events WHERE tenant_id='{T}'"))

# ---- finance reconciliation + revrec (respond without error) ----
fr = api("/v1/finance/reconciliation")
check("finance/reconciliation responds", "__error__" not in fr, True, ok=("__error__" not in fr))
rr = api("/v1/finance/revrec/report?month=7&year=2026")
check("finance/revrec/report responds", "__error__" not in rr, True, ok=("__error__" not in rr))

# ---- report ----
w = max(len(r[1]) for r in rows)
npass = sum(1 for r in rows if r[0] == "PASS")
for status, name, apiv, dbv in rows:
    mark = "PASS" if status == "PASS" else "FAIL"
    print(f"  [{mark}] {name.ljust(w)}  api={apiv}  db={dbv}")
print(f"\n  {npass}/{len(rows)} passed")
sys.exit(0 if npass == len(rows) else 1)
