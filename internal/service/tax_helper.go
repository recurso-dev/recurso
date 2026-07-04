package service

import (
	"strings"

	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/service/tax"
)

// calculateInvoiceGST applies Indian GST for INR amounts. Invoices in other
// currencies carry no GST — their jurisdictions (VAT, US sales tax) have
// engines in core/service/tax but are not yet wired into invoice generation,
// so charging 18% IGST on them would be wrong.
// Org state is fixed to "TN" pending per-tenant configuration.
func calculateInvoiceGST(currency string, amount int64, placeOfSupply *string) tax.TaxResult {
	if !strings.EqualFold(strings.TrimSpace(currency), "INR") {
		return tax.TaxResult{}
	}
	engine := tax.NewGSTEngine("TN")
	return engine.CalculateTaxLegacy(amount, domain.PtrToString(placeOfSupply))
}
