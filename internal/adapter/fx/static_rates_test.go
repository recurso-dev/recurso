package fx

import (
	"context"
	"math"
	"testing"
)

func TestStaticRates_DirectAndCrossRate(t *testing.T) {
	p := NewStaticRatesProvider()
	ctx := context.Background()

	// Identity
	rate, err := p.GetRate(ctx, "USD", "USD")
	if err != nil || rate != 1.0 {
		t.Errorf("USD->USD = %v, %v; want 1.0, nil", rate, err)
	}

	// Direct seeded pair
	rate, err = p.GetRate(ctx, "USD", "EUR")
	if err != nil || rate != 0.92 {
		t.Errorf("USD->EUR = %v, %v; want 0.92, nil", rate, err)
	}

	// Cross-rate via USD: EUR -> GBP = (EUR->USD) * (USD->GBP)
	rate, err = p.GetRate(ctx, "EUR", "GBP")
	if err != nil {
		t.Fatalf("EUR->GBP: %v", err)
	}
	want := (1.0 / 0.92) * 0.79
	if math.Abs(rate-want) > 1e-9 {
		t.Errorf("EUR->GBP = %v, want %v", rate, want)
	}

	// Unknown pair
	if _, err := p.GetRate(ctx, "USD", "XYZ"); err == nil {
		t.Error("expected error for unknown currency pair")
	}
}

func TestStaticRates_Metadata(t *testing.T) {
	p := NewStaticRatesProvider()
	meta := p.RateMetadata()
	if meta.Source != "static-fallback" {
		t.Errorf("Source = %q, want static-fallback", meta.Source)
	}
	if meta.AsOf.IsZero() {
		t.Error("AsOf should be set at seed time")
	}

	before := meta.AsOf
	p.SetRate("USD", "XYZ", 2.0)
	if got := p.RateMetadata().AsOf; got.Before(before) {
		t.Errorf("AsOf should advance on SetRate: %v < %v", got, before)
	}
}

func TestOXR_Metadata(t *testing.T) {
	p := NewOpenExchangeRatesProvider("test-key")
	meta := p.RateMetadata()
	if meta.Source != "live" {
		t.Errorf("Source = %q, want live", meta.Source)
	}
	if !meta.AsOf.IsZero() {
		t.Errorf("AsOf should be zero before first fetch, got %v", meta.AsOf)
	}
}
