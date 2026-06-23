package tax

import (
	"context"
	"strings"

	"github.com/recur-so/recurso/internal/core/port"
)

// HSN code to GST rate mapping
var hsnRateMap = map[string]float64{
	"998314": 0.18, // SaaS / IT Services (default)
	"998313": 0.18, // Licensing services for software
	"9983":   0.18, // IT services (broad category)
	"9984":   0.18, // Telecom services
	"9985":   0.18, // Transport
	"9982":   0.18, // Legal/accounting services
	"9981":   0.18, // R&D services
	"9973":   0.18, // Leasing/rental
	"9972":   0.12, // Real estate
	"9963":   0.05, // Accommodation (budget)
	"9964":   0.18, // Passenger transport
	"9992":   0.18, // Educational support
	"9991":   0.05, // Government services
	"9971":   0.12, // Financial services
	"8471":   0.18, // Computer hardware
	"4901":   0.05, // Books/periodicals
	"0":      0.05, // Essential goods
}

// GSTEngine handles India GST calculations
type GSTEngine struct {
	OrganizationState string
}

func NewGSTEngine(orgState string) *GSTEngine {
	if orgState == "" {
		orgState = "TN"
	}
	return &GSTEngine{OrganizationState: orgState}
}

// TaxResult is the legacy result type (kept for backward compatibility)
type TaxResult struct {
	IGST    int64
	CGST    int64
	SGST    int64
	Total   int64
	TaxType string // "inter_state" or "intra_state"
}

// CalculateTax implements port.TaxEngine
func (e *GSTEngine) CalculateTax(ctx context.Context, req *port.TaxRequest) (*port.TaxCalculation, error) {
	rate := e.rateForHSN(req.HSNCode)

	// SEZ/Export: 0% tax (zero-rated supply)
	if req.IsSEZ || req.IsExport {
		return &port.TaxCalculation{
			TotalTax: 0,
			TaxRate:  0,
			TaxType:  "exempt",
			Note:     "Zero-rated supply (SEZ/Export)",
		}, nil
	}

	// Reverse Charge Mechanism
	if req.IsRCM {
		totalTax := int64(float64(req.Amount) * rate)
		return &port.TaxCalculation{
			TotalTax:      totalTax,
			TaxRate:       rate,
			TaxType:       "rcm",
			ReverseCharge: true,
			Note:          "Tax under Reverse Charge Mechanism - buyer to remit",
		}, nil
	}

	customerState := strings.ToUpper(strings.TrimSpace(req.BuyerState))
	orgState := strings.ToUpper(strings.TrimSpace(e.OrganizationState))

	totalTax := int64(float64(req.Amount) * rate)

	calc := &port.TaxCalculation{
		TotalTax: totalTax,
		TaxRate:  rate,
	}

	if customerState == "" || customerState != orgState {
		// Inter-state supply (IGST)
		calc.IGST = totalTax
		calc.TaxType = "inter_state"
	} else {
		// Intra-state supply (CGST + SGST)
		halfTax := totalTax / 2
		calc.CGST = halfTax
		calc.SGST = totalTax - halfTax
		calc.TaxType = "intra_state"
	}

	return calc, nil
}

// GetApplicableRate returns the GST rate for a given request
func (e *GSTEngine) GetApplicableRate(ctx context.Context, req *port.TaxRequest) (float64, error) {
	if req.IsSEZ || req.IsExport {
		return 0, nil
	}
	return e.rateForHSN(req.HSNCode), nil
}

// rateForHSN looks up the GST rate for an HSN code, checking progressively shorter prefixes
func (e *GSTEngine) rateForHSN(hsn string) float64 {
	if hsn == "" {
		return 0.18 // Default SaaS rate
	}
	// Try exact match, then progressively shorter prefixes
	for len(hsn) > 0 {
		if rate, ok := hsnRateMap[hsn]; ok {
			return rate
		}
		hsn = hsn[:len(hsn)-1]
	}
	return 0.18 // Default
}

// CalculateTaxLegacy is the backward-compatible function with old signature
func (e *GSTEngine) CalculateTaxLegacy(amount int64, placeOfSupply string) TaxResult {
	calc, _ := e.CalculateTax(context.Background(), &port.TaxRequest{
		Amount:     amount,
		BuyerState: placeOfSupply,
		HSNCode:    "998314", // Default SaaS
	})
	if calc == nil {
		return TaxResult{}
	}
	return TaxResult{
		IGST:    calc.IGST,
		CGST:    calc.CGST,
		SGST:    calc.SGST,
		Total:   calc.TotalTax,
		TaxType: calc.TaxType,
	}
}
