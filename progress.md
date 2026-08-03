# Progress log

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
(production-readiness.md); bugs BUG-001..006 in bugs-found.md.

Remaining backlog: pagination-consistency sweep (docs/design-pagination.md —
start with documenting actual defaults in OpenAPI, zero behavior risk),
accessibility pass (axe-core; lowest score 6), performance baseline + charts
lazy-load, telemetry deploy (#215, founder), mock-widening cleanup, React 19.
