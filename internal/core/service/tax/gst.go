package tax

import (
	"context"
	"math"
	"strings"

	"github.com/recurso-dev/recurso/internal/core/port"
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
	rate := e.rateForHSN(req.HSNCode, req.FallbackRate)

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
		totalTax := int64(math.Round(float64(req.Amount) * rate))
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

	totalTax := int64(math.Round(float64(req.Amount) * rate))

	calc := &port.TaxCalculation{
		TotalTax: totalTax,
		TaxRate:  rate,
	}

	if customerState == "" || customerState != orgState {
		// Inter-state supply: the whole tax is a single IGST component.
		calc.IGST = totalTax
		calc.TaxType = "inter_state"
	} else {
		// Intra-state supply: CGST and SGST are each levied at half the GST rate
		// on the same base, so they are computed independently and are always
		// equal — Indian GST requires CGST == SGST on every invoice. The tax total
		// is their sum, which keeps TotalTax == CGST + SGST exactly. (Splitting a
		// pre-rounded combined total instead would make the halves differ by a
		// paisa on odd totals — e.g. 2 and 3 — which is non-compliant.)
		half := int64(math.Round(float64(req.Amount) * rate / 2))
		calc.CGST = half
		calc.SGST = half
		calc.TotalTax = half * 2
		calc.TaxType = "intra_state"
	}

	return calc, nil
}

// GetApplicableRate returns the GST rate for a given request
func (e *GSTEngine) GetApplicableRate(ctx context.Context, req *port.TaxRequest) (float64, error) {
	if req.IsSEZ || req.IsExport {
		return 0, nil
	}
	return e.rateForHSN(req.HSNCode, req.FallbackRate), nil
}

// rateForHSN looks up the GST rate for an HSN code, checking progressively
// shorter prefixes. When the code isn't recognized, it uses the caller's
// fallback rate (the tenant's configured gst_rate, as a fraction) if provided,
// otherwise the built-in SaaS default.
func (e *GSTEngine) rateForHSN(hsn string, fallback float64) float64 {
	// Try exact match, then progressively shorter prefixes.
	for len(hsn) > 0 {
		if rate, ok := hsnRateMap[hsn]; ok {
			return rate
		}
		hsn = hsn[:len(hsn)-1]
	}
	if fallback > 0 {
		return fallback
	}
	return 0.18 // Built-in default (SaaS)
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
