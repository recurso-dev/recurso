package service

import (
	"strings"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

func i64(v int64) *int64 { return &v }

func TestRateChargePerUnit(t *testing.T) {
	cases := []struct {
		name string
		rate string
		qty  int64
		want int64
	}{
		{"whole rate", "2", 10, 2000},                         // 10 × ₹2 = ₹20 = 2000p
		{"sub-minor rate", "0.0035", 1500, 525},               // 1500 × ₹0.0035 = ₹5.25 = 525p
		{"rounds half up", "0.005", 1, 1},                     // 0.5p → 1p
		{"rounds down below half", "0.004", 1, 0},             // 0.4p → 0p
		{"exact paise", "0.01", 250, 250},                     // 250 × 1p
		{"single unit", "29", 1, 2900},                        //
		{"large quantity no drift", "0.0001", 1000000, 10000}, // 10^6 × ₹0.0001 = ₹100
		{"zero quantity", "5", 0, 0},
		{"zero rate", "0", 500, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RateCharge(domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: tc.rate}, tc.qty)
			if err != nil {
				t.Fatalf("RateCharge: %v", err)
			}
			if got != tc.want {
				t.Fatalf("rate %s × %d = %d, want %d", tc.rate, tc.qty, got, tc.want)
			}
		})
	}
}

func TestRateChargePerUnitRejectsBadRates(t *testing.T) {
	for _, rate := range []string{"", "-1", "1/3", "1e-5", "1.2.3", "abc", "1,5", " 1", strings.Repeat("9", 41)} {
		if _, err := RateCharge(domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: rate}, 10); err == nil {
			t.Errorf("rate %q: expected error, got none", rate)
		}
	}
}

func TestRateChargeRejectsNegativeQuantity(t *testing.T) {
	if _, err := RateCharge(domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: "1"}, -1); err == nil {
		t.Fatal("expected error for negative quantity")
	}
}

func TestRateChargePackage(t *testing.T) {
	amounts := domain.ChargeAmounts{PackageAmount: 500, PackageSize: 1000} // ₹5 per 1000
	cases := []struct {
		qty  int64
		want int64
	}{
		{0, 0},       // no usage, nothing billed
		{1, 500},     // partial bundle rounds UP
		{999, 500},   //
		{1000, 500},  // exact bundle
		{1001, 1000}, // spills into second bundle
		{5000, 2500}, //
	}
	for _, tc := range cases {
		got, err := RateCharge(domain.ChargePackage, amounts, tc.qty)
		if err != nil {
			t.Fatalf("qty %d: %v", tc.qty, err)
		}
		if got != tc.want {
			t.Fatalf("qty %d = %d, want %d", tc.qty, got, tc.want)
		}
	}

	if _, err := RateCharge(domain.ChargePackage, domain.ChargeAmounts{PackageAmount: 500}, 10); err == nil {
		t.Fatal("expected error for missing package_size")
	}
	if _, err := RateCharge(domain.ChargePackage, domain.ChargeAmounts{PackageAmount: -1, PackageSize: 10}, 10); err == nil {
		t.Fatal("expected error for negative package_amount")
	}
}

// graduatedTiers: 0–100 @ ₹1, 101–1000 @ ₹0.50, 1001+ @ ₹0.10 (+₹5 flat).
func graduatedTiers() []domain.ChargeTier {
	return []domain.ChargeTier{
		{UpTo: i64(100), UnitAmount: "1"},
		{UpTo: i64(1000), UnitAmount: "0.50"},
		{UpTo: nil, UnitAmount: "0.10", FlatAmount: 500},
	}
}

