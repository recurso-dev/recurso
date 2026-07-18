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

// parseRate parses a decimal-string rate into an exact rational.
func parseRate(s string) (*big.Rat, error) {
	if s == "" {
		return nil, RatingError("unit_amount is required")
	}
	if len(s) > maxRateLen {
		return nil, RatingError("unit_amount is too long")
	}
	if !rateRe.MatchString(s) {
		return nil, RatingError(fmt.Sprintf("unit_amount %q is not a plain non-negative decimal", s))
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, RatingError(fmt.Sprintf("unit_amount %q is not a valid decimal", s))
	}
	return r, nil
}

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
