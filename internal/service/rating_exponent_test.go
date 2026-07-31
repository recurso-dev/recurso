package service

import (
	"testing"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestRateCharge_CurrencyExponent proves per-unit rating converts a MAJOR-unit
// rate to stored minor units using the currency's exponent, not a hardcoded 100.
// Before the fix, JPY (exp 0) was over-priced 100× and KWD (exp 3) under-priced
// 10×, because the engine always multiplied by 100.
func TestRateCharge_CurrencyExponent(t *testing.T) {
	// ¥5 per unit; 100 units = ¥500 = 500 minor units (JPY has no minor unit).
	jpy := domain.ChargeAmounts{UnitAmount: "5"}
	if got, err := RateCharge(domain.ChargePerUnit, jpy, 100, "JPY"); err != nil {
		t.Fatalf("JPY: %v", err)
	} else if got != 500 {
		t.Errorf("JPY per-unit = %d, want 500 (the pre-fix ×100 bug gave 50000)", got)
	}

	// $5 per unit; 100 units = $500 = 50000 minor units (2 decimals).
	usd := domain.ChargeAmounts{UnitAmount: "5"}
	if got, err := RateCharge(domain.ChargePerUnit, usd, 100, "USD"); err != nil {
		t.Fatalf("USD: %v", err)
	} else if got != 50000 {
		t.Errorf("USD per-unit = %d, want 50000", got)
	}

	// KWD 0.005 per unit; 1000 units = KWD 5.000 = 5000 minor units (3 decimals).
	kwd := domain.ChargeAmounts{UnitAmount: "0.005"}
	if got, err := RateCharge(domain.ChargePerUnit, kwd, 1000, "KWD"); err != nil {
		t.Fatalf("KWD: %v", err)
	} else if got != 5000 {
		t.Errorf("KWD per-unit = %d, want 5000 (the pre-fix ×100 bug gave 500)", got)
	}
}
