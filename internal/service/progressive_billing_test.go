package service

import (
	"math/big"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestProgressiveDelta_NeverDoubleOrUnderBills is the safety proof: for any
// non-decreasing sequence of cumulative quantities (interim points + the final
// close), the sum of the per-step deltas equals rate(final) exactly — for every
// charge model. So progressive billing never double-bills and never under-bills
// relative to classic arrears.
func TestProgressiveDelta_NeverDoubleOrUnderBills(t *testing.T) {
	tiers := []domain.ChargeTier{
		{UpTo: i64(100), UnitAmount: "1"},
		{UpTo: i64(1000), UnitAmount: "0.50"},
		{UpTo: nil, UnitAmount: "0.10", FlatAmount: 500},
	}
	pctTiers := []domain.ChargeTier{
		{UpTo: i64(1_000_000), Rate: "3"},
		{UpTo: nil, Rate: "2", FlatAmount: 500},
	}

	cases := []struct {
		name    string
		model   domain.ChargeModel
		amounts domain.ChargeAmounts
		// non-decreasing cumulative quantities: interim snapshots then the final.
		cumulative []int64
	}{
		{"per_unit", domain.ChargePerUnit, domain.ChargeAmounts{UnitAmount: "0.0035"},
			[]int64{0, 500, 500, 1500, 9999}},
		{"graduated crossing tiers", domain.ChargeGraduated, domain.ChargeAmounts{Tiers: tiers},
			[]int64{0, 50, 100, 150, 1000, 1001, 5000}},
		// volume is intentionally absent — its fee is non-monotonic, so it is
		// ProgressiveBillingEligible=false (see the dedicated test below).
		{"package (bundles)", domain.ChargePackage, domain.ChargeAmounts{PackageAmount: 500, PackageSize: 1000},
			[]int64{0, 1, 1000, 1001, 5000}},
		{"percentage of value", domain.ChargePercentage, domain.ChargeAmounts{Rate: "2.5", FixedAmount: 30},
			[]int64{0, 10000, 10000, 100000, 250000}},
		{"graduated_percentage", domain.ChargeGraduatedPercentage, domain.ChargeAmounts{Tiers: pctTiers},
			[]int64{0, 500_000, 1_000_000, 1_500_000, 3_000_000}},
		{"dynamic (summed amount)", domain.ChargeDynamic, domain.ChargeAmounts{},
			[]int64{0, 199, 199, 4200, 99999}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var billed int64 // the watermark
			var totalDeltas int64
			for i, cum := range tc.cumulative {
				delta, newWM, err := progressiveDelta(tc.model, tc.amounts, new(big.Rat).SetInt64(cum), billed)
				if err != nil {
					t.Fatalf("step %d (cum=%d): %v", i, cum, err)
				}
				if delta < 0 {
					t.Fatalf("step %d: negative delta %d", i, delta)
				}
				if newWM < billed {
					t.Fatalf("step %d: watermark rewound %d -> %d", i, billed, newWM)
				}
				billed = newWM
				totalDeltas += delta
			}

			// The sum of all deltas must equal what a single arrears bill of the
			// final cumulative quantity would produce — exactly.
			final := tc.cumulative[len(tc.cumulative)-1]
			want, err := RateCharge(tc.model, tc.amounts, final)
			if err != nil {
				t.Fatalf("final rate: %v", err)
			}
			if totalDeltas != want {
				t.Fatalf("Σ deltas = %d, want rate(final=%d) = %d (double/under-bill!)", totalDeltas, final, want)
			}
			if billed != want {
				t.Fatalf("final watermark = %d, want %d", billed, want)
			}
		})
	}
}

// TestProgressiveBillingEligibility documents which models are safe: every
// model except volume (whose whole-quantity re-tiering makes the fee drop as
// usage grows, which would over-bill under the watermark).
func TestProgressiveBillingEligibility(t *testing.T) {
	if domain.ProgressiveBillingEligible(domain.ChargeVolume) {
		t.Fatal("volume must NOT be progressive-billing eligible (non-monotonic fee)")
	}
	for _, m := range []domain.ChargeModel{
		domain.ChargePerUnit, domain.ChargeGraduated, domain.ChargePackage,
		domain.ChargePercentage, domain.ChargeGraduatedPercentage, domain.ChargeDynamic,
	} {
		if !domain.ProgressiveBillingEligible(m) {
			t.Fatalf("%s should be progressive-billing eligible", m)
		}
	}
}

// TestProgressiveDelta_RepeatedSnapshotBillsNothing: re-running at the same
// cumulative (a retry) bills 0 — the idempotency property that prevents
// double-billing on a re-swept period.
func TestProgressiveDelta_RepeatedSnapshotBillsNothing(t *testing.T) {
	amounts := domain.ChargeAmounts{UnitAmount: "1"}
	delta1, wm1, _ := progressiveDelta(domain.ChargePerUnit, amounts, big.NewRat(100, 1), 0)
	if delta1 != 10000 {
		t.Fatalf("first bill = %d, want 10000", delta1)
	}
	// Same cumulative, watermark already advanced -> bill 0.
	delta2, wm2, _ := progressiveDelta(domain.ChargePerUnit, amounts, big.NewRat(100, 1), wm1)
	if delta2 != 0 || wm2 != wm1 {
		t.Fatalf("retry bill = %d (wm %d), want 0 (wm %d)", delta2, wm2, wm1)
	}
}
