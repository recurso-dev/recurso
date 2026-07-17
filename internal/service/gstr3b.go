package service

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// GetGSTR3B assembles the GSTR-3B summary return for a calendar month from the
// same period inputs as GSTR-1 (finalized invoices + refund credit notes), so
// the two returns are consistent by construction.
func (s *GSTRService) GetGSTR3B(ctx context.Context, tenantID uuid.UUID, month, year int) (*domain.GSTR3BReturn, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	invoices, err := s.src.GetGSTR1Invoices(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	creditNotes, err := s.src.GetGSTR1CreditNotes(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}
	return BuildGSTR3B(tenantID, month, year, invoices, creditNotes), nil
}

// BuildGSTR3B assembles a return-ready GSTR-3B from a tenant's finalized
// invoices and credit notes for a tax period. Table 3.1(a) nets ALL credit
// notes (registered and unregistered) from the invoice totals — GSTR-3B is a
// net summary, unlike GSTR-1's separate CDNR section. Table 3.2 reports the
// inter-state (IGST) share supplied to unregistered buyers per place of
// supply, likewise net of that bucket's credit notes. Pure — no database — so
// the netting logic is unit-testable in isolation.
func BuildGSTR3B(tenantID uuid.UUID, month, year int, invoices []domain.GSTR1Invoice, creditNotes []domain.GSTR1CreditNote) *domain.GSTR3BReturn {
	ret := &domain.GSTR3BReturn{TenantID: tenantID, Month: month, Year: year}

	interUnreg := map[string]*domain.GSTR3BInterStateUnreg{}

	for _, inv := range invoices {
		ret.OutwardTaxable.TaxableValue += inv.TaxableValue
		ret.OutwardTaxable.IGST += inv.IGST
		ret.OutwardTaxable.CGST += inv.CGST
		ret.OutwardTaxable.SGST += inv.SGST
		ret.InvoiceCount++

		if inv.BuyerGSTIN == "" && inv.IGST > 0 {
			row := interUnreg[inv.PlaceOfSupply]
			if row == nil {
				row = &domain.GSTR3BInterStateUnreg{PlaceOfSupply: inv.PlaceOfSupply}
				interUnreg[inv.PlaceOfSupply] = row
			}
			row.TaxableValue += inv.TaxableValue
			row.IGST += inv.IGST
		}
	}

	for _, cn := range creditNotes {
		ret.OutwardTaxable.TaxableValue -= cn.TaxableValue
		ret.OutwardTaxable.IGST -= cn.IGST
		ret.OutwardTaxable.CGST -= cn.CGST
		ret.OutwardTaxable.SGST -= cn.SGST
		ret.CreditNoteCount++

		if cn.BuyerGSTIN == "" && cn.IGST > 0 {
			row := interUnreg[cn.PlaceOfSupply]
			if row == nil {
				row = &domain.GSTR3BInterStateUnreg{PlaceOfSupply: cn.PlaceOfSupply}
				interUnreg[cn.PlaceOfSupply] = row
			}
			row.TaxableValue -= cn.TaxableValue
			row.IGST -= cn.IGST
		}
	}

	for _, row := range interUnreg {
		ret.InterStateUnregistered = append(ret.InterStateUnregistered, *row)
	}
	sort.Slice(ret.InterStateUnregistered, func(i, j int) bool {
		return ret.InterStateUnregistered[i].PlaceOfSupply < ret.InterStateUnregistered[j].PlaceOfSupply
	})

	return ret
}
