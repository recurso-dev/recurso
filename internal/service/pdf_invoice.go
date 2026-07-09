package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"github.com/swapnull-in/recur-so/internal/core/domain"
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

	// Jurisdiction-aware rendering (populated by BuildInvoiceData)
	DocTitle       string // "TAX INVOICE" (India GST) or "INVOICE"
	ShowGST        bool   // render GST columns/rows (HSN, CGST/SGST/IGST, Place of Supply, Reverse Charge)
	SellerTaxLabel string // "GSTIN" (India) / "EIN" (US) / "Tax ID"
	SellerTaxID    string // the seller tax id value for the label above
	TaxLineLabel   string // non-GST single tax row, e.g. "Sales Tax" / "VAT"
	TaxLineRate    string
	TaxLineAmount  string
	BuyerCountry   string

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

// PDFLineItem represents a line item in the invoice. SACCode carries the line's
// HSN/SAC code. The per-line GST amounts (CGST/SGST/IGST) and taxable base are
// formatted currency strings, populated from the invoice's real line items.
type PDFLineItem struct {
	SNo           int
	Description   string
	SACCode       string
	Quantity      string
	UnitPrice     string
	Amount        string
	TaxableAmount string
	TaxRate       string
	TaxAmount     string
	CGSTAmount    string
	SGSTAmount    string
	IGSTAmount    string
}

// BuildPDFLineItems maps an invoice's persisted line items to PDF line items,
// each carrying its own HSN/SAC code, rate, and per-line CGST/SGST/IGST. When
// the invoice has no line items (legacy, pre-itemization) it falls back to a
// single synthetic line derived from the invoice totals — mirroring the
// e-invoice ItemList fallback so old invoices still render.
func BuildPDFLineItems(inv *domain.Invoice) []PDFLineItem {
	if inv == nil {
		return nil
	}
	cur := inv.Currency

	if len(inv.LineItems) > 0 {
		items := make([]PDFLineItem, 0, len(inv.LineItems))
		for i, li := range inv.LineItems {
			hsn := li.HSNCode
			if hsn == "" {
				hsn = domain.DefaultSACCode
			}
			qty := li.Quantity
			if qty <= 0 {
				qty = 1
			}
			items = append(items, PDFLineItem{
				SNo:           i + 1,
				Description:   li.Description,
				SACCode:       hsn,
				Quantity:      strconv.Itoa(qty),
				UnitPrice:     FormatAmount(li.UnitAmount, cur),
				Amount:        FormatAmount(li.Amount, cur),
				TaxableAmount: FormatAmount(li.TaxableAmount, cur),
				TaxRate:       formatTaxRate(li.TaxRate),
				TaxAmount:     FormatAmount(li.CGSTAmount+li.SGSTAmount+li.IGSTAmount, cur),
				CGSTAmount:    FormatAmount(li.CGSTAmount, cur),
				SGSTAmount:    FormatAmount(li.SGSTAmount, cur),
				IGSTAmount:    FormatAmount(li.IGSTAmount, cur),
			})
		}
		return items
	}

	// Legacy fallback: a single synthetic line from the invoice totals.
	hsn := inv.HSNCode
	if hsn == "" {
		hsn = domain.DefaultSACCode
	}
	return []PDFLineItem{
		{
			SNo:           1,
			Description:   "SaaS Subscription",
			SACCode:       hsn,
			Quantity:      "1",
			UnitPrice:     FormatAmount(inv.Subtotal, cur),
			Amount:        FormatAmount(inv.Subtotal, cur),
			TaxableAmount: FormatAmount(inv.Subtotal, cur),
			TaxRate:       formatTaxRate(domain.DefaultGSTRate),
			TaxAmount:     FormatAmount(inv.CGSTAmount+inv.SGSTAmount+inv.IGSTAmount, cur),
			CGSTAmount:    FormatAmount(inv.CGSTAmount, cur),
			SGSTAmount:    FormatAmount(inv.SGSTAmount, cur),
			IGSTAmount:    FormatAmount(inv.IGSTAmount, cur),
		},
	}
}

// formatTaxRate renders a GST rate percent (e.g. 18.0 -> "18%", 12.5 -> "12.5%").
func formatTaxRate(pct float64) string {
	return strconv.FormatFloat(pct, 'f', -1, 64) + "%"
}

// InvoicePDFService handles PDF invoice generation
type InvoicePDFService struct {
	sellerName    string
	sellerAddress string
	sellerGSTIN   string
	sellerPAN     string
	sellerState   string
	bankDetails   string
	sellerCountry string // ISO-2, e.g. "IN" / "US" — selects the invoice regime
	sellerTaxID   string // non-GST seller tax id (US EIN, etc.)
}

