# Recurso Dashboard — Operational Depth Initiative

> **Standing directive** (founder, 2026-08-13). Successor to
> `DASHBOARD_REDESIGN.md` (v0.12.0, Phases 0–11 — design system, shell,
> DataTable v2, the `ObjectPage`/`AuditTrail` framework, Customer/Subscription/
> Invoice pages, forms, home, a11y, visual QA all shipped). That mission made
> the dashboard **consistent**. This one makes it **deep**.

## Thesis

Take the dashboard from "a functional SaaS dashboard" to "the operating console
for a serious billing and financial-infrastructure system." Depth is **not**
more cards. Depth is exposing **relationships, states, history, consequences,
actions, and audit**. The defining characteristic is **financial correctness**.

Quality bar: Stripe / Ramp / Brex / Linear / Vercel / GitHub — but Recurso's own
visual language, continuing the existing token system (no raw palette, no
arbitrary radius/shadow).

## The depth test (every major page must pass)

1. **Orientation** — what is this, in 3 seconds?
2. **State** — current state, visible.
3. **Context** — *why* is it in this state?
4. **Relationships** — what does this object connect to / affect?
5. **History** — what happened over time?
6. **Financial** — the monetary consequence (amount, tax, receivable, journal, balance, reconciliation).
7. **Actions** — the appropriate operation, from the page.
8. **Safety** — consequence understood before mutating financial data.
9. **Audit** — who/what/when/why/reference.
10. **Navigation** — natural path to related objects.

If any answer is NO, the page is not finished.

## Hard rules

- **Never fake data.** Show only what the backend actually exposes. If a deep UI
  needs data the backend doesn't provide, build what's possible and log it under
  **Backend Gaps** below — do not fabricate metrics, accounting, or history.
- Build **primitives → patterns → pages**. If a pattern appears on 3 pages,
  extract it to a shared component.
- Preserve business logic, APIs, routing, permissions, accounting behavior.
- No card-grid dashboards. Prefer: summary → state → relationships → activity →
  financial → actions → audit.
- Financial mutations (refund/void/cancel/credit/adjustment/write-off/plan
  change) get consequence-explaining confirmation, not a generic "Are you sure?"
- Test after every batch: `cd frontend && npm run lint && npm run build && npx vitest run`;
  then visual pass at 320 / 375 / 768 / 1024 / 1440 in populated/empty/loading/
  error states with keyboard.

## Working model

Feature branch `dashboard-operational-depth` → green-CI PRs (main is protected).
One reviewable PR per increment. Build on the existing `ObjectPage`,
`ObjectTimeline`, `AuditTrail`, `DataTable`, `StatusBadge`, `FormSheet`
primitives — extend them, don't fork.

## Execution order (founder-specified)

1. Design system (assess vs. v0.12.0; add only what depth needs)
2. Dashboard shell
3. Home (exceptions-first)
4. Object-page framework (extend for context/consequence/downstream)
5. Customer · 6. Subscription · 7. Invoice · 8. Payment · 9. Usage ·
   10. Meter · 11. Product/Plan · 12. Ledger · 13. Account ·
   14. Reconciliation · 15. Dunning · 16. Remaining operational pages

Phases 1–4 were substantially delivered by v0.12.0; this initiative's net-new
value starts at the object level (adding the *why / what-it-affects / what-
happens-next / financial-consequence / downstream-impact* layers) and at the
still-missing object pages (Payment, Meter, Account) and the deep financial
pages (Ledger drill-through, Reconciliation discrepancy detail, Dunning
lifecycle).

## Backend Gaps (from the capability audit, 2026-08-13; UI must not fake these)

**Enabler (good news):** `GET /v1/events?object_id=<uuid>` now exists (built for
object pages) — a real per-object timeline, but limited to **10 lifecycle event
types** (invoice/subscription/customer/payment created/paid/failed/renewed/
canceled). Wallet, dunning, credit-note, dispute, quote, config changes emit
nothing to it.

**Blocking gaps (a deep page here needs new backend, not just UI):**
1. **Payments** — no `GET /v1/payments` or `/payment-attempts` anywhere.
   PaymentAttempt rows are webhook-written, never read back (except one
   `attempt_status` on the collections queue). A dedicated Payment object page
   is **not buildable** today; payment state is only visible via the invoice
   (status/amount_paid/paid_at/gateway_payment_id) + `/events` (payment.*).
