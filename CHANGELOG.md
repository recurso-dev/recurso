# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.11.0] - 2026-08-12 — The design release

Every screen, judged. This release closes out a full code-grounded design
review of all ~60 dashboard, portal, auth, and settings screens against the
bar set by Stripe, Linear, and Mercury (`docs/design-review-2026-08.md`). The
verdict was "trustworthy but uneven" — strong architecture, with the gap
concentrated in thirteen repeated, mostly shared-component patterns. All
thirteen are now closed: every number renders in tabular-mono with its real
currency, every failure is an honest error instead of a healthy-looking zero,
every raw UUID became a name or a picker, every figure drills to its source,
and the last statutory screen that dumped raw JSON now reads like a filing.
Alongside the design work: opt-in accrual revenue recognition, a period-scoped
GL export, and a set of correctness fixes where a form could quietly create
the wrong money.

### Added

- **Public landing page.** Signed-out visitors to `/` now see a marketing
  front door in the product's own design language instead of a login
  redirect; authenticated users land on the dashboard as before.
- **Settings has a persistent sub-nav.** All settings pages share one layout
  with a grouped rail (General / Billing documents / Taxes / Account), so
  moving between tax, branding, and account config no longer bounces through
  a hub page. Region-relevant pages are badged.
- **GST returns render as a filing, not a payload.** GSTR-1 shows control
  totals (gross outward vs credit notes), B2B invoice detail under each
  buyer's GSTIN, the B2CS rate-wise summary, CDNR, and the HSN rollup;
  GSTR-3B shows Table 3.1 rows (a)–(e) — with an explicit note on why
  (b)–(e) are zero — Table 3.2, and provenance counts. The GSTN upload JSON
  download is unchanged; a failed build now shows a retryable error in place.
- **Period-scoped general-ledger export.** `GET /v1/ledger/export` accepts
  `month` + `year` to export one calendar month's postings; the month-end
  close pack's export link carries its own period, so the file named
  `general-ledger-2026-08.csv` contains exactly August's ledger (it
  previously contained the entire ledger since inception).
- **Server-side list filters.** Subscriptions can be filtered by
  `plan_id` and `started_after`, plans by `currency` (any of the plan's
  prices) and `interval_unit`, and events by `type` — as real API query
  params. The dashboard's filter controls now use them, and the Events
  type filter offers the server's full type catalog.
- **Developer page code samples.** A "make your first request" quickstart
  (cURL/Node/Python) and webhook signature-verification samples (Node/Python,
  matching the real `X-Recurso-Signature` HMAC scheme) render in a new
  copyable code-sample component.
- **Every ambiguous metric is explainable.** KPI tiles across Collections,
  the dashboard, Executive Summary, Unit Economics, MRR Waterfall, Dunning,
  and Trial Balance carry a definition affordance stating exactly how the
  number is computed.
- **Drill-to-source everywhere.** Aging-report buckets link to the invoices
  inside them; ledger entries type their reference (invoice / credit note /
  schedule entry / wallet) and link invoice references to the invoice;
  reconciliation discrepancies link to their invoice; per-customer AR
  sub-accounts resolve to "Accounts Receivable — {customer}".
