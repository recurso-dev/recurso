# Progress log

## 2026-09-03 — cross-repo hygiene wave (SDK/docs re-sync, lint policy, migrations, route split)

Assessment first (all five repos): the API repo was healthy and its backlog
honestly empty, but the edges had rotted — `recurso-node` main carried nine
unresolved merge-conflict markers (1.8.0 was committed on top of a file that
did not compile; nothing shipped because publish is release-triggered and
npm still serves 1.2.0), the docs spec copy was 12 paths behind, Go/Node
covered 37%/45% of the spec, and no CI anywhere checked any of it.

Shipped, one branch per repo (`claude/recurso-repos-capabilities-jgz87l`):

- **recurso-node 1.9.0** — conflict resolved, schema regenerated, 47
  resources / 282 methods (220/220 in-scope paths), 298 tests, push/PR CI,
  CHANGELOG, lockfile re-synced.
- **recurso-go 1.7.0** — 45 resources / 276 methods (216/220; the four
  remaining are Prometheus/founder/marketing endpoints, skipped on purpose),
  112 test funcs, CI, CHANGELOG.
- **recurso-python 1.11.0** — the 15 accounting drill-down endpoints via
  openapi-python-client regen normalised to the existing style, MockTransport
  test, CI on 3.11/3.12, CHANGELOG.
- **docs** — byte-identical spec copy, 13 new reference pages, an
  explain-any-number concept page, "scoped keys (coming soon)" removed.
- **recurso** — `scripts/sdk_drift.py` + `sdk-drift` CI job (docs copy must
  match; SDK path coverage ratchets against `scripts/sdk_drift_baseline.json`);
  pinned `.golangci.yml` with 72 findings fixed or justified per site
  (real fixes: TLS 1.2 floor on SMTP, 25 missing `rows.Err()` checks,
  wrapped-error comparisons, context-bound DNS lookups); coverage floor
  (`scripts/coverage_gate.sh`, floor 41%, measured 43% with Postgres);
  `migrate down -all` verified 175→0→175 against Postgres 16 after adding
  six missing downs and fixing 000007's down (it dropped `tenants`, owned
  by 000001, so rollback had never worked); orphaned `migrations/` dir
  removed and its two unported organization indexes shipped as 000175;
  last three raw `{"error"}` sites moved to the envelope and the vestigial
  exported `Respond*` helpers deleted; hand-rolled limit parsing converged;
  `/v1` route table (273 registrations) extracted to `cmd/api/routes_v1.go`
  with a typed `v1Handlers` struct (main.go 2311→1972 lines; the OpenAPI
  drift test scans both files). Frontend: ESLint 9 flat config with React 19
  hook rules, 15 new page test suites (754→818 tests), raw-UUID inputs in
  BuyGift/CancelFlowStep replaced with pickers, EntitiesSettings empty state.
  Root case-duplicate reports merged (`bugs-found.md`→`BUGS_FOUND.md`,
  `production-readiness.md`→`PRODUCTION_READINESS.md`).

SDK publishing to npm/PyPI is parked by founder decision; nothing else is blocked.

## 2026-08-05 → 08-12 — the design initiative → v0.11.0, then SDK/docs sync

