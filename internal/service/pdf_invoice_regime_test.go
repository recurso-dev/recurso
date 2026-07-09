package service

import (
	"strings"
	"testing"
	"time"

	"github.com/swapnull-in/recur-so/internal/core/domain"
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