func TestRateChargeGraduated(t *testing.T) {
	cases := []struct {
		name string
		qty  int64
		want int64
	}{
		{"zero", 0, 0},
		{"inside first tier", 50, 5000},            // 50×₹1
		{"exact first bound", 100, 10000},          // 100×₹1
		{"spans two tiers", 150, 12500},            // 100×1 + 50×0.5 = ₹125
		{"exact second bound", 1000, 55000},        // 100 + 450 = ₹550
		{"reaches flat tier", 1001, 55010 + 500},   // + 1×0.1 + flat ₹5
		{"deep into last tier", 2000, 65000 + 500}, // 100+450+100 = ₹650 + ₹5
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RateCharge(domain.ChargeGraduated, domain.ChargeAmounts{Tiers: graduatedTiers()}, tc.qty)
			if err != nil {
				t.Fatalf("RateCharge: %v", err)
			}
			if got != tc.want {
				t.Fatalf("qty %d = %d, want %d", tc.qty, got, tc.want)
			}
		})
	}
}

// TestRateChargeGraduatedRoundsOnce locks the one-round-per-line rule: two
// tiers each producing 0.4p must round from the exact 0.8p sum (→ 1p), not
// per-tier (0 + 0 = 0p).
func TestRateChargeGraduatedRoundsOnce(t *testing.T) {
	tiers := []domain.ChargeTier{
		{UpTo: i64(1), UnitAmount: "0.004"},
		{UpTo: nil, UnitAmount: "0.004"},
	}
	got, err := RateCharge(domain.ChargeGraduated, domain.ChargeAmounts{Tiers: tiers}, 2)
	if err != nil {
		t.Fatalf("RateCharge: %v", err)
	}
	if got != 1 {
		t.Fatalf("expected exact-sum rounding to 1 paise, got %d", got)
	}
}

func TestRateChargeVolume(t *testing.T) {
	// Volume: whole quantity at the reached tier's rate.
	tiers := []domain.ChargeTier{
		{UpTo: i64(100), UnitAmount: "1"},
		{UpTo: i64(1000), UnitAmount: "0.50"},
		{UpTo: nil, UnitAmount: "0.10", FlatAmount: 500},
	}
	cases := []struct {
		qty  int64
		want int64
	}{
		{0, 0},
		{100, 10000},        // 100×₹1
		{101, 5050},         // ALL 101 × ₹0.50 — volume drops to the cheaper tier
		{1000, 50000},       //
		{1001, 10010 + 500}, // 1001×₹0.10 + flat
	}
	for _, tc := range cases {
		got, err := RateCharge(domain.ChargeVolume, domain.ChargeAmounts{Tiers: tiers}, tc.qty)
		if err != nil {
			t.Fatalf("qty %d: %v", tc.qty, err)
		}
		if got != tc.want {
			t.Fatalf("qty %d = %d, want %d", tc.qty, got, tc.want)
		}
	}
}

func TestTierValidation(t *testing.T) {
	cases := []struct {
		name  string
		tiers []domain.ChargeTier
	}{
		{"empty", nil},
		{"last tier bounded", []domain.ChargeTier{{UpTo: i64(10), UnitAmount: "1"}}},
		{"unbounded tier not last", []domain.ChargeTier{
			{UpTo: nil, UnitAmount: "1"},
			{UpTo: i64(10), UnitAmount: "1"},
		}},
		{"non-ascending bounds", []domain.ChargeTier{
			{UpTo: i64(10), UnitAmount: "1"},
			{UpTo: i64(10), UnitAmount: "0.5"},
			{UpTo: nil, UnitAmount: "0.1"},
		}},
		{"bad rate", []domain.ChargeTier{{UpTo: nil, UnitAmount: "1/3"}}},
		{"negative flat", []domain.ChargeTier{{UpTo: nil, UnitAmount: "1", FlatAmount: -5}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RateCharge(domain.ChargeGraduated, domain.ChargeAmounts{Tiers: tc.tiers}, 5); err == nil {
				t.Fatal("expected validation error, got none")
			}
			if _, err := RateCharge(domain.ChargeVolume, domain.ChargeAmounts{Tiers: tc.tiers}, 5); err == nil {
				t.Fatal("expected validation error, got none")
			}
		})
	}
}

func TestRateChargeUnsupportedModel(t *testing.T) {
	if _, err := RateCharge(domain.ChargeModel("mystery_model"), domain.ChargeAmounts{}, 5); err == nil {
		t.Fatal("expected error for unsupported model")
	}
}

