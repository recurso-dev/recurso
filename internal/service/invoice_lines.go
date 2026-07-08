package service

import (
	"sort"
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
		TaxableAmount: amount, // default: no per-line discount, taxable == amount
		CreatedAt:     createdAt,
	}
}

// distributeDiscount applies an invoice-level discount D across a set of gross
// line items and returns the invoice-level GST aggregates recomputed from the
// post-discount lines. It is the multi-line discount rule for itemized invoices.
//
// Rule (Phase 3):
//   - Each line's share d_i = round(D * amount_i / Σamount) with LARGEST-REMAINDER
//     rounding, so Σ d_i == D exactly (no paisa is created or lost).
//   - line.TaxableAmount = amount_i − d_i; line.Amount stays gross (a_i).
//   - Per-line GST is recomputed on the post-discount base at the line's own
//     rate, mirroring the GST engine's rule exactly: tax = trunc(taxable × rate);
//     intra-state CGST = tax/2, SGST = tax − CGST; inter-state all IGST. A line's
//     inter/intra shape is inferred from its existing split.
//   - The header (returned igst/cgst/sgst/total) aggregates from the real lines,
//     so Σ line tax == the returned total exactly with no drift.
//
// It mutates lines in place. A non-positive discount (or empty/zero-gross input)
// leaves each line's taxable base equal to its gross amount and simply
// re-aggregates the existing per-line tax.
func distributeDiscount(lines []domain.InvoiceItem, discount int64) (igst, cgst, sgst, total int64) {
	var gross int64
	for i := range lines {
		gross += lines[i].Amount
	}

	if discount <= 0 || gross <= 0 {
		for i := range lines {
			lines[i].TaxableAmount = lines[i].Amount
			igst += lines[i].IGSTAmount
			cgst += lines[i].CGSTAmount
			sgst += lines[i].SGSTAmount
		}
		return igst, cgst, sgst, igst + cgst + sgst
	}

	// Largest-remainder apportionment of the discount by gross line amount.
	shares := make([]int64, len(lines))
	remainders := make([]float64, len(lines))
	var allocated int64
	for i := range lines {
		exact := float64(discount) * float64(lines[i].Amount) / float64(gross)
		floor := int64(exact) // amounts are non-negative, so this truncates down
		shares[i] = floor
		remainders[i] = exact - float64(floor)
		allocated += floor
	}
	order := make([]int, len(lines))
	for i := range order {
		order[i] = i
	}
	// Deterministic: larger remainder first, ties broken by original position.
	sort.SliceStable(order, func(a, b int) bool {
		return remainders[order[a]] > remainders[order[b]]
	})
	for k := int64(0); k < discount-allocated && int(k) < len(order); k++ {
		shares[order[k]]++
	}

	// Post-discount taxable base + per-line GST recompute (engine rule).
	for i := range lines {
		taxable := lines[i].Amount - shares[i]
		if taxable < 0 {
			taxable = 0
		}
		lines[i].TaxableAmount = taxable

		rate := lines[i].TaxRate / 100.0 // stored as percent (18.0 -> 0.18)
		lineTax := int64(float64(taxable) * rate)

		// Inter-state lines carry IGST; intra-state carry CGST+SGST. A zero-tax
		// line is treated as inter-state (its recomputed tax is zero regardless).
		interState := lines[i].IGSTAmount > 0 || (lines[i].CGSTAmount == 0 && lines[i].SGSTAmount == 0)
		if interState {
			lines[i].IGSTAmount = lineTax
			lines[i].CGSTAmount = 0
			lines[i].SGSTAmount = 0
			igst += lineTax
		} else {
			half := lineTax / 2
			lines[i].CGSTAmount = half
			lines[i].SGSTAmount = lineTax - half
			lines[i].IGSTAmount = 0
			cgst += half
			sgst += lineTax - half
		}
	}
	return igst, cgst, sgst, igst + cgst + sgst
}
