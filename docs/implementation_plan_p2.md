# Implementation Plan - Phase 2: AI Intelligence & Self-Service 🧠

## Goal
Implement "Smart Retries" to recover failed payments using basic heuristics (mocking AI) and provide a Self-Service Portal for customers to view invoices and manage methods.

## User Review Required
> [!IMPORTANT]
> The "AI" for retries will be a heuristic-based mock service for this iteration. It will return "Optimal Retry Times" based on random logic or simple rules (e.g., "retry in 3 days").

## Proposed Changes

### 1. Smart Retry System (Worker & Logic)
We need a background process/service to check for `past_due` invoices and schedule retries.

#### [NEW] `internal/core/domain/retry.go`
- Domain model for `RetryAttempt` and `RetryPolicy`.

#### [NEW] `internal/adapter/worker/retry_worker.go`
- A simple Ticker-based worker that polls for unpaid invoices.
- Calls `SmartRetryService` to decide *when* to retry.

#### [NEW] `internal/service/smart_retry.go`
- **Logic**:
    - Input: Invoice History, Error Codes.
    - Output: Next Retry Time.
    - Mock Connection: Simulates calling an ML model.

### 2. Customer Portal (Self-Service)
A simple web dashboard for customers.

#### [NEW] `internal/adapter/handler/portal.go`
- `GET /portal/:customer_id/dashboard`
- `GET /portal/:customer_id/invoices`

#### [NEW] `internal/adapter/templates/portal_dashboard.html`
- Lists Subscriptions and Invoices.
- "Pay Now" links for open invoices.

### 3. Database Updates
#### [MODIFY] `internal/adapter/db/migrations/000003_add_retry_policy.up.sql`
- Add `next_retry_at` and `retry_count` to `invoices` table.

## Verification Plan

### Automated Tests
1.  **Retry Logic**: Unit test `SmartRetryService` to ensure it suggests future dates.
2.  **Worker**: Verify `RetryWorker` picks up an overdue invoice (mocking time/db state).

### Manual Verification
1.  **Smart Retries**:
    - Manually set an invoice to `past_due`.
    - Run the worker.
    - Verify `next_retry_at` is updated in DB.
2.  **Portal**:
    - Visit `/portal/<customer_id>`.
    - See list of invoices.
