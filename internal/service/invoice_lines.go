package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// newInvoiceLine builds a single itemized invoice line from an amount and its
// already-computed per-line tax. It is the shared accumulator the generation
// paths append to (base plan, add-ons, and single-amount paths) so every line
// is recorded with the same shape.
//
// Phase 1 invariant: the line's Amount and its CGST/SGST/IGST are taken verbatim
// from the values that were summed into the invoice totals, so
// Σ line.Amount == invoice.Subtotal and Σ line tax == invoice.TaxAmount hold
// exactly with no re-computation or rounding drift.
//
// hsn defaults to the tenant SAC (DefaultSACCode) when empty — a line is never
// emitted with an empty HSN, since the IRP rejects that.
func newInvoiceLine(invoiceID uuid.UUID, description, hsn string, quantity int, unitAmount, amount int64, tax InvoiceTax, createdAt time.Time) domain.InvoiceItem {
	if hsn == "" {
		hsn = domain.DefaultSACCode
	}
	if quantity <= 0 {
		quantity = 1
	}
	return domain.InvoiceItem{
		ID:            uuid.New(),
		InvoiceID:     invoiceID,
		Description:   description,
		HSNCode:       hsn,
		Quantity:      quantity,
		UnitAmount:    unitAmount,
		Amount:        amount,
		TaxRate:       tax.Rate * 100.0, // fraction -> percent (e.g. 0.18 -> 18.0)
		CGSTAmount:    tax.CGST,
		SGSTAmount:    tax.SGST,
		IGSTAmount:    tax.IGST,
		TaxableAmount: amount, // Phase 1: no per-line discount, taxable == amount
		CreatedAt:     createdAt,
	}
}
