package einvoice_eu_test

import (
	"context"
	"testing"

	"github.com/recurso-dev/recurso/internal/adapter/einvoice_eu"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func TestMockTransport_Transmit(t *testing.T) {
	transport := einvoice_eu.NewMockTransport()
	ctx := context.Background()

	docBytes := []byte("<Invoice><ID>INV-1001</ID></Invoice>")
	res, err := transport.Transmit(ctx, domain.EUInvoiceSyntaxUBL, "9915:test-vat", docBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil transmission result")
	}

	if res.Status != domain.EUInvoiceStatusSent {
		t.Errorf("expected status %s, got %s", domain.EUInvoiceStatusSent, res.Status)
	}

	if len(res.MessageID) == 0 {
		t.Error("expected non-empty message ID")
	}

	// Verify determinism for identical payload
	res2, err := transport.Transmit(ctx, domain.EUInvoiceSyntaxUBL, "9915:test-vat", docBytes)
	if err != nil {
		t.Fatalf("unexpected error on second transmit: %v", err)
	}

	if res.MessageID != res2.MessageID {
		t.Errorf("expected deterministic MessageID %s, got %s", res.MessageID, res2.MessageID)
	}
}
