# Implementation Plan - Execution P0 (Iron Core)

## Goal
Translate the approved documents (`feature_specs_p0.md`, `database_design.md`, `api_contract.md`) into working code.

## 1. Project Setup
- **Module**: `github.com/recurso-dev/recurso`
- **Structure**:
    - `cmd/api`: Main entry point.
    - `internal/core`: Domain types (Plans, Customers).
    - `internal/adapter`: Infrastructure (Postgres, Gin).
    - `internal/service`: Business logic.

## 2. Infrastructure
- **Web Framework**: Gin Gonic.
- **Database**: Postgres (with `golang-migrate`).
- **Config**: Env vars for DB connection.

## 3. Product Catalog (Plans)
- **Schema**: Create `plans` table.
- **Domain**: `Plan` struct.
- **API**: `POST /plans`, `GET /plans/:id` (as per OpenAPI).
- **Logic**: Validate currency and intervals.

## 4. Customer Management
- **Schema**: Create `tenants` (seed) and `customers` tables.
- **Domain**: `Customer` struct with `BillingAddress`.
- **API**: `POST /customers`.
- **Logic**: Ensure email uniqueness per tenant.

## 5. Subscription Setup
- **Schema**: Create `subscriptions` and `invoices` tables.
- **Domain**: `Subscription` state machine.
- **API**: `POST /subscriptions`.
- **Logic**: Mock payment success -> Create Invoice (Draft) -> Transition to Active.

## Verification
- **Run**: `go run cmd/api/main.go`
- **Test**: `curl` requests matching `api_contract.md` flows.
