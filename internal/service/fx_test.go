package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// fakeRateProvider is a configurable ExchangeRateProvider for the FX tests.
type fakeRateProvider struct {
	port.ExchangeRateProvider
	rate float64
	err  error
}

func (p *fakeRateProvider) GetRate(_ context.Context, _, _ string) (float64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return p.rate, nil
}

// metaRateProvider adds RateMetadata to the fake (separate type so the plain
// fake does NOT accidentally satisfy RateMetadataProvider).
type metaRateProvider struct {
	fakeRateProvider
	meta port.RateMetadata
}

func (p *metaRateProvider) RateMetadata() port.RateMetadata { return p.meta }

func TestFXNormalizer_SameCurrencyIsIdentity(t *testing.T) {
	n := newFXNormalizer(&fakeRateProvider{rate: 999}, nil)
	got, rate, err := n.convert(context.Background(), 12345, "USD", "USD")
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if got != 12345 || rate != 1.0 {
		t.Fatalf("same-currency convert = (%d, %v), want (12345, 1.0)", got, rate)
	}
}

func TestFXNormalizer_ConvertRoundsHalfAwayFromZero(t *testing.T) {
	cases := []struct {
		amount int64
		rate   float64
		want   int64
	}{
		{9200, 1.25, 11500}, // exact
		{3, 0.5, 2},         // 1.5 rounds away from zero -> 2
		{100, 0.833, 83},    // 83.3 -> 83
		{100, 0.836, 84},    // 83.6 -> 84
	}
	for _, c := range cases {
		n := newFXNormalizer(&fakeRateProvider{rate: c.rate}, nil)
		got, _, err := n.convert(context.Background(), c.amount, "EUR", "USD")
		if err != nil {
			t.Fatalf("convert(%d @ %v): %v", c.amount, c.rate, err)
		}
		if got != c.want {
			t.Errorf("convert(%d @ %v) = %d, want %d", c.amount, c.rate, got, c.want)
		}
	}
}

func TestFXNormalizer_FallsBackOnProviderError(t *testing.T) {
	provider := &fakeRateProvider{err: errors.New("live provider down")}
	fallback := &metaRateProvider{
		fakeRateProvider: fakeRateProvider{rate: 1.1},
		meta:             port.RateMetadata{Source: "static-fallback", AsOf: time.Unix(1_700_000_000, 0)},
	}
	n := newFXNormalizer(provider, fallback)

	got, rate, err := n.convert(context.Background(), 10000, "EUR", "USD")
	if err != nil {
		t.Fatalf("convert with fallback: %v", err)
	}
	if got != 11000 || rate != 1.1 {
		t.Fatalf("fallback convert = (%d, %v), want (11000, 1.1)", got, rate)
	}

	snap := n.snapshot()
	if snap.Source != "static-fallback" {
		t.Errorf("snapshot source = %q, want static-fallback (a fallback conversion must taint the report)", snap.Source)
	}
	if snap.Rates["EUR"] != 1.1 {
		t.Errorf("snapshot rates[EUR] = %v, want 1.1", snap.Rates["EUR"])
	}
}

func TestFXNormalizer_NoProviderIsError(t *testing.T) {
	n := newFXNormalizer(nil, nil)
	if _, _, err := n.convert(context.Background(), 100, "EUR", "USD"); err == nil {
		t.Fatal("convert with no providers: expected error, got nil")
	}
}