**`v0.11.0 — The design release` published 2026-08-12.** The arc: a founder
pivot to "VP of Design at Stripe" mode produced a code-grounded review of all
~60 screens (`docs/design-review-2026-08.md`, verdict ~74/100, thirteen
cross-cutting themes) — then ~60 PRs (#543–#602) closed every theme:

- **State contract (T1)** — silent failures that rendered as a healthy
  business (Dashboard total-outage $0 tiles, Collections analytics zeros,
  Developers empty-on-error) all route to explicit retryable errors.
- **Money (T2)** — every monetary cell renders through tabular-mono `Money`,
  right-aligned, in its real currency; Inter + JetBrains Mono self-hosted.
- **Correctness-of-trust (T9)** — the standouts: GSTReturns (scored 55; a
  statutory filing rendered as a raw JSON dump) rebuilt as GSTR-1/3B sections;
  the month-named GL export actually contained the ALL-TIME ledger →
  `/v1/ledger/export?month=&year=` (basis verified identical to the deferred
  rollforward, so the export ties to the close pack); amount-off coupons
  assumed USD (typed 500 on a JPY catalog → ¥50,000 coupon); three list pages
  filtered ONE fetched page client-side → real server-side filter params;
  Invoices loaded only the newest 250 so the CSV silently truncated.
- **Forms (T4)** — shared `useFormErrors` (inline role=alert + focus-first-
  invalid); settings gained a persistent sub-nav (T12); dev pages gained real
  code samples (T13); destructive actions confirm (T11); UUIDs became names
  and pickers (T5/T6); the remaining hand-rolled pages moved to react-query
  (T8, ADR-005 now universal).

**Then the supply chain synced to the new surface**: Go SDK v1.5.0, Node
1.7.0 (full schema regen), Python 1.9.0 (full regen), docs api-reference
resynced + a new ledger/export page. The sync itself found bugs upstream:
`listWebhookEndpointDeliveries` had duplicate params in openapi.yaml and the
Python generator SILENTLY dropped the whole endpoint (#600); the docs
advertised two query filters (`customer_id` on subscriptions, `active` on
plans) that no handler ever parsed.

**Post-release adversarial self-review (same day) caught 3 defects in the
session's own work**: the coupon form's dominant-currency default read
`p.currency` but currency lives on `prices[]` — inert for every tenant, and
the test had mocked the wrong shape (#602); the new filter SQL had zero
real-Postgres coverage → PG test added, which also exposed a hardcoded
unique id in the payment-attempt test (#601); the Python regen clobbered the
README and the `raise_on_unexpected_status=True` deviation (python#14 — the
generator resets three things every regen; checklist recorded).

Also: a fresh nanoid CVE broke Trivy on every PR mid-session (#593 unblocked
the train), and the week-old #590 (T10: dead portal code, revenue-view
cross-links) finally merged once the GitHub Actions outage cleared.

Remaining queue is founder-only: Organizations tenant-enumeration endpoint
design, pause/arrears semantics (L2), Idempotency-Key contract (S4), the
0.4%-vs-"never a cut" positioning, and the standing credentials list
(QuickBooks OAuth, GoCardless webhook, telemetry deploy, Peppol, demo
hosting, npm/PyPI publish secrets).

## 2026-08-03 owner-mode session — v0.8.0 released + product-grade polish wave

**`v0.8.0 — The correctness release` published** (tag at #427's changelog cut;
workflow clean: tests → GHCR image → release; curated notes). Then a standing
"think like an owner" directive kicked off a continuous audit loop (#428–#432):

- **#428 — statutory credit-note documents (B2/ENG-196, migration 000160).**
  The backlog's "no consumer — don't build" conclusion was stale (#279 had
  shipped the downloadable credit-note document days before it was written).
  Credit notes now store subtotal/tax/IGST/CGST/SGST/tax_type/HSN, populated at
  creation (downgrade credits carry the reversed proration tax; invoice-linked
  credits slice proportionally — the exact GSTR-1 CDNR math); the document
  renders a GST-grade CDN. Bonus real bug caught by the round-trip oracle:
  invoice `getByIDInternal` never scanned `tax_type` — every GetByID consumer
  saw it empty.
- **#429 — backlog truth-sync.** S1 ("card sync should be incremental") was
  ALREADY implemented (`force := provider == ""` + dirty-tracking) — struck
  with evidence. Lesson recorded: backlog rows go stale in both directions.
- **#430 — the last three unclamped list limits.** Dunning history accepted
  negative limits verbatim; both webhook-management lists had no upper cap.
  Routed through the house `parseLimitOffset` (50/500).
- **#431 — credit-note detail sheet** mirrors the recorded tax breakdown.
- **#432 — customer-portal money bug (end-customer facing).** Total Paid /
  Outstanding summed minor units across currencies and formatted as USD — an
  INR customer saw "$118,000.00"; JPY was divided by 100. Now per-currency.

HANDOFF.md refreshed to the v0.8.0 state.


## 2026-08-03 second wave — revrec deep-fixes + a money-out arbitrage

Three more merged fixes (#420–#422), the last one HIGH money-out. The arc:
soaking our own safety net found an edge in our own fix, repairing it properly
surfaced a latent interaction, closing that filed a lead which turned out to be
a real arbitrage.

**#420 — downgrade revenue reversal capped at genuinely-recognized events.**
A 32-seed soak of the invariant harness (CI runs 8 fixed seeds) caught seeds
23/39 driving Recognized Revenue wrong-sign: #413 attributed the whole schedule
shortfall to "already recognized," but an unpaid upgrade-charge invoice funds
Deferred with NO schedule — that shortfall is unscheduled deferral, not
recognized revenue. Fix: three-way split (pending → Deferred; genuinely
recognized → clawed back from 4100, capped at and MARKING the sub's recognized
events with a new 'reversed' status so repeat downgrades can never over-claw;
residual → Deferred, where the funding sits). Seeds 23/39 are now in the CI
seed list permanently. Lesson: soak beyond CI's fixed seeds after any
revrec/ledger change — 30 extra seeds cost ~2 minutes and found a real bug.

**#421 — schedule debt (ENG-191f).** Closed the latent interaction #420
documented: when a downgrade residual consumes unscheduled deferral, paying
that invoice later would schedule its FULL net — recognizing revenue already
credited back. New `revrec_schedule_debt` (migration 000158): the downgrade
records the residual; `CreateScheduleForInvoice` atomically consumes it and
shrinks the new schedule; a fully-consumed net creates no schedule.

**#422 — coupon-blind proration = money-out arbitrage (ENG-195, HIGH).**
Filed as backlog R3 during #421; the audit found it worse than filed:
plan-change proration used LIST prices while renewals honor the coupon, so an
80%-off customer who paid 40000 could downgrade mid-period into ~50000 of
spendable account credit — more than they ever paid, unbounded with steeper
discounts. Upgrades symmetrically over-charged discounted customers. Fix:
`coupon_applied_current_period` (migration 000159), set at all three
invoice-generation sites (create/renewal/trial; renewal clears it when the
coupon stops — the counter alone can't derive it), and
`computePlanChangeProration` discounts both plan prices when set. Preview
inherits automatically. Oracle: 80%-off downgrade credits ~10000 not ~50000;
the undiscounted control still credits list.

## 2026-08-03 early-AM continuation — metering & reporting correctness sweep

Three more merged fixes (#416–#418), each a real mis-billing or mis-reporting
bug with a failing-first oracle. Theme: seams where two features compose —
per-value pricing × progressive billing, FX rates × currency exponents,
per-event capture × non-additive aggregations.

**#416 — filtered charges on progressive subscriptions bypass the filter-blind
watermark.** A dimensional-pricing charge (`FilterKey` + per-value amounts) on
a progressive subscription was routed through the watermark path, whose
aggregate is filter-blind: ALL events billed at the charge's BASE amounts,
every per-value rate silently ignored (and if base amounts lacked the invoice
currency, the charge billed nothing at close). Unpreventable by validation —
the threshold lives on the subscription, the filters on the plan charge. Fix:
one shared `progressiveBillable` predicate (eligible model + arrears +
unfiltered) at all three sites (interim gate, interim bill, period-close
dispatch); filtered charges fall through to the classic filtered path exactly
as the volume model already did.

**#417 — FX conversion ignored currency exponents; JPY reporting off 100×,
KWD 10×.** FX rates are major-to-major, but every conversion site multiplied
minor-unit amounts by the raw rate — correct only for same-exponent pairs.
JPY→USD 100× understated, USD→JPY 100× overstated, KWD 10× off. Corrupted all
normalized reporting (MRR analytics, revenue segments, waterfall, invoice
aging, dunning recovery, org consolidation) for zero/three-decimal-currency
tenants; the static fallback seeds are true market rates, so the corruption
was real. No ledger impact (single-currency per tenant) — reporting only. Fix:
`domain.ConvertMinorUnits` (amount × rate × 10^(expTo−expFrom)) used by the
normalizer and both providers' `Convert`.

**#418 — pay-in-advance required additive aggregations; count metrics bill one
per event.** PIA validation guarded the charge model but accepted PIA on
max/latest/unique/percentile metrics — per-event captures SUM, so max
concurrent seats reported by heartbeats billed per heartbeat and unique users
billed per repeat visit. Also `BillEvent` billed `event.Quantity` on COUNT
metrics while arrears `COUNT(*)` counts each event once. Fix:
`PayInAdvanceAggregationEligible` (count, sum) enforced at the single
`resolveChargeInput` choke point (subsumes the weighted_sum-only guard), and
count metrics bill exactly one unit per event.

**Verified clean (non-findings):** payment-settle amount validation (correct by
construction — server creates the order/session at invoice total, HMAC-verified,
tenant-bound), Stripe/Chargebee/RevenueCat importers (already exponent-aware),
remaining hardcoded `/100`s (all percent/percentile/INR-only), SQL currency
bucketing (GROUP BY currency everywhere), credit-note void (atomic claim
reverses exactly the unspent balance), accounting adapters (all `MinorToMajor`),
usage alerts (quantity-based, no rating math).

## 2026-08-02 overnight session — latent revrec bug + reconciler tripwire

Two PRs, both merged on green CI, each with a failing-first oracle. The thread:
a warning the invariant harness had been *logging but not failing on* turned out
to be a real latent bug, and the fix came paired with a standing guardrail so
the class can't recur silently.

**#413 — downgrade credit no longer drives Deferred Revenue wrong-sign
(latent, revrec).** A mid-period plan downgrade booked the full net proration
credit as `DR Deferred / CR Customer-Credit` regardless of how much the
recognition schedule still held. When recognition had run ahead of the
proration boundary (lumpy/upfront recognition, or a fully-elapsed period),
Deferred no longer held it — so Deferred went **wrong-sign (a negative
liability)** and Recognized Revenue was left **overstated**. The trial balance
still netted to zero, so aggregated across subscriptions the harness only
logged `WARN downgrade schedule reduced less than the net credit … reduced=0`
and stayed green. Fix: split the net credit by what the schedule can actually
give back — the still-unrecognized part drains Deferred (code 16), the
already-recognized remainder is clawed back out of Recognized Revenue via a new
`RecordDowngradeRevenueReversal` (`LedgerCodeDowngradeRevenueReversal = 21`).
Deferred is never over-drawn; the Customer-Credit liability is unchanged.
Oracle `TestDowngradeCreditRevenueReversal_Postgres`: fully-recognize then
downgrade → old Deferred −50000 / RecRev 200000; fixed Deferred 0 / RecRev
150000. The sibling paths were already correct — `UnwindOnRefund` clamps the
reversal to what's deferred, and `UnwindOnCancel` derives the forfeit from the
pending events themselves; downgrade was the lone outlier.

**#414 — reconciler flags Deferred drained below scheduled recognition
(standing invariant).** New discrepancy `deferred_below_scheduled_revenue`:
Deferred Revenue must always be at least the sum of pending recognition events.
Unlike the abnormal-sign check (which needs a single account to go net-debit,
and is masked when other subscriptions' positive Deferred hides one going
negative), this compares two independently-computed totals, so a net shortfall
survives aggregation across subscriptions and entities. Wired into both
production reconciliation and the invariant-harness grade. Proven
never-false-positive: the full randomized harness stays green with it active
(the gap between Deferred and pending is exactly the recorded-but-unpaid
invoice deferrals, always ≥ 0). Unit tests cover both the shortfall and the
covered (incl. unpaid-slack) cases.

**Verified non-findings (left unchanged):** the `big.Rat` rating engine
(per-unit / graduated / volume / percentage / graduated-percentage) rounds
exactly once on the final money amount — no mischarge; the TaxJar float
`dollarsToCents`/`centsToDollars` is safe within its USD-only domain (no
currency-exponent bug). Convergence still holds: reachable correctness bugs are
essentially exhausted; the remainder is product decisions and founder-blocked
credential work.

## 2026-07-31 autonomous session — money-path correctness + safety-net hardening

~39 PRs merged (#382–#404), all on green CI, each fix with a failing-first
oracle (PG-backed for ledger paths). Theme: close the remaining money-path
correctness gaps, then build the guardrail that prevents the whole bug class
from recurring — and prove it by having that guardrail catch a real HIGH bug.

**Security — cross-tenant BYO webhooks (S2), fully closed.** Every BYO
per-connection webhook money-move is now bound to the connection's own tenant:
invoice settle/failure/reversal across all 3 gateways (#382), gateway
refund-event status advance (#384), and the Razorpay virtual-account reconcile
(#386, tenant-scoped atomic increment so a foreign `va_id` matches 0 rows).

**Lifecycle × metering / retention.**
- #388 (L1) — pay-in-advance usage no longer bills **paused/canceled** subs
  (was billing during a pause + accruing phantom charges on dead subs).
- #390 (L3, HIGH) — gift subscriptions no longer **auto-renew and dun the
  recipient** (set `CancelAtPeriodEnd` at redemption; a prepaid gift now
  cleanly expires instead of invoicing someone with no payment method).
- #392 (L4) — an accepted cancel-flow **discount offer is now actually applied**
  (it was logged and dropped, so retained customers paid full price).

**Ledger-leg completeness (money-in) — audited all invoice-creating paths.**
Found and fixed two flows that created an invoice with **no double-entry leg**
(AR would go negative on payment, revenue never recognized):
- #394 (Q1, HIGH) — quote→invoice conversion.
- #396 (Q2, HIGH) — the buyer's gift-purchase invoice.
The other 8 `invoiceRepo.Create` sites, and every money-**out** credit/refund
leg, were verified balanced. FK-ordering audit also clean (Q3 was the sole
stamp-before-create case).

**Safety-net hardening — the arc that pays for itself.**
- #399 (H1) — the ledger invariant harness called `RecordInvoice` *directly*,
  never the service create-paths — the blind spot that hid Q1/Q2. Hardened it to
  drive the **real** `QuoteService.ConvertToInvoice` + `GiftService.PurchaseGift`
  through the reconciler. **On its first run it caught Q3 (HIGH): quote
  conversion FK-violated against real Postgres and was outright broken in prod**
  (mock tests never exercised the FK). Fixed by wrapping create+claim in one tx.
- #403 (H2) — reconciler meta-audit confirmed the oracle is sound, then closed
  its one gap: credit-note leg-completeness (`missing_credit_note_transaction`),
  the credit-note analog of the invoice-leg check. Proven: neutering
  `RecordDowngradeCredit` now fails the harness. Both sides of the ledger —
  invoice legs and credit-note legs — are guarded.

**Investigated and *not* built (avoided needless change):** B2 (credit-note GST
split — no consumer: no credit-note PDF, and the GSTR-1 CDNR report already
prorates; #401) and the `invoice.go:260` "use first price" smell (consistent,
not a bug).

**Handoff — remaining items need a decision, not more bug-hunting:** L2 (pause
metering policy) and S4 (idempotency policy) are product calls; B2 is a no-op
until a credit-note document/e-invoice exists; W2/W3 are latent/unreachable;
QuickBooks OAuth / GoCardless webhook / telemetry / Peppol / demo hosting / the
Xero customer-email fix are founder-credential-blocked. No open correctness bugs.

# Progress log — overnight 2026-07-27 → 28

## Morning session (after founder wake-up)

- **QuickBooks verified live on prod** (sandbox company): OAuth → Connected →
  bulk customers + invoices pushed with QBO ids. Intuit gotcha: redirect-URI
  lists are **per keys tab** (Development vs Production) — the URI had been
  saved under Production while the creds are Development keys. Zero adapter
  code changes needed. `QBO_SANDBOX=true` picks the sandbox host.
- **#249** — GoCardless connect 500 fixed (provider CHECK constraint from
  000107 predated the third provider; migration 000150) + sync rows carry
  their provider, Integration column, filters, full error text.
- **#250** — provider rate limits honored (Xero 429 storm from the forced
  re-push): `doWithRateLimitRetry` waits Retry-After (capped), replays POST
  bodies, bounded retries; Xero + QuickBooks routed through it.
- **#251** — per-integration Sync buttons, server-side pagination (25/page),
  integration/status filters, debounced record-id search.
- **This PR** — sync rows resolve a human name (customer name / invoice
  number) at read time; search matches names too.


Running log of the autonomous overnight session. Newest first. Every item
merged on fully green CI (invariant harness + E2E + OpenAPI drift + frontend
gates) unless marked open.

## Completed

- **SDK sync (all three repos, merged)** — go #8 (mandate `Currency`,
  VPA omitempty), node #10 (schema regen), python #10 (full regen, 1.7.0,
  smoke 21/21): mandate currency rails, `gocardless` provider enum, async
  accounting-sync responses.
- **#245** — E2E step 6c: GoCardless webhook chain (fail-closed signature +
  fulfilment activates mandate and swaps BRQ→MD id). Compose gains a
  deterministic `GOCARDLESS_WEBHOOK_SECRET`. First run failed on
  heredoc-mangled JSON escapes — rebuilt with `jq -n` and re-verified against
  a local mock-gateway boot before repush.
- **Dependabot hygiene** — 3 alerts cleared by #243; the remaining
  react-router RSC-only advisory dismissed with justification (mirrors
  `.trivyignore`).
- **#244** — GoCardless `charged_back`/`late_failure` now reverse settlements
  via `ReverseSettledPayment` (paid→past_due + occurrence-aware code-19 leg,
  #209 machinery). No classification guards needed — GoCardless models
  merchant refunds as a separate resource, unlike Stripe's
  returns-as-refunds (#210).
- **#243** — react-router-dom 6.30 → 7.18: clears all three Dependabot
  alerts, zero source changes. v8 (fixes an RSC-only advisory) needs React
  ≥19.2 → documented `.trivyignore` + backlog item.
- **#242** — `docs/backlog.md`: ranked backlog, founder-blocked vs
  engineering-ready.
- **#241** — Mandates page: currency select (INR/EUR/GBP), VPA only for UPI,
  authorization-link copy toast, exponent-aware amounts.
- **#240** — GoCardless `payments.confirmed`/`paid_out` settle invoices
  (lookup by stored `PM…` id; `isGatewayPaymentID` accepts GC ids;
  `GetByGatewayPaymentIDPublic` consumed via capability assertion).
- **#239** — Manual accounting sync is async with per-tenant single-flight
  (202/`sync_already_running`; `sync_status: syncing`; 15-min cap). Closes
  the last known Cloudflare-~100s synchronous sweep.
- **#238** — GoCardless webhook receivers (platform + per-BYO-connection):
  HMAC-verified, per-event dedup (GC batches deliveries),
  `billing_requests.fulfilled` activates mandates + swaps BRQ→MD.
- **#237** — BYO GoCardless from the dashboard. Also fixed a latent
  cross-tenant leak: the resolver aliased the env router's
  `Extra`/`currencyOverrides` maps into every tenant build — now copied,
  with a regression test.
- **docs #25** — GoCardless setup + advanced guides (no longer
  "experimental"); **website #29** — integrations chip soon→BYO.

## Verification evidence (not testimony)

Runtime-verified on a fresh DB + real API boot with real sandbox billing
request: signed fulfilment webhook → mandate `active`, token swapped;
`payments.confirmed` → invoice paid + code-3 leg; `charged_back` →
past_due + exactly one code-19 leg; redelivered chargeback did **not**
double-reverse; bad signature → 401.

## Key decisions

- Accounting sync went **async** (not batched like CRM #231) because a forced
  full re-push has no natural batch boundary; single-flight per tenant.
- GoCardless chargebacks reuse the ACH reversal machinery **without**
  classification guards — a property of GoCardless's API (refunds are a
  separate resource), documented in the handler.
- react-router stays on v7 with a justified Trivy ignore rather than rushing
  a React 19 upgrade overnight.

## Open / next steps

- **Founder (morning)**: QuickBooks app creds (see `docs/backlog.md` P1.3);
  GoCardless webhook endpoint + `GOCARDLESS_WEBHOOK_SECRET` on Cloud Run
  (P1.4); telemetry deploy; `TRAFFIC_TOKEN`; demo-sandbox hosting; fix
  customer `bed15f4d…`'s Xero-invalid email.
- Engineering-ready: pagination-consistency sweep (design first — changing
  unbounded defaults silently truncates existing clients), interface-embedding
  mock cleanup, React 19 + react-router 8.

## Full-product verification (2026-07-28, gym-time sweep)

All six repos' gates green; 32-area fresh-boot API journey: 27 PASS, 3
findings fixed in this PR (EU e-invoice tenant-ctx 500, mandate guard 500→400,
Quotes raw-UUID customers), 4 PARTIALs backlogged (S1–S5 in docs/backlog.md).
Prod dashboard sweep clean (zero console errors). All seven third-party
integrations live-verified, incl. BYO GoCardless end-to-end (mandate
authorized on hosted page → webhook flipped it active). Full evidence:
docs/verification-2026-07-28.md.

## Hardening continued (2026-07-28)

Security audit clean; money-path audit → 2 fixes (#260 wallet double-credit
HIGH, #261 accounting currency-exponent MED). All five smoke findings closed:
S1 incremental card sync (#256), S2 nested billing_address (#259), S3 GET
subscription by id (#258), S4 lump-sum quote line (#264), S5 entity country
alias (#263). SDKs synced (node/go/python). Scorecard 8.5/10
(PRODUCTION_READINESS.md); bugs BUG-001..006 in BUGS_FOUND.md.

Remaining backlog: pagination-consistency sweep (docs/design-pagination.md —
start with documenting actual defaults in OpenAPI, zero behavior risk),
accessibility pass (axe-core; lowest score 6), performance baseline + charts
lazy-load, telemetry deploy (#215, founder), mock-widening cleanup, React 19.

## The correctness marathon → v0.10.1 (2026-08-02 → 08-04)

Five releases in three days, all bug-driven, all oracle-tested (see
CHANGELOG.md for the full ledgers): v0.7.0 bank debits → v0.8.0 correctness
sweep (coupon arbitrage, revrec downgrade family, FX exponents) → v0.9.0
paper trail (statutory credit notes, hot-path indexes) → v0.10.0 receipts
(migration Compare gate for Stripe/Chargebee/RevenueCat with printable
persisted reports, invoice branding, explain-any-number ledger deep links,
IA v2) → **v0.10.1 write-offs (2026-08-04)**.

The write-off arc is the one to study: the month-end close pack gained a
deferred tie-out identity (#473) — `ledger closing == schedule deferred +
awaiting payment` — and on its FIRST day live its unexplained-delta flagged
$7,345. The trail: write-offs were bare status flips (#474, codes 22/23 fix
it), a written-off invoice could be paid by a stale checkout link with
nowhere sound for the money to land (#475, recovery codes 24/25), and the
repeat cycle write-off→pay→bank-return→write-off was silently swallowed by
ledger idempotency (#476, occurrence-aware per design-ledger-occurrence.md).
Open policy question — bad-debt expense treatment for repeat write-offs on
partially-recognized invoices — is issue #477, guarded by a runtime warning.

Also since 07-28: demo tenant reseeded with coherent books (reconciler-clean,
survived overnight workers); docs+website carry real product screenshots with
zero placeholders; dashboard swept page-by-page with the founder live.

Founder-blocked queue: #466 (schedule-at-issuance accrual policy), #477
(bad-debt policy), SDK publish secrets, telemetry deploy, pricing.
