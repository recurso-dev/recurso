# Implementation Plan - Execution P1 (Growth & Payments)

## Goal
Enable actual revenue collection via Hosted Checkout and communicate with users via Email.

## User Review Required
> [!NOTE]
> We will use **Razorpay** as the primary gateway for India (INR) and a **Mock Gateway** for testing.
> For Email, we will start with a **Console Logger** (Mock) to avoid needing API keys immediately, but structure it for SendGrid/SES.

## Proposed Changes

### 1. Hosted Checkout (The "Pay Page")
#### [NEW] [internal/adapter/handler/checkout.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/handler/checkout.go)
- `GET /checkout/:invoice_id`: Renders an HTML page.
- Logic:
    1.  Fetch Invoice.
    2.  If Paid, show Success.
    3.  If Open, show Razorpay/Stripe JS button.

#### [NEW] [internal/adapter/templates/checkout.html](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/templates/checkout.html)
- Simple HTML template.

### 2. Payment Gateway Adapter
#### [NEW] [internal/core/port/payment_gateway.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/core/port/payment_gateway.go)
- Interface `PaymentGateway`:
    - `CreateOrder(amount, currency)`
    - `VerifyPayment(signature)`

#### [NEW] [internal/adapter/gateway/razorpay.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/gateway/razorpay.go)
- Implementation using `razorpay-go` SDK.

### 3. Incoming Webhooks (Payment Success)
#### [NEW] [internal/adapter/handler/webhook.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/handler/webhook.go)
- `POST /webhooks/razorpay`
- Logic:
    1.  Verify Signature.
    2.  Extract `invoice_id`.
    3.  Call `SubscriptionService.MarkInvoicePaid()`.

### 4. Notification Service
#### [NEW] [internal/core/port/notifier.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/core/port/notifier.go)
- Interface `Notifier`:
    - `SendEmail(to, subject, body)`

#### [NEW] [internal/adapter/notification/email.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/notification/email.go)
- Mock implementation logging to Stdout.

## Verification Plan
1.  **Manual Test**: Open `http://localhost:8080/checkout/INV-123` in browser.
2.  **Flow**: Click "Pay" -> Mock Success -> Check DB Invoice Status = `paid`.
