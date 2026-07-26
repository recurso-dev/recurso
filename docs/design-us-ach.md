# Design: ACH (US bank-account) payments via Stripe

**Status:** PROPOSED (US Market Readiness · Inc 3) — spec for review, no code yet
**Scope:** collect US invoices by bank debit (Stripe `us_bank_account`)
**Related:** [[design-us-invoice-presentation]], [[design-us-nexus]] · roadmap "ACH payments via Stripe"

## Objective

Let a US customer pay an invoice by **bank debit (ACH)** instead of card. ACH is
table-stakes for US B2B (cards cost 2.9%; ACH is ~$0.80 flat and preferred for
large invoices). The hard part is **not** the charge — it's that ACH is
**asynchronous and reversible**: funds settle over 1–5 business days, and a debit
can be **returned days after it settled** (insufficient funds, closed account).
Our payment model today is synchronous, so this needs a real settlement state
machine.

## Current state (grounded in code) — more is already wired than expected

A code map (Stripe adapter, webhook, ledger, mandate) found that the **online
one-time ACH charge is largely built**; the net-new work is the *bank-account
capture/authorization* and the *async state surfacing*, not the rail.

**Already there:**
- **`us_bank_account` is in the PaymentIntent path.** `CreateOrder`
  (`internal/adapter/gateway/stripe.go:84`) opens a Stripe PaymentIntent (not a
  Charge); `stripePaymentMethodTypes` (`stripe.go:56–82`) returns
  `[card, us_bank_account]` for USD, with a graceful **card-only fallback** if the
  account hasn't activated ACH (`stripe.go:48–54, 104–111`). stripe-go **v76**.
- **Async settlement is already modeled (via SEPA).** The invoice stays `open`;
  `CheckoutSuccess` (`internal/adapter/handler/checkout.go:361–390`) only settles
  on `status=="succeeded"` and otherwise returns an **ephemeral, non-persisted
  "processing"** with a comment that "ACH settles over days" (`checkout.go:365`).
  The `payment_intent.succeeded` webhook (`webhook_stripe.go:100`) →
  `MarkInvoicePaid` (`internal/service/subscription_payment.go:18`) does the
  atomic paid-claim.
- **The ledger already defers cash to settlement.** `MarkInvoicePaid` posts the
  **Cash leg only on `succeeded`** (`subscription_payment.go:67` →
  `ledger.RecordPaymentWithSettled`, `ledger.go:349`); the AR/Deferred leg posts
  at invoice creation (`RecordInvoice`, `ledger.go:266`). So "ledger-on-settle" —
  a decision below — is **already the behavior**, not a change (ADR-002).
- **Webhook idempotency** via `inbound_webhook_events` (migration 000086,
  `webhook_stripe.go:69/95`), signature verify fails-closed (`:35–39`).
- **A mandate lifecycle to mirror** (`internal/core/domain/mandate.go`
  `created → authorized → active → revoked`; the per-cycle at-most-once claim via
  `Invoice.MandateCycleKey` + partial-unique index migration 000090; the
  `chargeMandate` claim-first flow `internal/service/mandate.go:267`; the
  pre-debit notice scheduler `internal/scheduler/precharge.go`).

**The actual gaps (this is Inc 3):**
- **No bank-account capture / mandate save.** `CreateSetupIntent` (`stripe.go:197`)
  is **card-only** — the comment literally says "ACH-mandate save is a later
  enhancement" (`stripe.go:196`). No Financial-Connections flow. The Stripe
  mandate methods return `ErrNotSupported` (`stripe.go:411–421`) and the
  `SmartRouter` mandate/virtual-account methods are **Razorpay-hardcoded**
  (`smart_router.go:138–148`).
- **No persisted in-flight state.** The invoice enum (`invoice.go:11–20`:
  `draft, open, paid, void, uncollectible, past_due`) has no `processing`; the
  only "processing" today is the ephemeral checkout response, invisible to
  dunning and to the invoice list.
- **Missing webhooks.** `HandleStripe` (`webhook_stripe.go:74–87`) does **not**
  consume `payment_intent.processing`, `payment_intent.payment_failed`,
  `setup_intent.*`, or `mandate.updated`. It also can't yet reverse a **paid** ACH
  invoice on a late return.

