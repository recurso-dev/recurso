# Phase 20 Walkthrough: Comprehensive Dashboard Interactions

## Goal
Implement server-side search, filtering, and pagination for Customers, Plans, and Subscriptions to support a comprehensive dashboard experience.

## Changes

### Backend
1.  **Repository Interface Updates:**
    - Updated `PlanRepository` and `SubscriptionRepository` interfaces to accept filter structs (`PlanFilter`, `SubscriptionFilter`).
    - Added `Search`, `Limit`, `Offset` fields to `CustomerFilter`, `PlanFilter`, `SubscriptionFilter`.
    - `SubscriptionFilter` also includes `Status` and now supports searching by Customer Name/Email via JOIN.

2.  **Implementation Updates:**
    - Modified `List` methods in `CustomerRepository`, `PlanRepository`, and `SubscriptionRepository` to build dynamic SQL queries based on filters.
    - **Key Change:** `SubscriptionRepository.List` now performs a `LEFT JOIN customers` to allow searching subscriptions by customer name or email (`c.name ILIKE ...`).
    - Fixed a bug in `SubscriptionRepository` where query arguments were appended incorrectly for search parameters.

3.  **Handlers:**
    - Updated `ListCustomers`, `ListPlans`, `ListSubscriptions` handlers in `internal/adapter/handler` to parse query parameters (`q`, `limit`, `page`, `status`, `country`).
    - Added pagination logic (`limit`, `offset`) to all list endpoints.

### Frontend
1.  **API Client:**
    - Updated `frontend/src/lib/api.js` to pass `params` object in `getPlans`, `getCustomers`, and `getSubscriptions` calls.

2.  **UI Components:**
    - **`Customers.jsx`**: Added Search bar, debounced search state, and pagination controls. Wired up to `getCustomers` with params.
    - **`Products.jsx`**: Added Search bar, debounced search, and pagination. Wired up to `getPlans` with params.
    - **`Subscriptions.jsx`**: Added Search bar (supporting customer name search), Status filter dropdown, and pagination. Wired up to `getSubscriptions` with params.

## Verification
- Created `verify_p20.sh` to test:
    - Customer Search (by name)
    - Customer Filter (by country)
    - Subscription Search (by customer name using the new JOIN logic)
    - Pagination (limit verification)
- **Result**: All tests passed ✅.

## Artifacts
- `verify_p20.sh`: Verification script (in workspace root).
