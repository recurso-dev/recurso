# Remediation plan — 2026-08-04 full product audit

Owner: engineering (Claude, acting Principal Engineer + TPM). Source: the
2026-08-04 CTO-level audit (27 ranked findings; scores Product 82 / Eng 86 /
Security 84 / UX 80 / Perf 83 / Prod-readiness 80 / Accounting integrity 96).

**This file is the single source of truth for remediation status. It is
updated after every merged PR.** Each finding carries: business impact,
technical root cause, the long-term fix, PR breakdown (<500 LOC where
practical), affected files, migration risk, tests, acceptance criteria,
effort (S/M/L), and dependencies.

Legend — Status: ✅ done · 🔧 in progress · ⬜ planned · 🧭 product decision.
Effort: S (<½ day) · M (½–2 days) · L (>2 days).

---

## Milestones

| Milestone | Window | Theme | Findings |
|---|---|---|---|
| **M0 — Immediate** | before next release | customer-trust + the tax gap | #466, tax-preferred-provider, error-states (#7a), portal-auth follow-ups (#18) |
| **M1 — Short term** | ≤ 2 weeks | portal parity + hardening | #11, #8, #10, #12, #25 |
| **M2 — Medium term** | ≤ 1 month | DX + consistency | #7 (full), #9, #13, #19, #20, #21, #23, #24, #26 |
| **M3 — Long term** | technical debt | structural | #14, #15, #16, #17, #22, #27 |

Already shipped during the audit (baseline): #1 (#483), #2/#3 (#484),
#4/#5 (#482), chart-freeze (#481), pagination wave (#480). These are the six
fixes the audit counted as done.

---

## Progress log (newest first)

- **2026-08-04** — #466 Track B (tie-out reframe) + #7a (Usage error state)
  shipping as a frontend PR. **Correction on #7a**: on verification,
  OfflinePayments and Churn *already* render retryable error states (via
  `DataTable error=`/inline) — the agent finding was inaccurate for those two;
  only Usage's top-level stats fetch genuinely rendered nothing. Fixed Usage;
  #7a is now complete.
- **2026-08-04** — TAX-PREFERRED-PROVIDER **MERGED (#485)**.
- **2026-08-04** — Plan created. TAX-PREFERRED-PROVIDER fix implemented
  (one active provider per category; PG test green) — shipping as the first
  remediation PR.

---

## M0 — Immediate (before next release)

### TAX-PREFERRED-PROVIDER — multi-provider tax selection is silent 🔧
- **Impact**: a tenant connecting two tax providers had no control over which
  one computed tax; TaxJar silently won by iteration order — a wrong-rate risk
  and a confusing UX.
- **Root cause**: `IntegrationConnection.active` is per `(tenant, category,
  provider)`, so two tax providers could both be active; the resolver returned
  the first in a fixed list.
- **Fix (long-term)**: one active provider per category. `Upsert` deactivates
  the whole category before inserting the new active row, so the *connect*
  action is the selection. Tax is the only multi-provider category, so this is
  correct everywhere.
- **PRs**: single PR (~40 LOC).
- **Files**: `internal/adapter/db/integration_connection_repository.go`,
  `internal/service/salestax_resolver.go` (comment),
  `frontend/src/components/IntegrationConnections.jsx` (blurb),
  `internal/adapter/db/integration_connection_pg_test.go` (new).
- **Migration risk**: none (no schema change). Existing tenants with two active
  tax providers keep both until the next connect, which then normalizes — safe.
- **Tests**: PG test — connect TaxJar then Ziptax ⇒ exactly one active tax
  provider, Ziptax active, TaxJar deactivated. ✅ green.
- **Acceptance**: connecting a tax provider shows exactly one Connected badge in
  the Tax section; the resolver returns the just-connected provider.
- **Effort**: S. **Deps**: none. **Status**: ✅ MERGED #485.

### #466 — Deferred tie-out shows "Unexplained delta" 🧭 + 🔧
- **Impact**: HIGHEST trust-ROI. The flagship Month-End Close shows
  `Unexplained delta −$26,854` on live data → a prospect doubts the books.
- **Root cause**: revenue defers at *issuance* (ledger Code-1 AR→Deferred) but
  the recognition *schedule* is only built at *payment*. With $712k awaiting
  payment vs $4,274 scheduled, `ledger_deferred == scheduled + awaiting_payment`
  leaves a residual the UI labels "unexplained". On the demo tenant the residual
  is also inflated by pre-#474 write-offs lacking their code-22/23 legs (stale
  seed).
