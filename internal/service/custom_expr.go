package service

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/google/uuid"
)

// Custom aggregation (A2): a tenant-authored expression computes each usage
// event's numeric contribution, and the metric's period value is the SUM of
// those contributions. This lets a metric price on arithmetic over event
// fields (e.g. `quantity * properties.multiplier`, or
// `properties.bytes / 1000000`) without a bespoke aggregation type per formula.
//
// The expression runs under four guardrails (all enforced here, none optional):
//
//  1. Field allowlist — it is compiled against a FIXED environment exposing only
//     `quantity` (float64) and `properties` (map[string]float64 of the event's
//     numeric properties). Any other identifier is a COMPILE error, so an
//     expression can never reach host state, other events, or the wider program.
//  2. No host functions — the environment contains no functions, so there is
//     nothing host-side to call. expr's own pure builtins (arithmetic, math) are
//     the only callables, and they cannot touch the process.
//  3. Bounded resources — expr's built-in memory budget aborts an
//     array-materializing comprehension, and every evaluation additionally runs
//     under a context deadline (evalTimeout) that expr checks between VM steps.
//     A pathological expression therefore aborts the aggregation with an error
//     rather than hanging or exhausting memory; billing fails closed.
//  4. Strict numeric output — the expression is compiled with expr.AsFloat64(),
//     so a non-numeric result is a COMPILE error. Per-event results are summed in
//     float64 and rounded to int64 ONCE at the period boundary (see
//     AggregateCustom), mirroring the round-once discipline the rating layer uses.
//
// The expression is authored by a tenant admin configuring a metric — not by an
// end user — but the guardrails hold regardless of who wrote it.

// evalTimeout bounds a single per-event evaluation. Evaluations are arithmetic
// over a handful of fields and finish in microseconds; the deadline only exists
// to kill a pathological expression (e.g. a huge comprehension), never a normal one.
const evalTimeout = 100 * time.Millisecond

// customExprEnv is the ONLY environment an expression may reference. Compiling
// against this concrete type is what enforces the field allowlist: expr
// type-checks every identifier against it, so an unknown field fails to compile.
//
// Ctx carries the per-evaluation deadline that expr.WithContext("ctx") checks
// between VM steps; it is present so the timeout guard works, not for tenant use
// — a context has no float64 coercion, so AsFloat64 makes any expression that
// references it fail to compile.
type customExprEnv struct {
	Quantity   float64            `expr:"quantity"`
	Properties map[string]float64 `expr:"properties"`
	Ctx        context.Context    `expr:"ctx"`
}

// CustomEvaluator is a compiled, reusable custom-aggregation expression. It is
// safe to evaluate concurrently: Eval allocates its own environment per call and
// the compiled program is read-only.
type CustomEvaluator struct {
	program *vm.Program
	source  string
}

// CompileCustomExpression compiles expression under the guardrails above,
// returning an evaluator or a validation error. Call it both at metric-create
// time (to reject a bad expression with a 400) and before aggregating a period.
func CompileCustomExpression(expression string) (*CustomEvaluator, error) {
	if expression == "" {
		return nil, fmt.Errorf("custom aggregation requires a non-empty expression")
	}
	program, err := expr.Compile(expression,
		expr.Env(customExprEnv{}), // field allowlist + type check
		expr.AsFloat64(),          // strict numeric output
		expr.WithContext("ctx"),   // lets the VM observe the eval deadline
	)
	if err != nil {
		return nil, fmt.Errorf("invalid custom expression: %w", err)
	}
	return &CustomEvaluator{program: program, source: expression}, nil
}

// Eval computes one event's contribution. quantity is the event quantity and
// props are its numeric properties (non-numeric properties are omitted by the
// caller). A referenced-but-absent property reads as 0. The evaluation runs
// under a hard deadline; a timeout or runtime error is returned to the caller,
// which aborts the aggregation (billing fails closed rather than misbilling).
func (e *CustomEvaluator) Eval(ctx context.Context, quantity float64, props map[string]float64) (float64, error) {
	if props == nil {
		props = map[string]float64{}
	}
	ctx, cancel := context.WithTimeout(ctx, evalTimeout)
	defer cancel()
	env := customExprEnv{Quantity: quantity, Properties: props, Ctx: ctx}
	out, err := expr.Run(e.program, env)
	if err != nil {
		return 0, fmt.Errorf("custom expression %q failed: %w", e.source, err)
	}
	f, ok := out.(float64)
	if !ok {
		// AsFloat64 guarantees this at compile time; defense in depth.
		return 0, fmt.Errorf("custom expression %q returned non-float %T", e.source, out)
	}
	return f, nil
}

// eventStreamer is the subset of the usage repository the custom aggregation
// needs: it streams a period's events in occurrence order.
type eventStreamer interface {
	StreamEventsForMetric(ctx context.Context, subscriptionID uuid.UUID, dimension string, start, end time.Time, fn func(quantity int64, props map[string]string) error) error
}

// AggregateCustom evaluates a compiled custom expression against every event of
// the subscription's dimension in [start, end) and returns the SUM of the
// per-event results as an exact rational. The sum accumulates in big.Rat (each
// float64 result taken as its exact rational value) so a long period does not
// drift the way repeated float64 addition would; the result feeds RateChargeRat
// and is rounded to money exactly once. Zero events sum to 0.
//
// Event properties are stored as strings; only those that parse as a number are
// exposed to the expression (as float64). A non-numeric property is simply
// absent, so `properties.foo` on a non-numeric "foo" reads 0 — the same as a
// missing property. Any evaluation error aborts the aggregation (billing fails
// closed) rather than silently dropping an event.
func AggregateCustom(ctx context.Context, repo eventStreamer, ev *CustomEvaluator, subID uuid.UUID, dimension string, start, end time.Time) (*big.Rat, error) {
	sum := new(big.Rat)
	err := repo.StreamEventsForMetric(ctx, subID, dimension, start, end, func(quantity int64, props map[string]string) error {
		numeric := numericProps(props)
		f, err := ev.Eval(ctx, float64(quantity), numeric)
		if err != nil {
			return err
		}
		r := new(big.Rat).SetFloat64(f)
		if r == nil {
			// SetFloat64 returns nil for NaN/Inf — a malformed expression result.
			return fmt.Errorf("custom expression produced a non-finite value")
		}
		sum.Add(sum, r)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sum, nil
}

// numericProps returns the subset of props whose string value parses as a
// float, keyed the same. Non-numeric properties are dropped so the expression
// only ever sees numbers.
func numericProps(props map[string]string) map[string]float64 {
	if len(props) == 0 {
		return nil
	}
	out := make(map[string]float64, len(props))
	for k, v := range props {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			out[k] = f
		}
	}
	return out
}
