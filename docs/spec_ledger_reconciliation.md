# Spec: TigerBeetle Ledger Reconciliation

## Objective
Upgrade the daily ledger reconciliation job to effectively compare the TigerBeetle financial ledger against the PostgreSQL database. Currently, this is skipped because the TigerBeetle client lacks a convenient enumeration API, requiring pagination of `GetAccountTransfers` or timestamp-windowed queries to achieve the comparison.

## Tech Stack
- Go 1.25+
- TigerBeetle Go Client
- PostgreSQL

## Commands
Build: `make build`
Test: `make test`
Run specifically: `go test ./internal/service/reconciliation_test.go`
Lint: `golangci-lint run`

## Project Structure
```
internal/
  service/
    reconciliation.go       → Implement TB vs PG comparison logic
  adapter/
    tigerbeetle/
      ledger.go             → Add paginated transfer querying functions
```

## Code Style
```go
// FetchTransfers paginates through TigerBeetle transfers for an account
func (l *LedgerService) FetchTransfers(ctx context.Context, accountID uint128, startTimestamp uint64, limit int) ([]tigerbeetle.Transfer, error) {
	// Implement windowed query using TigerBeetle client API
	filter := tigerbeetle.AccountFilter{
		AccountId: accountID,
		TimestampMin: startTimestamp,
		Limit: uint32(limit),
	}
	return l.client.GetAccountTransfers(filter)
}
```

## Testing Strategy
- **Unit Tests**: Mock the TigerBeetle client to return a known set of transfers, and mock the PostgreSQL repository. Verify that the reconciliation service correctly flags missing, orphaned, or mismatched transactions.
- **Integration Tests**: Spin up a real TigerBeetle instance via testcontainers or docker-compose, write simulated data to both DBs (with intentional discrepancies), and ensure the reconciliation job detects them.

## Boundaries
- **Always**: Handle large datasets gracefully. The reconciliation job must stream or paginate data to avoid OOM (Out Of Memory) errors when comparing millions of transactions.
- **Ask first**: Before significantly changing the schema or account structure in TigerBeetle to support this.
- **Never**: Mutate data during the reconciliation read pass. Discrepancies should be alerted/logged, not automatically "fixed" without a human audit trail.

## Success Criteria
- [ ] The daily `GET /v1/finance/reconciliation` job successfully compares TigerBeetle balances/transfers against PostgreSQL invoices/payments.
- [ ] Any discrepancy (e.g., a payment recorded in PostgreSQL but missing in TigerBeetle) is logged and returned in the API response.
- [ ] The process runs efficiently without loading the entire ledger into memory.

## Open Questions
- What alerting mechanism should be used when the reconciliation job finds a discrepancy (e.g., Slack, Email, PagerDuty)?
