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

## Phase 3: "Everything Chargebee" (Feature Parity Expansion) — ✅ shipped
**Goal:** Close the gap on standard features expected by mature SaaS teams.

### 3.1 Marketing & Sales Enablement
- [x] **Coupons & Promotions:** Fixed amount, percentage, "forever" vs "once". (Coupons page)
- [x] **Gift Subscriptions:** Buy for a friend. (Gifts page, incl. cancel #218)
- [x] **Quotes:** Generate PDF quotes that convert to Invoices upon acceptance. (Quotes page)
- [x] **Referral System:** Credits for referring new users. (Referrals page)

### 3.2 Advanced Billing Logic
- [x] **Calendar Billing:** Align all customers to bill on the 1st of the month. (`billing_anchor_type=first_of_month`, month-end anchor restoration)
- [x] **Unbilled Charges:** Accumulate charges to bill at end of period. (`unbilled_charge` domain + advanced-billing service)
- [x] **Advance Invoicing:** Bill for N months ahead. (`pay_in_advance` + progressive-billing scheduler)
- [x] **Net D Terms:** Net-15, Net-30 payment terms for enterprise. (`payment_terms` on subscription + invoice)

### 3.3 Customer Experience
- [x] **Hosted Checkout Pages:** No-code, branded payment pages. (`/checkout/:id` SPA + pay-link)
- [x] **Customer Self-Service Portal:** magic-link login, download invoices, update cards, autopay mandates, pause.
- [x] **Email Notifications:** Transactional emails (Brevo) for payment succeeded/failed, portal magic-links, dunning.

## Phase 4: The AI Advantage (Innovation) — ✅ shipped
**Goal:** Leapfrog incumbents using GenAI and Reinforcement Learning.

| Feature Area | Recurso Implementation | Status |
| :--- | :--- | :--- |
| **Smart Dunning** | **RL (Bandits)** to optimize retry timing based on bank/error code, plus a Collections Intelligence operator layer (worklist, analytics, manual controls). | ✅ shipped (Dunning page) |
| **Analytics** | **Text-to-SQL (GenAI)**. "What is my Churn Rate?". | ✅ shipped (Ask AI page) |
| **Churn Prevention** | ML-based "Propensity to Churn" scoring triggered workflows. | ✅ shipped (Churn Risk page) |

## Phase 5: Finance & Ops (The CFO Suite) — ✅ shipped
**Goal:** Make Recurso the favorite tool of the Finance team.

- [x] **Revenue Recognition:** ASC 606 / IFRS 15 compliant reporting. (Revenue Waterfall, deferred rollforward)
- [x] **Third-party Accounting Sync:** QuickBooks Online, Xero, NetSuite, Tally integrations. (Xero + HubSpot live-verified; QBO adapter built, live OAuth verification pending founder creds — see docs/backlog.md P1)
- [x] **GST E-Invoicing:** Automated IRN generation and QR code embedding (India). (per-entity GSTR-1/3B, GSP connect)
- [x] **Multi-Entity:** Manage multiple business units under one login. (per-entity ledger, gapless invoice series, per-entity tax identity, consolidated reporting)

## Beyond the original roadmap — also shipped

These weren't in the initial plan but are live:

- **US ACH bank-debit** (capture → settle → dunning → late-return) and **GoCardless SEPA** (BYO mandates, webhooks, settlement, chargeback reversal — webhook registration pending founder infra).
- **BYO payment gateways** — per-tenant Stripe/Razorpay/GoCardless credentials.
- **EU e-invoicing** — EN 16931 / UBL export (real Peppol AP pending founder creds).
- **Collections Intelligence** — operator worklist + analytics over the dunning engine.
- **MCP server** — agent-operable billing with RBAC.
- **Three published SDKs** — Go, Node, Python (synced to the v1 API).

## Execution status

All five phases are shipped. Remaining work is **not new features** — it is
founder-blocked verification/infra (QuickBooks live OAuth, GoCardless webhook
registration, Peppol AP creds, telemetry deploy — see docs/backlog.md) plus a
short engineering-hygiene list (React 19 upgrade, pagination consistency, test-mock
refactor).
