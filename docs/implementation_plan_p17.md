# Implementation Plan - Phase 17: Backend Integration

**Goal**: Connect the "UI Polish" screens (Phase 16) to the real backend APIs to enable full functionality for Coupons, Products, and Usage Analytics.

## User Review Required
> [!NOTE]
> This phase focuses on wiring up existing UIs to the backend. We will be replacing mock data with real API calls.

## Proposed Changes

### 1. Coupons Integration
Connect the Coupons management UI to the `CouponHandler` endpoints.
- **Frontend Changes**:
    - [MODIFY] `frontend/src/lib/api.js`: Add `getCoupons`, `createCoupon` methods.
    - [MODIFY] `frontend/src/pages/Coupons.jsx`: Replace mock data with `api.getCoupons()`.
    - [MODIFY] `frontend/src/pages/CreateCoupon.jsx`: Submit form to `api.createCoupon()`.
    - [MODIFY] `frontend/src/components/slide-overs/CouponDetail.jsx`: Display real data.

### 2. Products Integration
Connect the Product Catalog UI to the `CatalogHandler` (Plans) endpoints. 
*Note: "Products" in the UI maps to "Plans" in our current backend domain.*
- **Frontend Changes**:
    - [MODIFY] `frontend/src/pages/Products.jsx`: Fetch data using `api.getPlans()` and map fields (Plan Name -> Product Name, etc.).

### 3. Usage Dashboard Integration
The current `AnalyticsService` only supports MRR. We need to expose usage statistics.
- **Backend Changes**:
    - [NEW] `internal/core/domain/analytics.go`: Define `UsageStats` structs.
    - [MODIFY] `internal/core/port/repository.go`: Add `GetUsageStats` to `UsageRepository` interface (or `AnalyticsRepository`).
    - [MODIFY] `internal/adapter/db/usage_repository.go`: Implement aggregation queries (e.g., usage by type, total units).
    - [MODIFY] `internal/service/analytics.go`: Add `GetUsageStats(ctx, tenantID)` method.
    - [MODIFY] `internal/adapter/handler/analytics.go`: Add `GetUsageStats` endpoint.
    - [MODIFY] `cmd/api/main.go`: Register new route.
- **Frontend Changes**:
    - [MODIFY] `frontend/src/lib/api.js`: Add `getUsageStats` method.
    - [MODIFY] `frontend/src/pages/Usage.jsx`: Fetch and display real stats.

## Verification Plan

### Manual Verification
- **Coupons**: Create a coupon via UI, see it in the list, verify it in DB/API response.
- **Products**: Verify existing plans show up in the Product Catalog.
- **Usage**:
    - Trigger some usage events via API.
    - Verify they appear in the Usage Dashboard.
