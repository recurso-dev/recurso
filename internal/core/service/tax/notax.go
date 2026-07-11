package tax

import (
	"context"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// NoTaxEngine is the default for jurisdictions the product isn't registered to
// collect tax in. It charges 0% rather than silently applying another country's
// rate (ENG-152: unsupported countries used to fall through to India GST and be
// billed 18%). The note makes the "no tax configured" decision auditable.
type NoTaxEngine struct {
	country string
}

// NewNoTaxEngine builds a zero-rate engine for the given (unsupported) country.
func NewNoTaxEngine(country string) *NoTaxEngine {
	return &NoTaxEngine{country: country}
}

// CalculateTax always returns zero tax with an explanatory note.
func (e *NoTaxEngine) CalculateTax(ctx context.Context, req *port.TaxRequest) (*port.TaxCalculation, error) {
	note := "No tax configured for this jurisdiction"
	if e.country != "" {
		note = "No tax configured for " + e.country
	}
	return &port.TaxCalculation{
		TotalTax: 0,
		TaxRate:  0,
		TaxType:  "none",
		Note:     note,
	}, nil
}

// GetApplicableRate is always 0 for an unsupported jurisdiction.
func (e *NoTaxEngine) GetApplicableRate(ctx context.Context, req *port.TaxRequest) (float64, error) {
	return 0, nil
}
