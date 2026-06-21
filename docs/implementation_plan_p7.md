# Implementation Plan - Phase 7: Advanced Billing (Coupons) 🎟️

## Goal
Implement a robust Coupon and Discount system allowing customers to apply codes for Percentage-based or Fixed-amount reductions on their subscriptions.

## User Review Required
> [!IMPORTANT]
> **Discount Logic**:
> - **Percentage**: e.g., 20% off.
> - **Fixed Amount**: e.g., $10 off.
> - **Duration**: "Forever" (applies to every invoice), "Once" (first invoice only), or "Repeating" (X months). *For MVP, we will start with Forever and Once.*
>
> **Ledger Impact**: Discounts reduce the `Revenue` credited. We will record the *Net* amount in the Ledger for now, rather than a separate "Discount Expense" line item, to keep the MVP simple.

## Proposed Changes

### 1. Database Schema
#### [NEW] `internal/adapter/db/migrations/000006_create_coupons_table.up.sql`
- `coupons`:
    - `id` (UUID)
    - `code` (string, unique) e.g., "SAVE20"
    - `discount_type` (enum: 'percent', 'amount')
    - `discount_value` (int) e.g., 20 or 1000 (cents)
    - `duration` (enum: 'forever', 'once', 'repeating')
    - `duration_months` (int, nullable)
- Change `subscriptions` table:
    - Add `coupon_id` (UUID, nullable)

### 2. Core Domain
#### [NEW] `internal/core/domain/coupon.go`
- Struct `Coupon`.
- Enum constants for types and durations.

### 3. Repository Layer
#### [NEW] `internal/adapter/db/coupon_repository.go`
- `Create(coupon)`
- `GetByCode(code)`

### 4. Service Layer
#### [MODIFY] `internal/service/subscription.go` (`CreateSubscription`)
- Input: Accept optional `coupon_code`.
- Logic: Validate code, fetch Coupon, link to Subscription.
- **Invoice Generation**: Calculate `discount_amount` based on coupon type and apply to `subtotal`. `total = subtotal - discount`.

### 5. API Layer
#### [NEW] `internal/adapter/handler/coupon.go`
- `POST /v1/coupons`: Admin endpoint to create coupons.

## Verification Plan

### Automated Tests
1.  **Unit Tests**: Test discount calculation logic (Percent vs Amount).
2.  **Integration**: Create Subscription with "SAVE20", verify Invoice total is reduced by 20%.

### Manual Verification
1.  Create a Coupon "DEMO50" (50% off).
2.  Create a new Subscription using this code.
3.  Check the generated Invoice amount.
4.  Check TigerBeetle ledger reflects the reduced revenue.
