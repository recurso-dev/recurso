# Spec: US Sales Tax Integration

## Objective
Replace the current 0% stub for US sales tax calculation with a real, jurisdiction-aware integration. The integration should dynamically calculate US state-by-state sales tax for SaaS products based on the seller's nexus and the buyer's billing address.

## Tech Stack
- Go 1.25+
- PostgreSQL
- Stripe/Razorpay (payment handling)
- *Proposed*: TaxJar or Avalara API client

## Commands
Build: `make build`
Test: `make test`
Test specific package: `go test ./internal/adapter/taxprovider/...`
Lint: `golangci-lint run`
Dev: `make run`

## Project Structure
```
internal/
  core/
    domain/             → Add US tax structures (Nexus, AddressValidation)
    port/               → `TaxProvider` interface
  service/              → Update `TaxResolver` to route US requests
  adapter/
    taxprovider/
      taxjar.go         → TaxJar implementation of `TaxProvider`
```

## Code Style
```go
// CalculateTax implements port.TaxProvider
func (t *TaxJarAdapter) CalculateTax(ctx context.Context, req domain.TaxRequest) (*domain.TaxResult, error) {
	if req.BuyerAddress.Country != "US" {
		return nil, errors.New("unsupported country for US tax provider")
	}
	
	// API call to TaxJar
	res, err := t.client.TaxForOrder(ctx, mapRequest(req))
	if err != nil {
		return nil, fmt.Errorf("tax calculation failed: %w", err)
	}

	return &domain.TaxResult{
		TotalTax: res.AmountToCollect,
		Rate:     res.Rate,
	}, nil
}
```

## Testing Strategy
- **Unit Tests**: Implement table-driven tests in `tax_resolver_test.go` to ensure routing rules direct US addresses to the new provider while keeping Indian addresses on the GST engine. Use mocks for the `TaxProvider` port.
- **Integration Tests**: A mocked HTTP server (`httptest.Server`) returning mock TaxJar/Avalara responses.
- **Coverage**: Require 90%+ coverage on `internal/adapter/taxprovider/`.

## Boundaries
- **Always**: Handle API rate limits and connection timeouts gracefully; fallback to a defined error state that does not silently charge 0% tax.
- **Ask first**: Before adding a new large dependency (like an official heavy SDK). Prefer using the standard `net/http` library to interact with simple REST APIs if the official SDK pulls in too many transitive dependencies.
- **Never**: Store API keys in plaintext in the database or hardcode them; always use environment variables (`TAXJAR_API_KEY`).

## Success Criteria
- [ ] Any invoice generated for a US customer correctly fetches the sales tax amount via the external provider.
- [ ] If the provider fails/times out, the invoice creation rolls back or transitions to a "Tax Calculation Failed" state rather than completing with 0% tax.
- [ ] Unit tests verify routing: IN -> GST engine, US -> US Tax engine, EU -> VAT table.

## Open Questions
- Are we committing to **TaxJar** or **Avalara** as the primary integration for the MVP?
- Should we cache tax rates for zip codes, or always perform a live lookup on every invoice generation?
