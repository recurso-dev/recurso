# Recurso - Open Source Billing Engine (MVP)

Recurso is a high-performance, developer-first billing engine built with **Go** and **PostgreSQL**.

## Features

### Priority 0: The Iron Core (Billing Engine)
- **Product Catalog**: Create Plans and Prices (Monthly/Yearly/One-time).
- **Customers**: Manage customer profiles with GSTIN/Tax ID support.
- **Subscriptions**: Lifecycle management (Active, Past Due, Canceled, Trialing).
- **Invoicing**: Automatic invoice generation with tax calculation (GST/VAT).
- **Credit Notes**: Full credit note lifecycle (Issued, Allocated, Refunded) for adjustments.

### Priority 1: Payments & Checkout
- **Hosted Checkout**: Ready-to-use HTML payment page.
- **Global Payments**: Smart routing:
    - **USD/EUR/GBP** -> Stripe (Mock/Simulated)
    - **INR** -> Razorpay
- **Webhooks**: Handling `payment.captured` events to mark invoices as PAID.
- **Notifications**: Email alerts on payment success.

### Priority 2: Intelligence & Self-Service
- **Smart Retries**: Background worker analyzes failed invoices and schedules retries using exponential backoff.
- **Customer Portal**: Self-service dashboard for customers to view billing history and pay invoices.

### Priority 3: Scale & Analytics
- **Usage Metering**: Ingest usage events (e.g., API calls, storage) for metered billing.
- **Analytics**: Real-time MRR (Monthly Recurring Revenue) calculation.
- **Financial Ledger**: Double-entry accounting system for audit-ready financial tracking.

### Compliance & Localization
- **India Stack**: Native GST calculation, Place of Supply rules, and HSN codes.
- **E-Invoicing**: Data structure readiness for IRP (Invoice Registration Portal) integration.
- **TDS Tracking**: Track Tax Deducted at Source obligations.

## Getting Started

### Prerequisites
- Go 1.23+
- PostgreSQL
- TigerBeetle (Optional, for Ledger)

### functionality Setup

1.  **Clone & Install Dependencies**
    ```bash
    go mod download
    ```

2.  **Database Setup**
    Ensure Postgres is running. The app connects to:
    `postgres://user:password@localhost:5432/recurso?sslmode=disable`
    *(See `cmd/api/main.go` to configure)*

3.  **Run the Server**
3.  **Run the Server**
    ```bash
    make run
    # Or build binary: make build
    ```
    *Migrations will apply automatically on startup.*

### Developer Commands
- `make build`: Compile the API
- `make test`: Run unit tests
- `make test-e2e`: Run end-to-end verification script
- `make docker-up`: Start Docker Compose (Dev)
    *Migrations will apply automatically on startup.*

## API Endpoints

### Core
- `POST /v1/plans` - Create a Plan
- `POST /v1/customers` - Create a Customer
- `POST /v1/subscriptions` - Create a Subscription
- `POST /v1/credit_notes` - Create a Credit Note

### Finance & Compliance
- `GET /v1/ledger/accounts` - View Ledger Accounts
- `POST /v1/tax/validate` - Validate GSTIN

### Checkout & Portal
- `GET /checkout/:invoice_id` - Payment Page
- `GET /portal/:customer_id` - Customer Dashboard

### Usage & Analytics
- `POST /v1/usage/events` - Ingest Metering Events
- `GET /v1/analytics/mrr` - Get MRR Metrics

## Architecture
- **Language**: Go (Gin Framework)
- **Database**: PostgreSQL
- **Architecture**: Hexagonal (Ports & Adapters)
