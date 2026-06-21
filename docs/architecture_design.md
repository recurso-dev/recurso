# Recurso: System Architecture Document

This document defines the high-level technical architecture of Recurso using the C4 Model.

## 1. System Context (Level 1)
**Scope**: Who uses Recurso and what external systems does it interact with?

- **Actors**:
    - **Merchant (Admin)**: Configures plans, views analytics.
    - **Subscriber**: Pays for subscriptions, manages payment methods.
    - **API Client**: The Merchant's application calling Recurso APIs.
- **System**:
    - **Recurso Platform**: The boundary.
- **External Systems**:
    - **Payment Gateways**: Razorpay, Stripe (Processing).
    - **Tax Providers**: AvaTax (Tax rules).
    - **Email Service**: SendGrid (Notifications).
    - **Banks**: For settlement & reconciliation.

---

## 2. Container Diagram (Level 2)
**Scope**: The deployable units (Applications/Services).

| Container | Technology | Responsibilities |
| :--- | :--- | :--- |
| **API Server** | Go (Gin) | Public REST API. Auth, Validation, Request Routing. |
| **Admin Dashboard** | React | Web UI for Merchants. |
| **Webhooks Worker** | Go | Reliable delivery of events to Merchant endpoints. |
| **Dunning Worker** | Go | Temporal Workflow. Retries failed payments. |
| **Proration Engine** | Go | Stateless lib. Calculates upgrades/downgrades. |
| **Smart Retry Service** | Python (FastAPI) | AI Model. Predicts best retry time. |
| **Primary DB** | PostgreSQL | Metadata (Plans, Users, Subs). |
| **Ledger DB** | TigerBeetle | Financial transactions (Double-Entry). |
| **Queue** | Kafka/Redpanda | Async events (`payment.failed`, `invoice.created`). |

---

## 3. Component Diagram (Level 3) - Core API
**Scope**: Components within the **API Server** container.

### 3.1 Catalog Component
- **Responsibility**: CRUD for Plans & Addons.
- **Collaborators**: `PostgresRepository`.

### 3.2 Subscription Engine
- **Responsibility**: State Machine management.
- **Logic**: 
    - `CreateSubscription()`: Calls `PaymentGateway` -> `Ledger` -> `Postgres`.
    - `CancelSubscription()`: Calculates refund -> `Ledger` -> `Postgres`.
- **Collaborators**: `LedgerClient`, `PaymentGatewayAdapter`.

### 3.3 Ledger Adapter
- **Responsibility**: Translates domain events into Double-Entry transfers.
- **Mapping**: `Subscription Created` -> `Dr Asset:Bank`, `Cr Liability:Deferred`.

---

## 4. Key Flows

### 4.1 Payment Success (Async)
1.  **Webhook** received from Stripe at `/webhooks/stripe`.
2.  **API Server** validates signature and pushes message to Kafka topic `payment.events`.
3.  **Billing Worker** consumes message.
4.  **Worker** calls `LedgerClient.RecordPayment()`.
5.  **Worker** updates Postgres `Invoice` status to `PAID`.
6.  **Worker** triggers `Webhooks Worker` to notify Merchant.

### 4.2 Proration Calculation
1.  **API Request**: POST `/subscriptions/{id}/change_plan`.
2.  **API Server** loads current subscription & new plan.
3.  **Proration Engine** runs `Calculate(old_price, new_price, days_used, days_total)`.
4.  **Response**: Returns `next_invoice_amount` preview to User.
