# Phase 23: Credit Notes Implementation Plan

## Goal
Implement Credit Notes functionality to allow issuing credits to customers, which can be applied to future invoices or refund transactions.

## Proposed Changes

### Backend
1.  **Domain Model (`internal/core/domain/credit_note.go`)**:
    - Struct `CreditNote`: ID, CustomerID, Reference, Amount (int64), Balance (int64), Currency, Status (Open, Applied, Void), Reason, CreatedAt.
2.  **Database Migration**:
    - Create `credit_notes` table.
3.  **Repository (`internal/adapter/db/credit_note_repository.go`)**:
    - `Create`, `GetByID`, `List` (by Tenant, Customer).
4.  **Service (`internal/service/credit_note.go`)**:
    - `IssueCreditNote`: Validates customer, creates entry.
    - `GetCreditNotes`: Lists notes.
5.  **Handler (`internal/adapter/handler/credit_note.go`)**:
    - `POST /v1/credit-notes`: Create.
    - `GET /v1/credit-notes`: List.
6.  **Register Routes**:
    - Add to `cmd/api/main.go`.

### Frontend
1.  **List View (`frontend/src/pages/CreditNotes.jsx`)**:
    - Data Grid showing ID, Customer, Amount, Status, Invoice, Date.
    - Match `credit_notes_data_grid` design.
2.  **Create View (`frontend/src/pages/CreateCreditNote.jsx`)**:
    - Form with Customer selection (async search), Amount, Reason, Linked Invoice (optional).
    - Match `create_new_credit_note_form` design.
3.  **Navigation**:
    - Add "Credit Notes" to Sidebar.
4.  **API Client**:
    - Add `getCreditNotes`, `createCreditNote` to `api.js`.

## Verification Plan

### Automated Tests
- Create `verify_p23.sh` to:
    1.  Create a customer.
    2.  Create a credit note for that customer.
    3.  List credit notes and verify existence.

### Manual Verification
- Go to `/credit-notes`.
- Click "Create Credit Note".
- Fill form and submit.
- Verify redirect and list update.
