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
  visual QA against a seeded stack still to run. Next: Subscription (reuse both
  primitives; its `/preview-change` proration is the financial-consequence layer).
