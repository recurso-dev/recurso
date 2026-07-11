# Spec: FX-Normalized Reporting

## Objective
Ensure Monthly Recurring Revenue (MRR) and general revenue analytics report accurately across different currencies by using real exchange rates. Currently, the analytics dashboard aggregates raw amounts without normalizing them to a base currency, which is inaccurate for multi-currency tenants.

## Tech Stack
- Go 1.25+
- PostgreSQL
- *Proposed*: OpenExchangeRates API (or similar)

## Commands
Build: `make build`
Test: `make test`
Test analytics: `go test ./internal/service/analytics_test.go`
Lint: `golangci-lint run`

## Project Structure
```
internal/
  core/
    domain/
      fx.go                  → Models for `ExchangeRate`
  service/
    fx_service.go            → Service to fetch and cache daily rates
    analytics.go             → Update MRR calculations to use fx_service
  adapter/
    fx/
      open_exchange.go       → Implementation of the FX provider
    db/
      exchange_rate_repo.go  → Persistence for historical rates
```

## Code Style
```go
// AnalyticsService MRR calculation
func (s *AnalyticsService) CalculateMRR(ctx context.Context, tenantID uuid.UUID, baseCurrency string) (int64, error) {
	subs, _ := s.subRepo.ListActive(ctx, tenantID)
	var totalMRR int64 = 0
	
	for _, sub := range subs {
		rate, _ := s.fxService.GetRate(ctx, sub.Currency, baseCurrency, time.Now())
		normalizedAmount := int64(float64(sub.PlanAmount) * rate)
		totalMRR += normalizedAmount
	}
	
	return totalMRR, nil
}
```

## Testing Strategy
- **Unit Tests**: Mock the FX service to return fixed rates (e.g., 1 USD = 80 INR). Write table-driven tests for the analytics calculations to ensure multi-currency subscriptions sum up correctly in the base currency.
- **Integration Tests**: Ensure the `open_exchange` adapter correctly parses the 3rd party API response.

## Boundaries
- **Always**: Cache daily exchange rates in the database. Do not hit the live FX API for every analytics query.
- **Ask first**: Before committing to a paid FX provider. We should evaluate if a free tier or a free open API (like Frankfurter) is sufficient for the MVP.
- **Never**: Overwrite historical exchange rates. A payment made in January should be reported using January's exchange rate for accurate historical revenue, even if viewed in July.

## Success Criteria
- [ ] A tenant with 1 subscription for $10 USD and 1 subscription for ₹800 INR reports an MRR of ~$20 USD (depending on the exact daily rate) when USD is selected as the base currency.
- [ ] The dashboard allows the tenant to select their preferred base currency for reporting.
- [ ] Exchange rates are fetched once per day via a cron worker and cached in PostgreSQL.

## Open Questions
- Do we want to use OpenExchangeRates, Frankfurter (free/open-source), or a manual static table for the MVP?
- Should the base currency be configurable per user, or per tenant?
