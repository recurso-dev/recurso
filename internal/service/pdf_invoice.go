package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

// PDFInvoiceData contains all data needed to generate a GST-compliant PDF invoice
type PDFInvoiceData struct {
	// Invoice details
	InvoiceNumber string
	InvoiceDate   string
	DueDate       string
	Status        string

	// Seller details
	SellerName      string
	SellerAddress   string
	SellerGSTIN     string
	SellerPAN       string
	SellerStateCode string

	// Buyer details
	BuyerName      string
	BuyerAddress   string
	BuyerGSTIN     string
	BuyerStateCode string
	PlaceOfSupply  string

	// Line items
	LineItems []PDFLineItem

	// Amounts (in formatted strings)
	Subtotal      string
	CGSTRate      string
	CGSTAmount    string
	SGSTRate      string
	SGSTAmount    string
	IGSTRate      string
	IGSTAmount    string
	TotalTax      string
	GrandTotal    string
	AmountInWords string

	// GST details
	SACCode       string
	HSNCode       string
	IsInterState  bool
	ReverseCharge string

	// E-Invoice (if applicable)
	IRN        string
	AckNo      string
	AckDate    string
	QRCodeData string

	// Footer
	BankDetails         string
	TermsAndConditions  string
	AuthorizedSignatory string
	SignatureImageURL   string // Base64 or URL to signature image
	SignedAt            string // Date/time when digitally signed
	SignedBy            string // Name of the signatory
}

// PDFLineItem represents a line item in the invoice
type PDFLineItem struct {
	SNo         int
	Description string
	SACCode     string
	Quantity    string
	UnitPrice   string
	Amount      string
	TaxRate     string
	TaxAmount   string
}

// InvoicePDFService handles PDF invoice generation
type InvoicePDFService struct {
	sellerName    string
	sellerAddress string
	sellerGSTIN   string
	sellerPAN     string
	sellerState   string
	bankDetails   string
}

// NewInvoicePDFService creates a new PDF service
func NewInvoicePDFService(sellerName, sellerAddress, sellerGSTIN, sellerPAN, sellerState, bankDetails string) *InvoicePDFService {
	return &InvoicePDFService{
		sellerName:    sellerName,
		sellerAddress: sellerAddress,
		sellerGSTIN:   sellerGSTIN,
		sellerPAN:     sellerPAN,
		sellerState:   sellerState,
		bankDetails:   bankDetails,
	}
}

