# Implementation Plan - Phase 20: Comprehensive Dashboard Interactions

The goal of this phase is to make the dashboard fully functional by implementing server-side search, filtering, and pagination for the core data grids. Currently, the "Search" and "Filter" UI elements exist but likely do not function or rely on client-side logic which is insufficient for production.

## User Review Required
> [!NOTE]
> This phase will touch multiple layers (Handler, Service, Repository) to plumb `limit`, `offset`, `search`, and `status` parameters.

## Proposed Changes

### Backend
#### [MODIFY] [customer_repository.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/db/customer_repository.go)
- Update `List` to accept `filter` struct (Search string, Status string, Limit int, Offset int).
- Update SQL query to use `ILIKE` for search and `WHERE` clauses for status.

#### [MODIFY] [customer.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/service/customer.go)
- Update `ListCustomers` signature to forward filters.

#### [MODIFY] [customer.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/handler/customer.go)
- Parse query parameters (`q`, `status`, `page`, `limit`) in `ListCustomers`.

#### [MODIFY] [plan_repository.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/db/plan_repository.go)
- Similar updates for Plans filtering.

#### [MODIFY] [subscription_repository.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/db/subscription_repository.go)
- Similar updates for Subscriptions filtering.

### Frontend
#### [MODIFY] [Customers.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/Customers.jsx)
- Update state to string search query and status filter.
- `useEffect` dependency on these states to refetch data.
- Debounce search input.

#### [MODIFY] [Products.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/Products.jsx)
- Wire up search and status filter.

#### [MODIFY] [Subscriptions.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/Subscriptions.jsx)
- Wire up search and status filter.

## Verification Plan
### Manual Verification
1.  **Search**: Type "Test" in Customer search bar -> Verify network request sends `?q=Test` -> Verify result list filters.
2.  **Filter**: Select "Active" status -> Verify network request sends `?status=active` -> Verify result list.
3.  **Pagination**: (optional) clicking "Next" sends `page=2`.
