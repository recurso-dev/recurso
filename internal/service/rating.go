package service

import (
	"fmt"
	"math/big"
	"regexp"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// Rating: the pure pricing library of usage-based billing v1
// (spec_usage_billing.md). RateCharge reduces (charge model, per-currency
// amounts, aggregated quantity) to an int64 minor-unit line amount.
//
// Precision rule (D1): per-unit rates are decimal strings in MAJOR currency
// units (e.g. "0.0035" rupees per call) so sub-minor-unit pricing is
// first-class. All unit-priced portions are computed exactly with big.Rat
// and rounded HALF-UP to minor units ONCE per line; tier/package flat
// amounts are already int64 minor units and added exactly. Every invoice
// line therefore stays int64 minor units and the existing Σ-line == subtotal
// invariant is untouched.

// minorUnitsPerMajor converts major-unit rates to minor units. The codebase
// treats all monetary int64s as 1/100 major units (paise/cents) throughout,
// so rating follows the same convention.
const minorUnitsPerMajor = 100

// rateRe restricts rates to plain non-negative decimals — big.Rat.SetString
// would also accept fractions ("1/3") and exponents ("1e-5"), which we do
// not want in stored pricing config.
var rateRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

const maxRateLen = 40

// RatingError marks invalid pricing configuration or an unrateable
// quantity (maps to HTTP 400 on config write; logged + skipped at
// invoice-generation time).
type RatingError string

func (e RatingError) Error() string { return string(e) }

// parseDecimalRate parses a decimal-string rate into an exact rational,
// attributing errors to the named field. The regex rejects fractions and
// exponents so stored pricing config stays plain decimals.
func parseDecimalRate(s, field string) (*big.Rat, error) {
	if s == "" {
		return nil, RatingError(field + " is required")
	}
	if len(s) > maxRateLen {
		return nil, RatingError(field + " is too long")
	}
	if !rateRe.MatchString(s) {
		return nil, RatingError(fmt.Sprintf("%s %q is not a plain non-negative decimal", field, s))
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, RatingError(fmt.Sprintf("%s %q is not a valid decimal", field, s))
	}
	return r, nil
}

// parseRate parses a per-unit rate (per_unit / tiers).
func parseRate(s string) (*big.Rat, error) { return parseDecimalRate(s, "unit_amount") }

// roundRatHalfUp rounds a non-negative exact amount (already scaled to
// minor units) half-up to int64.
func roundRatHalfUp(r *big.Rat) int64 {
	// floor(r + 1/2)
	half := big.NewRat(1, 2)
	sum := new(big.Rat).Add(r, half)
	return new(big.Int).Div(sum.Num(), sum.Denom()).Int64()
}

// ratePerUnit computes quantity × rate (major units) → exact minor units.
func ratePerUnit(rate *big.Rat, quantity int64) *big.Rat {
	q := new(big.Rat).SetInt64(quantity)
	minor := new(big.Rat).SetInt64(minorUnitsPerMajor)
	return new(big.Rat).Mul(new(big.Rat).Mul(rate, q), minor)
}

// RateCharge prices an aggregated quantity per the charge model, returning
// the line amount in int64 minor units. Quantity 0 always rates to 0 (no
// usage ⇒ nothing billed, tier flat amounts included). The amounts must
// already be the single-currency entry selected for the invoice currency.
func RateCharge(model domain.ChargeModel, amounts domain.ChargeAmounts, quantity int64) (int64, error) {
	if quantity < 0 {
		return 0, RatingError("quantity must not be negative")
	}
	if quantity == 0 {
		return 0, nil
	}

	switch model {
	case domain.ChargePerUnit:
		rate, err := parseRate(amounts.UnitAmount)
		if err != nil {
			return 0, err
		}
		return roundRatHalfUp(ratePerUnit(rate, quantity)), nil

	case domain.ChargePackage:
		if amounts.PackageSize <= 0 {
			return 0, RatingError("package_size must be greater than zero")
		}
		if amounts.PackageAmount < 0 {
			return 0, RatingError("package_amount must not be negative")
		}
		bundles := (quantity + amounts.PackageSize - 1) / amounts.PackageSize // ceil
		return bundles * amounts.PackageAmount, nil

	case domain.ChargeGraduated:
		return rateGraduated(amounts.Tiers, quantity)

	case domain.ChargeVolume:
		return rateVolume(amounts.Tiers, quantity)

	case domain.ChargePercentage:
		return ratePercentage(amounts, quantity)
	}
	return 0, RatingError(fmt.Sprintf("unsupported charge model %q", model))
}

// validateTiers checks the shared tier invariants: at least one tier,
// strictly ascending positive bounds, only the last tier unbounded
// (up_to null), all rates valid decimals, flat amounts non-negative.
func validateTiers(tiers []domain.ChargeTier) error {
	if len(tiers) == 0 {
		return RatingError("at least one tier is required")
	}
	var prev int64
	for i, t := range tiers {
		if t.UpTo == nil {
			if i != len(tiers)-1 {
				return RatingError("only the last tier may leave up_to unset")
			}
		} else {
			if *t.UpTo <= prev {
				return RatingError("tier up_to bounds must be strictly ascending and positive")
			}
			prev = *t.UpTo
		}
		if _, err := parseRate(t.UnitAmount); err != nil {
			return err
		}
		if t.FlatAmount < 0 {
			return RatingError("tier flat_amount must not be negative")
		}
	}
	if tiers[len(tiers)-1].UpTo != nil {
		return RatingError("the last tier must leave up_to unset (unbounded)")
	}
	return nil
}

// rateGraduated prices each tier's units at that tier's rate: with tiers
// 0-100 @ 1.00 and 101+ @ 0.50, 150 units = 100×1.00 + 50×0.50. A tier's
// FlatAmount is added once when at least one unit lands in it. The exact
// unit-priced total is rounded once at the end.
func rateGraduated(tiers []domain.ChargeTier, quantity int64) (int64, error) {
	if err := validateTiers(tiers); err != nil {
		return 0, err
	}

	unitTotal := new(big.Rat)
	var flatTotal int64
	remaining := quantity
	var lower int64 // units consumed by previous tiers
	for _, t := range tiers {
		if remaining <= 0 {
			break
		}
		inTier := remaining
		if t.UpTo != nil {
			width := *t.UpTo - lower
			if inTier > width {
				inTier = width
			}
			lower = *t.UpTo
		}
		rate, _ := parseRate(t.UnitAmount) // validated above
		unitTotal.Add(unitTotal, ratePerUnit(rate, inTier))
		flatTotal += t.FlatAmount
		remaining -= inTier
	}
	return roundRatHalfUp(unitTotal) + flatTotal, nil
}

// rateVolume prices the WHOLE quantity at the single tier it reaches
// (the first tier whose up_to ≥ quantity; the unbounded last tier catches
// the rest), plus that tier's flat amount.
func rateVolume(tiers []domain.ChargeTier, quantity int64) (int64, error) {
	if err := validateTiers(tiers); err != nil {
		return 0, err
	}

	for _, t := range tiers {
		if t.UpTo != nil && quantity > *t.UpTo {
			continue
		}
		rate, _ := parseRate(t.UnitAmount) // validated above
		return roundRatHalfUp(ratePerUnit(rate, quantity)) + t.FlatAmount, nil
	}
	// Unreachable: validateTiers guarantees an unbounded last tier.
	return 0, RatingError("quantity exceeds tier coverage")
}

// ratePercentage prices a percentage of the aggregated monetary base. Here
// quantity is the base in minor units (e.g. a sum of transaction amounts), not
// a unit count. FreeUnits are deducted from the base first; Rate (a percent
// decimal) is applied to the remainder exactly and rounded half-up once;
// FixedAmount (minor units) is added; and the line is clamped to
// [MinAmount, MaxAmount] (MaxAmount 0 = uncapped, MinAmount 0 = no floor).
// RateCharge already short-circuits quantity 0 to 0, so MinAmount floors only
// when there is usage.
func ratePercentage(a domain.ChargeAmounts, quantity int64) (int64, error) {
	rate, err := parseDecimalRate(a.Rate, "rate")
	if err != nil {
		return 0, err
	}
	if a.FreeUnits < 0 {
		return 0, RatingError("free_units must not be negative")
	}
	if a.FixedAmount < 0 {
		return 0, RatingError("fixed_amount must not be negative")
	}
	if a.MinAmount < 0 || a.MaxAmount < 0 {
		return 0, RatingError("min_amount and max_amount must not be negative")
	}
	if a.MaxAmount > 0 && a.MaxAmount < a.MinAmount {
		return 0, RatingError("max_amount must be greater than or equal to min_amount")
	}

	base := quantity - a.FreeUnits
	if base < 0 {
		base = 0
	}

	// line = base × (rate / 100), exact, rounded half-up once.
	pct := new(big.Rat).Quo(rate, big.NewRat(100, 1))
	lineRat := new(big.Rat).Mul(new(big.Rat).SetInt64(base), pct)
	line := roundRatHalfUp(lineRat) + a.FixedAmount

	if line < a.MinAmount {
		line = a.MinAmount
	}
	if a.MaxAmount > 0 && line > a.MaxAmount {
		line = a.MaxAmount
	}
	return line, nil
}