- **Fix — two tracks**:
  - **Track B (immediate, low-risk, SHIP FIRST)**: reframe the UI. When the
    delta is 0, show a green "Ties out". When non-zero, never say
    "unexplained" — show it as *Awaiting recognition (schedules build at
    payment)* with the awaiting-payment figure foregrounded and a one-line
    explanation. Removes the trust hit today.
  - **Track A (product decision, 🧭)**: build recognition schedules at
    *issuance* (accrual), so the tie-out is structurally zero. This changes
    revenue *timing* semantics (deferred-at-issuance recognized over the
    period regardless of payment) — a genuine accounting-policy call the
    founder must approve before implementation. Needs: schedule creation moved
    to invoice generation, migration/backfill for open invoices, revrec +
    reconciler alignment, and a decision on unpaid-then-written-off handling.
- **PRs**: Track B — 1 PR (~120 LOC, frontend + close_pack labels). Track A —
  epic (3–4 PRs, migration + revrec + tests), gated on founder sign-off.
- **Files (Track B)**: `frontend/src/pages/MonthEndClose.jsx`,
  `internal/service/close_pack.go` (field naming/label only).
- **Migration risk**: Track B none. Track A high (revenue-timing change +
  backfill) — do not start without the decision on issue #466.
- **Tests**: Track B — close-pack view test asserts zero-delta shows "Ties
  out" and non-zero shows the awaiting-recognition framing, never "unexplained".
- **Acceptance**: no screen ever shows the word "unexplained" to a customer;
  a healthy tenant shows a green tie-out.
- **Effort**: Track B S–M, Track A L. **Deps**: Track A needs the demo reseed
  (to confirm residual is data-only) + founder decision.
- **Status**: 🧭 Track A awaiting decision; 🔧 Track B shipping now.

### #7a — Three pages render nothing on API error ⬜
- **Impact**: `Usage`, `OfflinePayments`, `Churn` show a blank/empty view when
  the fetch fails → users read "no data" and file a ticket.
