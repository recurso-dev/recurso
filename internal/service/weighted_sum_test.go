package service

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// fakeWeightedRepo replays a carry-forward level and a set of timestamped delta
// events to the weighted-sum aggregation.
type fakeWeightedRepo struct {
	prior  int64
	events []struct {
		delta int64
		ts    time.Time
	}
}

func (f *fakeWeightedRepo) SumQuantityBefore(_ context.Context, _ uuid.UUID, _ string, _ time.Time) (int64, error) {
	return f.prior, nil
}

func (f *fakeWeightedRepo) StreamEventsForMetric(_ context.Context, _ uuid.UUID, _ string, _, _ time.Time, fn func(int64, time.Time, map[string]string) error) error {
	for _, e := range f.events {
		if err := fn(e.delta, e.ts, nil); err != nil {
			return err
		}
	}
	return nil
}

func evt(delta int64, ts time.Time) struct {
	delta int64
	ts    time.Time
} {
	return struct {
		delta int64
		ts    time.Time
	}{delta, ts}
}

var wsBase = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

func day(n int) time.Time { return wsBase.AddDate(0, 0, n) }

func TestAggregateWeightedSum(t *testing.T) {
	end := day(30) // 30-day period

	cases := []struct {
		name  string
		prior int64
		evs   []struct {
			delta int64
			ts    time.Time
		}
		want *big.Rat
	}{
		{
			// Founder's canonical example: 5 seats for 15 days, 10 for 15 days.
			// Deltas within the period: +5 at start, +5 at day 15.
			name:  "delta within period (5×15 + 10×15)/30 = 7.5",
			prior: 0,
			evs: []struct {
				delta int64
				ts    time.Time
			}{evt(5, day(0)), evt(5, day(15))},
			want: big.NewRat(15, 2),
		},
		{
			// Same result via carry-forward: level 5 before the period, +5 at day 15.
			name:  "carry-forward starting level",
			prior: 5,
			evs: []struct {
				delta int64
				ts    time.Time
			}{evt(5, day(15))},
			want: big.NewRat(15, 2),
		},
		{
			// A resource active all period with no changes: average == the level.
			name:  "constant carried level, no in-period events",
			prior: 8,
			evs:   nil,
			want:  big.NewRat(8, 1),
		},
		{
			// Fractional average: 10 for the first 10 days, 0 for the rest.
			// (10×10 + 0×20)/30 = 100/30 = 10/3.
			name:  "fractional average 10/3",
			prior: 0,
			evs: []struct {
				delta int64
				ts    time.Time
			}{evt(10, day(0)), evt(-10, day(10))},
			want: big.NewRat(10, 3),
		},
		{
			// Net-negative level clamps to 0 (usage can't bill negative).
			name:  "negative level clamps to zero",
			prior: 0,
			evs: []struct {
				delta int64
				ts    time.Time
			}{evt(-5, day(0))},
			want: new(big.Rat),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := &fakeWeightedRepo{prior: c.prior, events: c.evs}
			got, err := AggregateWeightedSum(context.Background(), repo, uuid.New(), "seats", wsBase, end)
			if err != nil {
				t.Fatalf("aggregate: %v", err)
			}
			if got.Cmp(c.want) != 0 {
				t.Fatalf("want %s, got %s", c.want.RatString(), got.RatString())
			}
		})
	}
}

// TestAggregateWeightedSum_PricesExactly ties the fractional average to money:
// 10/3 average seats at ₹30/seat is exactly ₹100.00 — the quantity is never
// pre-rounded (3 → ₹90, 4 → ₹120 would both be wrong).
func TestAggregateWeightedSum_PricesExactly(t *testing.T) {
	repo := &fakeWeightedRepo{prior: 0, events: []struct {
		delta int64
		ts    time.Time
	}{evt(10, day(0)), evt(-10, day(10))}}
	avg, err := AggregateWeightedSum(context.Background(), repo, uuid.New(), "seats", wsBase, day(30))
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	amount, err := RateChargeRat(domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: "30"}, avg)
	if err != nil {
		t.Fatalf("rate: %v", err)
	}
	if amount != 10000 {
		t.Fatalf("10/3 avg seats @ ₹30 want ₹100.00 (10000), got %d", amount)
	}
}

// TestAggregateWeightedSum_EmptyPeriod returns 0 for a zero-length period.
func TestAggregateWeightedSum_EmptyPeriod(t *testing.T) {
	repo := &fakeWeightedRepo{prior: 5}
	got, err := AggregateWeightedSum(context.Background(), repo, uuid.New(), "seats", wsBase, wsBase)
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if got.Sign() != 0 {
		t.Fatalf("empty period want 0, got %s", got.RatString())
	}
}
