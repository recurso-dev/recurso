# Implementation Plan: Revenue Recognition (ASC 606) 💹

## Overview
Automate the recognition of revenue for subscriptions by creating recognition schedules upon payment and executing ledger transfers over the service period.

## Phase 1: Domain & Data Modeling
1.  **Define Entities**: 
    - `RevenueSchedule`: Linked to an invoice/subscription, defines total amount and period.
    - `RecognitionEvent`: Individual slices of revenue (e.g., monthly buckets).
2.  **Schema Migration**: Create `revenue_schedules` and `recognition_events` tables.

## Phase 2: Core RevRec Service
1.  **Allocation Logic**: 
    - Function to split an invoice total into monthly components based on subscription `start_date` and `end_date`.
    - Handle edge cases (partial months at start/end).
2.  **Service Methods**:
    - `CreateSchedule(invoice)`: Generates the full schedule.
    - `GetReport(month, year)`: Aggregates recognition events for reporting.

## Phase 3: Repository & Worker
1.  **RevRec Repository**: Persistence for schedules and events.
2.  **RevRec Worker**:
    - Runs periodically (e.g., daily).
    - Finds events due for recognition (event_date <= now AND status = 'pending').
    - Calls TigerBeetle to move funds from `Deferred` to `Recognized`.
    - Marks event as `recognized`.

## Phase 4: Integration
1.  **Payment Hook**: Update `PaymentHandler` or `InvoiceService` to call `RevRecService.CreateSchedule` when an invoice is fully paid.
2.  **Cancellation Hook**: Update `SubscriptionService` to "cancel" or adjust future recognition events if a subscription is terminated early.

## Phase 5: Ledger Setup
1.  **Account Creation**: Ensure the tenant has "Deferred Revenue" and "Recognized Revenue" accounts in TigerBeetle.

## Risks & Mitigations
- **Rounding Errors**: Accumulate fractional cents in the final recognition event of a schedule to ensure total recognized exactly equals invoice total.
- **Clock Drift**: Use database server time for all recognition triggers.
