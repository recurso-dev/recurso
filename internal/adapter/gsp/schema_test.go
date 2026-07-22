package gsp

import (
	"testing"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// The NIC INV-01 schema requires a numeric 6-digit PIN for both parties.
// BuildInvoiceSchema must derive it from the seller/buyer PinCode (previously
// hardcoded to 0, which the IRP rejects).
func TestBuildInvoiceSchema_PinFromPinCode(t *testing.T) {
	req := &port.EInvoiceRequest{
		Invoice: &domain.Invoice{
			InvoiceNumber: "INV-1",
			Subtotal:      100000,
			CGSTAmount:    9000,
			SGSTAmount:    9000,
			Total:         118000,
			CreatedAt:     time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC),
		},
		Seller: port.EInvoiceSeller{GSTIN: "29ABCDE1234F1Z5", PinCode: "560001", StateCode: "29"},
		Buyer:  port.EInvoiceBuyer{GSTIN: "33XYZAB6789K1Z2", PinCode: "600 001", StateCode: "33", Place: "33"},
	}

	s := BuildInvoiceSchema(req)

	if s.SellerDtls.Pin != 560001 {
		t.Errorf("seller Pin = %d, want 560001 (from PinCode)", s.SellerDtls.Pin)
	}
	if s.BuyerDtls.Pin != 600001 {
		t.Errorf("buyer Pin = %d, want 600001 (digits parsed from '600 001')", s.BuyerDtls.Pin)
	}
}

func TestParsePin(t *testing.T) {
	for in, want := range map[string]int{
		"560001":  560001,
		"600 001": 600001,
		"110-011": 110011,
		"":        0,
		"abc":     0,
	} {
		if got := parsePin(in); got != want {
			t.Errorf("parsePin(%q) = %d, want %d", in, got, want)
		}
	}
}
