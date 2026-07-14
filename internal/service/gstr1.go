package service

import (
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// gstRate returns the combined GST rate as a whole-ish percentage (GST rates are
// 0/5/12/18/28), derived from the tax over the taxable value. Zero taxable value
// yields a 0 rate.
func gstRate(taxableValue, totalTax int64) float64 {
	if taxableValue <= 0 {
		return 0
	}
	return math.Round(float64(totalTax) / float64(taxableValue) * 100)
}

// BuildGSTR1 assembles a return-ready GSTR-1 from a tenant's finalized invoices
// for a tax period. Registered buyers (with a GSTIN) go to the invoice-level B2B
// section; unregistered buyers are summarized rate-wise per place of supply
// (B2CS); every invoice contributes to the HSN rollup and the control totals.
// Pure — no database — so the bucketing logic is unit-testable in isolation.
func BuildGSTR1(tenantID uuid.UUID, month, year int, invoices []domain.GSTR1Invoice) *domain.GSTR1Return {
	ret := &domain.GSTR1Return{TenantID: tenantID, Month: month, Year: year}

	b2b := map[string]*domain.GSTR1B2B{}
	b2cs := map[string]*domain.GSTR1B2CS{}
	hsn := map[string]*domain.GSTR1HSNSummary{}

	for _, inv := range invoices {
		tax := inv.IGST + inv.CGST + inv.SGST
		rate := gstRate(inv.TaxableValue, tax)

		ret.TotalTaxableValue += inv.TaxableValue
		ret.TotalIGST += inv.IGST
		ret.TotalCGST += inv.CGST
		ret.TotalSGST += inv.SGST
		ret.InvoiceCount++

		// HSN rollup (every invoice, regardless of buyer type).
		h := hsn[inv.HSNCode]
		if h == nil {
			h = &domain.GSTR1HSNSummary{HSNCode: inv.HSNCode}
			hsn[inv.HSNCode] = h
		}
		h.TaxableValue += inv.TaxableValue
		h.IGST += inv.IGST
		h.CGST += inv.CGST
		h.SGST += inv.SGST
		h.InvoiceCount++

		if inv.BuyerGSTIN != "" {
			g := b2b[inv.BuyerGSTIN]
			if g == nil {
				g = &domain.GSTR1B2B{GSTIN: inv.BuyerGSTIN}
				b2b[inv.BuyerGSTIN] = g
			}
			g.Invoices = append(g.Invoices, domain.GSTR1B2BInvoice{
				InvoiceNumber: inv.InvoiceNumber,
				Date:          inv.Date,
				PlaceOfSupply: inv.PlaceOfSupply,
				TaxableValue:  inv.TaxableValue,
				IGST:          inv.IGST,
				CGST:          inv.CGST,
				SGST:          inv.SGST,
				Rate:          rate,
			})
		} else {
			key := fmt.Sprintf("%s|%g", inv.PlaceOfSupply, rate)
			c := b2cs[key]
			if c == nil {
				c = &domain.GSTR1B2CS{PlaceOfSupply: inv.PlaceOfSupply, Rate: rate}
				b2cs[key] = c
			}
			c.TaxableValue += inv.TaxableValue
			c.IGST += inv.IGST
			c.CGST += inv.CGST
			c.SGST += inv.SGST
		}
	}

	// Deterministic ordering so the output (and any golden-file test) is stable.
	for _, g := range b2b {
		sort.Slice(g.Invoices, func(i, j int) bool { return g.Invoices[i].InvoiceNumber < g.Invoices[j].InvoiceNumber })
		ret.B2B = append(ret.B2B, *g)
	}
	sort.Slice(ret.B2B, func(i, j int) bool { return ret.B2B[i].GSTIN < ret.B2B[j].GSTIN })

	for _, c := range b2cs {
		ret.B2CS = append(ret.B2CS, *c)
	}
	sort.Slice(ret.B2CS, func(i, j int) bool {
		if ret.B2CS[i].PlaceOfSupply != ret.B2CS[j].PlaceOfSupply {
			return ret.B2CS[i].PlaceOfSupply < ret.B2CS[j].PlaceOfSupply
		}
		return ret.B2CS[i].Rate < ret.B2CS[j].Rate
	})

	for _, h := range hsn {
		ret.HSN = append(ret.HSN, *h)
	}
	sort.Slice(ret.HSN, func(i, j int) bool { return ret.HSN[i].HSNCode < ret.HSN[j].HSNCode })

	return ret
}
