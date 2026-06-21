# Walkthrough - Phase 17: Backend Integration

This phase connected the frontend "UI Polish" screens (Coupons, Products, Usage) to the real backend APIs.

## Changes

### 1. Coupons Integration
- **Frontend**: Updated `api.js` with `getCoupons`, `createCoupon`. Connected `Coupons.jsx` and `CreateCoupon.jsx` to real API.
- **Backend**: Existing `CouponHandler` endpoints are now fully utilized.

### 2. Products Integration
- **Frontend**: Updated `Products.jsx` to fetch `getPlans()` from backend and map them to the "Products" table layout.

### 3. Usage Dashboard Integration
- **Backend**:
    - Created `UsageStats` domain model and `UsageRepository` interface.
    - Implemented `GetUsageStats` in `UsageRepository` to aggregate usage by dimension for the tenant.
    - Updated `AnalyticsService` and `AnalyticsHandler` to expose `/analytics/usage`.
- **Frontend**:
    - Updated `Usage.jsx` to fetch stats from `/analytics/usage`.
    - "Total Units Consumed" and "Usage by Metric" widgets now show real aggregated data.
    - *Note: The time-series chart and event table remain simulated/mocked for this iteration as granular time-series API is a future optimization.*

## Verification Steps

### 1. Restart Backend
Since backend code (Service/Repository injection) was modified, you MUST restart the Go server:
```bash
# Stop running server (Ctrl+C)
go run cmd/api/main.go
# OR use the binary we just built:
# ./server
```

### 2. Verify Coupons
1. Navigate to **/coupons**.
2. Click **Create Coupon**.
3. Enter code `TEST-INTEGRATION`, choose "Percent Off", value `10`.
4. Click Create.
5. Verify you are redirected to the list and see the new coupon (fetched from DB).

### 3. Verify Products
1. Navigate to **/products**.
2. Verify you see the Plans referenced in your database (e.g., "Pro Plan", "Basic Plan").

### 4. Verify Usage
1. Navigate to **/usage**.
2. "Total Units Consumed" should reflect the sum of all usage events for your tenant in the DB (or 0 if empty).
3. "Usage by Metric" widget should show breakdown if data exists.
