# Implementation Plan - Phase 19: Backend Extension (Customer Fields)

The goal of this phase is to ensure the "Create Customer" form in the frontend is fully backed by the API. Currently, the UI collects Name, Email, Phone, Address, Country, State, and Tax ID, but the backend only fully supports Name, Email, and structured Address components. It treats Phone and Tax ID as missing.

## User Review Required
> [!IMPORTANT]
> This change involves a database migration to add `phone` and `tax_id` to the `customers` table.

## Proposed Changes

### Backend
#### [MODIFY] [customer.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/core/domain/customer.go)
- Add `Phone` (string) and `TaxID` (string) to `Customer` struct.

#### [NEW] [000006_add_customer_fields.up.sql](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/db/migrations/000006_add_customer_fields.up.sql)
- SQL migration to add `phone` and `tax_id` columns to `customers` table.

#### [MODIFY] [customer_repository.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/db/customer_repository.go)
- Update `CreateCustomer` and `ListCustomers` (scan/insert) to handle new fields.

#### [MODIFY] [customer.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/handler/customer.go)
- Update `createCustomerRequest` struct to include `Phone` and `TaxID`.
- Update mapping to service input.

### Frontend
#### [MODIFY] [CreateCustomer.jsx](file:///Users/swapnull/Documents/Workspace/recur-so/frontend/src/pages/CreateCustomer.jsx)
- Update `handleSubmit` to construct the full payload.
- Map `formData.address` (textarea) to `line1` for now (or simple split if logical).
- Include `phone`, `tax_id`, `country`, `state`.

## Verification Plan
### Automated Tests
- Run `go test ./...` to ensure no breaking changes in repository/service layers.

### Manual Verification
1. Start the app (`npm run dev` + `go run cmd/api/main.go`).
2. Navigate to "Customers" -> "Add New Customer".
3. Fill out ALL fields (Name, Email, Phone, Address, Country, State, Tax ID).
4. Submit form.
5. Verify in "Customers" list or DB that all fields are correctly saved.
