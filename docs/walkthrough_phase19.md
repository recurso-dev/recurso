# Phase 19: Backend Extension (Customer Fields)

## Changes
### Backend
- **Database**: Added `phone`, `tax_id`, `line1`, `city`, `state`, `zip`, `country` columns to `customers` table.
- **Domain**: Updated `Customer` struct to include new fields.
- **Handler**: Updated `CreateCustomer` to bind new fields from JSON request.
- **Repository**: Updated `Create`, `GetByID`, and `List` to handle new columns.
- **Service**: Updated `CreateCustomerInput` to accept new fields.

### Frontend
- **CreateCustomer.jsx**: Updated `handleSubmit` to send all form fields (`phone`, `tax_id`, etc.) to the API.

## Verification
### Manual Verification
1.  **Frontend**: Navigate to "Add New Customer".
2.  **Action**: Fill out all fields (Name, Email, Phone, Address, etc.) and submit.
3.  **Result**: Customer is created successfully.
4.  **Verification**: Check "Customers" list or database to confirm all fields are valid.
