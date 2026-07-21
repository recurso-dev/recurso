package service

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// fakeStreamer replays a fixed slice of events to StreamEventsForMetric.
type fakeStreamer struct {
	events []struct {
		qty   int64
		props map[string]string
	}
	err error
}

func (f *fakeStreamer) StreamEventsForMetric(_ context.Context, _ uuid.UUID, _ string, _, _ time.Time, fn func(int64, time.Time, map[string]string) error) error {
	if f.err != nil {
		return f.err
	}
	for _, e := range f.events {
		if err := fn(e.qty, time.Time{}, e.props); err != nil {
			return err
		}
	}
	return nil
}

func ev(qty int64, props map[string]string) struct {
	qty   int64
	props map[string]string
} {
	return struct {
		qty   int64
		props map[string]string
	}{qty, props}
}

// TestAggregateCustom_SumsFractionalContributions proves the custom aggregation
// sums per-event expression results exactly and that the fractional total,
// priced through per_unit, rounds money once. bytes/1e6 over three events
// (2.5 + 3.0 + 2.0 MB = 7.5 MB) at ₹10/MB = exactly ₹75.00.
func TestAggregateCustom_SumsFractionalContributions(t *testing.T) {
	evaluator, err := CompileCustomExpression("properties.bytes / 1000000")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	fs := &fakeStreamer{events: []struct {
		qty   int64
		props map[string]string
	}{
		ev(1, map[string]string{"bytes": "2500000"}),
		ev(1, map[string]string{"bytes": "3000000"}),
		ev(1, map[string]string{"bytes": "2000000"}),
	}}
	sum, err := AggregateCustom(context.Background(), fs, evaluator, uuid.New(), "storage", time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if sum.Cmp(big.NewRat(15, 2)) != 0 { // 7.5
		t.Fatalf("want 7.5 MB total, got %s", sum.RatString())
	}
	amount, err := RateChargeRat(domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: "10"}, sum)
	if err != nil {
		t.Fatalf("rate: %v", err)
	}
	if amount != 7500 {
		t.Fatalf("7.5 MB @ ₹10 want ₹75.00 (7500), got %d", amount)
	}
}

// TestAggregateCustom_NonNumericPropertyReadsZero proves a non-numeric or absent
// property reads as 0, so an expression over it doesn't error mid-billing.
func TestAggregateCustom_NonNumericPropertyReadsZero(t *testing.T) {
	evaluator, err := CompileCustomExpression("quantity * properties.rate")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	fs := &fakeStreamer{events: []struct {
		qty   int64
		props map[string]string
	}{
		ev(10, map[string]string{"rate": "1.5"}),       // 15
		ev(10, map[string]string{"rate": "not-a-num"}), // rate->0 => 0
		ev(10, nil), // no props => rate 0 => 0
	}}
	sum, err := AggregateCustom(context.Background(), fs, evaluator, uuid.New(), "d", time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if sum.Cmp(big.NewRat(15, 1)) != 0 {
		t.Fatalf("want 15, got %s", sum.RatString())
	}
}

// TestAggregateCustom_ZeroEvents sums to 0.
func TestAggregateCustom_ZeroEvents(t *testing.T) {
	evaluator, _ := CompileCustomExpression("quantity")
	sum, err := AggregateCustom(context.Background(), &fakeStreamer{}, evaluator, uuid.New(), "d", time.Time{}, time.Now())
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if sum.Sign() != 0 {
		t.Fatalf("want 0, got %s", sum.RatString())
	}
}

// TestCustomExpr_Guardrails proves each of the four guardrails the custom
// aggregation relies on. If a future expr upgrade weakens any of them, one of
// these fails rather than a tenant expression silently reaching host state or
// hanging billing.
func TestCustomExpr_Guardrails(t *testing.T) {
	// Field allowlist: only `quantity` and `properties` compile; anything else
	// is a compile error (can't reach host state or other identifiers).
	t.Run("unknown identifier rejected at compile", func(t *testing.T) {
		for _, src := range []string{
			"foo + 1",          // undeclared variable
			"os.Getenv(\"X\")", // no host packages
			"env.Quantity",     // internal field name, not the tagged alias
			"len(properties)",  // len over a map returns int, not the value we want... still compiles; see note
		} {
			_, err := CompileCustomExpression(src)
			if src == "len(properties)" {
				// len() is a pure builtin returning int; AsFloat64 coerces int->float,
				// so this DOES compile. It cannot touch the host. Just assert it's usable.
				if err != nil {
					t.Fatalf("len(properties) should compile: %v", err)
				}
				continue
			}
			if err == nil {
				t.Fatalf("expected %q to be rejected, but it compiled", src)
			}
		}
	})

	// Strict numeric output: a non-numeric result is a compile error.
	t.Run("non-numeric output rejected at compile", func(t *testing.T) {
		for _, src := range []string{
			`"hello"`,      // string
			`quantity > 5`, // bool
		} {
			if _, err := CompileCustomExpression(src); err == nil {
				t.Fatalf("expected non-numeric %q to be rejected", src)
			}
		}
	})

	// Empty expression is rejected.
	t.Run("empty rejected", func(t *testing.T) {
		if _, err := CompileCustomExpression(""); err == nil {
			t.Fatal("expected empty expression to be rejected")
		}
	})

	// Arithmetic over event fields evaluates correctly, and a referenced-but-
	// absent property reads as 0 (a typed map returns the zero value).
	t.Run("arithmetic and missing property", func(t *testing.T) {
		ev, err := CompileCustomExpression("quantity * properties.multiplier")
		if err != nil {
			t.Fatalf("compile: %v", err)
		}
		got, err := ev.Eval(context.Background(), 10, map[string]float64{"multiplier": 3})
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		if got != 30 {
			t.Fatalf("want 30, got %v", got)
		}
		// multiplier absent -> 0 -> whole product 0, no runtime error.
		got, err = ev.Eval(context.Background(), 10, map[string]float64{})
		if err != nil {
			t.Fatalf("eval missing prop: %v", err)
		}
		if got != 0 {
			t.Fatalf("missing property should read 0 (product 0), got %v", got)
		}
	})

	// Resource bound: a pathological array-materializing comprehension must
	// abort, not hang or exhaust memory. expr's built-in memory budget is the
	// primary guard (it fires here); the per-eval context deadline is a backstop
	// for CPU-bound loops. Either way Eval returns an error and billing fails
	// closed rather than misbilling.
	t.Run("pathological comprehension is bounded", func(t *testing.T) {
		ev, err := CompileCustomExpression("reduce(filter(1..30000000, true), #acc + #, 0)")
		if err != nil {
			t.Skipf("range builtins unavailable in this expr build: %v", err)
		}
		if _, err := ev.Eval(context.Background(), 0, nil); err == nil {
			t.Fatal("expected a huge comprehension to be bounded (memory budget / deadline)")
		}
	})
}
