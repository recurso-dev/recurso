package service

import (
	"math/big"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestRateChargeRat_FractionalQuantityRoundsMoneyOnce proves the Option-3
// precision guarantee: a fractional aggregate quantity (a weighted_sum average
// or a summed custom expression) is priced by rounding the MONEY once, never by
// pre-rounding the quantity to an integer. The canonical case: 7.5 seats at
// ₹10.00/seat must bill exactly ₹75.00 (7500 minor units) — not ₹80.00 (round
// qty up) or ₹70.00 (round qty down).
func TestRateChargeRat_FractionalQuantityRoundsMoneyOnce(t *testing.T) {
	// per_unit @ 10.00 major units/seat.
	perUnit := domain.ChargeAmounts{UnitAmount: "10"}

	got, err := RateChargeRat(domain.ChargePerUnit, perUnit, big.NewRat(15, 2)) // 7.5
	if err != nil {
		t.Fatalf("rate: %v", err)
	}
	if got != 7500 {
		t.Fatalf("7.5 seats @ 10.00 want 7500 minor units (₹75.00), got %d", got)
	}

	// A quantity that only resolves to whole minor units through exact rational
	// math: 1/3 of a unit at 0.03 major/unit = 0.01 major = 1 minor unit.
	third := domain.ChargeAmounts{UnitAmount: "0.03"}
	got, err = RateChargeRat(domain.ChargePerUnit, third, big.NewRat(1, 3))
	if err != nil {
		t.Fatalf("rate: %v", err)
	}
	if got != 1 {
		t.Fatalf("1/3 unit @ 0.03 want 1 minor unit, got %d", got)
	}

	// Real-world weighted_sum: 10 seats for 1/3 of the period = 3.333... average.
	// At 30.00/seat the exact line is 100.00 (3.333... × 30 = 100), which
	// pre-rounding the quantity to 3 (→90.00) or 4 (→120.00) would both miss.
	seat30 := domain.ChargeAmounts{UnitAmount: "30"}
	got, err = RateChargeRat(domain.ChargePerUnit, seat30, big.NewRat(10, 3))
	if err != nil {
		t.Fatalf("rate: %v", err)
	}
	if got != 10000 {
		t.Fatalf("10/3 avg seats @ 30.00 want 10000 minor units (₹100.00), got %d", got)
	}
}

// TestRateCharge_IntWrapperUnchanged proves the int64 wrapper produces the same
// result as before across every charge model — the existing aggregations
// (count/sum/max/unique/latest/percentile/dynamic) must be byte-identical.
func TestRateCharge_IntWrapperUnchanged(t *testing.T) {
	up := func(n int64) *int64 { return &n }
	cases := []struct {
		name   string
		model  domain.ChargeModel
		amt    domain.ChargeAmounts
		qty    int64
		expect int64
	}{
		{"per_unit", domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: "1.50"}, 100, 15000},
		{"package", domain.ChargePackage, domain.ChargeAmounts{PackageSize: 1000, PackageAmount: 500}, 2500, 1500},
		{"graduated", domain.ChargeGraduated, domain.ChargeAmounts{Tiers: []domain.ChargeTier{
			{UpTo: up(100), UnitAmount: "1.00"}, {UpTo: nil, UnitAmount: "0.50"},
		}}, 150, 12500},
		{"volume", domain.ChargeVolume, domain.ChargeAmounts{Tiers: []domain.ChargeTier{
			{UpTo: up(100), UnitAmount: "1.00"}, {UpTo: nil, UnitAmount: "0.50"},
		}}, 150, 7500},
		{"percentage", domain.ChargePercentage, domain.ChargeAmounts{Rate: "2.5"}, 1000000, 25000},
		{"graduated_percentage", domain.ChargeGraduatedPercentage, domain.ChargeAmounts{Tiers: []domain.ChargeTier{
			{UpTo: up(1000000), Rate: "3"}, {UpTo: nil, Rate: "2"},
		}}, 1500000, 40000},
		{"dynamic", domain.ChargeDynamic, domain.ChargeAmounts{}, 4200, 4200},
		{"zero", domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: "1.50"}, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := RateCharge(c.model, c.amt, c.qty)
			if err != nil {
				t.Fatalf("RateCharge: %v", err)
			}
			if got != c.expect {
				t.Fatalf("%s: want %d, got %d", c.name, c.expect, got)
			}
			// The rational core must agree with the int wrapper for integer input.
			gotRat, err := RateChargeRat(c.model, c.amt, new(big.Rat).SetInt64(c.qty))
			if err != nil {
				t.Fatalf("RateChargeRat: %v", err)
			}
			if gotRat != got {
				t.Fatalf("%s: rat core %d != int wrapper %d", c.name, gotRat, got)
			}
		})
	}
}

// TestRateChargeRat_Guards covers nil and negative quantities.
func TestRateChargeRat_Guards(t *testing.T) {
	if _, err := RateChargeRat(domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: "1"}, nil); err == nil {
		t.Fatal("nil quantity should error")
	}
	if _, err := RateChargeRat(domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: "1"}, big.NewRat(-1, 1)); err == nil {
		t.Fatal("negative quantity should error")
	}
	// Fractional package quantity ceils to a whole bundle.
	got, err := RateChargeRat(domain.ChargePackage, domain.ChargeAmounts{PackageSize: 1000, PackageAmount: 500}, big.NewRat(1, 10))
	if err != nil {
		t.Fatalf("rate: %v", err)
	}
	if got != 500 {
		t.Fatalf("0.1 units want 1 bundle (500), got %d", got)
	}
}
