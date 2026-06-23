package fx

import (
	"context"
	"fmt"
	"sync"

	"github.com/recur-so/recurso/internal/core/port"
)

// StaticRatesProvider stores admin-configured exchange rates in memory.
// Useful for development or when real-time rates are not needed.
type StaticRatesProvider struct {
	mu    sync.RWMutex
	rates map[string]float64 // key: "FROM:TO", value: rate
}

func NewStaticRatesProvider() *StaticRatesProvider {
	p := &StaticRatesProvider{
		rates: make(map[string]float64),
	}
	// Seed with common defaults (rates relative to USD)
	p.seedDefaults()
	return p
}

func (p *StaticRatesProvider) seedDefaults() {
	defaults := map[string]float64{
		"USD:INR": 83.50,
		"USD:EUR": 0.92,
		"USD:GBP": 0.79,
		"USD:JPY": 149.50,
		"USD:CAD": 1.36,
		"USD:AUD": 1.53,
		"USD:SGD": 1.34,
		"INR:USD": 1.0 / 83.50,
		"EUR:USD": 1.0 / 0.92,
		"GBP:USD": 1.0 / 0.79,
		"JPY:USD": 1.0 / 149.50,
		"CAD:USD": 1.0 / 1.36,
		"AUD:USD": 1.0 / 1.53,
		"SGD:USD": 1.0 / 1.34,
	}
	for k, v := range defaults {
		p.rates[k] = v
	}
}

// SetRate allows admin to configure a rate
func (p *StaticRatesProvider) SetRate(from, to string, rate float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rates[from+":"+to] = rate
}

func (p *StaticRatesProvider) GetRate(ctx context.Context, from, to string) (float64, error) {
	if from == to {
		return 1.0, nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Direct rate
	key := from + ":" + to
	if rate, ok := p.rates[key]; ok {
		return rate, nil
	}

	// Try cross-rate via USD
	fromUSD, hasFrom := p.rates[from+":USD"]
	usdTo, hasTo := p.rates["USD:"+to]
	if hasFrom && hasTo {
		return fromUSD * usdTo, nil
	}

	return 0, fmt.Errorf("exchange rate not available: %s -> %s", from, to)
}

func (p *StaticRatesProvider) Convert(ctx context.Context, amount int64, from, to string) (int64, float64, error) {
	rate, err := p.GetRate(ctx, from, to)
	if err != nil {
		return 0, 0, err
	}
	converted := int64(float64(amount) * rate)
	return converted, rate, nil
}

func (p *StaticRatesProvider) ListRates(ctx context.Context, baseCurrency string) ([]port.ExchangeRate, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var rates []port.ExchangeRate
	prefix := baseCurrency + ":"
	for key, rate := range p.rates {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			to := key[len(prefix):]
			rates = append(rates, port.ExchangeRate{
				From: baseCurrency,
				To:   to,
				Rate: rate,
			})
		}
	}
	return rates, nil
}
