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

// TestBuildCreditNoteData_TaxBreakdown proves B2 (ENG-196): a credit note that
// recorded its tax breakdown renders a statutory-grade CDN — taxable value plus
// the GST components that apply — while a legacy note (no breakdown) renders
// gross-only exactly as before.
func TestBuildCreditNoteData_TaxBreakdown(t *testing.T) {
	cn := &domain.CreditNote{
		ID:         uuid.New(),
		Amount:     59000,
		Balance:    59000,
		Subtotal:   50000,
		TaxAmount:  9000,
		CGSTAmount: 4500,
		SGSTAmount: 4500,
		TaxType:    "intra_state",
		HSNCode:    "998314",
		Currency:   "INR",
		Status:     domain.CreditNoteStatusIssued,
		Reason:     "downgrade_proration",
		Type:       domain.CreditNoteTypeAdjustment,
		CreatedAt:  time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
	}
	data := cnTestService().BuildCreditNoteData(cn, nil, "")
	if !data.HasTaxBreakdown {
		t.Fatal("HasTaxBreakdown = false, want true (Subtotal recorded)")
	}
	if data.IGST != "" || data.CGST == "" || data.SGST == "" {
		t.Errorf("intra-state split wrong: IGST=%q CGST=%q SGST=%q", data.IGST, data.CGST, data.SGST)
	}
	html, err := cnTestService().GenerateCreditNoteHTML(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Taxable value", "HSN 998314", "CGST reversed", "SGST reversed"} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered CDN missing %q", want)
		}
	}
	if strings.Contains(html, "IGST reversed") {
		t.Error("intra-state CDN must not render an IGST row")
	}

	// Legacy note: no breakdown recorded -> gross-only document, no tax rows.
	legacy := &domain.CreditNote{
		ID: uuid.New(), Amount: 3000, Balance: 3000, Currency: "USD",
		Status: domain.CreditNoteStatusIssued, Reason: "goodwill",
		Type: domain.CreditNoteTypeAdjustment, CreatedAt: time.Now(),
	}
	ldata := cnTestService().BuildCreditNoteData(legacy, nil, "")
	if ldata.HasTaxBreakdown {
		t.Fatal("legacy note must not claim a tax breakdown")
	}
	lhtml, err := cnTestService().GenerateCreditNoteHTML(ldata)
	if err != nil {
		t.Fatalf("render legacy: %v", err)
	}
	if strings.Contains(lhtml, "Taxable value") {
		t.Error("legacy gross-only document must not render a Taxable value row")
	}
}

// The tenant logo must survive html/template's URL filter — same ZgotmplZ
// regression class as the invoice QR/signature (typed template.URL).
func TestCreditNoteHTML_LogoDataURLSurvives(t *testing.T) {
	cn := &domain.CreditNote{
		ID:        uuid.New(),
		Amount:    5000,
		Currency:  "USD",
		Status:    domain.CreditNoteStatusIssued,
		Type:      domain.CreditNoteTypeAdjustment,
		CreatedAt: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
	}
	name := "Logo Buyer"
	cust := &domain.Customer{Name: &name, Email: "b@example.com"}
	data := cnTestService().BuildCreditNoteData(cn, cust, "")
	data.LogoDataURL = "data:image/png;base64,iVBORw0KGgo="
	data.SellerName = "Branded Co"

	html, err := cnTestService().GenerateCreditNoteHTML(data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "ZgotmplZ") {
		t.Fatal("logo data URL was sanitized away (ZgotmplZ)")
	}
	for _, want := range []string{"data:image/png;base64,iVBORw0KGgo=", "Branded Co"} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered credit note missing %q", want)
		}
	}
}
