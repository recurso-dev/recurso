package service

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func cnTestService() *InvoicePDFService {
	return NewInvoicePDFService("Acme Inc", "1 Market St, SF", "", "", "", "", "US", "")
}

func TestBuildCreditNoteData_RefundWithInvoice(t *testing.T) {
	ref := "CN-DEMO-0005"
	cn := &domain.CreditNote{
		ID:           uuid.New(),
		Reference:    &ref,
		Amount:       7582,
		Balance:      0,
		Currency:     "USD",
		Status:       domain.CreditNoteStatusIssued,
		Reason:       "downgrade_proration",
		Type:         domain.CreditNoteTypeRefund,
		RefundStatus: domain.RefundStatusProcessed,
		CreatedAt:    time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC),
	}
	name := "Rekall Group"
	cust := &domain.Customer{Name: &name, Email: "ap@rekall.example"}

	data := cnTestService().BuildCreditNoteData(cn, cust, "INV-2026-0007")

	if data.CreditNoteNumber != "CN-DEMO-0005" {
		t.Errorf("number = %q, want CN-DEMO-0005", data.CreditNoteNumber)
	}
	if !data.IsRefund || data.TypeLabel != "Refund" {
		t.Errorf("expected refund type, got IsRefund=%v label=%q", data.IsRefund, data.TypeLabel)
	}
	if data.Reason != "Downgrade Proration" {
		t.Errorf("reason = %q, want humanized 'Downgrade Proration'", data.Reason)
	}
	if data.OriginalInvoice != "INV-2026-0007" {
		t.Errorf("original invoice = %q", data.OriginalInvoice)
	}
	if data.BuyerName != "Rekall Group" {
		t.Errorf("buyer = %q", data.BuyerName)
	}

	html, err := cnTestService().GenerateCreditNoteHTML(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"CREDIT NOTE", "CN-DEMO-0005", "Rekall Group", "Acme Inc", "Downgrade Proration", "INV-2026-0007"} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered document missing %q", want)
		}
	}
}

func TestBuildCreditNoteData_AdjustmentNoInvoice(t *testing.T) {
	cn := &domain.CreditNote{
		ID:        uuid.New(),
		Amount:    5000,
		Balance:   5000,
		Currency:  "USD",
		Status:    domain.CreditNoteStatusIssued,
		Type:      domain.CreditNoteTypeAdjustment,
		CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	data := cnTestService().BuildCreditNoteData(cn, nil, "")

	if data.IsRefund || data.TypeLabel != "Account credit" {
		t.Errorf("expected account credit, got IsRefund=%v label=%q", data.IsRefund, data.TypeLabel)
	}
	// No reference → falls back to the UUID.
	if data.CreditNoteNumber != cn.ID.String() {
		t.Errorf("number = %q, want the UUID", data.CreditNoteNumber)
	}
	if _, err := cnTestService().GenerateCreditNoteHTML(data); err != nil {
		t.Fatalf("render: %v", err)
	}
}