- **Accrual revenue recognition (opt-in).** With
  `RECURSO_ACCRUAL_RECOGNITION=true`, recognition schedules are built at invoice
  **issuance** for subscription invoices, so revenue recognizes over the service
  period regardless of payment — accrual accounting (#466). The month-end
  deferred tie-out is then structurally zero (every issued invoice is
  scheduled, so nothing sits in the awaiting-payment bucket). **Default off**
  (the cash model — schedule at payment), so it is a per-deployment rollout.
  Writing off an accrual invoice cancels its schedule's not-yet-recognized
  events (so the reversed-out Deferred can't be re-recognized) and expenses the
  already-recognized portion as Bad Debt (the #477 split). No behavior change
  when the flag is off; the invariant harness stays green either way.

### Changed

- **Centralized input validation.** Currency and country codes were validated
  ad hoc (a bare `len==3`, or not at all). A new `internal/validate` package
  provides ISO-4217 currency and ISO-3166 country checks — backed by the
  maintained `golang.org/x/text` dataset rather than a hardcoded table, so the
  accepted set tracks the standard — plus amount/percentage/email predicates
  and gin `currency`/`country` binding tags. Plan, mandate, and advanced-charge
  currency fields now reject a non-ISO code (e.g. `ZZZ`, or the reserved
  `XXX`/`XTS`) at bind time instead of accepting any 3-character string.

- **Expensive endpoints now have a dedicated rate-limit bucket.** Import
  preview/commit/compare (each reprocesses an entire external billing
  account), invoice/credit-note PDF + HTML renders, the GL export, and the
  compare-report document previously sat only under the generous global
  500/min limit — one API key could issue 500 GL exports or import commits a
  minute. They now share a tight per-tenant `expensive` bucket (30/min); a
  caller past it gets 429, normal use is unaffected.
- **The dashboard's typography is self-hosted and money is uniform.** Inter
  (UI) and JetBrains Mono (money, code, IDs) ship with the app instead of
  loading from the Google Fonts CDN, and every monetary table cell renders
  through the shared tabular-mono `Money` component, right-aligned, in the
  amount's real currency — no more per-OS money fonts, hardcoded `$`
  prefixes, or headline figures assuming USD.
- **Amount-off coupons are created in a real currency.** The coupon form
  offers the currencies the plan catalog actually bills in and converts the
  typed amount with that currency's exponent. Previously the form assumed
  USD: a merchant billing in JPY who typed 500 minted a ¥50,000-off coupon
  (100×), and a KWD merchant got 10× less than intended.
- **The Invoices page loads the whole invoice set.** It previously fetched
  only the newest 250 invoices, so the status chips, the aging drill-down,
  search, and the CSV export all silently operated on a truncated set. The
  page now walks the paginated API (with an honest notice past a 10k safety
  cap), so the CSV export is complete.
- **List filters filter the whole list.** The subscriptions plan/date
  filters, plans currency/interval filters, and events type filter were
  applied client-side to a single fetched page — picking a plan quietly
  showed only that page's matches. All three now filter server-side.
- **Form validation follows one accessible contract.** Every create form
  shows inline `role="alert"` messages tied to the field via
  `aria-invalid`/`aria-describedby`, focuses the first invalid field on
  submit, and clears each message as it's fixed — replacing a mix of
  toast-only warnings and native browser tooltips.

### Security

- **Portal logout now revokes the session server-side.** Logging out of the
  customer portal only cleared the cookie — the `portal_sessions` row stayed
  valid for its full 7-day TTL, and the token-via-header auth path
  (`X-Portal-Session`) never sees the cleared cookie, so a captured token
  survived logout entirely. Logout now deletes the session row; the token
  dies immediately.
- **Magic-link request no longer reveals whether an email is a customer.** The
  portal login endpoint returned "Login link sent to your email" for a known
  address and a different "if this email exists…" message for an unknown one,
  letting an attacker enumerate customers by diffing the response. Both cases
  now return one identical generic message.
- **Portal magic-link hardening.** The sign-in token now travels in a POST
  body instead of the URL query string (which leaks via `Referer`, browser
  history, and proxy/access logs); the verify page strips the token from the
  URL and history the moment it reads it. And the verify endpoint now returns
  one generic message for every failure — invalid, expired, or already used —
  so the response can no longer be used to probe a token's state. The old
  query-string endpoint is kept for one release for links already in flight.

### Fixed

- **Write-offs of already-recognized revenue now expense Bad Debt, not reverse
  Deferred.** Part of the accrual revenue-recognition epic (#466/#477): when an
  invoice's revenue has been (partly) recognized, writing it off can't
  un-recognize that revenue — the service was delivered; the non-payment is a
  bad-debt expense. The write-off now splits — the recognized portion posts to a
  new Bad Debt Expense account (ledger code 26), the still-deferred portion
  reverses from Deferred (code 22) as before — and a later recovery inverts
  both. One-off invoices (whose revenue is recognized at issuance) now correctly
  expense bad debt on write-off instead of reversing Revenue. No behavior change
  under the current cash-recognition model (recognized is 0), and the invariant
  harness stays green.
- **The month-named GL export contained the all-time ledger.** The month-end
  close's "GL (CSV)" download was named `general-ledger-YYYY-MM.csv` but
  always exported every posting since inception — an auditor handed that file
  as close evidence got the wrong books. The export is now scoped to the
  period being closed (see the new period parameters above).
- **Silent failures no longer render as a healthy business.** A full data
  outage on the dashboard, a failed Collections analytics load, and failed
  Developers keys/webhooks/events loads used to render $0 tiles or empty
  lists — indistinguishable from a quiet account. Each now shows an explicit
  error state with a retry.
- **The Coupons list's "Redemptions: 0" column was fabricated** — a hardcoded
  zero for every coupon (the backend has no such field). Removed.
- **Profile's load error rendered inside the "Edit profile" button.** It now
  renders in the page body as a proper alert.
- **The dashboard's MRR hero assumed USD.** It now formats in the tenant's
  real reporting currency.
- **Consequential actions are consistently guarded.** Enabling money-moving
  MCP agent tools, changing a teammate's role, and clearing all declared tax
  nexus (a compliance-sensitive wipe) now require an explicit confirmation
  naming what will change.
- **The customer portal's Outstanding card ignored applied account credit.**
  An open invoice with account credit applied showed its face value as owed —
  a customer with a fully credit-covered invoice saw it as outstanding debt.
  Outstanding is now `total − amount_paid − credit_applied` (clamped at
  zero), per currency. Found by the post-release audit sweep.
- The ledger-entries read no longer falls back to the TigerBeetle mirror on a
  Postgres error: TB's transfer lookup is keyed by account id alone (no
  tenant scoping), so a transient DB outage could have served another
  tenant's transfers to a caller who guessed an account UUID. PG errors now
  surface as errors; the TB path serves only PG-less deployments. (Latent —
  prod runs without a TB client.)
- **Detached goroutines no longer risk crashing the whole API on a panic.**
  Fire-and-forget goroutines (invoice + credit-note issuance emails, the
  new-signup operator alert and marketing sync, the manual accounting-sync
  worker) ran with no panic recovery — and an unrecovered panic in ANY
  goroutine terminates the entire Go process, dropping every in-flight
  request. Gin's Recovery middleware only covers the request goroutine.
  A nil customer reaching a notification template, or a provider client
  panicking mid-sync, could take the API down. New `internal/safego.Go`
  wraps each detached goroutine in a recover-and-log guard; the panic is
  contained and logged with its stack instead of crashing the process.
  Found by the post-release audit sweep.
- **Charts mounted in a hidden tab rendered empty.** Chrome freezes
  `requestAnimationFrame` in hidden/occluded windows, so a chart that mounted
  there stuck at its entry animation's first frame — sub-pixel bars over a
  correctly scaled axis, indistinguishable from "no data" (seen live on Usage
  Explorer). Chart animation is now enabled only when the document is visible,
  and honors `prefers-reduced-motion`; charts booting hidden render at full
  height immediately.

- **Pagination consistency wave** (the silent-truncation bug class, from the
  backlog): the customer portal's invoice and dispute lists, the high-churn-
  risk list, and a subscription's unbilled-charges list were unbounded — they
  now take clamped `limit`/`offset` (DoS-bound at 1000; every realistic
  caller's results are unchanged). Compare-report history was hardcoded to 50
  with no way to page — `limit` is now caller-controllable (default 50, cap
  200). The accounting sync-log endpoint had its default and cap inverted
  (any requested limit above 25 was silently forced to 200) — it now honors
  the contract its OpenAPI spec always documented (default 25, cap 200). Ten
  list endpoints gained the paging parameters their spec entries were
  missing. Billing-side reads (invoice generation's unbilled-charge sweep,
  churn scoring) deliberately stay unbounded — a paged read there would
  silently leave charges unbilled.

## [0.10.1] - 2026-08-04 — The write-off release

Bad debt stops lying. Yesterday's close-pack tie-out flagged a $7,345 delta it
couldn't explain; the trail led to write-offs that were bare status flips —
AR carrying money that would never arrive, Deferred carrying revenue that
would never be earned. This release closes the whole arc: write-offs post
their reversal, a written-off invoice that gets paid after all recovers its
books before the cash lands, and the full cycle (write off, pay, bank
return, write off again) posts fresh legs instead of being silently
swallowed. Every step is proven end-to-end against real Postgres.

### Fixed

- **Invoice write-offs now post their ledger reversal.** Marking an invoice
  uncollectible — manually or via the dunning scheduler — was a bare status
  flip: AR kept carrying money that would never arrive and Deferred kept
  revenue that would never be earned, both overstated forever. Write-offs now
  post DR Deferred (or Revenue, for one-off invoices) + DR Tax Payable /
  CR the customer's AR (codes 22/23, idempotent per invoice); the close
  pack's awaiting-payment bucket counts legacy un-reversed write-offs so the
  deferred tie-out stays exact either way. Found by the tie-out's
  unexplained-delta on its first day live.
- **Paying a written-off invoice now recovers its books.** An uncollectible
  invoice can still get paid — a stale checkout link or a late bank transfer
  deliberately flips it to paid. But after the write-off reversal above, the
  payment's cash leg would have settled a receivable that no longer existed
  (driving AR negative) and the recognition schedule would have drained an
  already-reversed Deferred balance. Settlement now re-establishes AR,
  Deferred (or Revenue), and Tax Payable first (codes 24/25 — the exact
  mirror of 22/23; idempotent, posted only when a write-off leg exists), so
  the end state of the write-off → paid-after-all arc is identical to a plain
  paid invoice. Proven end-to-end by the Postgres write-off test.
- **Write-off cycles are occurrence-aware.** An invoice can be written off,
  paid after all, have the bank return that payment, and be written off
  again — but the second write-off's legs were silently swallowed by the
  `(reference, code)` idempotency pre-check, leaving AR and Deferred
  overstated invisibly. The write-off/recovery pair now follows the
  settle→reverse occurrence design (migration 000146): a fresh write-off
  posts only when every prior one has been recovered, at
  occurrence = completed cycles, and same-cycle duplicates still no-op.
  Repeat cycles log a books-review warning (recognized revenue, if any, is
  reversed from Deferred rather than expensed as bad debt — a tracked
  policy follow-up). Proven by a two-cycle Postgres test.

## [0.10.0] - 2026-08-03 — The receipts release

Nothing here asks to be believed. A migration into Recurso now proves itself
before you cut over — a Compare gate for Stripe, Chargebee, and RevenueCat
that checks coverage, money-critical fidelity, and billing continuity, then
persists as a dated, printable receipt. Every report figure is two clicks
from the journal legs behind it. Invoices carry your logo and signature, and
a latent bug that broke the statutory e-invoice QR is fixed. The dashboard's
navigation, onboarding, and a dozen pages were rebuilt around how operators
actually read them.

### Added

- **Migration Compare gate** for all three import sources — Stripe (#456),
  Chargebee (#458), RevenueCat (#459): `POST /v1/import/<source>/compare`
  re-diffs the committed export against live data, read-only, and reports
  per-record coverage, fidelity (plan amount/currency/interval, customer
  identity), and continuity (a period end drifted >1h is flagged as the
  double-billing risk it is). `ready` ⇔ zero issues; wired into the Import
  wizard's final step. (#456, #458, #459)
- **Compare receipts**: every Compare run persists (migration 000165) and
  renders as a printable, dated document — verdict, coverage, issues, and
  method — via `GET /v1/import/compare-reports/:id/document`; the wizard
  gains *Download report*. (#468)
- **Invoice branding**: Settings → Invoice branding — company name, logo,
  signature, signatory, bank details, and terms on invoice documents
  (migration 000164), applied on dashboard and portal renders; statutory
  GST/W-9 identity still wins on tax invoices. Credit notes carry the same
  letterhead. Images are strictly validated data URLs (PNG/JPEG, 300KB).
  (#452, #455)
- **Explain any number**: the Ledger is URL-addressable
  (`?account_id`/`?account_code`/`?code`), and Trial Balance rows, Month-End
  Close rows, and Revenue Recognition's headline cards deep-link to the
  postings behind them. (#467)
- **Navigation IA v2 + onboarding**: the sidebar reorganized around operator
  jobs (Billing / Usage / Revenue Recovery / Payments / Books / Reports /
  System, with Reconciliation leading Books), and a state-driven
  "Get to your first invoice" checklist on Home for new tenants. (#447)
- **Restore drill**: `scripts/restore_drill.sh` restores a live dump and
  proves row parity, invoice fidelity, and double-entry conservation; first
  published drill report included. (#448)
- Ledger entries API: `?code` posting-type filter plus real `limit`/`offset`
  paging (the page was silently capped at 100). (#457)
- `GET /v1/analytics/usage` returns `customers_metered` — a real distinct
  count. (#465)

### Fixed

- **Statutory e-invoice QR and signature images rendered broken** —
  `html/template` sanitized `data:` URLs to `#ZgotmplZ`; image fields are now
  typed `template.URL` with a regression test. (#452)
- The demo seeder now posts coherent books — deferred revenue funded at
  issuance, recognition legs per recognized event, credit-note issuance
  legs — verified against the reconciler's own checks; `--reset` purges by
  ownership with full FK coverage and keeps in-use demo plans. (#451, #460,
  #462, #464)
- Subscriptions list showed raw plan ids and `$0.00/mo` while plans loaded;
  invoice/subscription detail sheets showed raw UUIDs and machine dates;
  notifications dates humanized; Smart Dunning's error banner rendered
  inside a button. (#453, #463, #465, #471)

### Changed

- Ask AI first-run experience (example gallery, staged thinking state), Events
  detail and family-colored stream with a type filter, Wallets activity sheet,
  sync-record detail with deep links, grouped Settings hub, session-device
  labels, books-pages polish (AR rollups, totals footer, waterfall chart,
  de-jargoned dunning). (#449, #450, #453, #454)

## [0.9.0] - 2026-08-03 — The paper-trail release

The money you move now explains itself: credit notes become statutory-grade
documents with their tax breakdown recorded at creation, customers are emailed
the moment a refund or credit goes live (the one money event that never
notified), the customer portal shows every figure in its own currency, and a
schema-wide index audit makes the busiest paths — the invoice list, every
settlement webhook, the dunning sweep — index-served at any scale.

### Added

- **Statutory credit-note documents** — credit notes store their tax breakdown
  (taxable value, IGST/CGST/SGST, HSN; migration 000160), populated at
  creation: downgrade credits carry the reversed proration tax, invoice-linked
  credits slice the invoice's tax proportionally (the same math the GSTR-1
  CDNR report uses). The downloadable document renders a GST-grade CDN and the
  dashboard's detail sheet mirrors it. Standalone goodwill credits stay
  gross-only.
- **Credit-note issuance emails** — the customer is now notified when a credit
  note goes live: a refund on its way to their payment method, or account
  credit that will offset upcoming invoices. Direct-issued notes email
  immediately; maker-checker notes email at approval. This was the one money
  event that never notified.

### Fixed

- Invoice reads never surfaced `tax_type` (the column was missing from the
  single-invoice scan) — every `GetByID` consumer saw it empty.
- The customer portal's Total Paid / Outstanding cards summed minor units
  across currencies and formatted the result as USD; they are now per-currency.
- The last three unclamped list limits (dunning history — which accepted
  negative values verbatim — and both webhook-management lists) now clamp
  through the shared helper.

### Changed

- Reconciliation page labels the newer discrepancy types instead of showing
  raw identifiers; recognition sweeps no longer print a raw-log line per run
  (structured `slog` throughout the revrec repo and ledger handler).

### Performance

- **Hot-path indexes across the schema** (a systematic EXPLAIN audit;
  migrations 000161–000163). The invoices table — the busiest in the product —
  had no index on tenant, customer, subscription, gateway payment, or the
  dunning due-date shape; the dashboard list, every settlement webhook, and
  the dunning sweep could only sequential-scan. Rev-rec's reconciliation sum
  and per-payment schedule checks, and the webhook delivery worker's
  continuous claim loop, likewise gained the partial indexes they lacked.
  Every other hot table was verified already-indexed.

## [0.8.0] - 2026-08-03 — The correctness release

No new surface — a deep correctness pass over the money paths that already
exist. The headline: a money-out arbitrage (coupon-blind proration could mint
more account credit than a discounted customer ever paid) found and fixed, a
family of revenue-recognition edges around mid-period downgrades closed, FX
reporting made exponent-correct for zero/three-decimal currencies, and the
self-verifying ledger safety net extended far enough that the whole stack now
holds across 128 randomized billing sequences with coupons in the mix.

### Fixed

- **Revenue recognition around mid-period downgrades** — a downgrade credit
  could drive Deferred Revenue wrong-sign when recognition had run ahead of the
  proration boundary. The credit's net now splits by where its funding actually
  sits: the schedule's pending part drains Deferred; genuinely-recognized
  revenue is clawed back out of Recognized Revenue (capped at, and marking, the
  subscription's recognized events so repeated downgrades can never over-claw);
  any residual funded by an unpaid invoice's unscheduled deferral comes out of
  Deferred and is recorded as *schedule debt*, shrinking the schedule created
  when that invoice is later paid (migration 000158) — so credited-back service
  is never re-recognized as revenue.
- **Coupon-aware plan-change proration** — proration used list prices while
  renewals honor the subscription's coupon, so a heavily-discounted customer
  could downgrade into more spendable account credit than they ever paid
  (money-out over-credit), and upgrades over-charged the full list difference.
  A new per-subscription flag records whether the current period's invoice
  carried the discount (migration 000159); proration and its preview now
  credit/charge at the discounted prices.
- **Filtered charges on progressive subscriptions** — dimensional-pricing
  charges were routed through the filter-blind progressive watermark and billed
  every event at base rates (or nothing at all when base amounts lacked the
  invoice currency). They now fall through to the classic per-value path,
  exactly as the volume model already did.
- **FX normalization honors currency exponents** — reports converted minor
  units with major-to-major rates, so JPY figures were 100× off and KWD/BHD 10×
  off in every FX-normalized view (MRR analytics, revenue segments, waterfall,
  invoice aging, dunning recovery, consolidation). One exponent-aware
  conversion helper now backs the normalizer and both rate providers.
- **Pay-in-advance requires an additive aggregation** — per-event captures sum,
  so PIA on max/latest/unique/percentile metrics mis-billed (max concurrent
  seats billed per heartbeat; unique users billed per repeat visit); charge
  validation now restricts PIA to count and sum, and a count metric bills
  exactly one unit per event, matching the arrears `COUNT(*)`.
- **Inbound webhook dedup fails closed** — a dedup-store outage no longer lets
  a gateway webhook process on faith (risking duplicated side effects); the
  request 503s and the gateway's retry delivers it once the store recovers.

### Added

- **Reconciler: deferred-vs-scheduled invariant** — a standing
  `deferred_below_scheduled_revenue` finding fires when Deferred Revenue drops
  below the revenue still scheduled to recognize; unlike the wrong-sign check
  it survives aggregation across subscriptions and entities.
- **Invariant harness: couponed subscriptions + regression seeds** — the
  randomized ledger harness now creates a third of its subscriptions inside a
  discounted period and permanently carries the two seeds that exposed the
  downgrade-reversal edge, locking the whole coupon × proration × revrec
  surface into CI.
- Dashboard reconciliation page labels the newer discrepancy types
  (credit-note leg, unbalanced ledger, wrong-sign balance, deferred-below-
  scheduled) instead of showing raw identifiers.

## [0.7.0] - 2026-07-31 — The bank-debit release

Direct debit lands (ACH in the US, GoCardless SEPA/Bacs in the UK/EU), Recurso
becomes agent-operable over MCP, and a public demo sandbox ships — on top of a
deep money-path correctness sweep and a now self-verifying ledger safety net.

### Security

- **Cross-tenant BYO-webhook binding** — a bring-your-own per-connection
  webhook could reference another tenant's object. Every money-move is now
  bound to the connection's own tenant: invoice settle/failure/reversal across
  all three gateways, the gateway refund-event status advance, and the Razorpay
  virtual-account reconcile (tenant-scoped atomic increment). A `member` can no
  longer self-issue a refund credit note (money out without approval); wallet
  close and offline-payment recording now require a manager role.

### Added

- **NetSuite & Tally connections** — the sync adapters finally have a
  connection flow: `POST /v1/accounting/connect-token/{provider}` takes a
  pasted SuiteTalk OAuth 2.0 token for NetSuite (EXPERIMENTAL) and no
  credentials at all for Tally (local JSONL export, residency-safe). The
  Integrations page offers all four providers.
- **Webhook pause/resume** — `PUT /v1/webhooks/{id}/status`
  (`active`/`inactive`); paused endpoints keep secret and config.
- **Usage raw-event stream** — `GET /v1/usage/events` (customer/dimension
  filters) plus a "Recent events" inspector on the Usage page: verify
  ingestion is landing without opening the database.
- **Ledger invariant harness** — randomized billing sequences (upgrades,
  one-offs, recognition, cancels) must reconcile with zero discrepancies
  after every step; runs in CI against Postgres. E2E suite additionally
  gained coupon-math, usage round-trip, webhook lifecycle, and a
  zero-discrepancy reconciliation gate.
- **Invariant harness now drives real service create-paths** — the harness
  previously posted ledger legs directly, so it couldn't catch a *service*
  that forgot its leg. It now runs the real quote-conversion, gift-purchase,
  and trial-conversion flows through the reconciler (which immediately surfaced
  a quote-conversion path that was broken against Postgres). The reconciler also
  gained a credit-note leg-completeness check, so a forgotten credit leg is
  caught the same way a forgotten invoice leg is — the ledger safety net now
  self-verifies on both sides.
- **API monitoring** — Cloud Monitoring uptime check on `/health` plus
  alert policies for downtime and 5xx spikes.
- Organization rename UI; frontend lint/build/test CI job (previously CI
  ran no frontend checks at all).
- **Demo mode** — `DEMO_MODE=true` turns an instance into a safe public
  sandbox: every outward adapter (gateways, notifier, GSP, telemetry,
  webhook delivery, SaaS sync/export) is forced to its mock at the
  construction site; destructive identity edges 403 with code
  `demo_mode`; the sandbox bootstraps a demo tenant/user/key and a rich
  seeded data set (now including metering, a funded wallet, a commitment,
  and a usage alert) on first boot; `POST /auth/demo` + dashboard
  `?demo=1` land visitors logged in; and a reset worker restores pristine
  data every `DEMO_RESET_INTERVAL` (default 1h).
  `docker-compose.demo.yml` serves the whole sandbox with one command.

### Fixed

- **Two invoice-creating flows posted no ledger leg** — quote→invoice
  conversion and the buyer's gift-purchase invoice were created without the
  double-entry AR→Revenue leg every other flow posts, so their later payment
  drove AR negative and never recognized the revenue. Both now post the leg.
  Quote conversion was *also* outright broken against Postgres (it stamped a
  non-deferrable FK before the invoice row existed) — now done atomically.
- **Pay-in-advance usage billed paused/canceled subscriptions** — a paused sub
  kept charging per event during the pause and a canceled one accrued phantom
  charges; usage now only bills active subscriptions.
- **Gift subscriptions auto-renewed and dunned the recipient** — a redeemed
  gift is prepaid by the buyer, but it renewed and invoiced the recipient (who
  has no payment method) into dunning. Gifts now cleanly expire at period end.
- **Accepted cancel-flow discount offers were never applied** — a customer who
  stayed for a promised discount was billed full price; the discount is now
  minted as a coupon and applied to upcoming renewals.
- **Recurring coupons, US nexus gate, and plan-change tax** — recurring coupons
  were dropped on renewal/trial (the subscription's `coupon_id` was never
  persisted); auto-established economic nexus poisoned the US collection gate;
  mid-period plan-change proration taxed the net at a single HSN rate. All
  corrected, with per-currency exponent handling threaded through rating,
  invoicing, tax, and the accounting/e-invoice adapters.
- **Revenue recognition gaps** — a reversal→re-collect could create a second
  active schedule (double recognition), and a wallet/credit-covered invoice
  funded Deferred but never scheduled its recognition; both fixed.
- **Frontend money input** — the quote and referral forms asked for amounts in
  cents while every other form takes dollars; both now take dollars (exponent-
  aware, so JPY/KWD round-trip correctly).
- **Ledger completeness (audit-grade F1/F2/F3)** — mid-cycle upgrade
  proration and mandate debits never posted their invoice leg (DR AR / CR
  Deferred), drifting AR/Deferred permanently; the rev-rec worker could
  mis-mark a recognized event `failed` under concurrency (events are now
  claimed atomically, migration `000105`); one-off immediate recognition
  drained a never-funded Deferred balance at gross — it now records net
  of tax with no ledger posting. All three matched live reconciler
  discrepancies. (F1 ported from archive PR #82, lost in the repo split.)
- **Rate-limiter key collision** — the global limiter and the strict auth
  limiter shared one Redis key, so ~20 requests of any kind per minute
  locked users out of `/auth/*` ("Could not reach the API" on login).
  Limiters are now scope-namespaced; session endpoints get their own
  budget.
- **Coupon percent/amount enum** — seeded "percentage" coupons were
  billed as minor-unit discounts (20% → ₹0.20); normalized by migration
  `000104` and guarded by a new E2E check.
- `GET /v1/mandates` 500 on NULL gateway columns; login bounced on
  transient `/auth/me` failures (now 401-only with retries); shared
  plan/subscription lookups silently truncated at the API's limit=10
  default; webhook Pause button called an undefined binding; per-row
  customer hydration N+1s behind Credit Notes and Revenue-by-Geography;
  demo reseeds stranded orphaned per-customer AR ledger accounts.
- `google.golang.org/protobuf` → 1.33.0 (CVE-2024-24786).
- **OpenAPI spec accuracy** — documented two request fields the handlers
  already accept but the spec omitted: `CreateCreditNoteRequest.type`
  (`adjustment`/`refund`, where `refund` triggers the gateway refund) and
  `CreateSubscriptionRequest.trial_days`. Converted six OpenAPI 3.0
  `nullable: true` usages (Entitlement, TaxNexus, entitlement responses)
  to 3.1 `type: [x, "null"]` unions, so `redocly lint` passes with zero
  errors. All three SDKs regenerated/updated to match (Node, Python, Go).

### Changed

- **Dashboard consistency pass** — every page reviewed against
  production: create/add flows are right-side sheets with customer/
  subscription/invoice pickers (no more raw-UUID inputs); ledger AR
  sub-accounts labeled per customer; complete top-bar title map; archived
  customers badged; customer names resolved across Mandates, Wallets,
  Churn, Offline Payments, and the ledger.
- **Frontend performance** — route-level code splitting (entry chunk
  1,660 kB → 149 kB) and react-query caching for reference data and the
  big list pages (Invoices/Subscriptions/Customers).
- **API performance** — heavy finance reports (trial balance, deferred
  rollforward, rev-rec report/waterfall) share the 5-minute tenant-scoped
  Redis cache; the cache middleware no longer falls back to IP-keyed
  entries.
- CI actions on Node 24 (checkout@v5, setup-go@v6, upload-artifact@v5);
  SDKs (Go/Node/Python) refreshed to full API coverage in their repos.

## [0.6.0] - 2026-07-19

The usage-based billing release: everything between a usage event and the
money it becomes — billable metrics, tiered pricing, unattended renewals,
prepaid wallets, minimum commitments, usage alerts, batch ingestion, an
append-only audit trail — plus the first wave of ecosystem integrations
(GoCardless, Adyen, NetSuite, Avalara, HubSpot, S3 export) shipped as
experimental pending sandbox certification.

### Added

- **HubSpot CRM sync (EXPERIMENTAL)** — a daily worker upserts every
  customer as a HubSpot contact (keyed by email) carrying
  `recurso_customer_id` and `recurso_subscription_state`
  (active/churned). Private-app token via `HUBSPOT_ACCESS_TOKEN`;
  SaaS egress, so blocked under `RESIDENCY_MODE=self_hosted`.
  Salesforce follows as its own spec (JWT auth design needed).

- **NetSuite accounting sync (EXPERIMENTAL)** — a SuiteTalk REST adapter
  in the existing accounting-sync framework (customer / invoice / item
  upserts with Location-header id capture, `ErrExternalGone` remapping,
  per-line major-unit amounts). Residency-guarded like QuickBooks/Xero;
  connect with provider `netsuite` (RealmID = NetSuite account id).

- **Avalara AvaTax provider (EXPERIMENTAL)** — US sales-tax quotes via
  uncommitted SalesOrder transactions (`AVALARA_ACCOUNT_ID` /
  `AVALARA_LICENSE_KEY` / `AVALARA_COMPANY_CODE`), sharing TaxJar's error
  taxonomy, 24h rate cache, and the `RESIDENCY_MODE=self_hosted` guard.
  TaxJar takes precedence when both are configured.

- **S3 finance export (EXPERIMENTAL)** — a daily worker ships each
  tenant's general ledger CSV to operator-owned object storage
  (`S3_EXPORT_BUCKET`/`REGION`/`PREFIX`, `S3_EXPORT_ENDPOINT` for
  MinIO/R2), signed with a dependency-free SigV4 client. Idempotent keys
  per day; disabled unless configured.
- **Automation & Okta docs** — Zapier/n8n recipes over the signed webhook
  surface and the Okta SAML SSO walkthrough (Track D6/D7 ride existing
  code).

- **GoCardless + Adyen gateways (EXPERIMENTAL)** — bank-debit rails
  (SEPA/BACS via mandate-first billing requests) and global card
  processing (Checkout Sessions, off-session stored-method charges), both
  behind `GATEWAY_CURRENCY_OVERRIDES` (e.g. `EUR=gocardless,SGD=adyen`)
  with the existing INR→Razorpay / default→Stripe routing untouched.
  httptest-verified request shapes and idempotency keys; sandbox
  verification pending (founder-gated).

- **Append-only audit trail** — every successful config-grade mutation
  (plans, metrics, charges, coupons, webhooks, wallets, alerts, team,
  settings, ...) is recorded via middleware with actor (dashboard user or
  API key), route, entity, and the truncated request payload.
  `GET /v1/audit-logs` filters by entity/actor/time; the table rejects
  UPDATE and DELETE at the database level.

- **Ingestion hardening** — `POST /v1/usage/events/batch` ingests up to
  500 events with per-item results, and events accept an optional
  `transaction_id` idempotency key: a retried event with the same
  (subscription, transaction_id) collapses to the original, so SDK retries
  can never inflate usage. A covering index on
  (subscription_id, dimension, timestamp) backs the aggregation hot path.

- **Usage threshold alerts** — `/v1/usage-alerts` configures per-
  subscription thresholds on a billable metric (absolute quantity or
  percent of the matching entitlement limit). The billing-cycle sweep
  evaluates them and fires AT MOST once per billing period per threshold —
  a conditional claim dedups concurrent sweeps — emitting the
  `usage.alert.triggered` webhook event plus an email to the customer.

- **Minimum commitments** — `PUT /v1/subscriptions/{id}/commitment` sets a
  per-period floor (minor units); when a period's subtotal (flat + add-ons
  + metered usage) falls short, a "Minimum commitment true-up" line fills
  exactly the difference, taxed at the plan HSN. The usage-amount preview
  reports the projected true-up. Overage needs nothing: usage past the
  floor bills naturally.

- **Prepaid wallets** — money-denominated stored value per
  customer+currency (`/v1/wallets`, top-ups, transactions, auto-recharge).
  Invoices drain the wallet FIRST (wallet → credit notes → gateway), with
  residues consumed oldest-expiring-first so dated promotional credit is
  spent before it lapses; expired residue is written off by the sweep.
  Every movement is append-only with balance-after and posts balanced
  ledger legs (DR Cash / CR Customer Credit on top-up; DR Customer Credit /
  CR AR on drain). Auto-recharge tops up below-threshold wallets from the
  saved payment method on the billing-cycle tick.

- **Unattended renewals** — a billing-cycle scheduler claims due,
  locally-billed subscriptions (leased at-most-once claims), generates the
  renewal invoice (flat fee in advance + metered usage in arrears),
  advances the period anchor-preservingly, and best-effort charges the
  saved payment method; declines flow to dunning. `BILLING_CYCLE_INTERVAL`
  configures the tick (default `5m`, `0` disables). `cancel_at_period_end`
  subscriptions get their final usage rated, then cancel.
- **Metered mandate debits** — UPI-mandate debit invoices now carry the
  subscription's rated usage lines, and the subscription billing period
  advances with each cycle (it previously never advanced for mandate
  subscriptions, leaving current-period usage reporting stale). Usage that
  would exceed the mandate's authorized ceiling is billed on a separate
  open invoice instead of over-charging.
- **SDK metering methods** — Node `recurso@1.3.0`, Go `v1.1.0`, and Python
  `1.2.0` expose billable metrics, plan charges, the usage-amount preview,
  and event properties.

- **Usage-based billing v1** (`docs/spec_usage_billing.md`) — the metering
  engine: **billable metrics** (`POST/GET/PUT/DELETE /v1/billable-metrics`)
  aggregate usage events by `count`, `sum`, `max`, or `unique` (distinct
  values of an event property); **charges** attach `per_unit`, `graduated`,
  `volume`, or `package` pricing for a metric to a plan
  (`PUT/GET /v1/plans/{id}/charges`), with per-currency amounts and
  sub-minor-unit decimal rates (0.0035/call) computed exactly and rounded
  once per line; **rating at period close** bills the elapsed period's
  usage in arrears on the renewal invoice as tax-resolved line items, with
  a `usage_ratings` window claim making retried generation idempotent;
  immediate cancellation bills the partial window on a final usage invoice.
  `GET /v1/subscriptions/{id}/usage-amount` previews the running period,
  and `POST /v1/usage/events` accepts optional `properties`.

## [0.5.0] - 2026-07-17

The India release: the full statutory lifecycle (invoice → IRN → GSTR-1/3B →
TDS) from a billing engine you self-host, plus the correctness and security
hardening from a deep multi-agent review of the money paths.

### Added

- **GST returns, return-ready** — `GET /v1/india/gstr1` and
  `GET /v1/india/gstr3b` assemble both returns from the period's finalized
  invoices and credit notes: GSTR-1 with B2B/B2CS/CDNR sections and the HSN
  rollup; GSTR-3B as the net summary (Table 3.1(a) net of credit notes,
  Table 3.2 inter-state unregistered by place of supply, purchase-side
  sections as explicit zeros for the CA). Each response carries a
  `gov_schema` object in the official GSTN JSON shape — government field
  names, amounts in rupees — ready to validate with the Returns Offline Tool
  (ENG-203).
- **Self-hosted data residency** — `RESIDENCY_MODE=self_hosted` hard-disables
  every optional third-party egress: telemetry (even when opted in),
  QuickBooks/Xero sync (the connect flow and existing connections), and the
  TaxJar API. Financial data leaves the deployment only through the payment
  gateways, GSP, SMTP, and webhook endpoints the operator configures.
  `docs/india-data-residency.md` states the guarantee for security reviews;
  every enforcement point is unit-tested.
- **TDS record-on-receipts** — offline payments accept `tds_amount` for the
  portion a B2B customer withheld at source. It counts toward settling the
  invoice, accumulates on `invoices.tds_amount`, and posts
  DR TDS Receivable / CR AR in the ledger with the cash leg net of TDS.
- **Provable-ledger auditor outputs** — trial balance, deferred-revenue
  rollforward, GL export, and the revenue-recognition waterfall, wired into
  the dashboard's Finance sidebar (ENG-192, ENG-194); deeper analytics
  endpoints (MRR waterfall, invoice aging, unit economics, revenue by plan
  and geography).
- **Real Redis infrastructure** — distributed locking and idempotency backed
  by Redis, with `REQUIRE_REDIS` to fail closed on multi-instance
  deployments; the lock's mutual exclusion is proven by test (ENG-161,
  ENG-193).
- **Inbound webhook idempotency** for Stripe and Razorpay events (ENG-162),
  and **team email invites** where teammates set their own password
  (ENG-196).
- **Official Go SDK** — hand-crafted, stdlib-only `recurso-go` covering the
  full API surface.
- **Demo-data seeder** (`cmd/demo_seed`) — additive, tenant-scoped, with
  `--reset`, covering rev-rec, referrals, gifts, and reconciling ledger
  postings.

### Changed

- **SDKs moved to standalone repositories** — `recurso-go`, `recurso-node`
  (1.2.0, responses/requests typed from the OpenAPI spec), and
  `recurso-python` (1.1.0, regenerated; unexpected API statuses now raise
  instead of returning `None`). The in-repo `sdk/` directory and its CI job
  are gone; install/publishing docs point at the new repos.
- Ledger documentation now states plainly: PostgreSQL is authoritative,
  TigerBeetle is an optional mirror.
- Internal business material (pitch deck, review reports) moved out of the
  repository.

### Fixed

- **Billing correctness family (ENG-140–154)** — revenue is deferred NET of
  GST and unwound on cancel/refund/downgrade; signed double-entry balances;
  atomic trial-conversion billing; downgrade credits reverse GST and persist
  as adjustment credit notes applied at billing time; account credit is a
  real liability; cash postings are net of applied credit; first-period
  proration; no phantom revenue on UPI-Autopay debits.
- **Month-end billing dates** — interval math clamps instead of overflowing
  (Jan 31 + 1 month = Feb 28, never Mar 3), and anchored subscriptions
  restore their day in long months (Feb 28 → Mar 31) instead of sticking at
  28.
- **Atomic-claim sweep (ENG-160–200, PHASE2)** — one-shot semantics enforced
  by conditional updates everywhere a retry or race could double-fire:
  mandate debits (with the idempotency key proven to reach the gateway),
  trial activation, dunning steps and bandit weights, e-invoice IRN retries,
  gift redemption, quote→invoice conversion, cancel-flow retention offers,
  virtual-account credits, refund over-issue, and idempotency-key claims
  themselves; downgrade credit + plan flip commit in one transaction; the
  IRN is registered only after the invoice durably commits.
- **Graceful shutdown** — background workers drain under a bounded
  WaitGroup; all schedulers stop concurrently and idempotently (a double
  Stop previously deadlocked the exiting process); in-flight webhook
  deliveries respect cancellation.
- **Tax fixes** — CGST/SGST always split into equal halves; non-EU B2C
  digital exports are zero-rated instead of falling back to domestic VAT;
  export exemption requires an actual cross-border supply (GB→GB regression);
  single-digit GST state codes resolve; GST/VAT rounds instead of
  truncating.
- **Tenant isolation (ENG-157–169)** — repository-level tenant scoping for
  handler-reachable mutations, invoice settlement, and mandate writes;
  offline payments settle an invoice only when covered and only for the
  matching customer/currency.

### Security

- Stripe webhook verification fails closed when the secret is unset;
  outbound webhook URLs are SSRF-guarded (ENG-175, ENG-177).
- Portal session tokens hashed at rest; auth tokens atomically single-use;
  TOTP codes can't be replayed; per-account login lockout (ENG-145,
  ENG-151, ENG-176).
- Owner role protected from admin privilege escalation; API-key creation
  gated; multiple cross-tenant IDORs closed (ENG-164, ENG-165, ENG-178,
  ENG-160).
- O(1) API-key lookup replaces the bcrypt-per-key scan; trusted proxies
  configured so X-Forwarded-For can't bypass rate limits; `/health` no
  longer leaks connection errors (ENG-174, ENG-197, ENG-198).

## [0.4.0] - 2026-07-10

### Added

- **Real hosted checkout** — server-verified card/ACH collection via the Stripe
  Payment Element and UPI/cards/netbanking via Razorpay Checkout, smart-routed by
  currency (INR → Razorpay, else Stripe) (ENG-4).
- **Customer self-service portal** — magic-link login, card update via Stripe
  SetupIntent, UPI mandate re-authorization, and invoice history; payment-recovery
  emails deep-link straight into it (ENG-5).
- **Smart dunning depth** — off-session saved-card retries, retries settled
  through the double-entry ledger, and recovery deep-links (ENG-5).
- **US economic-nexus tracking** — per-state year-to-date sales/transaction
  tracking that auto-establishes economic nexus when a threshold is crossed;
  `GET /v1/settings/tax/nexus/status` reports proximity. Dataset seeded
  uncertified pending professional review (ENG-16).
- **Dashboard redesign** — shadcn foundation with stone design tokens, a
  monospace Money signature, a ⌘K command palette, and a Test-mode chip
  (ENG-135, ENG-136).
- **Cloud waitlist** — API-backed early-access signup (`POST /waitlist`) (ENG-12).
- Jurisdiction-aware invoice PDFs wired to real invoices.

### Changed

- **API keys are now mode-gated (test vs. live).** Newly created keys are issued
  as `rsk_test_` (test) or `rsk_live_` (live), and a key's mode must match the
  server's `gateway_mode` (reported at `GET /version`): a test key is accepted on
  a `none`/`test` server, a live key on a `live` server. Mismatches return
  `401 key_mode_mismatch`, so a test key can never move real money. New accounts
  get a test key by default; mint a live key with
  `POST /v1/developer/keys {"mode":"live"}`.

  **Breaking:** existing generated `sk_live_` keys are grandfathered as live and
  keep working against a live-gateway server — but a request from such a key to a
  **non-live** server (the mock `none` gateway or gateway test mode) now returns
  `401 key_mode_mismatch`. If you develop against a non-live instance with an old
  `sk_live_` key, either mint an `rsk_test_` key (Settings → API Keys, or the
  developer-keys endpoint) or configure live gateway keys. The demo key
  `sk_test_12345` is unaffected — it is grandfathered as a test key.

### Fixed

- **Tenant-context audit** — features that never worked against a real
  multi-tenant database: trial conversion (ENG-3), plan changes that silently
  never persisted (ENG-6), and four more background paths (ENG-134), plus
  `tenant_id` propagation across 11 handlers and silently-zero consolidated MRR.
- **Background-job robustness** — NULL / timestamptz scans that aborted the
  nexus, churn, pre-charge, dunning-retry, and accounting sweeps against real
  data (ENG-143).
- **Dashboard** — page crashes (Security, Referrals/Gifts on empty data), sparse
  or broken detail panels (Customer, Quote, Credit Note, Coupon), and
  non-functional create buttons/routes.
- **Live-key payment fixes** — Razorpay UPI-Autopay registration payload, Stripe
  inactive method types, checkout failure states, and invoice PDF currency
  symbols.

### Security

- Removed unauthenticated portal routes that exposed cross-tenant PII (ENG-139).
- GenAI text-to-SQL tenant isolation is now enforced by the database (dedicated
  schema + read-only role), not the prompt (ENG-137).
- Tenant-gated the invoice PDF route; patched dependency CVEs flagged by
  govulncheck / Trivy (jwt/v4 4.5.2, rollup 4.62.2, vite 7.3.6, x/oauth2 0.27.0,
  goxmldsig 1.6.0, Go 1.25.12).

## [0.3.0] - 2026-07-07

### Added

- **Opt-in anonymous telemetry** — measures self-hosted activation (how
  many instances reach their first real invoice) so the project can see
  adoption without a hosted service. Strictly opt-in (TELEMETRY_OPTIN=
  true); default OFF means zero network calls and zero data written.
  When enabled: a random instance ID, once-ever milestone events, and a
  24h heartbeat with range-bucketed counts — never amounts, names,
  emails, keys, IDs, or exact numbers. docs/telemetry.md documents every
  payload and the one-line opt-out.
- **One-click deploy** — Render, Railway, and DigitalOcean templates
  with README deploy buttons; a devcontainer for Codespaces; and a real
  Next.js starter in examples/ (pricing, signup, usage with entitlement
  headroom, feature gating) that builds and runs against `make demo`.
- **Competitor comparison pages** — honest /vs/ pages for Lago,
  Flexprice, Kill Bill, Chargebee, and Stripe Billing.
- **Cloud operations groundwork** — a keystroke-level provisioning
  runbook for the first manually-provisioned cloud customers, and a
  status-page plan.

## [0.2.3] - 2026-07-07

### Added

- **US sales tax via TaxJar** — set TAXJAR_API_KEY and US invoices get
  live jurisdiction rates (24h per-location cache, retry-once, invoices
  never blocked by lookup failures: they ship at 0% flagged
  sales_tax_error for review). Unconfigured stays the honest
  sales_tax_stub. Nexus-state configuration is not yet modeled.
- **Usage platform depth** — GET /v1/usage (time-bucketed windows),
  GET /v1/subscriptions/{id}/usage (current-period and lifetime usage
  per dimension with entitlement limit and remaining headroom),
  GET /v1/usage/dimensions catalog.
- **Accounting sync efficiency** — changed-since dirty tracking (daily
  sync pushes only what changed; manual sync forces a full re-push);
  Xero invoice lines now reference synced Items by code.
- SDKs: Node usage.query/usage.dimensions/subscriptions.usage (78
  tests); Python regenerated (12-check smoke). New docs: US sales tax
  compliance page, usage windowing and entitlement-headroom patterns,
  incremental-sync semantics.

### Fixed

- **Security**: POST /v1/usage/events accepted events against any
  tenant's subscription; it now enforces subscription ownership and
  customer match.
- Stale usage docs described fictional request fields; corrected to the
  real API.

## [0.2.2] - 2026-07-06

### Added

- **Webhook delivery tracking** — GET /v1/events/{id}/deliveries and
  GET /v1/webhooks/{id}/deliveries expose per-delivery status, attempts,
  response codes and errors; POST /v1/events/{id}/redeliver re-queues
  delivery idempotently.
- **FX-normalized MRR** — tenant and org MRR convert across currencies
  to a reporting currency (tenant BaseCurrency, else REPORTING_CURRENCY)
  with the rates, source (live / static-fallback), and timestamp in the
  response; unconvertible currencies are flagged, never silently mixed.
- **Refund lifecycle completion** — Stripe and Razorpay refund webhooks
  advance pending refunds to processed or refund_failed (with the
  gateway's reason); mandate-collected invoices now capture a
  refundable payment id.
- SDKs: Node gains delivery/redeliver/MRR methods (17 resources, 64
  methods, 75 tests); Python regenerated with the new endpoints
  (10-check smoke). Docs updated across API reference and guides.

### Fixed

- Payment-success webhook processing (Stripe and Razorpay) failed with
  a tenant-context error and returned 500 before recording anything —
  handlers now resolve the invoice's tenant themselves.
- The dashboard MRR tile always showed $0 (read a field the API never
  returned).
- Delivery worker retries no longer erase the failing HTTP status code.
- The new-customer form's oversized styling, stale state on country
  switch, and literal \n placeholder.

## [0.2.1] - 2026-07-06

### Added

- **Python SDK** (`sdk/python`) — generated from the served OpenAPI spec:
  typed models, sync and async clients, 32 API modules, quickstart README,
  no-network smoke test.
- **Node SDK test suite** — 71 tests with 100% client coverage, including
  a reflection guard that fails when a method ships untested; SDK builds
  and tests now gate CI.
- **Dashboard**: entitlements editing on plan detail (PUT-replace
  semantics with validation), a Finance > Reconciliation page (summary
  cards, TigerBeetle comparison badge, discrepancy table, "Books
  balanced" zero state), and event payload/type visibility on the
  Developers page.
- Docs: entitlements guide + API reference, recovered-revenue and
  reconciliation pages, performance numbers, error-envelope taxonomy,
  and an interactive API playground wired to the full spec.

### Fixed

- The create-API-key dialog displayed an empty key (read the wrong
  response field); key listings now show real prefix/type/status/date.
- Removed remaining mock content from the dashboard (fake usage tiers,
  dead buttons, mock pagination).
- OpenAPI corrections: PUT /v1/plans/{id}/entitlements takes a bare
  JSON array; reconciliation documents the TigerBeetle comparison.

## [0.2.0] - 2026-07-06

### Added

- **Entitlement engine v1** — plan-level feature grants (booleans and
  limits), effective-entitlement resolution per customer (union across
  active and trialing subscriptions: any-true booleans, max limits), a
  single-query `GET /v1/entitlements/check` fast path for feature gating,
  Node SDK support, and plan-detail UI.
- **Recovery attribution** — invoices that collect after failed attempts
  are recorded with amount, attempts, strategy, and days-to-recover;
  `GET /v1/analytics/dunning/recovered` serves totals and a 12-month
  series; the dunning dashboard shows recovered revenue.
- **TigerBeetle reconciliation** — the ledger reconciler now enumerates
  TigerBeetle transfers (paginated) and reports missing/mismatched
  entries against PostgreSQL instead of skipping the comparison.
- **Health alerting** — `ALERT_WEBHOOK_URL` (JSON or Slack format) fires
  on component state transitions; SEV1 incident runbook in
  `docs/incident-runbook.md`.
- **Jurisdiction-aware tax** — GST computed from each tenant's own
  registration state, zero-rated exports (LUT-aware), EU VAT with B2B
  reverse charge, US sales-tax engine (0% stub, explicitly marked).
- **Real accounting sync** — QuickBooks/Xero OAuth token refresh with
  rotation, provider external-ID mapping (no more duplicate books
  entries), true update pushes (QBO SyncToken sparse updates, Xero
  upserts), QBO invoice line ItemRefs.
- **Gateway refunds** — refund credit notes call the real Stripe/Razorpay
  refund APIs with over-refund guards, honest manual-required states,
  and a Refunds-vs-Cash ledger reversal.
- **Ledger reconciliation endpoint** — `GET /v1/finance/reconciliation`
  plus a daily scheduled drift check.
- Consistent API error envelope (`{"error": {"code", "message"}}`),
  hardened idempotency (scoped keys, no 5xx caching), OpenAPI spec
  covering the full 113-path surface, `RATE_LIMIT_PER_MINUTE` knob,
  published performance numbers (docs/performance.md), verified
  backup/restore drill.

### Fixed

- Per-request bcrypt capped the API at ~126 req/s — verified-key cache
  takes authenticated reads to ~7,800 req/s (p99 27ms).
- Only one tenant could register per database (unique constraint on the
  always-empty hashed key column).
- Ledger postings failed for all API-created tenants/customers — AR and
  chart-of-accounts provisioning is now self-healing on first posting.
- Portal magic-link login could never match a customer; links are now
  actually emailed.
- Razorpay mandate revocation really revokes (customer-scoped token
  deletion) instead of silently succeeding.

### Known limitations

- US sales tax remains a 0%-rate stub pending a TaxJar/Avalara
  integration; EU VAT rates are a static table.
- Accounting sync re-pushes all mapped entities daily (no dirty
  tracking); Xero invoice lines don't yet reference synced items.
- Refund webhooks (charge.refunded / refund.processed) are not consumed,
  so pending refunds don't auto-advance.

## [0.1.1] - 2026-07-05

### Added

- **Subscriber migration tool** (`cmd/import`) — import plans, customers,
  and subscriptions from another billing system (Stripe Billing, Chargebee,
  spreadsheets) via JSON or CSV. Writes directly to the database without
  generating invoices or calling payment gateways, so migrated customers
  are never double-billed mid-cycle; the renewal worker issues each
  subscription's next invoice at its imported `current_period_end`.
  Idempotent (plans by code, customers by email, subscriptions by
  `external_id`) with a `-dry-run` mode. See `cmd/import/example.json`
  and the "Migrating an existing subscriber base" section of
  `docs/deployment.md`.
- **`make seed`** — one-command demo dataset (tenant, plans, customers,
  subscriptions, invoices) for first-time exploration; prints the demo
  dashboard API key when done. Destructive: wipes the target database.

## [0.1.0] - 2026-07-04

First public release of Recurso, an open-source, self-hosted subscription
billing engine built with Go, PostgreSQL, and TigerBeetle.

### Added

- **Multi-tenant billing core** — plans, subscriptions (trials, upgrades,
  downgrades, cancellations, proration), invoicing, coupons, and usage-based
  (metered) billing.
- **Multi-currency payments** — Stripe and Razorpay integrations with smart
  gateway routing per currency, plus FX rate handling.
- **India compliance stack** — GST calculation with Place of Supply rules,
  HSN codes, TDS tracking, and e-invoicing (IRN/GSP) workflows.
- **Dunning** — configurable dunning campaigns and a smart retry engine with
  exponential backoff to maximize payment recovery.
- **Customer-facing surfaces** — hosted checkout pages and a customer
  self-service portal.
- **Revenue recognition** — deferred revenue schedules backed by an
  immutable double-entry ledger on TigerBeetle (optional component).
- **Billing documents** — quotes and credit notes with refund workflows.
- **Integrations** — outbound webhook delivery and email notifications.
- **Node.js SDK** (`sdk/node`) — typed client for the Recurso API.
- **React dashboard** (`frontend/`) — admin UI for plans, customers,
  subscriptions, invoices, and analytics (MRR, revenue).
- **Operations** — Dockerfile and docker-compose stack, Kubernetes manifests,
  CI pipeline, and versioned release builds (`/version` endpoint, ldflags
  version stamping).

### Known limitations

- Accounting sync (QuickBooks/Xero) runs in mock mode only; no real
  provider API calls are made yet.
- Razorpay mandate revocation is not implemented.
- TigerBeetle runs as a single node; no replication/HA setup is provided.
- The Node.js SDK is not yet published to npm; install it from this
  repository (`sdk/node`).

[0.1.0]: https://github.com/recurso-dev/recurso/releases/tag/v0.1.0
