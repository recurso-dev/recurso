- [x] Phase: Comprehensive Documentation & Planning <!-- id: 12 -->
    - [x] Create Business Strategy Document (Vision, Market, Strategy) <!-- id: 13 -->
    - [x] Create Master Product Roadmap (All features, detailed descriptions) <!-- id: 14 -->
    - [x] Create Feature Specifications (Deep dive per feature) <!-- id: 15 -->
        - [x] Specs: Priority 0 (Core Billing, Invoicing, API) <!-- id: 16 -->
        - [x] Specs: Priority 1 (Payments, Notifications, Checkout) <!-- id: 17 -->
        - [x] Specs: Priority 2 & 3 (AI, Self-Service, Advanced) <!-- id: 18 -->
    - [x] Create Database Design Document (Schema & Ledger) <!-- id: 19 -->
    - [x] Create API Contract (OpenAPI/Swagger) <!-- id: 20 -->
    - [x] Create System Architecture Document (C4 Model) <!-- id: 21 -->
    - [x] Create User Stories & Acceptance Criteria <!-- id: 22 -->
- [x] Phase: Execution - Priority 0 (Iron Core) <!-- id: 23 -->
    - [x] Create Implementation Plan P0 <!-- id: 24 -->
    - [x] Setup Project Structure & API Layer <!-- id: 25 -->
    - [x] Implement Product Catalog (Plans) <!-- id: 26 -->
    - [x] Implement Customer Management <!-- id: 27 -->
    - [x] Implement Subscription Lifecycle <!-- id: 28 -->

- [x] Phase: Execution - Priority 1 (Growth & Payments) <!-- id: 29 -->
    - [x] Create Implementation Plan P1 <!-- id: 30 -->
    - [x] Implement Hosted Checkout (HTML/JS) <!-- id: 31 -->
    - [x] Implement Payment Gateway Adapter (Razorpay) <!-- id: 32 -->
    - [x] Implement Webhook Handler (Incoming Payments) <!-- id: 33 -->
    - [x] Implement Notification Service (Email) <!-- id: 34 -->

- [x] Phase: Execution - Priority 2 (AI Intelligence & Self-Service) <!-- id: 35 -->
    - [x] Create Implementation Plan P2 <!-- id: 36 -->
    - [x] Implement Smart Retry Logic (Worker) <!-- id: 37 -->
    - [x] Implement Customer Portal (Dashboard) <!-- id: 38 -->

- [x] Phase: Execution - Priority 3 (Scale & Analytics) <!-- id: 39 -->
    - [x] Create Implementation Plan P3 <!-- id: 40 -->
    - [x] Implement Usage Metering (Event Ingestion) <!-- id: 41 -->
    - [x] Implement Analytics Service (MRR/Revenue) <!-- id: 42 -->

- [x] Phase: Post-MVP (Security & Infrastructure) <!-- id: 43 -->
    - [x] Create Implementation Plan P4 <!-- id: 44 -->
    - [x] Implement JWT Authentication <!-- id: 45 -->
    - [x] Dockerize Application (Dockerfile & Compose) <!-- id: 46 -->

- [x] Phase: Option 1 (TigerBeetle Ledger) <!-- id: 47 -->
    - [x] Create Implementation Plan P5 <!-- id: 48 -->
    - [x] Setup TigerBeetle in Docker Compose <!-- id: 49 -->
    - [x] Implement Ledger Adapter (Client & Models) <!-- id: 50 -->
    - [x] Integrate Ledger with Invoice/Payment Flow <!-- id: 51 -->

- [x] Phase: Option 2 (Frontend Admin Dashboard) <!-- id: 52 -->
    - [x] Create Implementation Plan P6 <!-- id: 53 -->
    - [x] Initialize React App (Vite) <!-- id: 54 -->
    - [x] Setup Design System (CSS Variables & Base Styles) <!-- id: 55 -->
    - [x] Implement Dashboard Home (MRR & Stats) <!-- id: 56 -->
    - [x] Implement Customer Management (List & Create) <!-- id: 57 -->
    - [x] Implement Plan Management (List & Create) <!-- id: 58 -->

