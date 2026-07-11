# Spec: Recovery Attribution

## Objective
Measure exactly what the smart dunning engine actually recovers and expose it via analytics. When a failed invoice is later collected via dunning, we must record the recovered amount, the number of attempts it took, and the specific strategy used. This is a prerequisite for proving the ROI of the dunning system.

## Tech Stack
- Go 1.25+
- PostgreSQL

## Commands
Build: `make build`
Test: `make test`
Test analytics: `go test ./internal/service/dunning_analytics_test.go`
Lint: `golangci-lint run`

## Project Structure
```
internal/
  core/
    domain/
      dunning_analytics.go   → Add `RecoveredPayment` model
  service/
    dunning_recovery.go      → Logic to attribute a paid invoice to dunning
  adapter/
    db/
      recovered_payment.go   → Persistence for recovered payments
```

## Code Style
```go
// RecordIfRecovered checks if a newly paid invoice was previously in dunning
func (s *RecoveryRecorder) RecordIfRecovered(ctx context.Context, invoice *domain.Invoice) error {
	dunningRecord, err := s.dunningRepo.GetByInvoiceID(ctx, invoice.ID)
	if err != nil || dunningRecord == nil {
		return nil // Not in dunning, no recovery to record
	}
	
	recovery := &domain.RecoveredPayment{
		ID:              uuid.New(),
		TenantID:        invoice.TenantID,
		InvoiceID:       invoice.ID,
		Amount:          invoice.Total,
		Currency:        invoice.Currency,
		Attempts:        dunningRecord.AttemptCount,
		StrategyUsed:    dunningRecord.StrategyCode,
		RecoveredAt:     *invoice.PaidAt,
	}
	
	return s.recoveryRepo.Create(ctx, recovery)
}
```

## Testing Strategy
- **Unit Tests**: Test the `RecordIfRecovered` logic. It should successfully create a `RecoveredPayment` record if the invoice was in dunning, and it should silently return (no-op) if the invoice was paid on the first attempt (never entered dunning).
- **Data Integrity**: Ensure the operation is idempotent. If `RecordIfRecovered` is called twice for the same invoice, it should not double-count the recovered revenue.

## Boundaries
- **Always**: Execute the recovery recording asynchronously or as a non-blocking background task after the invoice is marked paid. It should not block the main payment processing thread.
- **Ask first**: Before creating complex attribution models (e.g., fractional attribution). Stick to simple last-touch attribution for the MVP.
- **Never**: Double-count revenue if a single invoice is somehow marked paid, refunded, and marked paid again.

## Success Criteria
- [ ] Marking an invoice as paid that has `attempt_count > 1` creates a `RecoveredPayment` record.
- [ ] A new endpoint `GET /v1/analytics/recovery` returns the sum of recovered revenue for a given date range.

## Open Questions
- Do we want to expose this data in the main dashboard immediately, or just via the API for now?
