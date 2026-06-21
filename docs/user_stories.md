# Recurso: User Stories & Acceptance Criteria (Priority 0)

This document defines the functional requirements from the end-user's perspective. It serves as the **Definition of Done** for the engineering team.

## 1. Product Catalog

### Story 1: Merchant creates a Plan
**As a** Merchant,
**I want to** create a recurring pricing plan (e.g., "Gold Monthly $10"),
**So that** I can sell it to subscribers.

**Acceptance Criteria:**
- [ ] Merchant can POST JSON to `/plans` with `name`, `amount`, `currency`, `interval`.
- [ ] System validates that `amount` is positive integer.
- [ ] System validates `currency` is a valid ISO code (e.g., 'USD').
- [ ] Created plan returns a `UUID` and `status: active`.
- [ ] **Ledger Check**: No ledger impact yet.

---

## 2. Customer Management

### Story 2: Merchant registers a Customer
**As a** Merchant,
**I want to** create a customer record with billing details,
**So that** I can assign subscriptions to them.

**Acceptance Criteria:**
- [ ] Merchant can POST to `/customers` with `email`, `name`, `billing_address`.
- [ ] System checks if `email` is unique within the tenant.
- [ ] **Ledger Check**: System automatically creates a corresponding **TigerBeetle Account** (Liability Type) for this customer.
- [ ] The TigerBeetle Account ID is saved in the Postgres `Customer` record.

---

## 3. Subscription Lifecycle

### Story 3: Merchant subscribes a Customer to a Plan
**As a** Merchant,
**I want to** assign the "Gold Monthly" plan to "Customer A",
**So that** they are billed immediately.

**Acceptance Criteria:**
- [ ] POST to `/subscriptions`.
- [ ] System calculates `current_period_start` (Now) and `current_period_end` (Now + 1 Month).
- [ ] System creates an **Invoice** in `DRAFT` state.
- [ ] System attempts to collect payment (Mocked for P0).
- [ ] **Ledger Check**: 
    1.  Debit `Bank` (Asset), Credit `Deferred Revenue` (Liability).
- [ ] Subscription status moves to `active`.

---

## 4. Proration (The "Hard Math")

### Story 4: Customer upgrades mid-cycle
**As a** Customer,
**I want to** switch from "Silver" ($10) to "Gold" ($20) halfway through the month,
**So that** I get access to better features immediately.

**Acceptance Criteria:**
- [ ] Plan change requested at T=15 days (Total 30 days).
- [ ] **Credit**: Unused Silver = $5.00.
- [ ] **Debit**: Remaining Gold = $10.00.
- [ ] **Net Due**: $5.00.
- [ ] System generates an immediate **One-off Invoice** for $5.00.
- [ ] Subscription `plan_id` is updated to Gold.
- [ ] **Ledger Check**:
    1.  The $5.00 payment is recorded.

---

## 5. Billing

### Story 5: System generates renewal invoice
**As a** System (Cron),
**I want to** identify subscriptions expiring tomorrow,
**So that** I can charge them.

**Acceptance Criteria:**
- [ ] Job finds subscription where `current_period_end` == Tomorrow.
- [ ] System generates Invoice for next cycle.
- [ ] Webhook `invoice.created` is triggered.
