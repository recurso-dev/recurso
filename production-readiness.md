# Production readiness scorecard

Assessed 2026-07-28 after a full-product verification sweep (all-repo gates,
32-area API journey on a fresh boot, prod dashboard sweep, seven live
integrations). Scores are 1–10; anything below 10 lists what's missing.

| Dimension | Score | What's missing to reach 10 |
|---|---:|---|
| Architecture | 9 | Hexagonal Go core + React/react-query, ADR-backed decisions. Gap: a few endpoints diverge from conventions (unwrapped dunning/cancel responses, inconsistent list pagination defaults) — see docs/design-pagination.md. |
| Maintainability | 9 | Strong test culture, capability-assertion pattern, one-fact memory. Gap: interface-embedding mocks break on every port change (backlog #14). |
| Performance | 8 | Layered cache (ADR-005), atomic-claim workers, provider rate-limit backoff. **Money-path audit confirmed ledger legs balanced, no N+1 in the paths reviewed, and closed a HIGH concurrency double-credit.** Gap: no load/latency baseline captured; charts bundle ~968 kB — lazy-load candidate. |
| Security | 9 | Tenant-scoped fail-closed repos, sealed BYO secrets, HMAC-verified webhooks, credential sanitizer, granular OAuth scopes. **IDOR sweep complete: ~106 param-id sites audited, all tenant-scoped or ownership-checked; constant-time key comparison; all sensitive routes rate-limited; no token in localStorage.** Gap: no automated SAST/dependency-scan gate beyond Trivy; periodic re-audit needed. |
| Developer Experience | 9 | CLAUDE.md conventions, ADRs, three synced SDKs, OpenAPI 3.1, docs site. Gap: no one-command local bootstrap script documented end-to-end (docker compose exists). |
| Testing | 8 | Go unit + PG-backed harness + E2E + invariant harness + frontend vitest (161) + node SDK (173). Gap: coverage % not measured; some handler paths (EU e-invoice, mandate guards) were untested until this sweep. |
| Documentation | 8 | Mintlify docs (setup + dashboard + SDK guides), ADRs, README. Gap: architecture diagram and a "known limitations" page; API-reference completeness vs OpenAPI not audited. |
| Accessibility | 6 | Semantic components, keyboard-navigable Radix primitives. Gap: no axe-core pass, focus-management audit, or contrast check across pages. |
| Scalability | 8 | Multi-tenant row-level, per-tenant gateway routing, async sweeps, distributed locks (Redis). Gap: pagination inconsistency risks silent truncation at scale; no horizontal-scale load test. |
| Reliability | 9 | Fail-closed webhooks, terminal sync statuses, idempotent money path (now incl. wallet auto-recharge), occurrence-aware ledger, reconciliation gate, currency-exponent-correct exports. Gap: no chaos/timeout test suite; monitoring/alerting deploy pending (telemetry #215). |
| UX | 9 | Consistent Sheet/DataTable/StatCard patterns, money-precision-aware, empty/loading/error states, honest integration matrix. Gap: mobile-responsive audit; a few flows still surface generic errors. |

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
