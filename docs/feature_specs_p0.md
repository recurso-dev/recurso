# Recurso: Feature Specifications (Priority 0)

This document contains the deep-dive technical and functional specifications for the "Iron Core" features. These are the non-negotiables required to support the first subscription.

## 1. Product Catalog

### 1.1 Description
The foundational data layer describing *what* is being sold. It allows merchants to define Plans, Addons, and Charges.

### 1.2 Data Model
- **Plan**: `id`, `name`, `billing_interval` (day/week/month/year), `billing_interval_count` (e.g., every *3* months), `currency`, `price` (in lowest unit).
- **Addon**: `id`, `name`, `type` (recurring/one-time), `price`.
- **PricePoint**: Support for overridable prices per currency (USD price != INR price * FX).

### 1.3 Business Logic
- **Immutability**: Once a Plan has subscribers, its core attributes (interval, currency) cannot be changed. It must be `archived` and a `new version` created.
- **Versioning**: Implicit versioning. Creating "Gold Plan v2" does not automatically migrate "Gold Plan v1" users unless explicitly requested.

### 1.4 API Endpoints
- `POST /plans`: Create a plan.
- `GET /plans/{id}`: Retrieve details.
- `PATCH /plans/{id}`: Archive or update metadata (name/description only).

---

## 2. Customer Management

### 2.1 Description
The CRM-lite entity representing the buyer.

### 2.2 Data Model
- **Customer**: `id`, `email` (unique per tenant), `phone`, `company_name`.
- **BillingAddress**: `line1`, `city`, `zip`, `state` (Crucial for GST), `country`.
- **TaxInfo**: `gstin`, `vat_number`, `is_tax_exempt`.

### 2.3 Business Logic
- **Tax Location Determination**: "Place of Supply" logic. If `BillingAddress.State` != `Tenant.State`, apply IGST. Else CGST+SGST. This must be validated on creation.

---

## 3. Subscription Lifecycle (The State Machine)

### 3.1 Description
The core engine that manages the status of a recurring relationship.

### 3.2 States
1.  **Trialing**: Active, but free for $N$ days.
2.  **Active**: In good standing. Payment successful.
3.  **Past Due**: Payment failed. Retries in progress. Service *may* continue.
4.  **Paused**: Temporarily stopped by user/merchant. No billing.
5.  **Canceled**: Permanently stopped. One-way door (usually).
6.  **Unpaid**: Dunning failed. Service stopped.

### 3.3 Transitions
- `create()` -> `Active` (if paid) / `Trialing`.
- `payment_failed` -> `Past Due`.
- `dunning_success` -> `Active`.
- `dunning_exhausted` -> `Canceled` or `Unpaid` (Configurable).
- `pause()` -> `Paused`.

---

## 4. Calculated Billing & Proration

### 4.1 Description
The "Math Engine" for generating line items.

### 4.2 Proration Logic (Exact-Time)
**Scenario**: User moves from Plan A ($100) to Plan B ($200) on Day 15 of 30.
1.  **Unused Credit**: Plan A for 15 days = $50. (Credit)
2.  **New Charge**: Plan B for 15 days = $100. (Debit)
3.  **Net**: $50 Due.
    - *Immediate*: Charge $50 now.
    - *Term*: Add $50 to next invoice.

### 4.3 Rounding
- **Banker's Rounding**: Round half to even for midpoint ties.
- **Precision**: Store and calculate in `int64` (Paisa). Display as float.

---

## 5. Invoice Generation

### 5.1 Description
Generates the immutable record of debt.

### 5.2 Lifecycle
- **Draft**: Being built. Metered usage is being aggregated.
- **Posted/Open**: Finalized. Sent to user. Payment expected.
- **Paid**: Balance = 0.
- **Void**: Erroneously created. Value = 0.
- **Uncollectible**: Write-off.

### 5.3 PDF & Compliance
- Must include:
    - Merchant GSTIN & Address.
    - Customer GSTIN & Address (for B2B ITC).
    - HSN/SAC Codes for items.
    - Tax breakup (CGST/SGST/IGST).
    - **QR Code** (for e-invoicing).

---

## 6. API & Webhooks

### 6.1 Standards
- **Protocol**: REST (JSON) over HTTP/2.
- **Auth**: Basic Auth (API Key as username).
- **Idempotency**: `Idempotency-Key` header required for all non-GET requests.

### 6.2 Webhooks
- **Delivery**: At-least-once.
- **Security**: Signed payload (`HMAC-SHA256`).
- **Retries**: Exponential backoff (1m, 5m, 10m...) up to 24h.
