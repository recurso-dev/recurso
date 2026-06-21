# Implementation Plan - Phase 8: Multi-Tenancy & API Keys 🔑

## Goal
Transform Recurso into a true Multi-Tenant SaaS. Enable separate businesses ("Tenants") to sign up, generate unique API Keys, and manage their own Customers/Subscriptions in isolation.

## User Review Required
> [!IMPORTANT]
> **Tenant Isolation Strategy**:
> - We will use **Row-Level Security (Logical)**. Every table (`customers`, `plans`, `subscriptions`) already has a `tenant_id` column.
> - **Critical Refactor**: All Repository methods must be updated to enforce `WHERE tenant_id = ?` filters.
>
> **API Keys**:
> - Format: `pk_live_<random>` (Public, not implemented yet) and `sk_live_<random>` (Secret, implemented now).
> - Stored as hashed values in DB? For MVP reliability/debugging, we might store plain or simple hash. *Decision: Plain text for MVP simplicity, Hash recommended for Prod.* (We will store Plain for now to avoid complexity in verification).

## Proposed Changes

### 1. Database Schema
#### [NEW] `internal/adapter/db/migrations/000007_create_tenants_keys.up.sql`
- `tenants`:
    - `id` (UUID)
    - `name` (string)
    - `email` (string)
    - `created_at`
- `api_keys`:
    - `id` (UUID)
    - `tenant_id` (UUID)
    - `key_hash` (string) -- Storing the actual key for this MVP phase for simplicity: `key_value`
    - `type` (enum: 'public', 'secret')
    - `is_active` (bool)

### 2. Domain & Key Management
#### [NEW] `internal/core/domain/tenant.go`
- Structs for `Tenant` and `APIKey`.

### 3. Service Layer
#### [NEW] `internal/service/tenant.go`
- `Register(name, email) -> (Tenant, APIKey)`
- This is the "Sign Up" flow for your customers (developers).

### 4. Middleware Refactor
#### [MODIFY] `internal/adapter/middleware/auth.go`
- Remove environment variable check.
- **Lookup**: Extract Bearer token -> Query `api_keys` table -> Get `tenant_id`.
- **Context**: Set `tenant_id` in Gin Context.

### 5. Repository Refactor (The "Heavy Lift")
#### [MODIFY] `internal/adapter/db/*.go`
- Update `Create`, `Get`, `List` methods to accept `tenantID` from Context (or ensure the struct passed in has it set correctly).

## Verification Plan

### Manual Verification
1.  **Register**: Call `POST /v1/auth/register` (New Endpoint) -> Get `sk_live_123`.
2.  **Access**: Call `GET /v1/plans` with `Authorization: Bearer sk_live_123`.
3.  **Isolation**: Register Tenant B (`sk_live_456`). Verify Tenant B cannot see Tenant A's plans.
