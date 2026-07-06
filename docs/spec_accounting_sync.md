# Spec: Accounting Sync Efficiency

## Objective
Implement "changed-since" dirty tracking for the accounting synchronization jobs (QuickBooks and Xero). Currently, the daily sync re-pushes every mapped entity, which is inefficient and scales poorly. The sync should only push entities that have changed since the last successful sync.

## Tech Stack
- Go 1.25+
- PostgreSQL (for `updated_at` timestamps or an outbox table)
- Existing accounting provider interfaces (`internal/adapter/accounting`)

## Commands
Build: `make build`
Test: `make test`
Test sync logic: `go test ./internal/service/accounting_sync_test.go`
Lint: `golangci-lint run`

## Project Structure
```
internal/
  core/
    domain/
      accounting.go         → Add `LastSyncTime` to `AccountingConnection`
  service/
    accounting_sync.go      → Update sync logic to use `changed-since`
  adapter/
    db/
      accounting_connection_repository.go → Persist `LastSyncTime`
```

## Code Style
```go
func (s *AccountingSyncService) SyncInvoices(ctx context.Context, tenantID uuid.UUID) error {
	connection, err := s.connectionRepo.GetByTenant(ctx, tenantID)
	// ...
	
	lastSync := connection.LastInvoiceSyncTime
	if lastSync == nil {
		lastSync = &time.Time{} // Beginning of time for first sync
	}

	// Fetch only invoices updated since last sync
	invoices, err := s.invoiceRepo.ListUpdatedSince(ctx, tenantID, *lastSync)
	// ...
}
```

## Testing Strategy
- **Unit Tests**: Mock the `InvoiceRepository` to return specific sets of invoices based on the timestamp. Verify that the `AccountingAdapter` is only called for the changed invoices.
- **Data Integrity Tests**: Ensure that if the sync fails midway, the `LastSyncTime` is NOT updated, or only updated to the timestamp of the last successfully synced batch.

## Boundaries
- **Always**: Paginate the fetched entities; do not load thousands of changed invoices into memory at once.
- **Ask first**: Before adding complex event-sourcing or Debezium/Kafka outbox patterns. We should start with a simple timestamp-based (`updated_at`) query unless it proves insufficient.
- **Never**: Mark a sync as successful if the provider API returns a 5xx error.

## Success Criteria
- [ ] A sync job run twice in a row (with no data changes in between) results in 0 API calls to QuickBooks/Xero during the second run.
- [ ] Modifying an invoice updates its `updated_at` timestamp, and the next sync pushes only that invoice.
- [ ] Sync latency drops from O(N) where N is all data, to O(M) where M is modified data.

## Open Questions
- For "dirty tracking", should we rely purely on database timestamps (`updated_at`), or introduce a lightweight sync outbox table to guarantee no events are missed in edge cases (like clock skew)?