// NewInvoicePDFService creates a new PDF service. sellerCountry selects the
// invoice regime (India GST vs a plain sales-tax/VAT invoice); sellerTaxID is
// the seller's non-GST tax id (e.g. a US EIN) shown on non-GST invoices.
func NewInvoicePDFService(sellerName, sellerAddress, sellerGSTIN, sellerPAN, sellerState, bankDetails, sellerCountry, sellerTaxID string) *InvoicePDFService {
	return &InvoicePDFService{
		sellerName:    sellerName,
		sellerAddress: sellerAddress,
		sellerGSTIN:   sellerGSTIN,
		sellerPAN:     sellerPAN,
		sellerState:   sellerState,
		bankDetails:   bankDetails,
		sellerCountry: sellerCountry,
		sellerTaxID:   sellerTaxID,
	}
}

// BuildInvoiceData maps a real invoice and its customer to renderable PDF data,
// choosing the jurisdiction: an India-GST tax invoice (GSTIN, HSN, CGST/SGST or
// IGST, Place of Supply) when the seller is in India, or a plain invoice
// (single tax line, seller tax id / EIN, no HSN) otherwise — so a US invoice
// never shows GST fields.
func (s *InvoicePDFService) BuildInvoiceData(inv *domain.Invoice, cust *domain.Customer) PDFInvoiceData {
	cur := inv.Currency
	sellerIN := s.sellerCountry == "" || strings.EqualFold(s.sellerCountry, "IN")
	hasGSTSplit := inv.CGSTAmount > 0 || inv.SGSTAmount > 0 || inv.IGSTAmount > 0
	showGST := sellerIN && (hasGSTSplit || strings.EqualFold(cur, "INR"))

	// Buyer, from the customer's billing address.
	var buyerName, buyerAddr, buyerState, buyerGSTIN, buyerCountry string
	if cust != nil {
		if cust.Name != nil {
			buyerName = *cust.Name
		}
		ba := cust.BillingAddress
		var lines []string
		if ba.Line1 != "" {
			lines = append(lines, ba.Line1)
		}
		cityLine := strings.TrimSpace(strings.Trim(strings.Join([]string{ba.City, ba.State, ba.Zip}, " "), " "))
		if cityLine != "" {
			lines = append(lines, cityLine)
		}
		if ba.Country != "" {
			lines = append(lines, ba.Country)
		}
		buyerAddr = strings.Join(lines, "\n")
		buyerState = ba.State
		buyerCountry = ba.Country
		if cust.GSTIN != nil {
			buyerGSTIN = *cust.GSTIN
		}
	}

	data := PDFInvoiceData{
		InvoiceNumber:  inv.InvoiceNumber,
		InvoiceDate:    inv.CreatedAt.Format("January 2, 2006"),
		DueDate:        inv.DueDate.Format("January 2, 2006"),
		Status:         string(inv.Status),
		SellerName:     s.sellerName,
		SellerAddress:  s.sellerAddress,
		BuyerName:      buyerName,
		BuyerAddress:   buyerAddr,
		BuyerGSTIN:     buyerGSTIN,
		BuyerStateCode: buyerState,
		BuyerCountry:   buyerCountry,
		Subtotal:       FormatAmount(inv.Subtotal, cur),
		GrandTotal:     FormatAmount(inv.Total, cur),
		AmountInWords:  AmountToWords(inv.Total, cur),
		LineItems:      BuildPDFLineItems(inv),
		BankDetails:    s.bankDetails,
		ShowGST:        showGST,
	}

	if showGST {
		data.DocTitle = "TAX INVOICE"
		data.SellerTaxLabel = "GSTIN"
		data.SellerTaxID = s.sellerGSTIN
		data.IsInterState = inv.IGSTAmount > 0
		data.PlaceOfSupply = firstNonEmpty(derefPlaceOfSupply(cust), buyerState)
		data.ReverseCharge = "No"
		data.CGSTAmount = FormatAmount(inv.CGSTAmount, cur)
		data.SGSTAmount = FormatAmount(inv.SGSTAmount, cur)
		data.IGSTAmount = FormatAmount(inv.IGSTAmount, cur)
		data.CGSTRate = ratePercent(inv.CGSTAmount, inv.Subtotal)
		data.SGSTRate = ratePercent(inv.SGSTAmount, inv.Subtotal)
		data.IGSTRate = ratePercent(inv.IGSTAmount, inv.Subtotal)
		data.IRN = inv.IRN
	} else {
		data.DocTitle = "INVOICE"
		if s.sellerTaxID != "" {
			data.SellerTaxLabel = sellerTaxLabel(s.sellerCountry)
			data.SellerTaxID = s.sellerTaxID
		}
		if inv.TaxAmount > 0 {
			data.TaxLineLabel = taxLineLabel(s.sellerCountry)
			data.TaxLineAmount = FormatAmount(inv.TaxAmount, cur)
			data.TaxLineRate = ratePercent(inv.TaxAmount, inv.Subtotal)
		}
	}
	return data
}

