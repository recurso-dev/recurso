package tax

import (
	"context"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// USSalesTaxEngine is a stub implementation for US sales tax.
// US sales tax is jurisdiction-level (state, county, city) and requires
// integration with services like TaxJar, Avalara, or Vertex for accurate rates.
type USSalesTaxEngine struct {
	State string
}

func NewUSSalesTaxEngine(state string) *USSalesTaxEngine {
	return &USSalesTaxEngine{State: state}
}

// CalculateTax returns 0% tax with a note to integrate a tax provider.
// US SaaS sales tax varies by state, and many states exempt SaaS or have
// special rules. A real implementation requires TaxJar/Avalara integration.
func (e *USSalesTaxEngine) CalculateTax(ctx context.Context, req *port.TaxRequest) (*port.TaxCalculation, error) {
	return &port.TaxCalculation{
		TotalTax: 0,
		TaxRate:  0,
		TaxType:  "sales_tax_stub",
		Note:     "US sales tax requires TaxJar/Avalara integration for accurate jurisdiction-level rates. Currently returning 0% as placeholder.",
	}, nil
}

// GetApplicableRate returns 0 with a note about integration requirements
func (e *USSalesTaxEngine) GetApplicableRate(ctx context.Context, req *port.TaxRequest) (float64, error) {
	return 0, nil
}
