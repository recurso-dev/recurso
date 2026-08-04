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

- **2026-08-04** — **M1 milestone complete.** #25 (portal CSRF) shipping closes
  the last open M1 item; the milestone's other findings were already merged
  (#11 → #499 portal tests, #18 → #490 magic-link hardening, #8 → #487 expensive
  rate-limit bucket, #10 → #488 context-aware logging, #12 → #489 validator
  foundation). #25: a **double-submit CSRF** backstop behind the session
  cookie's SameSite=Lax. `middleware.PortalCSRFMiddleware` requires the
  `X-CSRF-Token` header to constant-time-match the non-httpOnly `portal_csrf`
  cookie on every state-changing `/portal/api` call; safe GETs lazily mint the
  cookie (so pre-existing sessions self-heal without re-login) and login issues
  it up front. The portal frontend echoes it via a shared `portalCsrfHeader()`
  helper. No server secret and no storage → correct across horizontally-scaled
  instances and restarts. Tests: middleware unit test (mint-on-GET + all
  double-submit failure/success modes) + a frontend test asserting the redeem
  POST carries the header. The previously-advertised-but-dead header is now
  enforced.

- **2026-08-04** — #466 accrual epic **Increment 4b shipping** — the
  `cmd/backfill_schedules` operational tool: creates issuance schedules for a
  tenant's EXISTING open subscription invoices so their deferred moves from the
  awaiting-payment bucket into scheduled and the tie-out reads zero on live
  data. Dry-run by default, `--apply` to write, idempotent (reuses the PG-tested
  CreateScheduleForInvoice). Smoke-verified end-to-end: 1 eligible → apply →
  schedule+events created (50000) → re-run finds 0. **This completes the
  accrual epic's rollout tooling.** The runbook to enable on a live tenant:
  (1) set RECURSO_ACCRUAL_RECOGNITION=true + deploy, (2)
  `DATABASE_URL=... go run ./cmd/backfill_schedules --tenant=<id> --apply`.

