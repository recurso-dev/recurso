# Implementation Plan - Phase 3: Scale & Analytics 📈

## Goal
Enable Usage-Based Billing (metered pricing) and provide high-level Revenue Analytics (MRR, Churn) to Tenants.

## User Review Required
> [!NOTE]
> For Usage-Based Billing, we will implement an "Event Ingestion API". The aggregation (Sum/Max/Last) will happen at the time of Invoice Generation (mocked in `SubscriptionService` for now).

## Proposed Changes

### 1. Usage Metering (Event Ingestion)
Allow tenants to report usage events (e.g., "API Calls", "Storage Used").

#### [NEW] `internal/core/domain/usage.go`
- `UsageEvent`: {CustomerID, Dimension, Quantity, Timestamp}

#### [NEW] `internal/adapter/db/usage_repository.go`
- Store events (initially in Postgres `usage_events` table).

#### [NEW] `internal/adapter/handler/usage.go`
- `POST /v1/usage/events`

### 2. Analytics Service
Provide key metrics for the dashboard.

#### [NEW] `internal/service/analytics.go`
- Methods to calculate:
    - **MRR**: Sum of active subscriptions * monthly price.
    - **Revenue**: Sum of paid invoices.

#### [NEW] `internal/adapter/handler/analytics.go`
- `GET /v1/analytics/mrr`
- `GET /v1/analytics/revenue`

### 3. Database Updates
#### [NEW] `internal/adapter/db/migrations/000005_create_usage_table.up.sql`
- Table `usage_events` (id, subscription_id, dimension, quantity, timestamp).

## Verification Plan

### Automated Tests
1.  **Ingestion**: `curl POST /v1/usage/events` -> 201 Created.
2.  **Analytics**: `curl GET /v1/analytics/mrr` -> Returns JSON number.

### Manual Verification
1.  **Usage**: Ingest 10 events. Query DB to confirm storage.
2.  **Analytics**: Create 2 new subscriptions ($50 each). Verify MRR = $100.