// TestRateChargeDynamic: the line is the pre-summed dynamic amount (identity),
// with no pricing config and no rate applied.
func TestRateChargeDynamic(t *testing.T) {
	cases := []struct {
		qty  int64 // Σ per-event dynamic_amount, minor units
		want int64
	}{
		{0, 0},       // no usage
		{4200, 4200}, // bills the sum exactly
		{1, 1},       //
		{999999, 999999},
	}
	for _, tc := range cases {
		got, err := RateCharge(domain.ChargeDynamic, domain.ChargeAmounts{}, tc.qty)
		if err != nil {
			t.Fatalf("qty %d: %v", tc.qty, err)
		}
		if got != tc.want {
			t.Fatalf("dynamic qty %d = %d, want %d", tc.qty, got, tc.want)
		}
	}
	// Negative aggregate is rejected by the shared quantity guard.
	if _, err := RateCharge(domain.ChargeDynamic, domain.ChargeAmounts{}, -1); err == nil {
		t.Fatal("expected error for negative dynamic quantity")
	}
}

func TestRateChargePercentage(t *testing.T) {
	cases := []struct {
		name    string
		amounts domain.ChargeAmounts
		qty     int64 // monetary base in minor units
		want    int64
	}{
		// 2.5% of ₹1000 (100000p) = ₹25 = 2500p
		{"plain percent", domain.ChargeAmounts{Rate: "2.5"}, 100000, 2500},
		// fractional percent, exact: 0.35% of 100000 = 350
		{"sub-percent exact", domain.ChargeAmounts{Rate: "0.35"}, 100000, 350},
		// rounds half up once: 1% of 150p = 1.5p → 2p
		{"rounds half up", domain.ChargeAmounts{Rate: "1"}, 150, 2},
		// fixed fee added after the percentage: 2% of 100000 = 2000, +30 = 2030
		{"fixed fee", domain.ChargeAmounts{Rate: "2", FixedAmount: 30}, 100000, 2030},
		// free units deducted first: 2% of (100000-20000) = 2% of 80000 = 1600
		{"free units", domain.ChargeAmounts{Rate: "2", FreeUnits: 20000}, 100000, 1600},
		// free units exceed base → base clamps to 0 → line 0 (+fixed if any)
		{"free units exceed base", domain.ChargeAmounts{Rate: "2", FreeUnits: 200000}, 100000, 0},
		{"free units exceed base with fixed", domain.ChargeAmounts{Rate: "2", FreeUnits: 200000, FixedAmount: 50}, 100000, 50},
		// min floor applies when there IS usage: 1% of 1000 = 10, floored to 500
		{"min floor", domain.ChargeAmounts{Rate: "1", MinAmount: 500}, 1000, 500},
		// max cap: 5% of 100000 = 5000, capped to 3000
		{"max cap", domain.ChargeAmounts{Rate: "5", MaxAmount: 3000}, 100000, 3000},
		// max 0 means uncapped
		{"max zero uncapped", domain.ChargeAmounts{Rate: "5", MaxAmount: 0}, 100000, 5000},
		// zero quantity short-circuits to 0 regardless of min (no usage)
		{"zero base no floor", domain.ChargeAmounts{Rate: "2", MinAmount: 500}, 0, 0},
		// zero rate
		{"zero rate", domain.ChargeAmounts{Rate: "0"}, 100000, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RateCharge(domain.ChargePercentage, tc.amounts, tc.qty)
			if err != nil {
				t.Fatalf("RateCharge: %v", err)
			}
			if got != tc.want {
				t.Fatalf("percentage %+v × %d = %d, want %d", tc.amounts, tc.qty, got, tc.want)
			}
		})
	}
}

