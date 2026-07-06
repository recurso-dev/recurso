package tax

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"
)

// SalesTaxProvider is the narrow port a US sales-tax rate service (TaxJar,
// Avalara, ...) must satisfy. It is deliberately smaller than port.TaxEngine:
// providers answer "what is the tax on this amount shipped from A to B",
// nothing more. Implementations live under internal/adapter/taxprovider.
type SalesTaxProvider interface {
	// Name identifies the provider ("taxjar") for invoice notes and logs.
	Name() string
	// LookupSalesTax returns the applicable rate and tax amount for the
	// query. Errors are returned as-is; callers decide degradation policy.
	LookupSalesTax(ctx context.Context, q *SalesTaxQuery) (*SalesTaxResult, error)
}

// SalesTaxQuery describes one taxable sale for rate lookup.
type SalesTaxQuery struct {
	FromCountry string // Seller country (ISO 2-letter, "US")
	FromState   string // Seller state ("CA")
	FromZip     string // Seller ZIP (optional; provider account nexus applies if empty)
	ToCountry   string // Buyer country (ISO 2-letter)
	ToState     string // Buyer state
	ToZip       string // Buyer ZIP
	Amount      int64  // Sale amount in the lowest currency unit (cents)
	Currency    string // ISO 3-letter code (US sales tax providers assume USD)
}

// SalesTaxResult is a provider's answer for one query.
type SalesTaxResult struct {
	Rate         float64 // Combined effective rate (0.0 to 1.0)
	TaxAmount    int64   // Tax to collect, lowest currency unit (cents)
	Jurisdiction string  // Cheap human-readable breakdown, e.g. "US/CA/LOS ANGELES"
	HasNexus     bool    // Whether the provider says the seller has nexus in the buyer's state
}

// DefaultSalesTaxRateTTL bounds how long a (state, zip) rate is reused before
// a fresh provider lookup. 24h keeps API spend flat across an invoice run
// while staying current enough for rate changes (which are effective-dated
// weeks in advance).
const DefaultSalesTaxRateTTL = 24 * time.Hour

// CachedSalesTaxProvider decorates a SalesTaxProvider with an in-memory
// per-(state,zip) rate cache. Cache hits recompute the tax amount locally
// from the cached rate, so repeat invoices to the same jurisdiction cost
// zero API calls inside the TTL.
//
// The cache lives in the provider (not the engine) because tax engines are
// constructed per invoice by the factory; the provider is the long-lived
// object, so this is where cached state survives.
type CachedSalesTaxProvider struct {
	inner SalesTaxProvider
	ttl   time.Duration

	mu      sync.Mutex
	entries map[string]cachedSalesTaxRate

	now func() time.Time // injectable clock for tests
}

type cachedSalesTaxRate struct {
	rate         float64
	jurisdiction string
	hasNexus     bool
	expiresAt    time.Time
}

// NewCachedSalesTaxProvider wraps inner with a rate cache. A non-positive
// ttl falls back to DefaultSalesTaxRateTTL.
func NewCachedSalesTaxProvider(inner SalesTaxProvider, ttl time.Duration) *CachedSalesTaxProvider {
	if ttl <= 0 {
		ttl = DefaultSalesTaxRateTTL
	}
	return &CachedSalesTaxProvider{
		inner:   inner,
		ttl:     ttl,
		entries: make(map[string]cachedSalesTaxRate),
		now:     time.Now,
	}
}

// Name reports the inner provider's name (the cache is transparent).
func (c *CachedSalesTaxProvider) Name() string { return c.inner.Name() }

// LookupSalesTax serves from cache when a fresh (state, zip) rate exists,
// otherwise delegates to the inner provider and caches the returned rate.
// Only successful lookups are cached; errors always pass through.
func (c *CachedSalesTaxProvider) LookupSalesTax(ctx context.Context, q *SalesTaxQuery) (*SalesTaxResult, error) {
	key := salesTaxCacheKey(q)

	c.mu.Lock()
	entry, ok := c.entries[key]
	if ok && c.now().Before(entry.expiresAt) {
		c.mu.Unlock()
		return &SalesTaxResult{
			Rate:         entry.rate,
			TaxAmount:    int64(math.Round(float64(q.Amount) * entry.rate)),
			Jurisdiction: entry.jurisdiction,
			HasNexus:     entry.hasNexus,
		}, nil
	}
	c.mu.Unlock()

	res, err := c.inner.LookupSalesTax(ctx, q)
	if err != nil || res == nil {
		return res, err
	}

	c.mu.Lock()
	c.entries[key] = cachedSalesTaxRate{
		rate:         res.Rate,
		jurisdiction: res.Jurisdiction,
		hasNexus:     res.HasNexus,
		expiresAt:    c.now().Add(c.ttl),
	}
	c.mu.Unlock()
	return res, nil
}

// salesTaxCacheKey keys rates by destination jurisdiction. Rates depend on
// where the sale lands (destination-sourced in most states), so (state, zip)
// is the right granularity; amount is deliberately excluded.
func salesTaxCacheKey(q *SalesTaxQuery) string {
	return strings.ToUpper(strings.TrimSpace(q.ToState)) + "|" + strings.TrimSpace(q.ToZip)
}
