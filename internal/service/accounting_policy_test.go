package service

import "testing"

// TestPolicyResolver_USDefault pins the increment-1 foundation: the US policy
// is accrual recognition with no bad-debt tax relief (US sales tax is a
// collected liability, not output tax on revenue), and an unknown jurisdiction
// falls back to accrual. Wiring the resolver changes no ledger behavior on its
// own — the schedule-at-issuance switch is gated separately (increment 2).
func TestPolicyResolver_USDefault(t *testing.T) {
	r := NewPolicyResolver()

	us := r.For("us") // case-insensitive
	if us.RevenueRecognition != RecognitionAccrual {
		t.Errorf("US recognition = %q, want accrual", us.RevenueRecognition)
	}
	if us.BadDebt.AllowTaxRelief {
		t.Error("US bad-debt tax relief should be false (sales tax is a collected liability)")
	}

	// Unknown jurisdiction → fallback (accrual).
	if got := r.For("ZZ").RevenueRecognition; got != RecognitionAccrual {
		t.Errorf("fallback recognition = %q, want accrual", got)
	}
}

// TestPolicyResolver_RegisterAdapter proves the jurisdiction-adapter seam: a
// per-country policy (e.g. a UK VAT adapter permitting bad-debt relief) can be
// registered and resolved, so international expansion is adding an adapter, not
// editing the ledger.
func TestPolicyResolver_RegisterAdapter(t *testing.T) {
	r := NewPolicyResolver()
	r.Register("gb", AccountingPolicy{
		RevenueRecognition: RecognitionAccrual,
		BadDebt: BadDebtTreatment{
			AllowTaxRelief:       true,
			RecognitionDelayDays: 180, // UK: 6 months overdue before VAT bad-debt relief
			RecoverableTaxes:     []string{"vat"},
		},
	})
	gb := r.For("GB")
	if !gb.BadDebt.AllowTaxRelief || gb.BadDebt.RecognitionDelayDays != 180 {
		t.Errorf("registered GB adapter not resolved: %+v", gb.BadDebt)
	}
	// Registering GB must not disturb the US default.
	if r.For("US").BadDebt.AllowTaxRelief {
		t.Error("US policy leaked GB's tax-relief setting")
	}
}

// TestNilPolicyResolver_SafeDefault confirms a nil resolver degrades to the
// safe current behavior (cash recognition) rather than panicking — the nil-safe
// wiring idiom.
func TestNilPolicyResolver_SafeDefault(t *testing.T) {
	var r *PolicyResolver
	if got := r.For("US").RevenueRecognition; got != RecognitionCash {
		t.Errorf("nil resolver = %q, want cash (safe default)", got)
	}
}
