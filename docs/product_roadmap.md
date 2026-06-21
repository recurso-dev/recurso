# Recurso: Product Roadmap & Chargebee Parity Strategy

This roadmap outlines the strategic execution to build **Recurso**, aiming for functional parity with market leaders like Chargebee while fundamentally improving the core architecture (TigerBeetle Ledger) and value proposition (AI-Native, India-Deep/Global-Wide).

## Phase 1: The Iron Core (Foundation & Correctness)
**Goal:** Build a billing engine more robust than the incumbents by using Double-Entry accounting from Day 1.

| Feature Area | Recurso Implementation | Chargebee Parity Check |
| :--- | :--- | :--- |
| **Ledger System** | **TigerBeetle (Immutable Double-Entry)**. Real-time auditability. | ✅ Superior (CB uses SQL state) |
| **Product Catalog** | Plans, Addons, Charges. Multi-currency support. | ✅ Parity |
| **Subscription Logic** | Exact-time proration (to the second), Upgrades/Downgrades, Grandfathering. | ✅ Parity |
| **Metering** | High-throughput usage ingestion (ClickHouse). | ✅ Parity |
| **Taxation** | Basic Tax engine (GST native, Flat rates for Global). | ⚠️ Basic (CB has AvaTax) |

## Phase 2: The India Stack & Global Payments (Differentiation)
**Goal:** Solve the specific pain points of the Indian market that Chargebee ignores, while enabling global collection.

| Feature Area | Recurso Implementation | Chargebee Parity Check |
| :--- | :--- | :--- |
| **India Payments** | **UPI AutoPay (Native)**, e-Mandates, pre-debit notifications (T-24h). | ✅ Superior |
| **Global Payments** | Stripe, Adyen, Braintree integrations. | ✅ Parity |
| **Offline Payments** | Virtual Accounts (NEFT/IMPS reconciliation), Cash logging. | ✅ Superior |
| **Compliance** | **Data Residency (India/Global Split)**, Tokenization (CoF). | ✅ Superior |

## Phase 3: "Everything Chargebee" (Feature Parity Expansion)
**Goal:** Close the gap on standard features expected by mature SaaS teams.

### 3.1 Marketing & Sales Enablement
- [ ] **Coupons & Promotions:** Fixed amount, percentage, "forever" vs "once".
- [ ] **Gift Subscriptions:** Buy for a friend.
- [ ] **Quotes:** Generate PDF quotes that convert to Invoices upon acceptance.
- [ ] **Referral System:** Credits for referring new users.

### 3.2 Advanced Billing Logic
- [ ] **Calendar Billing:** Align all customers to bill on the 1st of the month.
- [ ] **Unbilled Charges:** Accumulate charges to bill at end of period.
- [ ] **Advance Invoicing:** Bill for N months ahead.
- [ ] **Net D Terms:** Net-15, Net-30 payment terms for enterprise.

### 3.3 Customer Experience
- [ ] **Hosted Checkout Pages:** No-code, branded payment pages.
- [ ] **Customer Self-Service Portal:** Allow users to download invoices, update cards, pause subscriptions.
- [ ] **Email Notifications:** Transactional emails for "Payment Succeeded", "Payment Failed", "Card Expiring".

## Phase 4: The AI Advantage (Innovation)
**Goal:** Leapfrog incumbents using GenAI and Reinforcement Learning.

| Feature Area | Recurso Implementation | Competitive Advantage |
| :--- | :--- | :--- |
| **Smart Dunning** | **RL (Bandits)** to optimize retry timing based on bank/error code. | 🚀 Dynamic vs Static Rules |
| **Analytics** | **Text-to-SQL (GenAI)**. "What is my Churn Rate?". | 🚀 Conversational vs Dashboards |
| **Churn Prevention** | ML-based "Propensity to Churn" scoring triggered workflows. | 🚀 Predictive vs Reactive |

## Phase 5: Finance & Ops (The CFO Suite)
**Goal:** Make Recurso the favorite tool of the Finance team.

- [ ] **Revenue Recognition:** ASC 606 / IFRS 15 compliant reporting.
- [ ] **Third-party Accounting Sync:** QuickBooks Online, Xero, NetSuite integrations.
- [ ] **GST E-Invoicing:** Automated IRN generation and QR code embedding (India).
- [ ] **Multi-Entity:** Manage multiple business units under one login.

## Execution Timeline (Visual)

- **Months 1-2:** Phase 1 (Core Ledger, Subscriptions)
- **Months 3-4:** Phase 2 (Payments, India Stack)
- **Months 5-7:** Phase 3 (Parity: Coupons, Portal, Quotes)
- **Months 8-9:** Phase 4 (AI Layer)
- **Months 10+:** Phase 5 (ERP Integrations, RevRec)