**The gap in one sentence:** the *charge* is wired, but we can't capture/authorize
a bank account, we can't *see* an in-flight debit, and we can't handle the
days-later `failed`/return.

## The ACH lifecycle (what Stripe actually does)

1. **Collect + authorize the bank account.** A Stripe **SetupIntent** with
   `payment_method_types: [us_bank_account]` captures the account and the
   **mandate** (Nacha debit authorization — legally required, Stripe records the
   acceptance text/timestamp/IP). Verification is one of:
   - **Instant** via Financial Connections (buyer logs into their bank) — usable
     immediately; or
   - **Micro-deposits** — Stripe sends two small deposits; the buyer confirms the
     amounts **1–2 days later**, then the payment method is usable. (A multi-day
     onboarding step — see open questions.)
2. **Charge.** A **PaymentIntent** off the saved `us_bank_account` payment method
   moves to **`processing`** (not `succeeded`).
3. **Settle (days later).** Either `payment_intent.succeeded` (funds cleared) or
   `payment_intent.payment_failed` (e.g. `R01` insufficient funds at debit time).
4. **Late return (the nasty one).** Even after `succeeded`, the bank can **return**
   the debit for ~2 business days (`R01`, `R02` account closed, `R08` stop
   payment…). Stripe surfaces this as a **`charge.refunded`/dispute-style** event
   *after* we already marked the invoice paid. A "paid" invoice must be able to
   revert.

## Proposed design

### State model — a payment-attempt record, not an overloaded invoice status

Rather than add a `processing` value to the invoice enum (which would ripple
through every invoice consumer), introduce a **payment attempt** row that carries
the async lifecycle, and surface a lightweight flag on the invoice:

- `payment_attempts` (new): `id, invoice_id, tenant_id, gateway, method
  (us_bank_account), gateway_payment_intent_id, status (initiated | processing |
  succeeded | failed | returned), failure_code, amount, created_at, settled_at`.
- The invoice stays **`open`** while an attempt is `processing`, with a derived
  **"payment processing"** indicator in the API/dashboard (so dunning and the
  customer both see "don't re-charge, it's in flight").
- On `succeeded` → invoice `paid` + **ledger post** (see timing) + `settled_at`.
- On `failed` → invoice back to `open`/`past_due` with the ACH failure code
  surfaced; dunning resumes.
- On **late return** → reverse: a `refund`-class ledger reversal (mirrors the
  existing gateway-refund path) moves the invoice `paid → open/past_due` and
  records the R-code. This reuses the refund plumbing already in the webhook.

### Mandate / authorization