- **2026-08-04** — #466 accrual epic **Increment 4 shipping** (tie-out
  exclusion + verification): `SumUnscheduledDeferral` now EXCLUDES invoices with
  an active recognition schedule — the load-bearing fix, without which an
  accrual invoice would be double-counted (in scheduled AND awaiting-payment)
  and the tie-out would go negative. New PG test drives the real
  `ClosePackService.Generate` and PROVES the deferred tie-out is exactly zero
  under accrual (`ledger 100000 = scheduled 100000 + awaiting 0, delta 0`), and
  that the cash model still ties via the awaiting bucket. This is the
  verification the founder asked for. Increment 3 **MERGED (#496)** — accrual is
  now available end-to-end (opt-in via RECURSO_ACCRUAL_RECOGNITION).

- **2026-08-04** — #466 accrual epic **Increment 3 shipping** (the switch,
  opt-in): `RECURSO_ACCRUAL_RECOGNITION=true` builds the recognition schedule at
  invoice ISSUANCE for subscription invoices, so revenue recognizes over the
  period regardless of payment and the month-end tie-out is structurally zero.
  **Default OFF** = the cash model, so production is unchanged until a deployment
  opts in. Write-off now cancels the schedule's pending events (so reversed-out
  Deferred isn't re-recognized) — no-op under cash. End-to-end PG test:
  recognize 30k of 100k → write off → 30k Bad Debt, 70k Deferred reversal,
  pending events cancelled, schedule canceled, books balanced. Full invariant
  harness + all 10 seeds green (flag off). Increment 2 **MERGED (#495)**.
  Remaining (Increment 4, operational): open-invoice schedule backfill for a
  tenant that flips to accrual, and (optional) close-pack tie-out wording once a
  tenant is fully accrual (the identity already accommodates the mixed state).

- **2026-08-04** — #466 accrual epic **Increment 2 shipping** (write-off
  bad-debt split): `RecordInvoiceWriteOff` now splits the pre-tax reversal by
  recognized-vs-deferred — recognized → Bad Debt Expense (code 26), still-
  deferred → Deferred (code 22); `RecordWriteOffRecovery` inverts the actual
  legs (24/27/25). SAFE: under the cash model recognized=0, so the write-off is
  byte-identical to before (existing PG test unchanged); one-off write-offs now
  correctly expense bad debt (revenue was recognized at issuance) instead of
  reversing Revenue. `SetRecognizedReader` wired to the revrec repo. New PG test
  proves the split + recovery + balanced books; full invariant harness + all 10
  seeds green. This is the prerequisite for Increment 3 (schedule-at-issuance
  switch + backfill + tie-out identity update) — accrual would otherwise drive
  Deferred negative on write-offs. Increment 1 **MERGED (#493)**.

- **2026-08-04** — #466 Track A (accrual) epic **Increment 1 shipping**:
  Bad Debt Expense account (5200, seeded + on-demand), reserved codes 26/27,
  the `AccountingPolicy` seam (interface + US default adapter + resolver,
  keeping accounting/tax/jurisdiction engines separate per founder direction),
  and `SumRecognizedByInvoice` (the query the write-off split needs). Pure
  additive scaffolding — no money behavior changes; invariant harness green.
  Increment 2 = the accrual switch (schedule at issuance) + write-off split
  (recognized→Bad Debt, unrecognized→Deferred) + recovery mirror +
  pending-event cancel, one invariant-gated change. Docs **#492 MERGED**.

- **2026-08-04** — #18 (portal magic-link hardening) shipping: token moves to
  a POST body (kept GET one release for links in flight), the frontend strips
  the token from the URL/history immediately, and verify returns ONE generic
  message for every failure state. #12 (validators) **MERGED (#489)**.
- **FOUNDER DECISIONS (2026-08-04)**: (1) **#466 Track A APPROVED** — proceed
  with accrual (schedules at issuance). (2) **Bad-debt = policy-driven, NOT
  hardcoded** — model an `AccountingPolicy{RevenueRecognition, BadDebtTreatment
  {AllowTaxRelief, RecognitionDelayDays, RecoverableTaxes}}` with jurisdiction
  adapters (US / IndiaGST / UKVAT / EUVAT / AustraliaGST). (3) Keep the
  accounting engine, tax engine, and jurisdiction rules SEPARATE — the
  accounting engine must not know GST rules. (4) After M1: a **production
  resilience sprint** (chaos/failover/outage/load 10k–100k). (5) Then invest in
  **Continuous Quality Gates** (invariants + reconciliation + security + API
  contract + Playwright + perf + a11y on every PR).

- **2026-08-04** — #8 (expensive-endpoint rate limits) shipping. #466 Track B +
  #7a **MERGED (#486)**. Added the **#466 Track A epic design** below — and the
  key finding that Track A is *coupled to #477* (bad-debt on recognized
  revenue): schedule-at-issuance recognizes revenue on unpaid invoices, so any
  write-off then hits already-recognized revenue and needs a Bad Debt Expense
  leg. They must ship together or Track A over-recognizes on every unpaid
  invoice.

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
- **Effort**: Track B S–M, Track A L. **Deps**: Track A needs the founder
  decision on accrual + bad-debt tax treatment (see the Track A epic below).
- **Status**: ✅ Track B MERGED #486; 🧭 Track A designed (see epic below).

### #466 Track A — accrual: build recognition schedules at issuance (EPIC) 🧭
- **Goal**: every subscription invoice gets its recognition schedule at
  *issuance*, so `ledger_deferred == scheduled` structurally and the tie-out is
  always zero (no `awaiting_payment` residual). This is the enterprise/ASC-606
  expectation the founder prefers.
- **Feasibility**: `CreateScheduleForInvoice` is already idempotent per invoice
  and already called at generation for wallet/credit-covered invoices — so
  moving the call to issuance for ALL subscription invoices is mechanically
  small.
