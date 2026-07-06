package tax

import (
	"strings"

	"github.com/swapnull-in/recur-so/internal/core/port"
)

// EU country codes
var euCountries = map[string]bool{
	"AT": true, "BE": true, "BG": true, "HR": true, "CY": true,
	"CZ": true, "DK": true, "EE": true, "FI": true, "FR": true,
	"DE": true, "GR": true, "HU": true, "IE": true, "IT": true,
	"LV": true, "LT": true, "LU": true, "MT": true, "NL": true,
	"PL": true, "PT": true, "RO": true, "SK": true, "SI": true,
	"ES": true, "SE": true,
}

// NewTaxEngine creates the appropriate tax engine based on the seller's country and state.
func NewTaxEngine(country, state string) port.TaxEngine {
	return NewTaxEngineWithSalesTaxProvider(country, state, nil)
}

// NewTaxEngineWithSalesTaxProvider is NewTaxEngine with an optional US
// sales-tax rate provider threaded through. Only the US engine uses it; a
// nil provider keeps the US engine as the historical 0%-rate stub.
func NewTaxEngineWithSalesTaxProvider(country, state string, salesTax SalesTaxProvider) port.TaxEngine {
	country = strings.ToUpper(strings.TrimSpace(country))
	state = strings.ToUpper(strings.TrimSpace(state))

	switch {
	case country == "IN":
		return NewGSTEngine(state)
	case country == "US":
		return NewUSSalesTaxEngineWithProvider(state, salesTax)
	case euCountries[country]:
		return NewEUVATEngine(country)
	case country == "GB":
		// UK uses VAT but post-Brexit has its own rules
		return NewEUVATEngine(country)
	default:
		// Default to GST for unsupported countries (India-focused product)
		return NewGSTEngine(state)
	}
}