Reuse the mandate concept: the SetupIntent's Nacha acceptance is stored like a
mandate (`created → active`), and each debit is claimed at-most-once per invoice
via the existing `MandateCycleKey` idempotency (so a webhook re-fire or retry
can't double-debit). Revocation maps to `RevokeMandate`.

### Ledger timing — already correct, keep it

The cash leg **already** posts only on `payment_intent.succeeded`
(`subscription_payment.go:67`), and the AR/Deferred leg at invoice creation. So no
change: an ACH invoice sits DR AR / CR Deferred while `processing` and gains its
Cash leg on settlement. A late return posts a **reversal** leg (Refunds-vs-Cash,
the existing ADR-002 pattern the `charge.refunded` handler already uses). Never
post cash at initiation.

### Verification — instant only (Financial Connections) for 3a (decided)

Inc 3a uses **Stripe Financial Connections** (the buyer logs into their bank; the
account is verified and the Nacha mandate captured in one flow, usable
immediately). This covers most major US banks and avoids the multi-day
micro-deposit wait. **Micro-deposits are deferred** to a follow-up (only if a
design partner hits an unsupported bank) — that path is what adds the
`setup_intent`/micro-deposit webhooks and a multi-day onboarding state.

### Webhook events to add (`webhook_stripe.go`)

| Event | Action |
|---|---|
| `payment_intent.processing` | create/advance the attempt → `processing`; invoice shows "payment processing" |
| `payment_intent.succeeded` | attempt → `succeeded`; invoice `paid`; **ledger post** (extend the existing handler to be ACH-aware) |
| `payment_intent.payment_failed` | attempt → `failed` (+ code); invoice reopens; dunning resumes |
| `charge.refunded` w/ ACH return reason | attempt → `returned`; **reverse** the paid invoice + ledger |
| `setup_intent.succeeded` / `payment_method.updated` | micro-deposit verification completed → payment method usable |
| `mandate.updated` | authorization revoked/expired → block further debits |

All handled idempotently (the webhook layer already dedupes; keep per-event
idempotency keyed on the Stripe event id + attempt).

## Test-mode plan (build without live keys)

Stripe **test mode** fully simulates `us_bank_account`: test bank accounts +
special test amounts/payment-methods that drive `processing → succeeded`,
`processing → failed`, and **post-settlement returns** (R-codes). So Inc 3 can be
**built and verified end-to-end against test keys** — the compose E2E can exercise
the whole state machine. **Live go-live is founder-gated**: Stripe ACH must be
activated on the account and the Nacha authorization language reviewed.

## Phasing

- **Inc 3a — bank-account capture (Financial Connections).** Extend
  `CreateSetupIntent`/`FinalizeSetupIntent` (`stripe.go:197/219`) to
  `us_bank_account` + Financial Connections, storing the verified payment method
  (`default_payment_method`/`pm_gateway_connection_id`, migrations 000074/000118)
  and the Nacha mandate acceptance. Portal + dashboard "add bank account". No new
  charge path — the existing `us_bank_account` PaymentIntent already charges.
  *Fully test-mode verifiable.*
- **Inc 3b — surface the async state.** A persisted **`payment_attempts`** row
  (`initiated → processing → succeeded | failed | returned`) + an invoice
  "payment processing" indicator; consume **`payment_intent.processing`** and
  **`payment_intent.payment_failed`** in `HandleStripe`; **dunning must skip** an
  invoice with a live processing attempt. Ledger unchanged (already on-settle).
  *The core correctness work.*
- **Inc 3c — late returns + reconciliation.** Handle a post-settlement return
  (reverse a paid invoice via the existing refund/ledger path, record the R-code,
  reopen + resume dunning); a reconciler sweeps attempts stuck `processing` past
  the ACH window and reconciles them against Stripe.

**Founder-gated (parallel, not blocking the test-mode build):** activate ACH on
the live Stripe account and review the Nacha authorization copy.

## Boundaries

- **Always:** post the ledger only on settlement; claim each debit at-most-once
  (`MandateCycleKey`); handle every webhook idempotently.
- **Ask first / founder-gated:** live Stripe ACH activation; Nacha authorization
  copy (legal); enabling ACH as a *default* for US tenants.
- **Never:** mark an invoice paid at initiation; auto-retry an ACH that failed
  `R02`/`R08` (account closed / stop payment) without human intent.

## Risks

- **"Paid then returned"** is the hardest correctness case — a settled invoice
  reverts days later. The reversal + dunning-resume path must be tested against
  the invariant harness (the ledger must net to zero across post + reversal).
- **Dunning double-charge:** the dunning engine must treat an invoice with a
  `processing` attempt as "do not touch."
- **Micro-deposit onboarding latency** (days) — a poor first-payment UX; instant
  verification (Financial Connections) is strongly preferred where available.
- **Stuck `processing`:** ACH can silently hang; a reconciler must reconcile
  attempts against Stripe after N days.

## Decisions & open questions

- ✅ **Verification: instant-only (Financial Connections) for 3a**, micro-deposits
  deferred. *(decided)*
- ✅ **Ledger-on-settle** — already the behavior; no change. *(confirmed by code)*
- ⬜ **State model:** persisted `payment_attempts` row (recommended — keeps the
  invoice enum untouched and models the attempt lifecycle cleanly) **vs.** adding
  a `processing` value to `InvoiceStatus` (+ the status CHECK-constraint
  migration). Confirm the table approach.
- ⬜ **Test-mode-first sequencing:** build + verify 3a/3b/3c end-to-end against
  Stripe **test** keys (test `us_bank_account` + Financial Connections sandbox
  drive the whole lifecycle incl. failures/returns); live go-live gated on your
  Stripe ACH activation + Nacha copy. Confirm.
- ⬜ **Scope of 3a payment surface:** ACH from the **customer portal**
  (`portal_api.go` SetupIntent flow) is the clear first target. Also expose
  "charge saved bank account" for **dunning/renewal** off-session in 3a, or defer
  to 3b? (Off-session ACH has its own mandate/notice nuances.)
