package service

import (
	"context"
	"math/big"
	"time"

	"github.com/google/uuid"
)

// Weighted-sum aggregation (A2): each event quantity is a signed DELTA to a
// running level (e.g. +5 seats provisioned, -2 deprovisioned), and the metric's
// period value is the TIME-WEIGHTED AVERAGE of that level:
//
//	Σ(level_i · Δt_i) / (end − start)
//
// where level_i is the running level during interval i. The starting level
// carries forward the net of all events before the period, so a resource still
// active at period start counts from t0 (not from its next event). The average
// is generally fractional, so it feeds RateChargeRat and rounds to money once.
//
// Correctness: the level is piecewise-constant between events; each interval
// [prev, next) contributes level × duration exactly. Durations are taken in
// nanoseconds (int64 from time.Duration) and accumulated in big.Int so a large
// level × a month of nanoseconds cannot overflow; the ratio over the period
// length is then exact. A net-negative average (more deprovisioning than
// provisioning + carry-forward) is clamped to 0 — usage can't be billed
// negative, and RateChargeRat rejects a negative quantity.

// weightedSumRepo is the subset of the usage repository the weighted-sum
// aggregation needs: the carry-forward starting level plus the period's events
// with timestamps, in occurrence order.
type weightedSumRepo interface {
	SumQuantityBefore(ctx context.Context, subscriptionID uuid.UUID, dimension string, before time.Time) (int64, error)
	StreamEventsForMetric(ctx context.Context, subscriptionID uuid.UUID, dimension string, start, end time.Time, fn func(quantity int64, ts time.Time, props map[string]string) error) error
}

// AggregateWeightedSum computes the time-weighted average level over [start,
// end) as an exact rational. An empty period (end <= start) or no level yields
// 0. The result is never negative.
func AggregateWeightedSum(ctx context.Context, repo weightedSumRepo, subID uuid.UUID, dimension string, start, end time.Time) (*big.Rat, error) {
	totalNs := end.Sub(start).Nanoseconds()
	if totalNs <= 0 {
		return new(big.Rat), nil
	}

	// Carry-forward: the level at period start is the net of all prior events.
	level, err := repo.SumQuantityBefore(ctx, subID, dimension, start)
	if err != nil {
		return nil, err
	}

	// numerator = Σ level · Δt (nanoseconds), accumulated exactly.
	numerator := new(big.Int)
	prev := start
	addSegment := func(levelAt int64, from, to time.Time) {
		dur := to.Sub(from).Nanoseconds()
		if dur <= 0 {
			return
		}
		// numerator += levelAt * dur
		term := new(big.Int).Mul(big.NewInt(levelAt), big.NewInt(dur))
		numerator.Add(numerator, term)
	}

	err = repo.StreamEventsForMetric(ctx, subID, dimension, start, end, func(delta int64, ts time.Time, _ map[string]string) error {
		// The level held over [prev, ts); then this event changes it.
		addSegment(level, prev, ts)
		level += delta
		prev = ts
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Final segment: the level held from the last event (or start) to period end.
	addSegment(level, prev, end)

	avg := new(big.Rat).SetFrac(numerator, big.NewInt(totalNs))
	if avg.Sign() < 0 {
		return new(big.Rat), nil
	}
	return avg, nil
}