- **Root cause**: hand-rolled `useEffect` fetch with no error branch in render.
- **Fix**: add explicit Error state (retry) to each; part of the broader
  ADR-005 migration (#7) but shipped standalone first because it's the
  support-ticket generator.
- **PRs**: 1 PR (~90 LOC).
- **Files**: `frontend/src/pages/{Usage,OfflinePayments,Churn}.jsx`.
- **Tests**: each page's test renders an errored query → asserts an "Unable to
  load" + retry control is shown.
- **Acceptance**: forcing the API to 500 shows an error state with retry, not a
  blank page. **Effort**: S. **Deps**: none. **Status**: ⬜ next after #466-B.

### #18 — Portal magic-link hardening ⬜
- **Impact**: token in URL query leaks via Referer/history/logs; verify errors
  are a state oracle (expired vs used vs unknown).
- **Fix**: consume the token from a POST body; collapse verify errors to one
  generic message (dashboard already does both).
- **PRs**: 1 PR (~80 LOC).
- **Files**: `internal/adapter/handler/portal_api.go`, portal verify page.
- **Migration risk**: the emailed link shape changes — support the old
  `?token=` GET for one release, then remove.
- **Tests**: verify handler accepts POST body; error responses are identical
  across states. **Effort**: M. **Deps**: coordinate with the portal test PR
  (#11). **Status**: ⬜.

---

## M1 — Short term (≤ 2 weeks)

### #11 — Portal pages untested ⬜
- **Impact**: payment-method + gift-redeem are money paths with zero test
  coverage. **Fix**: behavioral tests for the 5 portal pages (login,
  payment-method, redeem, verify, dashboard). **Files**:
  `frontend/src/pages/portal/__tests__/*` (new). **Tests**: happy + error +
  empty per page; assert money figures and the redeem/payment flows.
  **Acceptance**: each portal page has a dedicated test asserting behavior, not
  just render. **Effort**: M. **Deps**: pairs with #18. **Status**: ⬜.

### #8 — Expensive endpoints only under the global rate limit ⬜
- **Impact**: one key can issue 500 GL exports / import commits per minute.
- **Root cause**: import-commit/compare, PDF & GL-export inherit only the
  `api` 500/min bucket. **Fix**: add an `expensive` scope (10–30/min) and apply
  it to those routes. **Files**: `cmd/api/main.go`,
  `internal/adapter/middleware/rate_limit.go`. **Migration risk**: low —
  legitimate callers stay well under the cap; log when throttled. **Tests**:
  middleware test — Nth+1 request in the window gets 429. **Acceptance**: the
  named routes 429 past the bucket; normal use unaffected. **Effort**: M.
  **Deps**: none. **Status**: ⬜.

### #10 — Request/tenant IDs absent from service logs ⬜
- **Impact**: a production incident can't be reconstructed — ledger/sync/email
  failures aren't correlatable. **Root cause**: only 3 of 273 service log calls
  carry `request_id`. **Fix**: carry `request_id`/`tenant_id`/`user_id` on
  `ctx`, add an slog handler that extracts them, so every log line downstream is
  tagged. **Files**: `internal/adapter/middleware/*`, a shared `logctx` helper,
  service call sites (mechanical). **Migration risk**: none. **Tests**: a
  handler-level test asserts a service log emitted during the request carries
  the request id. **Acceptance**: grepping logs by `request_id` returns the full
  chain across middleware→service. **Effort**: M. **Deps**: none. **Status**: ⬜.

### #12 — Currency validation not centralized ⬜
- **Impact**: inconsistent, no ISO-4217 set check. **Fix**: shared
  `validator.Currency/Country/Amount/Percentage/Email` (gin binding tags +
  helpers); replace ad-hoc `len==3`. **Files**: new `internal/validate` pkg,
  binding structs across handlers. **Migration risk**: low (stricter input —
  audit callers first). **Tests**: table test of the validators. **Acceptance**:
  an invalid currency/country is rejected at bind with a clear message.
  **Effort**: M. **Deps**: none. **Status**: ⬜.

### #25 — Portal CSRF header advertised but not enforced ⬜
- **Impact**: `X-CSRF-Token` in CORS allow-headers with no token issued/checked
  — misleading, no defense-in-depth beyond SameSite. **Fix**: either issue +
  verify a double-submit CSRF token on portal state-changing POSTs, or drop the
  header from CORS. **Files**: `cmd/api/main.go`, portal middleware. **Effort**:
  M. **Deps**: none. **Status**: ⬜.

---

## M2 — Medium term (≤ 1 month)

- **#7 (full)** — migrate the remaining 16 pages to react-query (ADR-005). M
  per page; do in batches of ~4. Benefits: cache, retry, dedup, background
  refresh. Files: the 16 pages + their test mocks.
- **#9** — server-side pagination / virtualization on unbounded list pages
  (`Wallets`, `Mandates`, `Developers`, `Disputes`, `CreditNotes`). M.
- **#13** — wrap native tables in `overflow-x-auto`; fix
  `FinanceReconciliation`'s `overflow-hidden` clip. S.
- **#19** — response-envelope consistency (`{data:}`), 2 raw error escapes,
  ~16 missing `operationId`. S–M; API-contract care.
- **#20** — add `>=0` guards to `dispute.CreditAmount`, `referral.RewardAmount`,
  `usage.DynamicAmount`, `plan.Amount`. S.
- **#21** — confirm `fxNormalizer.rates` is per-report (or add a mutex). S.
- **#23** — verify CI provisions Postgres so `_pg_test.go` coverage is real,
  not skipped. S (CI config).
- **#24** — unit tests for `subscription_payment/cancel/pause/retention/
  upgrade`, `smart_retry`, `salestax_resolver`. L (spread across PRs).
- **#26** — remove stray `console.*`; use shared `formatDate`. S.

## M3 — Long term (technical debt)

- **#14** — wrap scheduler/worker per-tick bodies in `safego.Run` (follow-up to
  #483). M.
- **#15** — wrap invoice-create + ledger post in one tx (or a formal outbox).
  M; ADR update.
- **#16** — audit the 14 `exhaustive-deps` disables. M.
- **#17** — split oversized components (`SubscriptionDetail`, `Developers`,
  `PlanCharges`) into feature folders. L.
- **#22** — kill the acknowledged N+1 in `plan_repository`; replace `SELECT *`
  in 3 repos with explicit columns. S–M.
- **#27** — decompose `main()` (2076 lines) into `bootstrap/router/middleware/
  workers`; split the largest money functions for unit-testability. L.

---

## Follow-on

After this plan is worked down, commission a **second independent audit** (a
different model/engineer) focused on adversarial testing — security, scale, API
abuse, edge-case workflows — to validate the fixes addressed root causes, not
symptoms.
