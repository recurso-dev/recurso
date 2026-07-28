# Engineering backlog

Ranked by ROI (impact ÷ effort). Updated 2026-07-28 (overnight session).
Struck as already-shipped on re-audit: per-entity GSTR-1/3B (`?entity_id=` +
primary-aware GSTIN, #205/#208) and per-entity MRR UI (ExecutiveSummary +
Entities pages).
Items marked **founder** are blocked on credentials/infrastructure only the
founder can provide; everything else is engineering-ready.

## P0 — money-path correctness

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| 2 | **Xero-invalid customer email** (`bed15f4d…`) | MED — one customer's invoices never sync (QuickBooks rejects it too) | — | **founder** fixes the email in the dashboard; sync rows now show the customer name + id. |

## P1 — verification & parity (mostly founder-blocked)

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| 3 | **QuickBooks live OAuth verification** | HIGH — parity claim untested; Xero verification found 3 real bugs, QBO likely has rot too | LOW once creds exist | **founder**: developer.intuit.com → create app → redirect URIs `http://localhost:8199/v1/accounting/callback/quickbooks` + `https://api.recurso.dev/v1/accounting/callback/quickbooks` → `QBO_CLIENT_ID`/`QBO_CLIENT_SECRET` into `recur-so/.env`. |
| 4 | **GoCardless webhook registration** | HIGH — mandate activation + settlement (#238/#240) are dead until GC can reach us | LOW | **founder**: GC dashboard → webhook endpoint `https://api.recurso.dev/webhooks/gocardless` + secret → `GOCARDLESS_WEBHOOK_SECRET` on Cloud Run (BYO tenants use the per-connection URL on the Payment Gateways card). |
| 5 | Telemetry receiver deploy (#215) | MED — adoption visibility | LOW | **founder**: 4 wrangler commands in `telemetry-worker/README.md`. |
| 6 | `TRAFFIC_TOKEN` org secret | MED — traffic history beyond GitHub's rolling 14 days | LOW | **founder**: classic PAT with repo scope, org-level secret, then re-dispatch `traffic-snapshot.yml` in all 6 repos. |
| 7 | Real Peppol AP creds (EU e-invoicing inc 2) | MED | MED | **founder** account; retry worker (#89) is merged and waiting. |
| 8 | Demo sandbox hosting + `VITE_DEMO_URL` | MED — website CTA is dark | LOW | **founder**: hosting + DNS; code 100% ready (#214, website #24). |

## P2 — product completeness

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| 11 | Gift-subscription cancel + wallet-close UI edge cases | LOW | LOW | Deferred from roadmap run 2026-07-20. |
| 12 | Dunning alert edit UI | LOW | LOW | Deferred from roadmap run 2026-07-20. |

## P2b — smoke-sweep findings (2026-07-28, see docs/verification-2026-07-28.md)

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| S1 | Card-level accounting Sync should be incremental (force=false) | HIGH — forced full re-push exceeds the 15-min budget on real tenants | LOW | Keep force=true only on the header Sync-now; dirty-tracking already exists. |
| S2 | Nested `billing_address` silently ignored on customer create/update | MED — silent data loss for API users | LOW | Accept nested (map to flat) or reject with 400. |
| S3 | No `GET /v1/subscriptions/:id` | MED — API-completeness gap | LOW | Add route + OpenAPI; SDKs regenerate. |
| S4 | Quote totals stay 0 when created with line items via API | MED — CPQ math broken for API path (UI path computes) | MED | Compute totals server-side on create/update. |
| S5 | Entity create ignores `country` in request/response | LOW-MED | LOW | Bind + persist + echo country_code. |

## P3 — engineering hygiene

| # | Item | Impact | Effort | Notes |
|---|------|--------|--------|-------|
| 12b | React 19 + react-router 8 upgrade | MED — clears the Trivy-ignored RSC advisory (GHSA-qwww-vcr4-c8h2, not exploitable in this SPA) and unblocks future deps | MED | react-router 8.3.0 needs React >=19.2; remove `.trivyignore` entry when done. |
| 13 | Pagination consistency on list endpoints | MED — silent truncation has bitten twice (CLAUDE.md) | MED | A few endpoints default `limit=10`, some 50/100/200, many unbounded. Normalize on `ParsePagination` + document defaults in OpenAPI. |
| 14 | Interface-embedding test mocks | LOW-MED — every port widening breaks/panics mocks (`mockLedgerRepoFor*`, `stubCollectionsAgg`, …) | MED | Either generate mocks or convert to narrow per-test interfaces (capability-assertion pattern used by webhook/CRM paths is the house style now). |
| 15 | Dunning-campaign + cancel-flow responses are unwrapped (no `{data:}`) | LOW — known API quirk, clients must stay tolerant | LOW | Breaking change; batch with a future v2 or additive alias. |
| 16 | `head` HTTP-tool alias footgun, iCloud `" 2"` duplicate files | — | — | Environment quirks, documented in memory; no code change. |

## Recently closed (context for the ranking)

- BYO GoCardless from the dashboard (#237), mandate activation webhooks
  (#238), payment settlement webhooks (#240), currency-aware mandate UI
  (#241), chargeback/late-failure settlement reversal (ACH-parity, reusing
  the #209 occurrence-aware ledger machinery) — overnight 2026-07-27/28.
- Manual accounting sync async + single-flight (#239) — last known
  Cloudflare-timeout landmine.
- QA sweep of #199–#207: all findings closed (#208–#212).
- Live-verified integrations: Stripe (test), Razorpay (test), TaxJar
  (sandbox), Xero (production, at scale), HubSpot (production), GoCardless
  (sandbox, mandate + debit).