// GenerateInvoiceHTML generates HTML for a GST-compliant invoice
func (s *InvoicePDFService) GenerateInvoiceHTML(data PDFInvoiceData) (string, error) {
	// Set seller defaults
	if data.SellerName == "" {
		data.SellerName = s.sellerName
	}
	if data.SellerAddress == "" {
		data.SellerAddress = s.sellerAddress
	}
	if data.SellerGSTIN == "" {
		data.SellerGSTIN = s.sellerGSTIN
	}
	if data.BankDetails == "" {
		data.BankDetails = s.bankDetails
	}

	tmpl, err := template.New("invoice").Parse(GSTInvoicePDFTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// FormatAmount formats an amount in paise to rupees string
func FormatAmount(amountPaise int64, currency string) string {
	amount := float64(amountPaise) / 100
	symbol := "₹"
	if currency == "USD" {
		symbol = "$"
	}
	return fmt.Sprintf("%s%.2f", symbol, amount)
}

// AmountToWords converts amount to words (simplified)
func AmountToWords(amount int64, currency string) string {
	rupees := amount / 100
	paise := amount % 100

	if currency == "INR" {
		if paise > 0 {
			return fmt.Sprintf("Rupees %d and Paise %d Only", rupees, paise)
		}
		return fmt.Sprintf("Rupees %d Only", rupees)
	}
	return fmt.Sprintf("%d.%02d %s", rupees, paise, currency)
}

// GenerateInvoiceNumber generates a unique invoice number
func GenerateInvoiceNumber(tenantPrefix string, sequence int) string {
	year := time.Now().Year()
	return fmt.Sprintf("%s/INV/%d/%04d", tenantPrefix, year, sequence)
}

// GenerateIRN generates a dummy IRN (in production, this comes from GST portal)
func GenerateIRN() string {
	return uuid.New().String()
}

// GenerateQRCode generates a Base64 encoded PNG QR code
func GenerateQRCode(content string) (string, error) {
	var png []byte
	png, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(png)
	return fmt.Sprintf("data:image/png;base64,%s", encoded), nil
}

// GSTInvoicePDFTemplate is the HTML template for GST-compliant invoices
const GSTInvoicePDFTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Tax Invoice - {{.InvoiceNumber}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: Arial, sans-serif; font-size: 12px; color: #333; }
        .invoice { max-width: 800px; margin: 0 auto; padding: 20px; }
        .header { border: 1px solid #000; margin-bottom: 0; }
        .header-row { display: flex; border-bottom: 1px solid #000; }
        .header-row:last-child { border-bottom: none; }
        .header-cell { padding: 10px; }
        .header-left { width: 50%; border-right: 1px solid #000; }
        .header-right { width: 50%; }
        .company-name { font-size: 18px; font-weight: bold; color: #000; margin-bottom: 5px; }
        .label { color: #666; font-size: 10px; text-transform: uppercase; margin-bottom: 2px; }
        .value { font-weight: bold; }
        .title { text-align: center; font-size: 14px; font-weight: bold; background: #f0f0f0; padding: 8px; border: 1px solid #000; border-top: none; }
        
        .parties { display: flex; border: 1px solid #000; border-top: none; }
        .party { width: 50%; padding: 10px; }
        .party:first-child { border-right: 1px solid #000; }
        .party-title { font-weight: bold; margin-bottom: 5px; text-transform: uppercase; font-size: 10px; color: #666; }
        
        .items { border: 1px solid #000; border-top: none; }
        .items table { width: 100%; border-collapse: collapse; }
        .items th, .items td { padding: 8px; text-align: left; border: 1px solid #000; }
        .items th { background: #f0f0f0; font-size: 10px; text-transform: uppercase; }
        .items td { font-size: 11px; }
        .text-right { text-align: right; }
        .text-center { text-align: center; }
        
        .totals { border: 1px solid #000; border-top: none; }
        .totals table { width: 100%; border-collapse: collapse; }
        .totals td { padding: 8px; border: 1px solid #000; }
        .total-label { text-align: right; width: 70%; }
        .total-value { text-align: right; width: 30%; font-weight: bold; }
        .grand-total { font-size: 14px; background: #f0f0f0; }
        
        .amount-words { border: 1px solid #000; border-top: none; padding: 10px; }
        .amount-words-label { font-size: 10px; color: #666; }
        .amount-words-value { font-weight: bold; font-style: italic; }
        
        .footer { display: flex; border: 1px solid #000; border-top: none; }
        .footer-left { width: 60%; padding: 10px; border-right: 1px solid #000; }
        .footer-right { width: 40%; padding: 10px; text-align: center; }
        .bank-title { font-weight: bold; margin-bottom: 5px; font-size: 10px; }
        .signature-box { min-height: 60px; display: flex; flex-direction: column; justify-content: flex-end; }
        .signature-image { max-width: 150px; max-height: 50px; margin: 0 auto 5px; }
        .signature-line { border-top: 1px solid #000; padding-top: 5px; }
        .signed-info { font-size: 9px; color: #666; margin-top: 3px; }
        
        .e-invoice { border: 1px solid #000; border-top: none; padding: 10px; display: flex; }
        .qr-code { width: 100px; height: 100px; border: 1px solid #000; margin-right: 20px; display: flex; align-items: center; justify-content: center; font-size: 10px; color: #666; }
        .irn-details { flex: 1; }
        
        .terms { border: 1px solid #000; border-top: none; padding: 10px; font-size: 10px; color: #666; }
        .terms-title { font-weight: bold; margin-bottom: 5px; }
        
        @media print {
            body { print-color-adjust: exact; -webkit-print-color-adjust: exact; }
            .invoice { padding: 0; }
        }
    </style>
</head>
<body>
    <div class="invoice">
        <!-- Header -->
        <div class="header">
            <div class="header-row">
                <div class="header-cell header-left">
                    <div class="company-name">{{.SellerName}}</div>
                    <div>{{.SellerAddress}}</div>
                </div>
                <div class="header-cell header-right">
                    <div class="label">Invoice Number</div>
                    <div class="value">{{.InvoiceNumber}}</div>
                    <div class="label" style="margin-top: 10px;">Invoice Date</div>
                    <div class="value">{{.InvoiceDate}}</div>
                </div>
            </div>
            <div class="header-row">
                <div class="header-cell header-left">
                    <div class="label">GSTIN</div>
                    <div class="value">{{.SellerGSTIN}}</div>
                </div>
                <div class="header-cell header-right">
                    <div class="label">Due Date</div>
                    <div class="value">{{.DueDate}}</div>
                </div>
            </div>
        </div>
        
        <div class="title">TAX INVOICE</div>
        
        <!-- Parties -->
        <div class="parties">
            <div class="party">
                <div class="party-title">Bill To</div>
                <div class="value">{{.BuyerName}}</div>
                <div>{{.BuyerAddress}}</div>
                {{if .BuyerGSTIN}}
                <div style="margin-top: 5px;"><span class="label">GSTIN:</span> {{.BuyerGSTIN}}</div>
                {{end}}
            </div>
            <div class="party">
                <div class="party-title">Place of Supply</div>
                <div class="value">{{.PlaceOfSupply}}</div>
                <div style="margin-top: 5px;"><span class="label">State Code:</span> {{.BuyerStateCode}}</div>
                <div><span class="label">Reverse Charge:</span> {{.ReverseCharge}}</div>
            </div>
        </div>
        
        <!-- Line Items -->
        <div class="items">
            <table>
                <thead>
                    <tr>
                        <th style="width: 5%;">S.No</th>
                        <th style="width: 35%;">Description</th>
                        <th style="width: 10%;">SAC</th>
                        <th style="width: 10%;" class="text-center">Qty</th>
                        <th style="width: 15%;" class="text-right">Unit Price</th>
                        <th style="width: 10%;" class="text-right">Tax %</th>
                        <th style="width: 15%;" class="text-right">Amount</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .LineItems}}
                    <tr>
                        <td class="text-center">{{.SNo}}</td>
                        <td>{{.Description}}</td>
                        <td>{{.SACCode}}</td>
                        <td class="text-center">{{.Quantity}}</td>
                        <td class="text-right">{{.UnitPrice}}</td>
                        <td class="text-right">{{.TaxRate}}</td>
                        <td class="text-right">{{.Amount}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        
        <!-- Totals -->
        <div class="totals">
            <table>
                <tr>
                    <td class="total-label">Subtotal</td>
                    <td class="total-value">{{.Subtotal}}</td>
                </tr>
                {{if not .IsInterState}}
                <tr>
                    <td class="total-label">CGST @ {{.CGSTRate}}</td>
                    <td class="total-value">{{.CGSTAmount}}</td>
                </tr>
                <tr>
                    <td class="total-label">SGST @ {{.SGSTRate}}</td>
                    <td class="total-value">{{.SGSTAmount}}</td>
                </tr>
                {{else}}
                <tr>
                    <td class="total-label">IGST @ {{.IGSTRate}}</td>
                    <td class="total-value">{{.IGSTAmount}}</td>
                </tr>
                {{end}}
                <tr class="grand-total">
                    <td class="total-label">Grand Total</td>
                    <td class="total-value">{{.GrandTotal}}</td>
                </tr>
            </table>
        </div>
        
        <!-- Amount in Words -->
        <div class="amount-words">
            <span class="amount-words-label">Amount in Words: </span>
            <span class="amount-words-value">{{.AmountInWords}}</span>
        </div>
        
        {{if .IRN}}
        <!-- E-Invoice Details -->
        <div class="e-invoice">
            <div class="qr-code">
				{{if .QRCodeData}}
				<img src="{{.QRCodeData}}" style="width: 100%; height: 100%;" />
				{{else}}
				[QR Code]
				{{end}}
			</div>
            <div class="irn-details">
                <div><span class="label">IRN:</span></div>
                <div class="value" style="word-break: break-all; font-size: 10px;">{{.IRN}}</div>
                <div style="margin-top: 5px;"><span class="label">Ack No:</span> {{.AckNo}}</div>
                <div><span class="label">Ack Date:</span> {{.AckDate}}</div>
            </div>
        </div>
        {{end}}
        
        <!-- Footer -->
        <div class="footer">
            <div class="footer-left">
                <div class="bank-title">Bank Details</div>
                <div style="white-space: pre-line;">{{.BankDetails}}</div>
            </div>
            <div class="footer-right">
                <div class="label">For {{.SellerName}}</div>
                <div class="signature-box">
                    {{if .SignatureImageURL}}
                    <img src="{{.SignatureImageURL}}" class="signature-image" alt="Digital Signature" />
                    {{end}}
                    <div class="signature-line">{{if .SignedBy}}{{.SignedBy}}{{else}}Authorized Signatory{{end}}</div>
                    {{if .SignedAt}}
                    <div class="signed-info">Digitally signed on {{.SignedAt}}</div>
                    {{end}}
                </div>
            </div>
        </div>
        
        <!-- Terms -->
        <div class="terms">
            <div class="terms-title">Terms & Conditions</div>
            <div>{{.TermsAndConditions}}</div>
        </div>
    </div>
</body>
</html>`
