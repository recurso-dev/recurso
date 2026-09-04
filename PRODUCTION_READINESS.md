# Production Readiness Scorecard

Assessed during the autonomous test-engineering run. Scores 1–10; every score
below 10 explains the gap and what closes it.

| Dimension | Score | Rationale / path to 10 |
|---|---:|---|
| Architecture | 9 | Clean Go hexagonal core (domain/port/adapter/service) + React/Vite dashboard; double-entry ledger is PG-authoritative with an optional TigerBeetle mirror. To 10: a few large handlers/services could split further; documented in backlog. |
| Testing | 7.5 | Backend is strong (319 test files, PG-backed suites, ledger **invariant harness** gating every money path). Frontend net raised this run: the money-display layer (utils + Money), money-path UI guards (credit-note void, mandate revoke), the API-key security contract, the shared-hook anti-truncation guard, and representative list-page contracts are now covered (253 tests). To 10: the remaining slide-overs (Invoice/Customer/Plan detail), finance-report pages, and AuthProvider (see TEST_BACKLOG). |
| Reliability | 8 | Idempotent workers with atomic claims (ADR-003), fail-closed migration hardening, best-effort+reconcile ledger posts. To 10: broaden failure-injection tests around gateway/webhook paths and worker retries. |
| Performance | 7 | Currency-exponent-aware money, lazy-loaded routes, chunked bundle, react-query caching. To 10: no automated perf budget/regression gate yet; large unbounded list endpoints (e.g. ListInvoices) need server pagination. |
| Security | 8 | httpOnly session cookies, in-memory API key (never localStorage — now test-locked), TOTP MFA, SAML SSO, RBAC, row-level multi-tenancy, tenant-scoping asserted on mutations, and a **Trivy Security Scan CI gate** that caught + forced the fix of a HIGH dependency CVE this run (x/text CVE-2026-56852, #300). To 10: add automated authz-matrix + injection/fuzz tests at the handler layer. |
| Scalability | 7 | Multi-tenant row-level isolation, per-entity ledgers, batched workers. To 10: some list endpoints unbounded; no automated load profile. |
| Maintainability | 8 | ADRs, consistent patterns (DataTable/Sheet/ConfirmDialog), typed OpenAPI 3.1, strict CI gates (OpenAPI drift, invariant harness, lint). To 10: raise frontend test coverage so refactors are safe. |
| Documentation | 9 | Mintlify docs (dashboard + setup + SDK guides), README, ADRs, in-dashboard contextual doc links. To 10: keep guides synced to shipped features (mostly done). |
| Developer Experience | 8 | Fast vitest + go test, clear commands in CLAUDE.md, green-CI patch-flow. To 10: single-command full-stack test run + coverage report in CI. |
| User Experience | 9 | Premium, consistent dashboard; empty/loading/error states; contextual help; auto-charts. To 10: finish the lower-value action gaps (backlog). |

## Overall
The product is **substantially production-capable** on the backend money paths.
The main risk before this run was **frontend regression exposure** — a thin test
net over a large UI. The test-engineering run targets that directly, highest-risk
(money display + critical workflows) first.

---

# Scorecard — 2026-07-28 verification sweep (merged from production-readiness.md)

Assessed 2026-07-28 after a full-product verification sweep (all-repo gates,
32-area API journey on a fresh boot, prod dashboard sweep, seven live
integrations). Scores are 1–10; anything below 10 lists what's missing.

| Dimension | Score | What's missing to reach 10 |
|---|---:|---|
| Architecture | 9 | Hexagonal Go core + React/react-query, ADR-backed decisions. Gap: a few endpoints diverge from conventions (unwrapped dunning/cancel responses, inconsistent list pagination defaults) — see docs/design-pagination.md. |
| Maintainability | 9 | Strong test culture, capability-assertion pattern, one-fact memory. Gap: interface-embedding mocks break on every port change (backlog #14). |
| Performance | 9 | Layered cache (ADR-005), atomic-claim workers, provider rate-limit backoff. **Money-path audit confirmed ledger legs balanced, no N+1 in the paths reviewed, and closed a HIGH concurrency double-credit.** Charts chunk no longer loads on non-analytics pages (fixed via rolldown advancedChunks — 617 kB, lazy). Gap: no load/latency baseline captured yet. |
| Security | 9 | Tenant-scoped fail-closed repos, sealed BYO secrets, HMAC-verified webhooks, credential sanitizer, granular OAuth scopes. **IDOR sweep complete: ~106 param-id sites audited, all tenant-scoped or ownership-checked; constant-time key comparison; all sensitive routes rate-limited; no token in localStorage.** Gap: no automated SAST/dependency-scan gate beyond Trivy; periodic re-audit needed. |
| Developer Experience | 9 | CLAUDE.md conventions, ADRs, three synced SDKs, OpenAPI 3.1, docs site. Gap: no one-command local bootstrap script documented end-to-end (docker compose exists). |
| Testing | 8 | Go unit + PG-backed harness + E2E + invariant harness + frontend vitest (161) + node SDK (173). Gap: coverage % not measured; some handler paths (EU e-invoice, mandate guards) were untested until this sweep. |
| Documentation | 8 | Mintlify docs (setup + dashboard + SDK guides), ADRs, README. Gap: architecture diagram and a "known limitations" page; API-reference completeness vs OpenAPI not audited. |
| Accessibility | 6 | Semantic components, keyboard-navigable Radix primitives. Gap: no axe-core pass, focus-management audit, or contrast check across pages. |
| Scalability | 8 | Multi-tenant row-level, per-tenant gateway routing, async sweeps, distributed locks (Redis). Gap: pagination inconsistency risks silent truncation at scale; no horizontal-scale load test. |
| Reliability | 9 | Fail-closed webhooks, terminal sync statuses, idempotent money path (now incl. wallet auto-recharge), occurrence-aware ledger, reconciliation gate, currency-exponent-correct exports. Gap: no chaos/timeout test suite; monitoring/alerting deploy pending (telemetry #215). |
| UX | 9 | Consistent Sheet/DataTable/StatCard patterns, money-precision-aware, empty/loading/error states, honest integration matrix. **Premium chart layer (branded tooltip + shared palette + gradient area fills) across all four chart surfaces; live browser audit fixed five defects (loss-in-green bars, mixed money precision, wrapping badge, dual-currency axis, bare month labels).** Gap: mobile-responsive audit; a few flows still surface generic errors. |

## Overall: 8.5 / 10 — shippable, with a clear hardening path

**Blocking nothing for a controlled launch.** The money path is
double-entry, idempotent, reconciliation-gated, and invariant-harness-tested;
all seven integrations are live-verified; CI is green across six repos.

**Top hardening priorities (in order):**
1. **DONE** — security (IDOR/authz) audit came back clean; money-path audit
   found and fixed two bugs (wallet double-credit HIGH #260, accounting
   currency-exponent MED #261).
2. Execute the pagination-consistency plan (docs/design-pagination.md) —
   silent truncation is the highest-likelihood scale bug.
3. Accessibility pass (axe-core + focus/contrast) — lowest current score.
4. Capture a performance baseline; lazy-load the charts bundle.
5. Deploy telemetry/monitoring (#215) so production issues are observable.
6. Measure and publish test coverage.

