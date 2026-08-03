package service

import (
	"strings"
	"testing"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

func pdfStr(s string) *string { return &s }

// A US seller's invoice must render as a plain sales-tax invoice — no GST
// fields anywhere.
func TestBuildInvoiceData_USRegime(t *testing.T) {
	svc := NewInvoicePDFService("Acme Inc", "1 Market St, San Francisco, CA", "", "", "", "Bank: ...", "US", "12-3456789")
	inv := &domain.Invoice{
		InvoiceNumber: "INV-US-1",
		Currency:      "USD",
		Subtotal:      100000, // $1000.00
		TaxAmount:     8750,   // $87.50 sales tax
		Total:         108750,
		CreatedAt:     time.Now(),
		DueDate:       time.Now().Add(720 * time.Hour),
	}
	cust := &domain.Customer{
		Name:           pdfStr("Jane Buyer"),
		BillingAddress: domain.BillingAddress{Line1: "5 King St", City: "Austin", State: "TX", Zip: "78701", Country: "US"},
	}

	data := svc.BuildInvoiceData(inv, cust)
	if data.ShowGST {
		t.Fatal("US invoice must not show GST")
	}
	if data.DocTitle != "INVOICE" {
		t.Errorf("DocTitle = %q, want INVOICE", data.DocTitle)
	}
	if data.SellerTaxLabel != "EIN" || data.SellerTaxID != "12-3456789" {
		t.Errorf("seller tax = %q/%q, want EIN/12-3456789", data.SellerTaxLabel, data.SellerTaxID)
	}
	if data.TaxLineLabel != "Sales Tax" || data.TaxLineAmount == "" {
		t.Errorf("tax line = %q/%q, want Sales Tax/non-empty", data.TaxLineLabel, data.TaxLineAmount)
	}

	html, err := svc.GenerateInvoiceHTML(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, bad := range []string{"GSTIN", "HSN/SAC", "CGST", "SGST", "IGST", "Place of Supply"} {
		if strings.Contains(html, bad) {
			t.Errorf("US invoice HTML must not contain %q", bad)
		}
	}
	if !strings.Contains(html, "Sales Tax") || !strings.Contains(html, "EIN") {
		t.Error("US invoice HTML should show Sales Tax and EIN")
	}
}

// An India seller's INR invoice keeps the full GST tax invoice.
func TestBuildInvoiceData_IndiaGSTRegime(t *testing.T) {
	svc := NewInvoicePDFService("Bharat Co", "MG Road, Bengaluru", "29ABCDE1234F1Z5", "ABCDE1234F", "KA", "Bank: ...", "IN", "")
	inv := &domain.Invoice{
		InvoiceNumber: "INV-IN-1",
		Currency:      "INR",
		Subtotal:      100000,
		TaxAmount:     18000,
		CGSTAmount:    9000,
		SGSTAmount:    9000,
		Total:         118000,
		HSNCode:       "998314",
		CreatedAt:     time.Now(),
		DueDate:       time.Now().Add(360 * time.Hour),
	}
	cust := &domain.Customer{
		Name:           pdfStr("Ravi Buyer"),
		BillingAddress: domain.BillingAddress{Line1: "1 MG Road", City: "Bengaluru", State: "KA", Zip: "560001", Country: "IN"},
	}

	data := svc.BuildInvoiceData(inv, cust)
	if !data.ShowGST {
		t.Fatal("INR/IN invoice must show GST")
	}
	if data.DocTitle != "TAX INVOICE" || data.SellerTaxLabel != "GSTIN" {
		t.Errorf("title/label = %q/%q, want TAX INVOICE/GSTIN", data.DocTitle, data.SellerTaxLabel)
	}

	html, err := svc.GenerateInvoiceHTML(data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"TAX INVOICE", "GSTIN", "HSN/SAC", "CGST", "SGST"} {
		if !strings.Contains(html, want) {
			t.Errorf("GST invoice HTML should contain %q", want)
		}
	}
}

// Data-URL images (the e-invoice QR, tenant logo/signature) must survive
// html/template's URL filter. Before the fields were typed template.URL,
// they rendered as the "#ZgotmplZ" sanitizer marker — a broken image on a
// statutory GST invoice. This is the regression oracle.
func TestInvoiceHTML_DataURLImagesSurvive(t *testing.T) {
	svc := NewInvoicePDFService("Seller", "Addr", "", "", "", "", "US", "")
	inv := &domain.Invoice{
		InvoiceNumber: "INV-IMG-1",
		Currency:      "USD",
		Subtotal:      100000,
		Total:         100000,
		CreatedAt:     time.Now(),
		DueDate:       time.Now().Add(720 * time.Hour),
	}
	cust := &domain.Customer{
		Name:           pdfStr("Img Buyer"),
		BillingAddress: domain.BillingAddress{Line1: "5 King St", City: "Austin", State: "TX", Zip: "78701", Country: "US"},
	}
	data := svc.BuildInvoiceData(inv, cust)
	data.QRCodeData = "data:image/png;base64,iVBORw0KGgo="
	data.LogoDataURL = "data:image/png;base64,iVBORw0KGgo="
	data.SignatureImageURL = "data:image/jpeg;base64,/9j/4AAQ"
	html, err := svc.GenerateInvoiceHTML(data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "ZgotmplZ") {
		t.Fatal("data-URL image was sanitized away (ZgotmplZ) — image fields must be template.URL")
	}
	for _, want := range []string{"data:image/png;base64,iVBORw0KGgo=", "data:image/jpeg;base64,/9j/4AAQ"} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered HTML is missing image %q", want)
		}
	}
}

// The printable Compare receipt must render both verdicts and carry the
// coverage numbers + issues verbatim.
func TestCompareReportHTML_Renders(t *testing.T) {
	data := CompareReportDocData{
		TenantName:  "Acme Labs",
		Source:      "stripe",
		Ready:       false,
		GeneratedAt: time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
		Report: CompareReport{
			Customers:     CompareCount{Source: 10, Matched: 9, Missing: 1},
			Plans:         CompareCount{Source: 3, Matched: 3},
			Subscriptions: CompareCount{Source: 8, Matched: 8},
			Issues: []CompareIssue{
				{Kind: "customer", ExternalID: "cus_x", Field: "missing", Source: "x@acme.com"},
			},
		},
	}
	html, err := RenderCompareReportHTML(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"NOT READY", "Acme Labs", "cus_x", "x@acme.com", ">10<", ">9<", ">1<"} {
		if !strings.Contains(html, want) {
			t.Errorf("document missing %q", want)
		}
	}
	data.Ready = true
	data.Report.Issues = nil
	html, err = RenderCompareReportHTML(data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "READY") || strings.Contains(html, "NOT READY") {
		t.Error("ready verdict not rendered")
	}
}
