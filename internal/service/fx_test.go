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

// TestFXNormalizer_CrossExponentConversion proves conversion honors each
// currency's minor-unit exponent. FX rates are MAJOR-to-MAJOR (1 JPY = 0.0067
// USD); the stored amounts are MINOR units with per-currency exponents (JPY 0,
// USD 2, KWD 3). Multiplying minor units by the raw rate — the old behavior —
// is only correct when both exponents match: JPY→USD came out 100× too small,
// USD→JPY 100× too large, KWD→USD 10× too large, silently corrupting every
// normalized report (MRR, revenue segments, aging, dunning recovery, org
// consolidation) for zero- and three-decimal currencies.
func TestFXNormalizer_CrossExponentConversion(t *testing.T) {
	cases := []struct {
		name string
		amt  int64
		from string
		to   string
		rate float64
		want int64
	}{
		// ¥1000 (1000 minor, exp 0) at 0.0067 USD/JPY = $6.70 = 670 minor (old: 7).
		{"JPY to USD", 1000, "JPY", "USD", 0.0067, 670},
		// $10.00 (1000 minor, exp 2) at 149 JPY/USD = ¥1490 = 1490 minor (old: 149000).
		{"USD to JPY", 1000, "USD", "JPY", 149, 1490},
		// KWD 1.500 (1500 minor, exp 3) at 3.26 USD/KWD = $4.89 = 489 minor (old: 4890).
		{"KWD to USD", 1500, "KWD", "USD", 3.26, 489},
		// $4.89 (489 minor) at 0.3067 KWD/USD = KWD 1.4998 = 1500 minor (old: 150).
		{"USD to KWD", 489, "USD", "KWD", 0.3067, 1500},
		// Same-exponent pair is unchanged by the fix: €10.00 at 1.08 = $10.80.
		{"EUR to USD", 1000, "EUR", "USD", 1.08, 1080},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := newFXNormalizer(&fakeRateProvider{rate: tc.rate}, nil)
			got, _, err := n.convert(context.Background(), tc.amt, tc.from, tc.to)
			if err != nil {
				t.Fatalf("convert: %v", err)
			}
			if got != tc.want {
				t.Errorf("convert(%d %s -> %s @ %v) = %d, want %d",
					tc.amt, tc.from, tc.to, tc.rate, got, tc.want)
			}
		})
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