- [x] Phase: Option 3 (Advanced Billing - Coupons) <!-- id: 59 -->
    - [x] Create Implementation Plan P7 <!-- id: 60 -->
    - [x] DB Migration: Coupons & Redemptions <!-- id: 61 -->
    - [x] Implement Coupon Domain & Repository <!-- id: 62 -->
    - [x] Update Subscription Logic to Accept Coupons <!-- id: 63 -->
    - [x] Update Invoice Calculation for Discounts <!-- id: 64 -->
    - [x] Verify Coupon Flow <!-- id: 65 -->

- [x] Phase: Startup Scale (Multi-Tenancy) <!-- id: 66 -->
    - [x] Create Implementation Plan P8 <!-- id: 67 -->
    - [x] DB Migration: Tenants & API Keys <!-- id: 68 -->
    - [x] Implement Tenant Service & Registration <!-- id: 69 -->
    - [x] Upgrade Auth Middleware (DB-based Keys) <!-- id: 70 -->
    - [x] Refactor Repositories for Tenant Isolation <!-- id: 71 -->

- [x] Phase: Developer Experience (SDKs) <!-- id: 72 -->
    - [x] Create Implementation Plan P9 <!-- id: 73 -->
    - [x] Create/Update OpenAPI v3 Spec <!-- id: 74 -->
    - [x] Generate Node.js SDK <!-- id: 75 -->
    - [x] Create 'Quickstart' Example for Developers <!-- id: 76 -->

- [x] Phase: Premium UI/UX Overhaul (Tailwind) <!-- id: 77 -->
    - [x] Install & Configure Tailwind CSS <!-- id: 78 -->
    - [x] Design System Setup (Components/Layout) <!-- id: 79 -->
    - [x] Rebuild Dashboard Home <!-- id: 80 -->
    - [x] Rebuild Customers & Plans Pages <!-- id: 81 -->
    - [x] Implement Developer Settings (API Keys) <!-- id: 82 -->
- [x] Phase 11: Completing UI & Updating Deps <!-- id: 83 -->
    - [x] Update Dependencies (Vite, PostCSS) <!-- id: 84 -->
    - [x] Implement Subscriptions Page <!-- id: 85 -->
    - [x] Implement Invoices Page <!-- id: 86 -->
    - [x] Implement Developers & Settings Pages <!-- id: 87 -->

- [x] Phase 12: Frontend API Integration <!-- id: 88 -->
    - [x] Setup Axios Client & Interceptors <!-- id: 89 -->
    - [x] Integrate Customers Page <!-- id: 90 -->
    - [x] Integrate Plans Page <!-- id: 91 -->
    - [x] Integrate Subscriptions Page <!-- id: 92 -->
    - [x] Integrate Invoices Page <!-- id: 93 -->
    - [x] Integrate Usage Dashboard UI with Backend API
    - [x] Update `Usage.jsx` to fetch stats from `/analytics/usage`

- [x] Phase 18: UI Polish - Create Screens
    - [x] Refactor `CreateCustomer.jsx` (Stitch Design)
    - [x] Refactor `CreatePlan.jsx` (Stitch Design)
    - [x] Refactor `CreateSubscription.jsx` (Stitch Design)
    - [x] Refactor `CreateCoupon.jsx` (Stitch Design)
    - [x] Update `Products.jsx` navigations (API Keys) <!-- id: 95 -->

- [x] Phase 13: Functional Create Actions & Settings <!-- id: 96 -->
    - [x] Backend: Add API Key Management (List/Create) <!-- id: 97 -->
    - [x] Frontend: Integrate Developers Page (API Keys) <!-- id: 98 -->
    - [x] Frontend: Implement Create Plan Modal <!-- id: 99 -->
    - [x] Frontend: Implement Create Customer Modal <!-- id: 100 -->
    - [x] Frontend: Implement Create Subscription Modal <!-- id: 101 -->

