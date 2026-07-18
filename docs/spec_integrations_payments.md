# Spec: Payment Integrations — GoCardless + Adyen (Track D1)

> **Status: APPROVED under the Lago-parity program (spec_lago_parity.md
> Track D, founder directive 2026-07-18: "support the same integrations as
> Lago"). Ships as EXPERIMENTAL until sandbox-verified (founder-gated:
> needs sandbox credentials).**

## Objective

Close Lago's payments-integration edge: Lago ships Stripe, Adyen,
GoCardless (+ MoneyHash). Recurso has Stripe + Razorpay. This spec adds:

- **GoCardless** — bank-debit rails (SEPA/BACS/ACH): mandate-first, the
  European/UK analog of our UPI AutoPay flows.
- **Adyen** — global card/wallet processing via Checkout Sessions.

MoneyHash (aggregator) is deferred: no customer signal.

## Design

Both implement `port.PaymentGateway` on the applicable subset and return
explicit "not supported by <gateway>" errors elsewhere — exactly how
Stripe handles UPI-mandate methods today. No new dependencies: plain
`net/http` + `encoding/json` like the existing adapters.

### GoCardless (`internal/adapter/gateway/gocardless.go`)

Bank debit is mandate-first, so the UPI-mandate surface maps naturally:

| Port method | GoCardless API |
| --- | --- |
| CreateMandate | `POST /billing_requests` (mandate_request) + `POST /billing_request_flows` → AuthURL for customer authorisation |
| ExecuteMandateDebit | `POST /payments` `{links: {mandate}}` with `Idempotency-Key` header |
| RevokeMandate | `POST /mandates/{id}/actions/cancel` (already-cancelled = success) |
| Refund | `POST /refunds` |
| everything else | not-supported error |

Auth: `Authorization: Bearer <GOCARDLESS_ACCESS_TOKEN>`,
`GoCardless-Version: 2015-07-06`. Host by `GOCARDLESS_ENV`
(`sandbox` → api-sandbox.gocardless.com, else api.gocardless.com).

### Adyen (`internal/adapter/gateway/adyen.go`)

| Port method | Adyen API (Checkout v71) |
| --- | --- |
| CreateOrder | `POST /sessions` → PaymentOrder{ID: session id, ClientSecret: sessionData} |
| RetryPayment / ChargeSavedPaymentMethod | `POST /payments` with `storedPaymentMethodId`, `shopperReference`, `shopperInteraction: ContAuth`, `recurringProcessingModel: UnscheduledCardOnFile` |
| Refund | `POST /payments/{pspReference}/refunds` with `Idempotency-Key` |
| CancelSubscription | no-op (Recurso owns the cycle; nothing gateway-side) |
| mandates / VAs / VerifyPayment | not-supported error |

Auth: `x-api-key: <ADYEN_API_KEY>`; `merchantAccount` from
`ADYEN_MERCHANT_ACCOUNT`; host by `ADYEN_ENV` (`test` →
checkout-test.adyen.com, live requires `ADYEN_LIVE_URL_PREFIX`).

### Routing (`smart_router.go`)

`SmartRouter` gains an `Extra` registry plus env-driven currency overrides:

```
GATEWAY_CURRENCY_OVERRIDES="EUR=gocardless,GBP=gocardless,SGD=adyen"
```

`gatewayFor` consults overrides first, then the existing INR→Razorpay /
default→Stripe rule, so nothing changes for existing deployments. Refund
keeps prefix routing (GoCardless payment ids start `PM`), falling back to
currency routing (Adyen pspReferences carry no prefix).

### Residency

Both adapters are payment gateways — operator-configured egress, the same
class as Stripe/Razorpay — so they are NOT residency-blocked (matching the
approved residency policy: gateways/GSP/SMTP/webhooks remain).

## Testing

httptest-backed unit suites per adapter asserting: auth headers, the
idempotency key, exact request JSON shape, response mapping, error
mapping (declines → PaymentResult{Success:false, ErrorCode}), and the
already-cancelled-mandate = success rule. **No live/sandbox calls in CI.**

## Founder-gated

Sandbox verification (GoCardless sandbox org + Adyen test merchant
account), then dropping the EXPERIMENTAL flag. Until then main.go logs
the experimental status at boot when either is configured.
