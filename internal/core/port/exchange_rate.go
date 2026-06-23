package port

import "context"

// ExchangeRate represents a currency conversion rate
type ExchangeRate struct {
	From string  // Source currency (ISO 3-letter)
	To   string  // Target currency (ISO 3-letter)
	Rate float64 // Conversion rate (1 unit of From = Rate units of To)
}

// ExchangeRateProvider fetches and converts between currencies
type ExchangeRateProvider interface {
	GetRate(ctx context.Context, from, to string) (float64, error)
	Convert(ctx context.Context, amount int64, from, to string) (int64, float64, error) // returns (converted amount, rate, error)
	ListRates(ctx context.Context, baseCurrency string) ([]ExchangeRate, error)
}
