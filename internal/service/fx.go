package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// FXSnapshot records the exchange rates used to normalize a report, plus
// their provenance, so normalized figures are auditable.
type FXSnapshot struct {
	// Rates maps source currency -> rate into the reporting currency.
	Rates map[string]float64 `json:"rates"`
	// Source is "live" (fetched from the FX provider) or "static-fallback"
	// (seeded/admin-configured rates were used for at least one conversion).
	Source string `json:"source"`
	// AsOf is when the rates were last refreshed.
	AsOf time.Time `json:"as_of"`
}

// fxNormalizer converts per-currency amounts into a single reporting
// currency, tracking every rate it used and whether the static fallback
// provider had to be consulted.
type fxNormalizer struct {
	provider port.ExchangeRateProvider
	fallback port.ExchangeRateProvider

	rates        map[string]float64
	usedFallback bool
}

func newFXNormalizer(provider, fallback port.ExchangeRateProvider) *fxNormalizer {
	return &fxNormalizer{
		provider: provider,
		fallback: fallback,
		rates:    make(map[string]float64),
	}
}

// rate resolves the conversion rate from -> to, preferring the primary
// provider and falling back to the static provider on error.
func (n *fxNormalizer) rate(ctx context.Context, from, to string) (float64, error) {
	if from == to {
		n.rates[from] = 1.0
		return 1.0, nil
	}

	if n.provider != nil {
		rate, err := n.provider.GetRate(ctx, from, to)
		if err == nil {
			n.rates[from] = rate
			return rate, nil
		}
		if n.fallback == nil {
			return 0, err
		}
	}

	if n.fallback == nil {
		return 0, fmt.Errorf("no FX provider configured for %s -> %s", from, to)
	}

	rate, err := n.fallback.GetRate(ctx, from, to)
	if err != nil {
		return 0, err
	}
	n.usedFallback = true
	n.rates[from] = rate
	return rate, nil
}

// convert converts a minor-unit amount from -> to, rounding half away from zero.
func (n *fxNormalizer) convert(ctx context.Context, amount int64, from, to string) (int64, float64, error) {
	rate, err := n.rate(ctx, from, to)
	if err != nil {
		return 0, 0, err
	}
	return int64(math.Round(float64(amount) * rate)), rate, nil
}

// snapshot returns the audit trail for all conversions performed so far.
func (n *fxNormalizer) snapshot() *FXSnapshot {
	source := "unknown"
	asOf := time.Now().UTC()

	active := n.provider
	if n.usedFallback {
		// At least one conversion relied on static rates; flag the whole
		// snapshot so consumers know the total is not fully live.
		source = "static-fallback"
		active = n.fallback
	}

	if meta, ok := active.(port.RateMetadataProvider); ok {
		m := meta.RateMetadata()
		if !n.usedFallback {
			source = m.Source
		}
		if !m.AsOf.IsZero() {
			asOf = m.AsOf
		}
	}

	return &FXSnapshot{
		Rates:  n.rates,
		Source: source,
		AsOf:   asOf,
	}
}
