# Recurso: Master Product Roadmap (Product Note)

This document serves as the **comprehensive functional inventory** of Recurso. It details every feature we intend to build to achieve parity with incumbents (Chargebee) while delivering our unique startup-first, AI-native value proposition.

## 1. Core Billing Engine (The "Iron Core")
The fundamental machinery that models time and money.

| Feature | Description | Priority |
| :--- | :--- | :--- |
| **Product Catalog** | Create Plans (Gold, Silver), Addons (Priority Support), and Charges (Setup Fee). Support for multiple currencies per plan. | P0 |
| **Customer Management** | CRM-lite for subscribers. Store tax details (GSTIN), billing address, shipping address, and contacts. | P0 |
| **Subscription Lifecycle** | Finite state machine: `Active` -> `Past Due` -> `Canceled`. Handles logic for `Reactivation`, `Pause`, and `Resume`. | P0 |
| **Calculated Billing** | Logic to generate line items. Support for: <br>• **Flat Fee:** $10/mo.<br>• **Per Unit:** $5 per seat.<br>• **Tiered:** First 10 @ $5, Next 100 @ $4.<br>• **Volume:** All units @ $4 if > 10. | P0 |
| **Proration Engine** | Exact-time calculation for mid-cycle changes. <br>• **Upgrade:** Credit unused time, charge remaining time (Net debit).<br>• **Downgrade:** Credit unused, charge remaining (Net credit).<br>• **Switch:** Generate credit note. | P0 |
| **Metering** | Ingestion API for usage-based billing. Support for aggregation strategies: `Sum`, `Max`, `Last`, `Unique Count`. | P1 |

## 2. Payments & Checkout
The interface between Recurso and the financial world.

| Feature | Description | Priority |
| :--- | :--- | :--- |
| **Hosted Checkout** | No-code, secure payment page. Customizable branding (Logo, Colors). Support for adding/removing addons during checkout. | P1 |
| **Payment Gateway Routing**| Abstracted interface to switch providers. <br>• **India:** Razorpay, Cashfree (UPI AutoPay support).<br>• **Global:** Stripe, Adyen, PayPal.<br>• **Smart Routing:** Route based on currency or card type. | P1 |
| **Offline Payments** | Support for high-value B2B. Generate Virtual Account Numbers (VAN) for NEFT/IMPS. Auto-reconcile when bank webhook is received. | P1 |
| **Dunning Management** | Logic for failed payments. <br>• **Day 1:** Retry.<br>• **Day 3:** Email + Retry.<br>• **Day 7:** Mark Past Due.<br>Configurable retry schedule. | P1 |
| **Smart Retries (AI)** | **[Differentiator]** RL model to override static dunning rules. Predicts optimal time (e.g., Friday 2 PM) to retry a specific card based on BIN data. | P2 |

## 3. Invoice & Tax Management
Ensuring compliance and financial record-keeping.

| Feature | Description | Priority |
| :--- | :--- | :--- |
| **Invoice Generation** | PDF generation with customizable templates. HTML emails with "Pay Now" buttons. | P0 |
| **Credit & Debit Notes** | Handling refunds, chargebacks, and adjustments correctly. Never "delete" an invoice; always issue a Credit Note. | P0 |
| **Tax Engine (Global)** | Integration with tax providers (AvaTax) or manual tax tables (VAT, Sales Tax) based on customer region. | P2 |
| **India Compliance** | **[Differentiator]** Native GST handling.<br>• **GST Rules:** IGST vs CGST/SGST based on Place of Supply.<br>• **E-Invoicing:** Auto-generate IRN and QR Code via GSP integration. | P1 |

## 4. Customer Self-Service
Reducing support burden for startups.

| Feature | Description | Priority |
| :--- | :--- | :--- |
| **Customer Portal** | Embeddable portal for end-users to:<br>• Download Invoices.<br>• Update Payment Method (Card/UPI).<br>• Upgrade/Downgrade Subscription.<br>• Cancel/Pause. | P2 |
| **Email Notifications** | Transactional emails for every lifecycle event: `Welcome`, `Payment Success`, `Payment Fail`, `Card Expiring`, `Invoice Available`. | P1 |

## 5. Sales & Marketing Tools
Helping startups grow revenue.

| Feature | Description | Priority |
| :--- | :--- | :--- |
| **Coupons & Discounts** | Flexible engine.<br>• **Type:** Fixed Amount or Percentage.<br>• **Duration:** Once, Forever, Limited Period (3 months).<br>• **Scope:** Specific Plan, Addon, or Global. | P2 |
| **Gift Subscriptions** | Allow user A to buy a subscription for user B. Handling non-recurring "Access Passes". | P3 |
| **Quotes** | For B2B sales. Create a draft invoice (Quote) -> User accepts -> Converts to Subscription + Invoice. | P3 |
| **Referral System** | Two-sided incentives. "Give $10, Get $10". Auto-apply credits to ledger. | P3 |

## 6. Accounting & Reporting (The CFO Suite)
Visibility and financial correctness.

| Feature | Description | Priority |
| :--- | :--- | :--- |
| **Revenue Recognition** | **[Differentiator]** ASC-606 compliant revenue schedules. Track `Deferred Revenue` vs `Recognized Revenue` daily. | P3 |
| **Standard Reports** | MRR, Churn Rate, ARPU, LTV, Tax Liability, Failed Transaction Analysis. | P2 |
| **"Ask Data" (AI)** | **[Differentiator]** Natural Language Interface (Text-to-SQL) to query specific financial questions not covered by standard dashboards. | P2 |
| **ERP Sync** | Connectors for QuickBooks Online, Xero, NetSuite, Tally (India). | P3 |

## 7. Developer Experience
Making it easy for engineers to integrate.

| Feature | Description | Priority |
| :--- | :--- | :--- |
| **API & Webhooks** | REST/gRPC APIs. Reliable webhooks with retry logic and signature verification. | P0 |
| **SDKs** | Client-side (React, JS) and Server-side (Go, Node, Python) libraries. | P2 |
| **Testing Sandbox** | "Time Machine" feature to simulate passage of time (e.g., "Fast forward to renewal date") to test integration logic. | P1 |
