package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// GSTR1Source is the read side the GSTR-1 export needs: a period's finalized
// invoices and refund credit notes, flattened with buyer GST identity.
type GSTR1Source interface {
	GetGSTR1Invoices(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]domain.GSTR1Invoice, error)
	GetGSTR1CreditNotes(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]domain.GSTR1CreditNote, error)
}

// GSTRService produces the GSTR-1 return for a tenant's tax period.
type GSTRService struct {
	src GSTR1Source
}

func NewGSTRService(src GSTR1Source) *GSTRService { return &GSTRService{src: src} }

// GetGSTR1 assembles the return for a calendar month from that month's finalized
// invoices and refund credit notes.
func (s *GSTRService) GetGSTR1(ctx context.Context, tenantID uuid.UUID, month, year int) (*domain.GSTR1Return, error) {
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
	return BuildGSTR1(tenantID, month, year, invoices, creditNotes), nil
}

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
// and credit notes for a tax period. Registered buyers (with a GSTIN) go to the
// invoice-level B2B section; unregistered buyers are summarized rate-wise per
// place of supply (B2CS); registered credit notes go to CDNR; every invoice
// contributes to the HSN rollup and the outward-supply totals. Pure — no
// database — so the bucketing logic is unit-testable in isolation.
func BuildGSTR1(tenantID uuid.UUID, month, year int, invoices []domain.GSTR1Invoice, creditNotes []domain.GSTR1CreditNote) *domain.GSTR1Return {
	ret := &domain.GSTR1Return{TenantID: tenantID, Month: month, Year: year}

	b2b := map[string]*domain.GSTR1B2B{}
	b2cs := map[string]*domain.GSTR1B2CS{}
	cdnr := map[string]*domain.GSTR1CDNR{}
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

	// Credit notes to registered buyers -> CDNR (a separate GSTR-1 section;
	// GSTR-1 does not net them from B2B). Unregistered credit notes (no GSTIN)
	// belong in CDNUR, which this v1 does not build — they are skipped here and
	// counted separately so the omission is visible, not silent.
	for _, cn := range creditNotes {
		if cn.BuyerGSTIN == "" {
			continue // CDNUR (unregistered) — out of scope for v1
		}
		tax := cn.IGST + cn.CGST + cn.SGST
		ret.TotalCreditTaxableValue += cn.TaxableValue
		ret.TotalCreditIGST += cn.IGST
		ret.TotalCreditCGST += cn.CGST
		ret.TotalCreditSGST += cn.SGST
		ret.CreditNoteCount++

		g := cdnr[cn.BuyerGSTIN]
		if g == nil {
			g = &domain.GSTR1CDNR{GSTIN: cn.BuyerGSTIN}
			cdnr[cn.BuyerGSTIN] = g
		}
		g.Notes = append(g.Notes, domain.GSTR1CDNRNote{
			NoteNumber:            cn.NoteNumber,
			Date:                  cn.Date,
			OriginalInvoiceNumber: cn.OriginalInvoiceNumber,
			PlaceOfSupply:         cn.PlaceOfSupply,
			TaxableValue:          cn.TaxableValue,
			IGST:                  cn.IGST,
			CGST:                  cn.CGST,
			SGST:                  cn.SGST,
			Rate:                  gstRate(cn.TaxableValue, tax),
		})
	}

	// Deterministic ordering so the output (and any golden-file test) is stable.
	for _, g := range cdnr {
		sort.Slice(g.Notes, func(i, j int) bool { return g.Notes[i].NoteNumber < g.Notes[j].NoteNumber })
		ret.CDNR = append(ret.CDNR, *g)
	}
	sort.Slice(ret.CDNR, func(i, j int) bool { return ret.CDNR[i].GSTIN < ret.CDNR[j].GSTIN })

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
