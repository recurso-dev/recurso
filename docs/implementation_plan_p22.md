# Phase 22: Financial Ledger Implementation Plan

## Goal
Implement a Financial Ledger view in the dashboard to allow users to inspect the double-entry ledger transactions (credits/debits) recorded by the system.

## Proposed Changes

### Backend
1.  **Expose Ledger Data**:
    - Check `internal/service/ledger.go` for available methods (likely `GetEntries` or similar).
    - Create a new `LedgerHandler` in `internal/adapter/handler/ledger.go`.
    - Implement `GET /v1/ledger/entries` endpoint.
    - Parsing query params for paging/filtering (account_id, date range).
    - Register route in `cmd/api/main.go`.

### Frontend
1.  **New Page: `Ledger.jsx`**:
    - Create `frontend/src/pages/Ledger.jsx`.
    - Implement a Data Grid to display entries:
        - Columns: ID, Timestamp, Account, Debit/Credit, Amount, Reference (Invoice/Payment ID).
    - Use `financial_ledger_overview` design as reference.
2.  **Navigation**:
    - Add "Financials" or "Ledger" to `DashboardLayout.jsx` sidebar.
3.  **API Client**:
    - Add `getLedgerEntries` to `frontend/src/lib/api.js`.

## Verification Plan

### Automated Tests
- Create `verify_p22.sh` script to:
    1.  Create an invoice (which should trigger ledger entries if integrated).
    2.  Query `GET /v1/ledger/entries` to verify entries exist.
    3.  Check for specific accounts (Accounts Receivable, Revenue).

### Manual Verification
- Navigate to `/ledger` in the dashboard.
- Verify the table renders correctly.
- Verify pagination works.
