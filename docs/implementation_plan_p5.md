# Implementation Plan - Phase 5: The TigerBeetle Ledger 🐯

## Goal
Integrate **TigerBeetle** to provide an immutable, double-entry financial ledger for all transactions. This moves Recurso from a "Database-based" billing engine to a proper "Financial System of Record".

## User Review Required
> [!IMPORTANT]
> **Architecture Split**: We will maintain the **PostgreSQL** DB as the "System of Metadata" (Customer profiles, Plan names, Invoice states) but use **TigerBeetle** as the "System of Finance" (Account Balances, Transaction History).
> **Dual-Write**: For this MVP, we will dual-write to Postgres and TigerBeetle. In a production system, we might use an event bus (CDC) to ensure consistency, but direct calls are sufficient for now.

## Proposed Changes

### 1. Infrastructure (Docker)
Add TigerBeetle to our stack.

#### [MODIFY] `docker-compose.yml`
- Add `tigerbeetle` service (Image: `ghcr.io/tigerbeetle/tigerbeetle`).
- Command to format and start the replica.

### 2. Core Domain (Ledger)
Define the financial models.

#### [NEW] `internal/core/domain/ledger.go`
- `AccountType` (Asset, Liability, Equity, Revenue, Expense).
- `Transaction` struct.

### 3. Adapter (TigerBeetle Client)
The bridge between Go and TB.

#### [NEW] `internal/adapter/tigerbeetle/client.go`
- Wrapper around `github.com/tigerbeetle/tigerbeetle-go`.
- Methods: `CreateAccounts`, `CreateTransfers`.

### 4. Integration points
Where the ledger gets updated.

#### [MODIFY] `internal/service/subscription.go` (or PaymentHandler)
- When `CreateSubscription` -> Create Customer "Accounts Receivable" (Asset) and Revenue (Revenue) accounts in TB.
- When `Invoice` is created -> Debit AR, Credit Revenue.
- When `Payment` is received -> Debit Cash, Credit AR.

## Verification Plan

### Automated Tests
1.  **Ledger Integrity**: Create a transfer and verify `LookupAccounts` returns updated balances.
2.  **Double-Entry**: Verify that every Debit has a matching Credit.

### Manual Verification
1.  Run `docker-compose up`.
2.  Perform a Checkout flow.
3.  Check TigerBeetle logs or query balances to see the movement of funds.
