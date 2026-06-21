# Implementation Plan - Phase 1: The Iron Core

## Goal Description
Establish the foundational "Iron Core" of Recurso. This includes setting up the project structure, integrating the TigerBeetle ledger for immutable financial tracking, and implementing the core subscription domain logic (Plans, Customers, Proration). We will use **Go** for the service layer and **TigerBeetle** for the ledger.

## User Review Required
> [!IMPORTANT]
> **TigerBeetle Dependency**: reliable execution requires running a TigerBeetle cluster. For local development, we will use a dockerized TigerBeetle instance.
> **Database Choice**: We are proceeding with PostgreSQL for metadata (System of Reference) as defined in the research.

## Proposed Changes

### Project Structure
We will adopt a standard Go project layout:
```text
recurso/
├── cmd/
│   └── api/            # Main entry point
├── internal/
│   ├── core/           # Domain logic (Platform agnostic)
│   │   ├── ledger/     # Ledger interfaces & models
│   │   └── subscription/ # Subscription state machine
│   ├── adapter/        # Interface implementations
│   │   ├── tigerbeetle/# TigerBeetle client implementation
│   │   └── postgres/   # Postgres repositories
│   └── service/        # Application services (Use cases)
├── pkg/                # Public shared code
└── api/                # Proto/OpenAPI definitions
```

### [NEW] Core Module
#### [NEW] [go.mod](file:///Users/swapnull/Documents/Workspace/recur-so/go.mod)
- Initialize new Go module `github.com/recur-so/recurso`.

### [NEW] Ledger Service (TigerBeetle)
#### [NEW] [internal/core/ledger/types.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/core/ledger/types.go)
- Define `Account` and `Transfer` structs matching TigerBeetle primitives.
- Define `AccountType` enums (Asset, Liability, Revenue, Expense).

#### [NEW] [internal/adapter/tigerbeetle/client.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/adapter/tigerbeetle/client.go)
- Implement the client to connect to the TigerBeetle cluster.
- Implement `CreateAccounts` and `CreateTransfers`.

### [NEW] Subscription Domain
#### [NEW] [internal/core/subscription/model.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/core/subscription/model.go)
- Define `Plan`, `Customer`, `Subscription` structs.
- Implement `BillingCycle` calculation logic.

#### [NEW] [internal/core/subscription/proration.go](file:///Users/swapnull/Documents/Workspace/recur-so/internal/core/subscription/proration.go)
- Implement `CalculateProration` function (Exact-time, integer math).

## Verification Plan

### Automated Tests
- **Unit Tests**:
    - `go test ./internal/core/...` to verify Proration logic and State Machine transitions.
    - Test the "Exact-time Proration" with various scenarios (Upgrade halfway, Downgrade near end).
- **Integration Tests**:
    - Spin up TigerBeetle container.
    - Verify `CreateAccount` and `CreateTransfer` actually persist to ledger.
    - Verify Double-Entry constraint (Sum of debits == Sum of credits).

### Manual Verification
- None for this backend-focused phase.
