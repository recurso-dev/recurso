package port

import (
	"context"
	"time"
)

// ExchangeRate represents a currency conversion rate
type ExchangeRate struct {
	From string  // Source currency (ISO 3-letter)
	To   string  // Target currency (ISO 3-letter)
	Rate float64 // Conversion rate (1 unit of From = Rate units of To)
}

// RateMetadata describes the provenance of exchange rates so that
// FX-normalized figures (e.g. MRR reporting) are auditable.
type RateMetadata struct {
	Source string    // "live" or "static-fallback"
	AsOf   time.Time // when the rates were last refreshed
}

// RateMetadataProvider is optionally implemented by ExchangeRateProvider
// implementations to expose where their rates came from and how fresh they are.
type RateMetadataProvider interface {
	RateMetadata() RateMetadata
}

// ExchangeRateProvider fetches and converts between currencies
type ExchangeRateProvider interface {
	GetRate(ctx context.Context, from, to string) (float64, error)
	Convert(ctx context.Context, amount int64, from, to string) (int64, float64, error) // returns (converted amount, rate, error)
	ListRates(ctx context.Context, baseCurrency string) ([]ExchangeRate, error)
}