- [x] Phase 15: Advanced Billing Features <!-- id: 102 -->
    - [x] Create Implementation Plan P15 <!-- id: 103 -->
    - [x] DB Migration: Advanced Billing Fields <!-- id: 104 -->
    - [x] Implement Calendar Billing (Align to 1st) <!-- id: 105 -->
    - [x] Implement Unbilled Charges <!-- id: 106 -->
    - [x] Implement Advance Invoicing <!-- id: 107 -->
    - [x] Implement Net-D Payment Terms <!-- id: 108 -->

- [x] **Phase 16: UI Polish (Update 2)**
  - [x] Dashboard: Implement 4-card grid and new "Recent Activity" table.
  - [x] Customers: Implement new Data Grid with filters and badges.
  - [x] Plans: Implement new Data Grid and Slide-Over Detail View.
  - [x] Subscriptions: Implement new Data Grid and Slide-Over Detail View.
  - [x] Coupons: Implement Management Screen and Detail View.
  - [x] Products: Implement Product Catalog Screen.
  - [x] Usage: Implement Usage Metering Dashboard. <!-- id: 115 -->

- [x] **Phase 17: Backend Integration**
  - [x] Create Implementation Plan P17
  - [x] Integrate Coupons UI (List & Create)
  - [x] Integrate Products/Plans UI
  - [x] Backend: Implement Usage Aggregation API
  - [x] Integrate Usage Dashboard UI

- [x] **Phase 19: Backend Extension (Customer Fields)**
  - [x] DB Migration: Add Address, Phone, TaxID to Customers
  - [x] Update Domain & DTOs
  - [x] Update Repository & Service
  - [x] Verify End-to-End Customer Creation

- [x] **Phase 20: Comprehensive Dashboard Interactions**
    - [x] Backend: Add Filter/Search to List APIs (Customers, Plans, Subscriptions)
    - [x] Backend: Join Subscriptions with Customers for Search
    - [x] Frontend: Update API client for params
    - [x] Frontend: Implement Search/Filter UI in Customers.jsx
    - [x] Frontend: Implement Search/Filter UI in Products.jsx
    - [x] Frontend: Implement Search/Filter UI in Subscriptions.jsx
    - [x] Verification: Test End-to-End: Wire up Subscriptions Search/Filter UI

- [ ] **Phase 23: Credit Notes**
    - [x] Backend: Create `CreditNote` domain and DB migration
    - [x] Backend: Implement `CreditNoteRepository` and `CreditNoteService`
    - [x] Backend: Implement `CreditNoteHandler` (Create/List)
    - [x] Frontend: Implement `CreditNotes.jsx` (List View)
    - [x] Frontend: Implement `CreateCreditNote.jsx` (Form)
    - [x] Verification: Test End-to-End

- [x] **Phase 21: Authentication UI**
    - [x] Frontend: Create `Register.jsx` (Tenant Onboarding) matching design
    - [x] Frontend: Update `Login.jsx` matching design
    - [x] Frontend: Integrate Register with `POST /auth/register`
    - [x] Verify Register -> Login flow

- [ ] **Phase 22: Financial Ledger**
    - [x] Backend: Check if `LedgerHandler` exists or needs creation (Expose `/v1/ledger` endpoints)
    - [x] Frontend: Implement `Ledger.jsx` matching `financial_ledger_overview` design
    - [x] Frontend: Add 'Financials' item to navigation
    - [x] Verification: Test End-to-End

- [x] **Phase 24: Webhooks & Events**
    - [x] Backend: Create DB migrations (`webhook_endpoints`, `events`, `event_deliveries`)
    - [x] Backend: Implement domain, repository, service, handler
    - [x] Backend: Add routes (`POST/GET/DELETE /webhooks`, `GET /events`)
    - [x] Frontend: Update `api.js` with webhook/event endpoints
    - [x] Frontend: Update `Developers.jsx` with real API integration
    - [x] Verification: Build checks passed
