package tax

import (
	"context"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// EU VAT standard rates by country (as of 2024)
var euVATRates = map[string]float64{
	"AT": 0.20, // Austria
	"BE": 0.21, // Belgium
	"BG": 0.20, // Bulgaria
	"HR": 0.25, // Croatia
	"CY": 0.19, // Cyprus
	"CZ": 0.21, // Czech Republic
	"DK": 0.25, // Denmark
	"EE": 0.22, // Estonia
	"FI": 0.24, // Finland
	"FR": 0.20, // France
	"DE": 0.19, // Germany
	"GR": 0.24, // Greece
	"HU": 0.27, // Hungary
	"IE": 0.23, // Ireland
	"IT": 0.22, // Italy
	"LV": 0.21, // Latvia
	"LT": 0.21, // Lithuania
	"LU": 0.17, // Luxembourg
	"MT": 0.18, // Malta
	"NL": 0.21, // Netherlands
	"PL": 0.23, // Poland
	"PT": 0.23, // Portugal
	"RO": 0.19, // Romania
	"SK": 0.20, // Slovakia
	"SI": 0.22, // Slovenia
	"ES": 0.21, // Spain
	"SE": 0.25, // Sweden
	"GB": 0.20, // UK (post-Brexit, for reference)
}

// EUVATEngine handles EU VAT calculations
type EUVATEngine struct {
	SellerCountry string
}

func NewEUVATEngine(sellerCountry string) *EUVATEngine {
	if sellerCountry == "" {
		sellerCountry = "DE"
	}
	return &EUVATEngine{SellerCountry: sellerCountry}
}

// CalculateTax implements port.TaxEngine for EU VAT
func (e *EUVATEngine) CalculateTax(ctx context.Context, req *port.TaxRequest) (*port.TaxCalculation, error) {
	buyerCountry := req.BuyerCountry
	if buyerCountry == "" {
		buyerCountry = e.SellerCountry
	}

	// B2B cross-border within EU: reverse charge (0% VAT, buyer self-assesses)
	if req.IsBusiness && buyerCountry != e.SellerCountry && isEU(buyerCountry) {
		return &port.TaxCalculation{
			TotalTax:      0,
			TaxRate:       0,
			TaxType:       "vat_reverse_charge",
			ReverseCharge: true,
			Note:          "B2B cross-border EU: reverse charge applies",
		}, nil
	}

	// Export outside EU: 0% VAT
	if !isEU(buyerCountry) {
		if req.IsBusiness {
			return &port.TaxCalculation{
				TotalTax: 0,
				TaxRate:  0,
				TaxType:  "export_exempt",
				Note:     "Export outside EU: zero-rated",
			}, nil
		}
		// B2C outside EU for digital services: MOSS/OSS rules
		// Seller must register in buyer's country or use OSS
		// For simplicity, we apply buyer country rate
	}

	// Determine which country's rate to apply
	var rate float64
	if req.IsBusiness || buyerCountry == e.SellerCountry {
		// B2B domestic or same-country: seller country rate
		rate = euVATRates[e.SellerCountry]
	} else {
		// B2C cross-border (MOSS/OSS): buyer country rate
		rate = euVATRates[buyerCountry]
		if rate == 0 {
			rate = euVATRates[e.SellerCountry] // Fallback
		}
	}

	vatAmount := int64(float64(req.Amount) * rate)

	return &port.TaxCalculation{
		TotalTax:  vatAmount,
		TaxRate:   rate,
		TaxType:   "vat",
		VATRate:   rate,
		VATAmount: vatAmount,
	}, nil
}

// GetApplicableRate returns the VAT rate for a given request
func (e *EUVATEngine) GetApplicableRate(ctx context.Context, req *port.TaxRequest) (float64, error) {
	buyerCountry := req.BuyerCountry
	if buyerCountry == "" {
		buyerCountry = e.SellerCountry
	}

	// B2B cross-border: reverse charge
	if req.IsBusiness && buyerCountry != e.SellerCountry && isEU(buyerCountry) {
		return 0, nil
	}

	// Export outside EU
	if !isEU(buyerCountry) && req.IsBusiness {
		return 0, nil
	}

	if req.IsBusiness || buyerCountry == e.SellerCountry {
		return euVATRates[e.SellerCountry], nil
	}

	rate := euVATRates[buyerCountry]
	if rate == 0 {
		rate = euVATRates[e.SellerCountry]
	}
	return rate, nil
}

func isEU(country string) bool {
	_, ok := euVATRates[country]
	// Exclude GB post-Brexit
	return ok && country != "GB"
}
