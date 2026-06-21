# Implementation Plan - Phase 15: Advanced Billing Features

## Goal Description
Implement sophisticated billing capabilities to reach parity with enterprise-grade billing systems. This phase focuses on **Calendar Billing** (aligning cycles to specific dates), **Unbilled Charges** (accumulating usage/add-ons before invoicing), and **Net-D Terms** (credit periods).

## User Review Required
> [!IMPORTANT]
> **Schema Changes**: We will need to modify the `subscriptions` table to support custom billing anchors and add a new `unbilled_charges` table.

## Proposed Changes

### Database & Migrations
#### [NEW] [internal/adapter/db/migrations/000005_advanced_billing.up.sql](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/db/migrations/000005_advanced_billing.up.sql)
- `subscriptions`: Add `billing_anchor_type` (e.g., 'acquisition', 'first_of_month') and `billing_anchor_day` (1-31).
- `subscriptions`: Add `payment_terms` (e.g., 'net0', 'net15', 'net30').
- `unbilled_charges`: New table to store pending charges (amount, currency, description) linked to a subscription.

### Core Domain
#### [MODIFY] [internal/core/domain/subscription.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/core/domain/subscription.go)
- Update `CalculateNextBillingDate` to respect `billing_anchor_type`.
    - If `first_of_month`: Prorate partial period from Start Date to 1st of next month, then align cycles.
    
#### [MODIFY] [internal/core/domain/invoice.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/core/domain/invoice.go)
- Update `GenerateInvoice` to include "Unbilled Charges" when finalizing a period.
- Set `due_date` based on `payment_terms`.

### Service Layer
#### [MODIFY] [internal/service/subscription.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/service/subscription.go)
- Add method `AddUnbilledCharge(subscriptionID, amount, description)`.
- Update `CreateSubscription` to accept billing alignment preferences.

### API & Frontend
#### [MODIFY] [internal/adapter/handler/subscription.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/handler/subscription.go)
- Expose new fields in Create/Update endpoints.

#### [MODIFY] [frontend/src/pages/CreateSubscription.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/CreateSubscription.jsx)
- Add UI controls for "Align to Calendar Month" and "Payment Terms".

## Verification Plan

### Automated Tests
- **Unit Tests**: Verify proration logic for "Align to 1st of Month" (short first period).
- **Integration Tests**: Create a subscription with Net-30 terms and verify the generated invoice `due_date`.