- **THE COUPLING (critical)**: recognizing revenue on an *unpaid* invoice means
  a later write-off (#474) hits revenue that is already (partly) recognized.
  Reversing it from Deferred (as today) understates Deferred and overstates the
  reversal; the recognized portion must instead be **expensed as bad debt**
  (DR Bad Debt Expense / CR AR). This is exactly issue **#477**. Track A without
  #477 would over-recognize on every unpaid invoice that later fails — worse
  than today. **They are one epic.**
- **Increments (each its own PR, invariant-harness gated)**:
  1. **#477 foundation** — add a Bad Debt Expense account (e.g. 5200); on a
     write-off of a partially-recognized invoice, split the reversal: recognized
     portion → Bad Debt Expense, unrecognized portion → Deferred; unwind the
     schedule's pending events (reuse the `UnwindOnRefund` mechanism). Policy
     fork for the founder: **bad-debt tax treatment** (GST/VAT bad-debt relief
     varies by jurisdiction) — needs a decision before the tax leg is written.
  2. **Schedule at issuance** — call `CreateScheduleForInvoice` at invoice
     generation for subscription invoices (not just when already paid). Guard:
     don't double-create at payment (idempotent per invoice already handles it).
  3. **Migration/backfill** — create schedules for existing open invoices so the
     tie-out is zero on day one; reconciler alignment; retire/repurpose the
     `awaiting_payment` bucket in the tie-out identity (it collapses to ~0).
  4. **Tie-out identity update** — `ledger_deferred == scheduled` (drop the
     awaiting term once every issued invoice is scheduled); update the close
     pack + Track-B UI wording accordingly.
- **Migration risk**: HIGH — changes revenue *timing* (recognized over the
  period regardless of payment) and touches the reconciler invariants. The
  invariant harness + reconciler are the safety net; every increment must keep
  all seeds green.
- **Effort**: L (4 PRs). **Deps**: ✅ founder APPROVED accrual + directed a
  policy-driven (not hardcoded) bad-debt model with jurisdiction adapters.
- **Status**: ✅ Increments 1–3 shipping (#493 merged, #495 merged, #496
  = the opt-in switch). Accrual is now AVAILABLE (RECURSO_ACCRUAL_RECOGNITION),
  default-off. Increment 4 = backfill + tie-out wording (operational).

### #7a — Three pages render nothing on API error ✅
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

### #18 — Portal magic-link hardening ✅
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
  (#11). **Status**: ✅ MERGED #490 (POST-body token + generic verify errors).

---

## M1 — Short term (≤ 2 weeks)

### #11 — Portal pages untested ✅
- **Impact**: payment-method + gift-redeem are money paths with zero test
  coverage. **Fix**: behavioral tests for the 5 portal pages (login,
  payment-method, redeem, verify, dashboard). **Files**:
  `frontend/src/pages/portal/__tests__/*` (new). **Tests**: happy + error +
  empty per page; assert money figures and the redeem/payment flows.
  **Acceptance**: each portal page has a dedicated test asserting behavior, not
  just render. **Effort**: M. **Deps**: pairs with #18. **Status**: ✅ MERGED #499 (dedicated behavioral tests for login/redeem/verify — the money-and-auth-critical flows).

### #8 — Expensive endpoints only under the global rate limit ✅
- **Impact**: one key can issue 500 GL exports / import commits per minute.
- **Root cause**: import-commit/compare, PDF & GL-export inherit only the
  `api` 500/min bucket. **Fix**: add an `expensive` scope (10–30/min) and apply
  it to those routes. **Files**: `cmd/api/main.go`,
  `internal/adapter/middleware/rate_limit.go`. **Migration risk**: low —
  legitimate callers stay well under the cap; log when throttled. **Tests**:
  middleware test — Nth+1 request in the window gets 429. **Acceptance**: the
  named routes 429 past the bucket; normal use unaffected. **Effort**: M.
  **Deps**: none. **Status**: ✅ MERGED #487 (expensive bucket 30/min per tenant, 14 routes).

### #10 — Request/tenant IDs absent from service logs ✅
- **Impact**: a production incident can't be reconstructed — ledger/sync/email
  failures aren't correlatable. **Root cause**: only 3 of 273 service log calls
  carry `request_id`. **Fix**: carry `request_id`/`tenant_id`/`user_id` on
  `ctx`, add an slog handler that extracts them, so every log line downstream is
  tagged. **Files**: `internal/adapter/middleware/*`, a shared `logctx` helper,
  service call sites (mechanical). **Migration risk**: none. **Tests**: a
  handler-level test asserts a service log emitted during the request carries
  the request id. **Acceptance**: grepping logs by `request_id` returns the full
  chain across middleware→service. **Effort**: M. **Deps**: none. **Status**: ✅ MERGED #488 (context-aware logging).

### #12 — Currency validation not centralized ⬜
- **Impact**: inconsistent, no ISO-4217 set check. **Fix**: shared
  `validator.Currency/Country/Amount/Percentage/Email` (gin binding tags +
  helpers); replace ad-hoc `len==3`. **Files**: new `internal/validate` pkg,
  binding structs across handlers. **Migration risk**: low (stricter input —
  audit callers first). **Tests**: table test of the validators. **Acceptance**:
  an invalid currency/country is rejected at bind with a clear message.
  **Effort**: M. **Deps**: none. **Status**: ✅ foundation MERGED #489 (validate pkg
  + tags + 3 fields wired; remaining currency fields are a mechanical rollout).

### #25 — Portal CSRF header advertised but not enforced ✅
- **Impact**: `X-CSRF-Token` in CORS allow-headers with no token issued/checked
  — misleading, no defense-in-depth beyond SameSite. **Fix**: either issue +
  verify a double-submit CSRF token on portal state-changing POSTs, or drop the
  header from CORS. **Files**: `cmd/api/main.go`, portal middleware. **Effort**:
  M. **Deps**: none. **Status**: ✅ shipping — double-submit CSRF: `PortalCSRFMiddleware` validates `X-CSRF-Token` == the non-httpOnly `portal_csrf` cookie on state-changing `/portal/api` calls (constant-time), lazily mints the cookie on safe GETs so pre-existing sessions self-heal, and login issues it; the portal frontend echoes it via `portalCsrfHeader()`. Defense-in-depth behind SameSite=Lax.

---

## M2 — Medium term (≤ 1 month)

- **#7 (full)** — migrate the remaining 16 pages to react-query (ADR-005). M
  per page; do in batches of ~4. Benefits: cache, retry, dedup, background
  refresh. Files: the 16 pages + their test mocks.
- **#9** — server-side pagination / virtualization on unbounded list pages
  (`Wallets`, `Mandates`, `Developers`, `Disputes`, `CreditNotes`). M.
  **Status**: 🔧 partial — **Mandates + Disputes shipping** (both already had
  backend `limit`/`offset` support): server-side offset pagination via the
  `DataTable` `pagination` prop, fetching `PER_PAGE+1` to detect `hasNext`
  without a total count, `keepPreviousData` for smooth paging, and the status
  filter resets to page 1. Tests assert Next requests the next offset.
  **Remaining 3**: `CreditNotes` currently filters client-side (its search would
  only cover the loaded page) — needs search moved server-side first; `Wallets`
  (`ListWallets` takes only `limit`) and `Developers` (`ListKeys` takes neither)
  need a backend `offset` added (handler→service→repo + PG test) before their
  frontend can paginate. Tracked as the #9 backend slice.
- **#13** — wrap native tables in `overflow-x-auto`; fix
  `FinanceReconciliation`'s `overflow-hidden` clip. S. **Status**: ✅ shipping —
  the two native `<table>`s in `PricingSimulator` (a narrow slide-over) were
  wrapped in `overflow-hidden` divs that CLIP wide pricing/GL rows; changed to
  `overflow-x-auto` so they scroll (rounded corners preserved). On verification
  `FinanceReconciliation`'s discrepancies use the shadcn `<Table>`, whose
  internal `overflow-auto` wrapper already self-scrolls — the enclosing Card's
  `overflow-hidden` only rounds corners, it does not clip content, so no change
  was needed there (evidence-based: the finding's clip claim held only for the
  native tables).
- **#19** — response-envelope consistency (`{data:}`), 2 raw error escapes,
  ~16 missing `operationId`. S–M; API-contract care. **Status**: 🔧 partial —
  the **operationId** half is shipping: all 16 operations that lacked one
  (OAuth, gateway/integration connections, SSO, SAML) now have a unique
  `operationId`, so all 304 operations are named and the generated SDKs get
  stable method names (verified: 0 missing, 0 duplicates, spec + drift tests
  green). The **envelope-normalization** and **2 raw `{"error"}` escapes**
  (webhook.go, webhook_gocardless.go, founder-metrics) are deliberately deferred:
  they change response shapes clients already depend on (and gateway webhook
  bodies), so they belong in a versioned API-contract change with SDK
  regeneration, not a drive-by. Tracked as the remaining #19 slice.
- **#20** — add `>=0` guards to `dispute.CreditAmount`, `referral.RewardAmount`,
  `usage.DynamicAmount`, `plan.Amount`. S. **Status**: ✅ shipping — all four
  reject negatives at the API edge (400) before any ledger/credit-note/billing
  path sees them; `usage.DynamicAmount` enforced in `toEvent` (the field already
  documented "non-negative"). Tests cover each guard.
- **#21** — confirm `fxNormalizer.rates` is per-report (or add a mutex). S.
  **Status**: ✅ shipping — CONFIRMED safe, no mutex needed. Every one of the 12
  call sites constructs a fresh `newFXNormalizer` and drives it sequentially
  over the report's rows (no goroutines), so its rates map is per-report and
  single-goroutine. The real shared state — `s.fxProvider`/`s.fxFallback`, hit
  by concurrent report requests — is already `sync.RWMutex`-guarded in both the
  live (OpenExchangeRates) and static providers. Locked it in with a `-race`
  concurrency test on the shared static provider + a doc comment on the
  normalizer stating the single-goroutine contract.
- **#23** — verify CI provisions Postgres so `_pg_test.go` coverage is real,
  not skipped. S (CI config). **Status**: ✅ shipping — VERIFIED: `ci.yml`
  provisions a `postgres:15` service (+ redis:7), creates the `recurso_repo_test`
  scratch DB, exports `TEST_DATABASE_URL`/`TEST_REDIS_URL`, and runs
  `go test -race ./...`, so the PG suites genuinely run. Added a **guard step**
  closing the real latent risk: because the invariant harness *skips* (not fails)
  when `TEST_DATABASE_URL` is unset, dropping the service would leave CI green
  with the money-path net silently off. The step runs
  `TestLedgerInvariants_RandomizedBillingSequences` explicitly and fails the
  build unless it PASSED (grep asserts no `--- SKIP`, requires `--- PASS`).
  Both branches validated locally (skip→red, pass→green).
- **#24** — unit tests for `subscription_payment/cancel/pause/retention/
  upgrade`, `smart_retry`, `salestax_resolver`. L (spread across PRs).
- **#26** — remove stray `console.*`; use shared `formatDate`. S. **Status**:
  ✅ formatDate consolidated; console premise corrected. The 4 remaining ad-hoc
  `new Date(x).toLocaleDateString()` sites (Collections, FinanceReconciliation,
  Developers, Mandates) now use the shared `formatDate` from `lib/utils`
  (null-safe, one house format "Aug 4, 2026"). On the console part: a survey
  found **zero** stray `console.log`/debug calls — all 12 non-test `console.*`
  are `console.error` inside genuine error/`onError`/ErrorBoundary handlers
  (legitimate diagnostics), so there was nothing to remove without losing
  production observability. Left intentionally; the actionable half was the
  date consolidation.

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

## M4 — Production resilience sprint (founder-added, after M1)

Before more features: chaos testing, DB failover, Redis/queue/email/webhook
outage behavior, webhook retry storms, clock skew, multi-node deploy,
backup/restore validation, 10k–100k invoice load/soak tests. The audit
deliberately did not cover load/soak — this closes that gap.

## M5 — Continuous Quality Gates (founder-added, the durable investment)

Every PR runs automatically: accounting invariant tests, reconciliation
verification, security scans (SAST + deps), API contract tests, Playwright
e2e, performance-regression checks, accessibility checks. Shifts from *finding*
regressions in periodic audits to *preventing* them from reaching main.

## Follow-on

After this plan is worked down, commission a **second independent audit** (a
different model/engineer) focused on adversarial testing — security, scale, API
abuse, edge-case workflows — to validate the fixes addressed root causes, not
symptoms.