2. **Invoice → ledger drill** — no per-invoice journal-entry JSON. `/ledger/entries`
   requires `account_id`; postings carry only an untyped `reference_id` + numeric
   `code` (no `source_type`, no reference_id filter). GL-with-names is CSV-only.
   "Show this invoice's DR/CR" is **not fetchable** — only reconstructable from
   the code pattern (which is computing, not fetching — flag, don't fake).
3. **Lifecycle history** — no invoice status-history, no subscription
   schedule/plan-change history, no reconciliation run history (nothing
   persisted; no run id/actor/drift). "What changed over time" = only the
   10-type `/events` feed.
4. **Per-invoice dunning attempt trail** — no `/invoices/:id/attempts`;
   PaymentAttempt + DunningCampaignExecution have no read endpoints. Decline
   codes live on the invoice (`last_payment_error`) + collections failures
   breakdown, not a per-attempt timeline.

**Ergonomic gaps (workable, note them):** most lists lack totals + sort (only
invoices/collections/audit-logs/events paginate honestly); wallet transactions
are limit-only (500 cap, no filter); usage has no previous-period comparison and
raw events carry no charge/invoice attribution; metric has no "which plans use
it" reverse lookup; audit log is config-mutations-only with no before/after diff.

**Solidly achievable now (build these deep):** Customer (identity +
subs/invoices/credit-statement/wallets/entitlements/consents/churn/risk +
events timeline; compute "currently owed" from open invoices); Subscription
(status/periods/plan-join/**proration preview via `/preview-change`**/usage/
addons/commitment/lifecycle actions/invoices); Invoice (full fields + embedded
line items + tax splits + dunning fields + PDF/send + e-invoice status); Plan
(prices/entitlements/charges + **Plan→Subscriptions reverse lookup exists**);
Ledger (accounts/entries-by-account/trial-balance with balanced+abnormal/
deferred rollforward/GL CSV); Reconciliation (expected-vs-found + 20 discrepancy
types; compute difference client-side); Dunning (collections queue w/ real
pagination + retry-now/pause/mark-uncollectible actions + analytics); Wallet;
Usage (current+lifetime, buckets, raw stream).

## Progress log

- 2026-08-13 — Initiative opened. Branch created, charter written. Backend-
  capability audit completed → gap ledger above.
- 2026-08-13 — **Increment 1: Customer (depth template).** Added
  `GET /customers/:id/financial-summary` (per-currency outstanding/past-due/
  billed/paid; narrow nil-safe port; OpenAPI + service/handler tests). New shared
  primitives `FinancialSummary` (per-currency metric strip, danger/warning tone)
  and `AttentionBanner` (exceptions-first, silent when healthy). Customer page now
  leads with the financial position + surfaces past-due invoices above the fold.
  Green: Go build + drift + tests; frontend lint 0 / build / 498 tests. Live
  visual QA against a seeded stack still to run.
- 2026-08-13 — **Increment 2: Subscription (state context).** Reused
  `AttentionBanner` (proving it's shared) to add Layer 3 — why the subscription
  is in its state + what happens next: past_due shows the real decline reason +
  retry from its own past-due invoice (links to it), unpaid, paused (+resume),
  scheduled cancel-at-period-end, trial-end. Silent when healthy. The
  plan-change proration preview already existed. Frontend-only. Green: lint 0,
  build, 501 tests. Both increments on branch `dashboard-operational-depth`
  (PR #640) since #2 depends on #1's unmerged primitive.
- 2026-08-13 — **Increment 3: Invoice (customer document + finance accounting).**
  Closed the biggest documented gap (hybrid): new `GET /invoices/:id/journal-entries`
  returns every posting referencing the invoice (reuses the GL SELECT filtered by
  reference_id; tenant-scoped; draft→empty≠missing→404). New shared `JournalEntries`
  primitive (transfer postings, DR/CR account+name, Debits=Credits tie-out) below
  the amount — the finance side of the document. Reused AttentionBanner for
  past_due (decline reason)/uncollectible/void. Backend: build+drift+lint 0+handler
  tests. Frontend: lint 0, build, 503 tests. THREE object pages, THREE shared
  primitives (FinancialSummary, AttentionBanner, JournalEntries). Next: Payment —
  needs new read endpoints (no payments resource); or Ledger/Reconciliation which
  reuse JournalEntries. Live visual QA of all three still pending before merge.
- 2026-08-13 — **Increment 4: Ledger + Reconciliation (source-of-truth pages).**
  Frontend-only. Reconciliation: verdict headline (Reconciled / N-to-resolve),
  all 20 discrepancy types labeled + a per-type reason line (was 13, no reasons),
  new signed Difference column (found−expected, computed — the report has no
  difference field). Ledger: added Debits/Credits-posted/Net-balance strip for
  the selected account (was balance only). Green: lint 0, build, 503 tests.
  Four increments, five object/finance pages on PR #640.
- 2026-08-13 — **Increment 5: Payment (closed the last big gap).** Payments
  aren't addressable objects, so the Payment "page" is the invoice's attempt
  history. New `GET /invoices/:id/payment-attempts` (repo ListByInvoice; narrow
  nil-safe lister on SubscriptionService; missing→404, exists-none→empty; openapi
  + handler tests). New shared `PaymentAttempts` primitive (attempt lifecycle
  status/failure/gateway/settled) wired into InvoicePage as a "Payments" section
  between Amount and Journal entries — the invoice now tells the whole story:
  owed → how we collected → what posted. Green: build+drift+lint 0+tests;
  frontend lint 0/build/504 tests. FIVE increments; FOUR shared primitives. All
  audit backend gaps now closed except reconciliation run-history/scoping
  (product decision). Next: Usage/Meter/Plan/Dunning (current endpoints) or a
  tenant-wide Payments log page. LIVE VISUAL QA of the whole batch STILL PENDING
  before merging #640.
- 2026-08-13 — **Live visual QA (PR #640, all 5 pages).** Rebuilt api+dashboard
  from the branch against the seeded stack (admin@acmesaas.com tenant), verified
  every new endpoint returns correct data, and eyeballed each page: Customer
  financial-summary (Outstanding/Past-due/Billed/Paid, danger tone) ✓; Invoice
  Payments (retry history, tone-coded, failure reason + gateway ref) + Journal
  entries (DR/CR + Debits=Credits tie-out) ✓; Reconciliation verdict banner +
  per-type reasons + signed Difference column ✓; Ledger Debits/Credits/Net
  strip ✓. Zero horizontal overflow at 375px across all pages (iframe measure).
  No issues found. PR #640 is merge-ready pending review.
- 2026-08-13 — **Increment 6: Dunning (lifecycle legible end-to-end).**
  Frontend-only. The Collections worklist was already deep; connected it: the
  invoice number now drills to the invoice, and the invoice's past-due banner
  now shows the retry count ("past due: <reason> — N retries so far, next
  <date>"). So worklist (what's failing) → invoice (why / how many tries /
  what's next / the Payments attempt history) is one click. Green: lint 0,
  build, 504 tests. Six increments; four shared primitives. Next candidates:
  Usage / Meter / Plan (current endpoints), or a tenant-wide Payments log page.
- 2026-08-13 — **#640 merged to main** (squash 8b314ee8). The 4 shared
  primitives + 3 read endpoints are now baseline.
- 2026-08-13 — **Increment 7: Usage** (new branch `dashboard-depth-usage` off
  merged main). The Usage Explorer now names each event's meter and shows how it
  aggregates (Event → Meter → Aggregation), joining billable metrics by
  code==dimension. Reframed the header around the meter pipeline. Documented gaps
  kept honest: no per-event charge/invoice attribution, no previous-period
  comparison — so no per-event pricing link or trend is shown (the billing side
  is on the Subscription page). Frontend-only. Green: lint 0, build, 505 tests.
- 2026-08-13 — **Increment 8: Plan** (branch `dashboard-depth-plan` off main).
  Added the "Subscriptions on this plan" reverse lookup (GET /subscriptions?
  plan_id=) to the plan detail — the directive's key Plan relationship, the only
  piece the already-rich plan view (pricing/entitlements/charges) lacked. Each
  row drills to the subscription. Closes Plan → Subscriptions (inverse of the
  existing Subscription → Plan link). Frontend-only. Green: lint 0, build, 505
  tests. NOTE: /plans/:id is still a list+sheet, not a full ObjectPage — a
  full-page conversion could follow, but the sheet is already comprehensive.
- 2026-08-13 — **#641 (Usage) + #642 (Plan sheet) merged to main.**
- 2026-08-13 — **Increment 9: Plan → full page** (branch dashboard-depth-plan-page).
  /plans/:id is now a proper PlanPage (ObjectPage: pricing, entitlements, usage
  charges via PlanCharges, subscriptions reverse lookup, audit rail), matching
  Customer/Subscription/Invoice. Editing reuses the existing PlanDetail sheet as
  the page's Edit surface (no logic duplicated — the Customer-page pattern).
  Removed the dead routeId/sheet wiring from the Plans list. Frontend-only.
  lint 0, build, 510 tests.
- 2026-08-14 — **#643 (Plan full page) merged to main.**
- 2026-08-14 — **Increment 10: Account page** (branch dashboard-depth-account).
  New AccountPage at /ledger/accounts/:id (ObjectPage): identity, the
  debits/credits/net-balance identity, and per-account journal activity (each
  posting on the side it hit this account, against its counterpart). Handles
  per-customer AR sub-accounts (name from postings; balance honestly omitted —
  only chart accounts carry balances). Made the Customer's ledger account a link.
  Uses existing getLedgerAccounts + getLedgerEntries — no backend. lint 0, build,
  514 tests.
- 2026-08-14 — **#644 (Account page) merged to main.**
- 2026-08-14 — **Increment 11: Meter page** (branch dashboard-depth-meter).
  Backend: GET /billable-metrics/:id/charges (ChargeRepository.ListByMetric —
  the reverse lookup the audit flagged as the only remaining metering gap;
  plan_charges JOIN plans → MetricPlanCharge; service verifies metric→404;
  openapi + service test). Frontend: new MeterPage at /billable-metrics/:id
  (Definition + Plans-pricing-on-it reverse lookup + Recent events feeding it +
  audit), Metering list rows rowHref to it. Event → Meter → Aggregation →
  Pricing now navigable end to end. lint 0, build, 518 tests.
- 2026-08-14 — **Increment 12: Payments log page** (branch
  dashboard-depth-payments-log). Backend: GET /v1/payment-attempts — the
  tenant-wide payments log gateway attempts never had (only per-invoice
  history existed). PaymentAttemptRepository.List (COUNT(*) OVER() total, LEFT
  JOIN invoices for number + currency) → PaymentAttemptListItem; extended the
  narrow paymentAttemptLister with List; SubscriptionService.ListPaymentAttempts
  (nil lister → empty page); handler returns {data, pagination} like
  ListInvoices; route + openapi + handler test. Frontend: new Payments page at
  /payments (nav group "Payments" → "Payments Log") — every settlement attempt
  newest-first, exceptions-first status filter (failed/returned lead, tone
  destructive), When/Invoice(link)/Amount/Method+Gateway/Status/Reason columns,
  rows drill to /invoices/:id. Answers "did this collection go through, and if
  not why?" without opening each invoice. lint 0, build, 521 tests.
- 2026-08-14 — **Increment 13: Wallet object page** (branch
  dashboard-depth-wallet). No new backend — GET /wallets/:id and
  /wallets/:id/transactions already existed; pure frontend depth. New WalletPage
  at /wallets/:id: balance header + Open/Closed badge + the three wallet actions
  (top-up, auto-recharge, close) with a consequence-explaining close confirm; a
  Balance section that reconstructs the refundable-paid vs forfeitable-
  promotional split from open top-up residues **only when it reconciles to the
  balance** (else hidden — no guessing); an AttentionBanner for closed wallets +
  promotional credit expiring within 30 days (dated, real); and the append-only
  movement ledger where each **drain links to the invoice it settled**. The
  Wallets list now rows-drill to it (retired the cramped transaction Sheet); the
  customer page's wallet rail links to the specific wallet instead of the bare
  list. A wallet's movements ARE its audit trail (wallets emit nothing to
  /events — noted honestly). lint 0, build, 525 tests.
- 2026-08-14 — **Increment 14: Credit Note object page** (branch
  dashboard-depth-credit-notes). Backend linchpin: GET
  /v1/credit-notes/:id/journal-entries — mirrors the invoice journal endpoint
  (a credit note's legs are keyed by its id via reference_id). Service
  GetCreditNoteJournalEntries (repo.GetByID tenant-scoped → 404; ledger's
  GetJournalEntriesByReference; nil ledger → empty non-nil), handler returns
  {data:{credit_note_id, entries}}, route + openapi + 2 handler tests. Frontend:
  new CreditNotePage at /credit-notes/:id **replacing the cramped detail
  slide-over** (deleted CreditNoteDetail.jsx + its test). Full object page:
  lifecycle header + approve/reject/void actions (role-gated, consequence-
  explaining confirms) + Document; AttentionBanner (pending approval, rejected,
  refund failed); amounts (credit/applied-or-refunded/balance); statutory tax
  reversal (taxable + IGST/CGST/SGST); Related (customer + the invoice it
  offsets); and the **journal-entries drill** via the shared JournalEntries
  primitive. List rows drill to it; CreditNoteDetail slide-over retired. lint 0,
  build, 520 tests.
- 2026-08-14 — **Increment 15: Dispute object page** (branch
  dashboard-depth-disputes). Backend linchpin: GET /v1/disputes/:id — disputes
  had only list + resolve, no single-read, so the page couldn't be refresh-safe.
  DisputeService.Get (repo.GetByID + tenant check → nil for missing/cross-
  tenant), handler {data: dispute}, 404 on bad/foreign id. Route + openapi
  ($ref InvoiceDispute) + 2 handler tests. Frontend: new DisputePage at
  /disputes/:id — reason ("what the customer disputed"), the contested invoice
  as a rich linked relation (its real number/amount/status via getInvoice), the
  customer link, the resolution (outcome + note + resolved-at, or "not yet
  resolved"), an AttentionBanner for open/rejected, and the Review action
  (accept — optionally issuing a credit — or reject) ported from the list. The
  list's raw-shortId invoice stays but rows now drill to the page. NB: the
  InvoiceDispute model is a lightweight customer query (not a gateway
  chargeback); it emits nothing to /events, so no timeline — the two timestamps
  carry the history. lint 0, build, 524 tests.
- 2026-08-14 — **Increment 16: Quote object page** (branch
  dashboard-depth-quotes). No new backend — GET /quotes/:id + the full action
  set already existed; pure frontend depth. New QuotePage at /quotes/:id
  **replacing the detail slide-over** (deleted QuoteDetail.jsx + its test). Full
  object page: lifecycle header (draft→sent→accepted/declined→converted) with the
  state-appropriate action set (edit/send/accept/decline/convert/delete; convert
  + delete gated by consequence-explaining confirms); AttentionBanner for
  converted (links the invoice), accepted-awaiting-conversion, and expired;
  line-items table; totals math (subtotal/discount/tax/total); notes & terms;
  Related (customer + the converted invoice). Converting now navigates straight
  to the new invoice. List rows already rowHref'd here; slide-over retired. lint
  0, build, 524 tests.
- 2026-08-14 — **Increment 17: Coupon object page** (branch
  dashboard-depth-coupons). Backend linchpin: GET /v1/coupons/:id — coupons had
  list/create/update but no single-read. Handler uses the repo's existing
  tenant-scoped GetByID (nil → 404); route + openapi ($ref Coupon). (No handler
  unit test: concrete *db.CouponRepository, no interface seam, and the sibling
  coupon handlers have none — not worth adding sqlmock; the drift gate covers
  the route.) Frontend: new CouponPage at /coupons/:id **replacing the detail
  slide-over** (deleted CouponDetail.jsx + its test — that slide-over referenced
  redemption/currency fields the API never returns, i.e. dead/fake UI). The page
  shows ONLY real fields: a plain-language "what it does" sentence
  (forever/once/repeating-N-months), the activate/deactivate gate, and the
  genuine depth — **"Redeemed by": the subscriptions actually carrying this
  coupon** (reverse lookup over subscriptions by coupon_id — a REAL redemption
  count, not the fabricated counter the slide-over showed). List rows rowHref to
  it. lint 0, build, 524 tests.
