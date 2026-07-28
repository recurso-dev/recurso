# Full-product verification — 2026-07-28

Whole-product sweep run by the autonomous session: build/test gates across all
six repos, a 32-area API journey against a fresh local boot (mock gateway),
a production dashboard page sweep, and live third-party integration checks.
Evidence-based: every PASS below was exercised, not assumed.

## Gates (all repos)

| Gate | Result |
|---|---|
| recurso: `go build` + full `go test` | PASS |
| recurso: PG-backed service suite (ledger/rev-rec/invariant harness) | PASS |
| recurso frontend: eslint (max-warnings 0) / vite build / vitest (161) | PASS |
| recurso-go SDK: build + tests | PASS |
| recurso-node SDK: typecheck + tests (173) | PASS |
| recurso-website: SSG build | PASS |
| Prod surfaces (api/app/site/docs) | 200; OpenAPI 3.1 served |
| Main-branch CI (recurso) | green |

## API journey (fresh boot, 32 areas)

PASS (27): auth/tenancy (incl. 401s), customers, invoices+GST math
(50000 + 9000 IGST = 59000), coupons (20% → 40000 + 7200 GST), usage
ingestion + charge simulation, wallets (create/top-up/balance), credit-note
surface, gifts, referrals, dunning + campaigns, cancel flows, churn,
mandates (EUR mock + INR guards), gateway connections (incl. zero-width
sanitizer + non-ASCII 400), outbound webhook lifecycle, inbound webhook
fail-closed (503 w/o secrets), multi-entity, ledger reconciliation
(0 discrepancies), rev-rec + month-end close, analytics suite, collections
intelligence, accounting-sync surface (202/400/status+total), CRM-sync
guard, US nexus, GSTR-1 gov schema, portal magic-link, API keys/team,
disputes, OpenAPI.

FIXED during the sweep:
- **EU e-invoice GET/RETRY 500'd on every call** — `ownedInvoice` read the
  tenant-scoped invoice repo without injecting the tenant (tenant-context
  bug class). Fixed + regression test.
- **INR mandate guard failures returned generic 500** — phone/VPA guard
  errors are now classified 400 with their message. Fixed + regression test.
- **Quotes page showed raw customer UUIDs** — now `CustomerName` (same
  pattern as every other page).

PARTIAL (tracked in backlog):
- Customer create/update accept only FLAT address fields; a nested
  `billing_address` object is silently ignored.
- No `GET /v1/subscriptions/:id` route (list/cancel only).
- Quote totals stay 0 when created with line items via API.
- Entity create ignores/doesn't echo `country`.

## Production dashboard sweep

Home, Invoices, Metering, Dunning, Collections, Wallets, Quotes all render
with live data; zero console errors; unknown routes fall back to Home.

## Live integrations (verified this session, production/sandbox)

| Integration | Status |
|---|---|
| Stripe (test keys) | verified (earlier session, checkout/ACH) |
| Razorpay (test keys) | verified (earlier session) |
| TaxJar (sandbox) | verified — destination-based TX calc |
| Xero (production org) | verified at scale; 429 pacing fixed (#250) |
| QuickBooks (sandbox company, prod API) | verified — customers+invoices pushed |
| HubSpot (production) | verified — 25-contact batched sync |
| GoCardless (BYO, founder sandbox) | verified END-TO-END: connect → EUR mandate → hosted authorization → webhook activation (`active`) |

## Known non-blockers

- One customer (`bed15f4d…`, "Pied Piper Inc") has an email both Xero and
  QuickBooks reject — founder to correct.
- Card-level accounting Sync force-re-pushes everything; incremental card
  sync is the top backlog item.
- Xero connection row shows Last sync "—" until its next sync stamps it.
