package tax

import (
	"context"
	"fmt"
	"strings"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// USSalesTaxEngine calculates US sales tax. US sales tax is
// jurisdiction-level (state, county, city, district), so accurate rates
// come from an external SalesTaxProvider (TaxJar). Without a provider the
// engine is an honest 0%-rate stub and marks calculations sales_tax_stub.
type USSalesTaxEngine struct {
	State    string // Seller's state (from-address for provider lookups)
	provider SalesTaxProvider
}

// NewUSSalesTaxEngine creates the provider-less stub engine (0% rate,
// TaxType sales_tax_stub).
func NewUSSalesTaxEngine(state string) *USSalesTaxEngine {
	return NewUSSalesTaxEngineWithProvider(state, nil)
}

// NewUSSalesTaxEngineWithProvider creates the engine with an optional live
// rate provider. A nil provider yields the stub behavior. Callers that
// construct engines per invoice should pass a long-lived provider (wrapped
// in CachedSalesTaxProvider) so rate caching survives engine recreation.
func NewUSSalesTaxEngineWithProvider(state string, provider SalesTaxProvider) *USSalesTaxEngine {
	return &USSalesTaxEngine{State: strings.ToUpper(strings.TrimSpace(state)), provider: provider}
}

// HasProvider reports whether a live rate provider is wired in.
func (e *USSalesTaxEngine) HasProvider() bool { return e.provider != nil }

// ProviderName returns the wired provider's name, or "" for the stub.
func (e *USSalesTaxEngine) ProviderName() string {
	if e.provider == nil {
		return ""
	}
	return e.provider.Name()
}

// CalculateTax computes US sales tax for the request.
//
// With a provider: the buyer's state+zip (TaxRequest.BuyerState/BuyerZip)
// and the seller's state are sent to the provider; the result carries
// TaxType "sales_tax" and a note naming the provider. Provider errors are
// returned to the caller, which decides degradation policy (the
// TaxResolver degrades to 0% with TaxType sales_tax_error rather than
// failing the invoice).
//
// Without a provider: returns 0% with TaxType "sales_tax_stub", preserving
// the historical placeholder behavior.
func (e *USSalesTaxEngine) CalculateTax(ctx context.Context, req *port.TaxRequest) (*port.TaxCalculation, error) {
	if e.provider == nil {
		return &port.TaxCalculation{
			TotalTax: 0,
			TaxRate:  0,
			TaxType:  "sales_tax_stub",
			Note:     "US sales tax requires TaxJar/Avalara integration for accurate jurisdiction-level rates. Currently returning 0% as placeholder.",
		}, nil
	}

	res, err := e.provider.LookupSalesTax(ctx, e.queryFor(req, req.Amount))
	if err != nil {
		return nil, fmt.Errorf("us sales tax lookup via %s: %w", e.provider.Name(), err)
	}

	note := fmt.Sprintf("US sales tax via %s", e.provider.Name())
	if res.Jurisdiction != "" {
		note += " (" + res.Jurisdiction + ")"
	}
	if !res.HasNexus {
		note += "; no nexus in buyer state per provider — nothing to collect"
	}
	return &port.TaxCalculation{
		TotalTax: res.TaxAmount,
		TaxRate:  res.Rate,
		TaxType:  "sales_tax",
		Note:     note,
	}, nil
}

// GetApplicableRate returns the combined rate for the request's destination.
// Stub engines report 0. Provider lookups use a nominal amount when the
// request carries none, since rates are amount-independent.
func (e *USSalesTaxEngine) GetApplicableRate(ctx context.Context, req *port.TaxRequest) (float64, error) {
	if e.provider == nil {
		return 0, nil
	}
	amount := req.Amount
	if amount <= 0 {
		amount = 10000 // nominal $100.00; rate does not depend on amount
	}
	res, err := e.provider.LookupSalesTax(ctx, e.queryFor(req, amount))
	if err != nil {
		return 0, fmt.Errorf("us sales tax rate via %s: %w", e.provider.Name(), err)
	}
	return res.Rate, nil
}

// queryFor maps a port.TaxRequest to the provider query. The seller side is
// the engine's configured state; nexus configuration beyond that is the
// provider account's concern (see ROADMAP: nexus is not modeled here yet).
func (e *USSalesTaxEngine) queryFor(req *port.TaxRequest, amount int64) *SalesTaxQuery {
	toCountry := strings.ToUpper(strings.TrimSpace(req.BuyerCountry))
	if toCountry == "" {
		toCountry = "US"
	}
	return &SalesTaxQuery{
		FromCountry: "US",
		FromState:   e.State,
		ToCountry:   toCountry,
		ToState:     strings.ToUpper(strings.TrimSpace(req.BuyerState)),
		ToZip:       strings.TrimSpace(req.BuyerZip),
		Amount:      amount,
		Currency:    req.Currency,
	}
}
