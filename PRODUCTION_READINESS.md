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
