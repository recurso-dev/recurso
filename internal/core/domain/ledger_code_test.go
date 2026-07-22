package domain

import "testing"

// TestLedgerCodesAreUnique guards against the class of bug the downgrade codes
// hit: the (reference_id, code) unique index (uq_ledger_tx_reference_code)
// dedups on code alone, so two posting types that share a value would silently
// drop one leg via ON CONFLICT DO NOTHING if they ever posted against the same
// reference_id. Every transaction code — the named constants AND the well-known
// literals used at the posting sites (1 invoice, 2 recognition, …) — must be
// distinct. Add a new code here whenever you add one; a duplicate fails the test.
func TestLedgerCodesAreUnique(t *testing.T) {
	codes := map[string]uint16{
		// Literals posted directly at ledger.go call sites (no named constant).
		"Invoice":            1,
		"RevenueRecognition": 2,
		"Payment":            3,
		"Refund":             4,
		"DeferredReversal":   5,
		"CreditApplication":  7,
		"Adjustment":         8,

		// Named constants.
		"OutputTax":            LedgerCodeOutputTax,
		"RefundTaxReversal":    LedgerCodeRefundTaxReversal,
		"TDSReceivable":        LedgerCodeTDSReceivable,
		"DowngradeCredit":      LedgerCodeDowngradeCredit,
		"DowngradeTaxReversal": LedgerCodeDowngradeTaxReversal,
		"WalletTopUp":          LedgerCodeWalletTopUp,
		"WalletDrain":          LedgerCodeWalletDrain,
		"WalletRefund":         LedgerCodeWalletRefund,
		"WalletForfeit":        LedgerCodeWalletForfeit,
	}

	seen := make(map[uint16]string, len(codes))
	for name, code := range codes {
		if prev, dup := seen[code]; dup {
			t.Errorf("ledger code %d is shared by %q and %q — the (reference_id, code) index would silently drop one leg", code, prev, name)
			continue
		}
		seen[code] = name
	}
}
