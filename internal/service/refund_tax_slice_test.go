package service

import "testing"

// A series of partial refunds must never reverse more tax than the invoice
// carried. Independently rounded slices do (two 50% refunds of a 101-tax
// invoice each round to 51 → 102); cumulative allocation sums to exactly 101.
func TestRefundTaxSlice_CumulativeNeverExceedsInvoiceTax(t *testing.T) {
	// 101 tax on a 202 total: each half-refund's share is 50.5, which rounds
	// (half away from zero) to 51 — twice over, 102 > 101.
	const invoiceTax, invoiceTotal = int64(101), int64(202)
	half := invoiceTotal / 2 // 101
	first := refundTaxSlice(half, 0, invoiceTax, invoiceTotal)
	second := refundTaxSlice(invoiceTotal-half, half, invoiceTax, invoiceTotal)
	if first+second != invoiceTax {
		t.Fatalf("slices %d + %d = %d, want exactly %d", first, second, first+second, invoiceTax)
	}
	if independent := refundTaxPortion(half, invoiceTax, invoiceTotal) + refundTaxPortion(invoiceTotal-half, invoiceTax, invoiceTotal); independent <= invoiceTax {
		t.Fatalf("test premise broken: independent rounding gave %d, expected over-reversal", independent)
	}

	// Many small refunds on a larger invoice: the running sum tracks the
	// cumulative proportion and never exceeds the invoice tax.
	const bigTax, bigTotal = int64(101), int64(1101)
	var sum, refunded int64
	for i := 0; i < 11; i++ {
		s := refundTaxSlice(100, refunded, bigTax, bigTotal)
		sum += s
		refunded += 100
		if sum > bigTax {
			t.Fatalf("after %d refunds tax reversed %d exceeds %d", i+1, sum, bigTax)
		}
	}
	if sum != refundTaxPortion(refunded, bigTax, bigTotal) {
		t.Fatalf("running sum %d != cumulative proportion %d", sum, refundTaxPortion(refunded, bigTax, bigTotal))
	}

	// A single refund is unchanged from the proportional rule.
	if got, want := refundTaxSlice(118000, 0, 18000, 118000), refundTaxPortion(118000, 18000, 118000); got != want {
		t.Fatalf("single refund: %d, want %d", got, want)
	}
	// Negative prior is treated as zero.
	if refundTaxSlice(500, -1, 100, 1000) != refundTaxSlice(500, 0, 100, 1000) {
		t.Fatalf("negative prior must clamp to zero")
	}
}
