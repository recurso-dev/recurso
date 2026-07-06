# Spec: Entitlement Engine v1

## Objective
Build an Entitlement Engine that allows SaaS companies to define feature grants (booleans) and usage limits (quotas) at the Plan level. Customers can then query their effective entitlements based on their active subscriptions, making Recurso load-bearing in the customer's application beyond just accounting.

## Tech Stack
- Go 1.25+
- PostgreSQL
- Optional: Redis (if high-throughput caching is required)

## Commands
Build: `make build`
Test: `make test`
Test specific package: `go test ./internal/service/entitlement_test.go`
Lint: `golangci-lint run`

## Project Structure
```
internal/
  core/
    domain/
      entitlement.go        → `Feature`, `Entitlement`, `EffectiveEntitlement` models
  service/
    entitlement.go          → Logic to compute effective entitlements
  adapter/
    handler/
      entitlement.go        → High-performance `GET /v1/entitlements` endpoint
    db/
      entitlement_repository.go
```

## Code Style
```go
// CheckAccess determines if a customer has access to a specific feature
func (s *EntitlementService) CheckAccess(ctx context.Context, customerID uuid.UUID, featureCode string) (bool, error) {
	entitlements, err := s.GetEffectiveEntitlements(ctx, customerID)
	if err != nil {
		return false, err
	}
	
	for _, e := range entitlements {
		if e.FeatureCode == featureCode {
			return e.HasAccess, nil
		}
	}
	return false, nil
}
```

## Testing Strategy
- **Unit Tests**: Thoroughly test `GetEffectiveEntitlements` with various edge cases: overlapping subscriptions, expired subscriptions, paused subscriptions, and trial periods.
- **Integration Tests**: Verify database persistence of Features and Plan-Feature mappings.
- **Performance Benchmarking**: The entitlement check endpoint will be called frequently. Write `go test -bench` benchmarks to ensure the resolution time is <5ms.

## Boundaries
- **Always**: Cache the results of entitlement checks if they are requested frequently. Use appropriate cache invalidation when a subscription changes status.
- **Ask first**: Before mandating Redis as a hard dependency. Can we use an in-memory local cache (like groupcache or bigcache) for the MVP?
- **Never**: Return a 500 error on the entitlement check endpoint if the DB is down; fallback to gracefully cached data to prevent bringing down the customer's application.

## Success Criteria
- [ ] Admins can create Features and attach them to Plans with specific limits (e.g. `max_users = 10`, `premium_support = true`).
- [ ] Customers can query `GET /v1/entitlements/:feature` and get a boolean or integer response indicating their access level.
- [ ] If a customer upgrades their plan, their effective entitlements update immediately.
- [ ] The API responds in <10ms for entitlement checks.

## Open Questions
- Will this run entirely within the Go backend as a simple boolean/limit checker, or do we need an externalized high-performance cache (like Redis) for checking entitlements on every request?
- How do we handle usage tracking against the defined limits (e.g. counting current users vs `max_users`)?
