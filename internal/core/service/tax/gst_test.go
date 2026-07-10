package tax

import (
	"testing"
)

func TestGSTCalculator(t *testing.T) {
	// Org is in TN
	engine := NewGSTEngine("TN")

	tests := []struct {
		name          string
		amount        int64
		pos           string
		expectedTotal int64
		expectedIGST  int64
		expectedCGST  int64
		expectedSGST  int64
	}{
		{
			name:          "Intra-State (TN to TN)",
			amount:        100000, // 1000.00
			pos:           "TN",
			expectedTotal: 18000, // 18% of 100000
			expectedIGST:  0,
			expectedCGST:  9000,
			expectedSGST:  9000,
		},
		{
			name:          "Inter-State (TN to KA)",
			amount:        100000,
			pos:           "KA",
			expectedTotal: 18000,
			expectedIGST:  18000,
			expectedCGST:  0,
			expectedSGST:  0,
		},
		{
			name:          "Unknown State (Defaults to IGST for now)",
			amount:        100000,
			pos:           "",
			expectedTotal: 18000,
			expectedIGST:  18000,
			expectedCGST:  0,
			expectedSGST:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := engine.CalculateTaxLegacy(tt.amount, tt.pos)

			if res.Total != tt.expectedTotal {
				t.Errorf("Total: got %d, want %d", res.Total, tt.expectedTotal)
			}
			if res.IGST != tt.expectedIGST {
				t.Errorf("IGST: got %d, want %d", res.IGST, tt.expectedIGST)
			}
			if res.CGST != tt.expectedCGST {
				t.Errorf("CGST: got %d, want %d", res.CGST, tt.expectedCGST)
			}
			if res.SGST != tt.expectedSGST {
				t.Errorf("SGST: got %d, want %d", res.SGST, tt.expectedSGST)
			}
		})
	}
}

// TestGSTCalculator_RoundsNotTruncates proves the ENG-146 fix: tax is rounded,
// not floored, so it doesn't systematically under-collect.
func TestGSTCalculator_RoundsNotTruncates(t *testing.T) {
	engine := NewGSTEngine("TN")
	// 100003 × 18% = 18000.54 — must round to 18001, not truncate to 18000.
	// Inter-state (KA) puts the whole tax in IGST, avoiding the CGST/SGST split.
	res := engine.CalculateTaxLegacy(100003, "KA")
	if res.Total != 18001 {
		t.Errorf("Total = %d, want 18001 (rounded, not truncated 18000)", res.Total)
	}
	if res.IGST != 18001 {
		t.Errorf("IGST = %d, want 18001", res.IGST)
	}
}