func derefPlaceOfSupply(cust *domain.Customer) string {
	if cust != nil && cust.PlaceOfSupply != nil {
		return *cust.PlaceOfSupply
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ratePercent derives a display rate from a tax amount over its taxable base,
// e.g. 900 over 10000 -> "9%". Returns "" when the base is zero.
func ratePercent(taxMinor, baseMinor int64) string {
	if baseMinor <= 0 || taxMinor <= 0 {
		return ""
	}
	pct := float64(taxMinor) / float64(baseMinor) * 100
	return strconv.FormatFloat(pct, 'f', -1, 64) + "%"
}

func sellerTaxLabel(country string) string {
	if strings.EqualFold(country, "US") {
		return "EIN"
	}
	return "Tax ID"
}

func taxLineLabel(country string) string {
	switch {
	case strings.EqualFold(country, "US"):
		return "Sales Tax"
	case isEUCountry(country):
		return "VAT"
	default:
		return "Tax"
	}
}

func isEUCountry(country string) bool {
	switch strings.ToUpper(country) {
	case "AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR", "DE", "GR",
		"HU", "IE", "IT", "LV", "LT", "LU", "MT", "NL", "PL", "PT", "RO", "SK",
		"SI", "ES", "SE", "GB":
		return true
	default:
		return false
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
    <title>{{.DocTitle}} - {{.InvoiceNumber}}</title>
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
                    {{if .SellerTaxID}}
                    <div class="label">{{.SellerTaxLabel}}</div>
                    <div class="value">{{.SellerTaxID}}</div>
                    {{end}}
                </div>
                <div class="header-cell header-right">
                    <div class="label">Due Date</div>
                    <div class="value">{{.DueDate}}</div>
                </div>
            </div>
        </div>
        
        <div class="title">{{.DocTitle}}</div>
        
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
                {{if .ShowGST}}
                <div class="party-title">Place of Supply</div>
                <div class="value">{{.PlaceOfSupply}}</div>
                <div style="margin-top: 5px;"><span class="label">State Code:</span> {{.BuyerStateCode}}</div>
                <div><span class="label">Reverse Charge:</span> {{.ReverseCharge}}</div>
                {{else}}
                <div class="party-title">Details</div>
                {{if .BuyerStateCode}}<div><span class="label">State:</span> {{.BuyerStateCode}}</div>{{end}}
                {{if .BuyerCountry}}<div><span class="label">Country:</span> {{.BuyerCountry}}</div>{{end}}
                {{end}}
            </div>
        </div>
        
        <!-- Line Items -->
        <div class="items">
            <table>
                <thead>
                    <tr>
                        <th style="width: 4%;">S.No</th>
                        <th style="width: 26%;">Description</th>
                        {{if .ShowGST}}<th style="width: 9%;">HSN/SAC</th>{{end}}
                        <th style="width: 6%;" class="text-center">Qty</th>
                        <th style="width: 13%;" class="text-right">Unit Price</th>
                        <th style="width: 13%;" class="text-right">Taxable</th>
                        <th style="width: 8%;" class="text-right">Tax %</th>
                        <th style="width: 11%;" class="text-right">Tax Amt</th>
                        <th style="width: 10%;" class="text-right">Amount</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .LineItems}}
                    <tr>
                        <td class="text-center">{{.SNo}}</td>
                        <td>{{.Description}}</td>
                        {{if $.ShowGST}}<td>{{.SACCode}}</td>{{end}}
                        <td class="text-center">{{.Quantity}}</td>
                        <td class="text-right">{{.UnitPrice}}</td>
                        <td class="text-right">{{.TaxableAmount}}</td>
                        <td class="text-right">{{.TaxRate}}</td>
                        <td class="text-right">{{.TaxAmount}}</td>
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
                {{if .ShowGST}}
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
                {{else if .TaxLineAmount}}
                <tr>
                    <td class="total-label">{{.TaxLineLabel}}{{if .TaxLineRate}} @ {{.TaxLineRate}}{{end}}</td>
                    <td class="total-value">{{.TaxLineAmount}}</td>
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