func TestRateChargePercentageRejectsBadConfig(t *testing.T) {
	cases := []struct {
		name    string
		amounts domain.ChargeAmounts
	}{
		{"missing rate", domain.ChargeAmounts{}},
		{"bad rate", domain.ChargeAmounts{Rate: "abc"}},
		{"negative rate string", domain.ChargeAmounts{Rate: "-1"}},
		{"negative free units", domain.ChargeAmounts{Rate: "1", FreeUnits: -1}},
		{"negative fixed", domain.ChargeAmounts{Rate: "1", FixedAmount: -1}},
		{"negative min", domain.ChargeAmounts{Rate: "1", MinAmount: -1}},
		{"negative max", domain.ChargeAmounts{Rate: "1", MaxAmount: -1}},
		{"max below min", domain.ChargeAmounts{Rate: "1", MinAmount: 500, MaxAmount: 300}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RateCharge(domain.ChargePercentage, tc.amounts, 100000); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

// graduatedPercentTiers: 0-1,000,000 @ 3%, 1,000,000-2,000,000 @ 2%,
// 2,000,000+ @ 1% + flat ₹5. Bounds and base are minor units.
func graduatedPercentTiers() []domain.ChargeTier {
	return []domain.ChargeTier{
		{UpTo: i64(1_000_000), Rate: "3"},
		{UpTo: i64(2_000_000), Rate: "2"},
		{UpTo: nil, Rate: "1", FlatAmount: 500},
	}
}

func TestRateChargeGraduatedPercentage(t *testing.T) {
	cases := []struct {
		name string
		base int64 // monetary base in minor units
		want int64
	}{
		{"zero", 0, 0},
		{"inside first band", 500_000, 15_000},           // 3%×500,000
		{"exact first bound", 1_000_000, 30_000},         // 3%×1,000,000
		{"spans two bands", 1_500_000, 40_000},           // 30,000 + 2%×500,000
		{"exact second bound", 2_000_000, 50_000},        // 30,000 + 2%×1,000,000
		{"reaches flat band", 2_000_001, 50_000 + 500},   // +1%×1 (0.01→0 in the sum) + flat
		{"deep into last band", 3_000_000, 60_000 + 500}, // 30,000+20,000+1%×1,000,000 + flat
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RateCharge(domain.ChargeGraduatedPercentage, domain.ChargeAmounts{Tiers: graduatedPercentTiers()}, tc.base)
			if err != nil {
				t.Fatalf("RateCharge: %v", err)
			}
			if got != tc.want {
				t.Fatalf("base %d = %d, want %d", tc.base, got, tc.want)
			}
		})
	}
}

// TestRateChargeGraduatedPercentageRoundsOnce locks one-round-per-line: two
// bands each producing 0.4p must round from the exact 0.8p sum (→ 1p), not
// per-band (0 + 0 = 0p).
func TestRateChargeGraduatedPercentageRoundsOnce(t *testing.T) {
	tiers := []domain.ChargeTier{
		{UpTo: i64(1), Rate: "40"}, // 40% × 1 = 0.4p
		{UpTo: nil, Rate: "40"},    // 40% × 1 = 0.4p
	}
	got, err := RateCharge(domain.ChargeGraduatedPercentage, domain.ChargeAmounts{Tiers: tiers}, 2)
	if err != nil {
		t.Fatalf("RateCharge: %v", err)
	}
	if got != 1 {
		t.Fatalf("expected exact-sum rounding to 1 paise, got %d", got)
	}
}

func TestRateChargeGraduatedPercentageRejectsBadConfig(t *testing.T) {
	cases := []struct {
		name  string
		tiers []domain.ChargeTier
	}{
		{"empty", nil},
		{"missing rate", []domain.ChargeTier{{UpTo: nil}}},
		{"bad rate", []domain.ChargeTier{{UpTo: nil, Rate: "abc"}}},
		{"non-ascending bounds", []domain.ChargeTier{{UpTo: i64(100), Rate: "1"}, {UpTo: i64(100), Rate: "1"}, {UpTo: nil, Rate: "1"}}},
		{"last tier bounded", []domain.ChargeTier{{UpTo: i64(100), Rate: "1"}}},
		{"negative flat", []domain.ChargeTier{{UpTo: nil, Rate: "1", FlatAmount: -1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RateCharge(domain.ChargeGraduatedPercentage, domain.ChargeAmounts{Tiers: tc.tiers}, 1_000_000); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}
