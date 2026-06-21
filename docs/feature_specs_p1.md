# Recurso: Feature Specifications (Priority 1)

This document contains the deep-dive technical and functional specifications for the "Growth & Payments" features. These enable actual revenue collection and customer communication.

## 1. Payments & Checkout

### 1.1 Hosted Checkout Page (HCP)
A secure, PCI-compliant page where customers enter payment details.
- **Customization**:
    - **Branding**: Logo, Brand Color, Font.
    - **Domain**: `pay.recurso.com` or CNAME `billing.startup.com`.
- **Functionality**:
    - **Dynamic Totals**: Reflected immediately when Addons/Coupons are applied toggled.
    - **Methods**: Show/Hide cards/UPI/Netbanking based on currency choice.

### 1.2 Payment Gateway Routing (The Orchestrator)
The abstraction layer that routes transactions to the correct financial provider.
- **Rules Engine**:
    - `IF currency == INR THEN use Razorpay/Cashfree`
    - `IF currency == USD THEN use Stripe/Adyen`
    - `IF method == UPI THEN use Razorpay`
- **Failover Logic**:
    - On `5xx` error from Primary Gateway, retry on Secondary Gateway (if card token is portable).

### 1.3 Offline Payments (B2B)
- **Virtual Accounts**:
    - Generate distinct Virtual Account Number (VAN) for each Customer-Currency pair (e.g., via Razorpay Smart Collect).
    - **Reconciliation**: Webook `payment.received` -> Find Customer by VAN -> Match Open Invoice -> Mark Paid.
- **Cash Recording**:
    - UI for agents to log cash payments -> Creates Ledger Entry (Debit Cash, Credit Customer Balance).

---

## 2. Notification System

### 2.1 Transactional Emails
Triggered by system events.
- **Events**:
    - `subscription_created`: Welcome email.
    - `payment_succeeded`: Receipt/Invoice attachment.
    - `payment_failed`: "Action Required" email with link to update payment method.
    - `card_expiring`: Warning sent 7 days before expiry.
- **Tech Stack**: SendGrid / AWS SES.

### 2.2 Webhooks (Outbound)
Notifying the merchant's backend.
- **Events**: `subscription.*`, `invoice.*`, `payment.*`.
- **Payload**: JSON envelope with `event_id`, `type`, `data` object.
- **Security**: `Wh-Signature` header (HMAC-SHA256 of payload + shared secret).

---

## 3. Dunning Management (Revenue Recovery)

### 3.1 The Schedule
Configurable rules for what happens when a payment fails.
- **Default Schedule**:
    - **T+0 (Fail)**: Email User + Retry in 24h.
    - **T+3**: Email User + Retry.
    - **T+7**: Email User (Urgent) + Retry.
    - **T+14**: Mark Subscription `Unpaid`/`Canceled` + Revoke Access hook.

### 3.2 Logic
- **Hard Declines** (Invalid Account, Fraud): Do not retry. Mark `Action Required`.
- **Soft Declines** (Insufficient Funds, Timeout): Retry according to schedule.

---

## 4. India Specifics (The "India Stack")

### 4.1 UPI AutoPay
- **Mandate Type**: `Recurring`.
- **Flow**:
    1.  **Intent** (Mobile): App switch to PhonePe/GPay.
    2.  **Collect** (Desktop): Notification to User's VPA.
- **Execution**:
    - **Pre-Debit Notification**: **MUST** be sent 24 hours before debit via Gateway API.
    - **Debit**: Triggered at T+24h.

### 4.2 GST Compliance
- **Logic**:
    - If `Customer.GSTIN` is present -> B2B Invoice.
    - If `Customer.GSTIN` is missing -> B2C Invoice.
- **E-Invoicing**:
    - For B2B > ₹5Cr turnover.
    - Async job -> Send Invoice JSON to IRP (Invoice Registration Portal) -> Receive `IRN` + `SignedQR`.
    - Embed `SignedQR` on the PDF.
