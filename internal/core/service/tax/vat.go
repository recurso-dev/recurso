package tax

import (
	"context"
	"errors"
	"math"

	"github.com/recurso-dev/recurso/internal/core/port"
)

// euVATRates holds the STANDARD VAT rate for every EU member state (plus GB
// for post-Brexit reference). One entry per country, kept in a single
// maintainable table so rate changes are a one-line edit.
//
// As of: 2026-01 (verified against European Commission / Tax Foundation
// 2026 VAT tables). All 27 EU member states are present; the rate is the
// headline STANDARD rate only.
//
// SCOPE: reduced, super-reduced, and parking rates are intentionally OUT OF
// SCOPE — this table is standard-rate-only. Categorising a supply into a
// reduced band requires product-level classification the billing engine does
// not carry, so digital-service SaaS invoices always fall under the standard
// rate here.
//
// Recent standard-rate changes already reflected: EE 22->24% (Jul 2025),
// FI 24->25.5% (Sep 2024), SK 20->23% (Jan 2025), RO 19->21% (Aug 2025).
var euVATRates = map[string]float64{
	"AT": 0.20,  // Austria
	"BE": 0.21,  // Belgium
	"BG": 0.20,  // Bulgaria
	"HR": 0.25,  // Croatia
	"CY": 0.19,  // Cyprus
	"CZ": 0.21,  // Czech Republic
	"DK": 0.25,  // Denmark
	"EE": 0.24,  // Estonia (24% since 1 Jul 2025)
	"FI": 0.255, // Finland (25.5% since 1 Sep 2024)
	"FR": 0.20,  // France
	"DE": 0.19,  // Germany
	"GR": 0.24,  // Greece
	"HU": 0.27,  // Hungary (highest standard rate in the EU)
	"IE": 0.23,  // Ireland
	"IT": 0.22,  // Italy
	"LV": 0.21,  // Latvia
	"LT": 0.21,  // Lithuania
	"LU": 0.17,  // Luxembourg (lowest standard rate in the EU)
	"MT": 0.18,  // Malta
	"NL": 0.21,  // Netherlands
	"PL": 0.23,  // Poland
	"PT": 0.23,  // Portugal
	"RO": 0.21,  // Romania (21% since 1 Aug 2025)
	"SK": 0.23,  // Slovakia (23% since 1 Jan 2025)
	"SI": 0.22,  // Slovenia
	"ES": 0.21,  // Spain
	"SE": 0.25,  // Sweden
	"GB": 0.20,  // United Kingdom — NOT EU (post-Brexit reference only)
}

// VAT-number validation contract. Mirrors the SalesTaxProvider port: a narrow
// interface the EU-VAT path depends on, with a concrete implementation (VIES)
// living under internal/adapter/vatprovider. Sentinel errors live here (not in
// the adapter) so the resolver can classify failures — definitive-invalid vs.
// service-outage — without importing the adapter.
var (
	// ErrVATInvalidFormat: the number failed local per-country format
	// validation and was never sent to the network. Treat as definitively
	// not eligible for reverse charge.
	ErrVATInvalidFormat = errors.New("vat: number failed local format validation")
	// ErrVATInvalidInput: the validation service rejected the input
	// (unsupported country code / malformed number). Also definitive.
	ErrVATInvalidInput = errors.New("vat: validation service rejected input")
	// ErrVATUnavailable: network failure or the member-state registry was
	// unreachable (5xx / MS_UNAVAILABLE) after the single retry. The
	// validity is UNKNOWN — callers should degrade, not deny.
	ErrVATUnavailable = errors.New("vat: validation service unavailable")
)

// VATValidator is the narrow port a VAT-number validation service (VIES, ...)
// must satisfy. Implementations live under internal/adapter/vatprovider.
type VATValidator interface {
	// Name identifies the validator ("vies") for invoice notes and logs.
	Name() string
	// ValidateVAT format-checks locally first, then confirms the number is
	// registered. countryCode is the ISO 3166-1 alpha-2 member-state code;
	// vatNumber is the national number WITHOUT the country prefix. Errors are
	// one of the sentinels above (via errors.Is).
	ValidateVAT(ctx context.Context, countryCode, vatNumber string) (*VATValidation, error)
}

// VATValidation is a validator's answer for one VAT number.
type VATValidation struct {
	Valid   bool   // Whether the number is registered and active
	Name    string // Registered trader name if disclosed ("" or "---" otherwise)
	Address string // Registered address if disclosed
}

// IsEUVATCountry reports whether cc is an EU member state for VAT purposes
// (GB excluded — it left the EU VAT area post-Brexit). Exported so the tax
// resolver can identify intra-EU cross-border reverse-charge candidates
// without duplicating the country table.
func IsEUVATCountry(cc string) bool { return isEU(cc) }

// StandardVATRate returns the standard VAT rate for cc and whether cc is in
// the table. Exported for tests and callers that need the raw rate.
func StandardVATRate(cc string) (float64, bool) {
	r, ok := euVATRates[cc]
	return r, ok
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

	// Export outside the EU: a genuine CROSS-BORDER supply to a non-EU buyer is out
	// of scope for EU VAT — 0% for both B2B and B2C. This must NOT fire for a
	// DOMESTIC sale by a non-EU seller that this engine still serves (e.g. a GB
	// seller to a GB buyer — isEU("GB") is false, but GB->GB is domestic 20% UK
	// VAT, not an export), so require buyer != seller (PHASE2 #5).
	if !isEU(buyerCountry) && buyerCountry != e.SellerCountry {
		note := "Export outside EU: zero-rated"
		if !req.IsBusiness {
			note = "B2C digital service outside EU: out of scope (no EU VAT)"
		}
		return &port.TaxCalculation{
			TotalTax: 0,
			TaxRate:  0,
			TaxType:  "export_exempt",
			Note:     note,
		}, nil
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

	vatAmount := int64(math.Round(float64(req.Amount) * rate))

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

	// Export outside EU: zero-rated for both B2B and B2C, but only for a genuine
	// cross-border supply — a domestic non-EU-seller sale (GB->GB) is not an
	// export (PHASE2 #5).
	if !isEU(buyerCountry) && buyerCountry != e.SellerCountry {
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
