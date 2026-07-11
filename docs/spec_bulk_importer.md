# Spec: Bulk Operations in Importer

## Objective
Enhance the existing subscriber migration importer (`cmd/import`) to support bulk updates and bulk cancellations. The current importer only supports creating new customers and subscriptions. We need the ability to process a CSV file to update existing records or cancel subscriptions en masse.

## Tech Stack
- Go 1.25+
- PostgreSQL
- `encoding/csv` (Standard library)

## Commands
Build: `make build`
Test: `go test ./cmd/import/...`
Lint: `golangci-lint run`

## Project Structure
```
cmd/
  import/
    main.go            → Add CLI flags for `--mode=update` or `--mode=cancel`
    updater.go         → Logic for processing update rows
    canceler.go        → Logic for processing cancel rows
```

## Code Style
```go
func (i *Importer) ProcessCancelSync(ctx context.Context, reader *csv.Reader) error {
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		
		subID := record[0] // Assuming column 0 is the subscription ID
		reason := record[1]
		
		// Attempt to cancel
		_, err = i.subService.Cancel(ctx, i.tenantID, uuid.MustParse(subID), false, reason, "")
		if err != nil {
			i.logger.Error("Failed to cancel subscription", "sub_id", subID, "error", err)
			i.stats.Failed++
			continue
		}
		i.stats.Success++
	}
	return nil
}
```

## Testing Strategy
- **Unit Tests**: Create mock CSV payloads in memory (using `strings.NewReader`) and pass them to the importer. Verify that the correct underlying service methods (`Update`, `Cancel`) are called the expected number of times.
- **End-to-End**: Run the compiled binary against a local test database with a test CSV and assert the database state post-execution.

## Boundaries
- **Always**: Provide a summary report at the end of the run (e.g., "Processed: 100, Success: 98, Failed: 2").
- **Ask first**: Before loading the entire CSV into memory. The importer must process files iteratively (streaming) to support large migrations (100k+ rows) without OOM crashing.
- **Never**: Halt the entire import process on a single row failure. Log the error and continue to the next row.

## Success Criteria
- [ ] Running `go run cmd/import/main.go --mode=cancel --file=cancels.csv` successfully cancels the subscriptions listed in the file.
- [ ] Running `go run cmd/import/main.go --mode=update --file=updates.csv` successfully updates customer/subscription metadata.
- [ ] Errors on individual rows are printed to stderr or a log file without stopping the script.

## Open Questions
- Should the `update` mode support updating the plan/pricing, or is it strictly for updating metadata (like customer names, billing addresses, tax IDs)?
