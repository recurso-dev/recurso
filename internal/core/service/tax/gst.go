package tax

import (
	"strings"
)

// GSTEngine handles India GST calculations
type GSTEngine struct {
	OrganizationState string
}

func NewGSTEngine(orgState string) *GSTEngine {
	// Default to a state if not provided, e.g. "TN" for Chennai based HQ
	if orgState == "" {
		orgState = "TN" 
	}
	return &GSTEngine{OrganizationState: orgState}
}

type TaxResult struct {
	IGST    int64
	CGST    int64
	SGST    int64
	Total   int64
	TaxType string // "inter_state" or "intra_state"
}

// CalculateTax calculates GST based on customer's Place of Supply
// amount is in cents
func (e *GSTEngine) CalculateTax(amount int64, placeOfSupply string) TaxResult {
	// Default GST Rate: 18% (Standard for SaaS)
	// Can be made configurable per plan/product later via HSN mapping
	const GSTRate = 0.18

	// Clean up input
	customerState := strings.ToUpper(strings.TrimSpace(placeOfSupply))
	orgState := strings.ToUpper(strings.TrimSpace(e.OrganizationState))

	totalTax := int64(float64(amount) * GSTRate)
	
	res := TaxResult{
		Total: totalTax,
	}

	if customerState == "" || customerState != orgState {
		// Inter-state supply (IGST)
		// Or if customer state is unknown, usually fallback to IGST or treat as export (Zero rated)
		// For domestic MVP, we assume IGST if different.
		res.IGST = totalTax
		res.TaxType = "inter_state"
	} else {
		// Intra-state supply (CGST + SGST)
		halfTax := totalTax / 2
		res.CGST = halfTax
		res.SGST = totalTax - halfTax // Handle odd numbers by giving remainder to SGST? Or just floor. 
		// Actually best to maintain integer math: 
		// CGST = 9%
		// SGST = 9%
		res.TaxType = "intra_state"
	}

	return res
}
