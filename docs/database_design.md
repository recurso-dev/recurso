# Recurso: Database Design Document

This document defines the data persistence layer for Recurso. We use a **Polyglot Persistence** strategy:
1.  **PostgreSQL (System of Reference)**: Stores metadata, relationships, and mutable state (Users, Plans).
2.  **TigerBeetle (System of Record)**: Stores immutable financial history (Balances, Transactions).

---

## 1. PostgreSQL Schema (Metadata)

**Standards**:
- **Primary Keys**: `UUIDv7` (Time-sortable, index-friendly).
- **Timestamps**: `created_at`, `updated_at` (with triggers) on all tables.
- **Tenancy**: `tenant_id` on ALL tables for isolation.

### 1.1 Core Hierarchy

#### `tenants`
The merchant utilizing Recurso.
```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    api_key_hash VARCHAR(64) NOT NULL,
    default_currency CHAR(3) DEFAULT 'USD',
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### `customers`
The end-user / subscriber.
```sql
CREATE TABLE customers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    
    -- Billing Info
    billing_address JSONB, -- { line1, city, state, zip, country }
    tax_id VARCHAR(50),    -- GSTIN / VAT ID
    
    -- Ledger Mapping
    ledger_account_id UUID NOT NULL, -- Pointer to TigerBeetle Account (Liability/Prepaid)
    
    deleted_at TIMESTAMPTZ,
    UNIQUE(tenant_id, email)
);
```

### 1.2 Product Catalog

#### `plans`
```sql
CREATE TABLE plans (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL, -- e.g. "gold-monthly"
    
    -- Recurring Logic
    interval_unit VARCHAR(10) CHECK (interval_unit IN ('day', 'week', 'month', 'year')),
    interval_count INT DEFAULT 1, -- e.g., 3 months = Quarterly
    
    active BOOLEAN DEFAULT TRUE,
    UNIQUE(tenant_id, code)
);
```

#### `prices`
Separates "What you sell" from "How much it costs".
```sql
CREATE TABLE prices (
    id UUID PRIMARY KEY,
    plan_id UUID NOT NULL REFERENCES plans(id),
    currency CHAR(3) NOT NULL,
    amount BIGINT NOT NULL, -- In lowest unit (cents/paisa)
    type VARCHAR(20) DEFAULT 'recurring', -- or 'one_time'
    UNIQUE(plan_id, currency)
);
```

### 1.3 Subscription Engine

#### `subscriptions`
The central state machine.
```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    plan_id UUID NOT NULL REFERENCES plans(id),
    
    -- State
    status VARCHAR(20) CHECK (status IN ('trialing', 'active', 'past_due', 'paused', 'canceled', 'unpaid')),
    
    -- Billing Cycles
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    billing_anchor TIMESTAMPTZ NOT NULL, -- The "Renewal Day"
    
    -- Payment Method
    payment_method_id VARCHAR(100), -- Token ID
    
    canceled_at TIMESTAMPTZ,
    metadata JSONB
);
```

#### `invoices`
Immutable snapshots of debt.
```sql
CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    subscription_id UUID REFERENCES subscriptions(id),
    customer_id UUID REFERENCES customers(id),
    
    -- Financials
    currency CHAR(3) NOT NULL,
    subtotal BIGINT NOT NULL, -- Sum of lines
    tax_amount BIGINT DEFAULT 0,
    total BIGINT NOT NULL, -- Subtotal + Tax
    amount_paid BIGINT DEFAULT 0,
    amount_remaining BIGINT GENERATED ALWAYS AS (total - amount_paid) STORED,
    
    status VARCHAR(20) CHECK (status IN ('draft', 'open', 'paid', 'void', 'uncollectible')),
    due_date TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    
    -- Compliance
    invoice_number VARCHAR(50) NOT NULL, -- Sequential (INV-001) per tenant
    pdf_url TEXT,
    irn VARCHAR(100), -- India GST Reference
    qr_code TEXT
);
```

---

## 2. TigerBeetle Ledger (Financials)

**Standards**:
- **IDs**: 128-bit Integers (Converted from Postgres UUIDs).
- **Precision**: `uint64` (Base units).

### 2.1 Chart of Accounts (Codes)

We define a standardized chart of accounts for every Tenant.

| Category | Code Range | specific Account | TigerBeetle Code |
| :--- | :--- | :--- | :--- |
| **Assets** | 1000-1999 | Bank (Clearance) | `1001` |
| **Liabilities**| 2000-2999 | Customer Prepaid (Credits) | `2001` |
| | | GST Payable | `2002` |
| | | Deferred Revenue | `2003` |
| **Equity** | 3000-3999 | Capital | `3001` |
| **Revenue** | 4000-4999 | Subscription Revenue (Recognized) | `4001` |
| **Expenses** | 5000-5999 | Gateway Fees | `5001` |
| | | Bad Debt (Write-off) | `5002` |

### 2.2 Transaction Flows

#### Scenario A: Subscription Payment (Pre-paid)
User pays $100 + $18 Tax for a Month.
**Transfer (Linked Chain)**:
1.  **Debit**: Asset:Bank (`1001`) — $118
2.  **Credit**: Liability:DeferredRevenue (`2003`) — $100
3.  **Credit**: Liability:GSTPayable (`2002`) — $18

#### Scenario B: Revenue Recognition (Month End)
We recognize the $100 service has been delivered.
**Transfer**:
1.  **Debit**: Liability:DeferredRevenue (`2003`) — $100
2.  **Credit**: Revenue:Subscription (`4001`) — $100

#### Scenario C: Refund
User asks for refund.
**Transfer**:
1.  **Debit**: Liability:DeferredRevenue (`2003`) — $100
2.  **Debit**: Liability:GSTPayable (`2002`) — $18
3.  **Credit**: Asset:Bank (`1001`) — $118

---

## 3. ID Mapping Strategy
How to link Postgres UUIDs to TigerBeetle u128s.

- **Algorithm**: Treat the 16 bytes of the UUID directly as a 128-bit integer (Little Endian).
- **Lookup**:
    - `Postgres -> TB`: `uuid.ToUint128()`
    - `TB -> Postgres`: `u128.ToUUID()` stored in `user_data_128` field of TigerBeetle for reverse lookups.
