package service

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// PDFCreditNoteData is the view model for a printable credit note document.
// Seller identity is reused from the InvoicePDFService so a credit note carries
// the same letterhead as the invoices it offsets.
type PDFCreditNoteData struct {
	SellerName    string
	SellerAddress string

	CreditNoteNumber string
	IssueDate        string

	BuyerName    string
	BuyerAddress string
	BuyerEmail   string

	Amount   string
	Balance  string
	Currency string

	Reason          string
	TypeLabel       string
	IsRefund        bool
	RefundStatus    string
	OriginalInvoice string
	StatusLabel     string
}

// BuildCreditNoteData assembles the document view model. originalInvoiceNumber
// is the human number of the invoice this note offsets ("" when the note isn't
// tied to a specific invoice).
func (s *InvoicePDFService) BuildCreditNoteData(cn *domain.CreditNote, cust *domain.Customer, originalInvoiceNumber string) PDFCreditNoteData {
	number := cn.ID.String()
	if cn.Reference != nil && strings.TrimSpace(*cn.Reference) != "" {
		number = *cn.Reference
	}

	data := PDFCreditNoteData{
		SellerName:       s.sellerName,
		SellerAddress:    s.sellerAddress,
		CreditNoteNumber: number,
		IssueDate:        cn.CreatedAt.Format("January 2, 2006"),
		Amount:           FormatAmount(cn.Amount, cn.Currency),
		Balance:          FormatAmount(cn.Balance, cn.Currency),
		Currency:         cn.Currency,
		Reason:           titleFromSnake(cn.Reason),
		IsRefund:         cn.Type == domain.CreditNoteTypeRefund,
		OriginalInvoice:  originalInvoiceNumber,
		StatusLabel:      titleFromSnake(string(cn.Status)),
	}
	if data.IsRefund {
		data.TypeLabel = "Refund"
		data.RefundStatus = titleFromSnake(string(cn.RefundStatus))
	} else {
		data.TypeLabel = "Account credit"
	}

	if cust != nil {
		if cust.Name != nil {
			data.BuyerName = *cust.Name
		}
		data.BuyerEmail = cust.Email
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
		data.BuyerAddress = strings.Join(lines, "\n")
	}

	return data
}

// GenerateCreditNoteHTML renders the credit note as printable HTML (matching
// how invoices are served — the browser's print-to-PDF produces the document).
func (s *InvoicePDFService) GenerateCreditNoteHTML(data PDFCreditNoteData) (string, error) {
	if data.SellerName == "" {
		data.SellerName = s.sellerName
	}
	if data.SellerAddress == "" {
		data.SellerAddress = s.sellerAddress
	}
	tmpl, err := template.New("creditnote").Parse(CreditNotePDFTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// titleFromSnake turns "downgrade_proration" into "Downgrade Proration".
func titleFromSnake(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

const CreditNotePDFTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Credit Note {{.CreditNoteNumber}}</title>
<style>
  * { box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; color: #1c1917; margin: 0; padding: 40px; background: #fff; }
  .doc { max-width: 720px; margin: 0 auto; }
  .head { display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #10b981; padding-bottom: 20px; margin-bottom: 28px; }
  .seller-name { font-size: 18px; font-weight: 700; }
  .seller-addr { font-size: 12px; color: #57534e; white-space: pre-line; margin-top: 4px; }
  .title { text-align: right; }
  .title h1 { font-size: 22px; margin: 0; letter-spacing: 0.5px; }
  .title .num { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 13px; color: #57534e; margin-top: 4px; }
  .meta { display: flex; gap: 40px; margin-bottom: 28px; }
  .meta .col { flex: 1; }
  .label { font-size: 10px; text-transform: uppercase; letter-spacing: 0.6px; color: #78716c; font-weight: 600; margin-bottom: 4px; }
  .val { font-size: 13px; }
  .addr { white-space: pre-line; font-size: 13px; line-height: 1.5; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 24px; }
  th { text-align: left; font-size: 10px; text-transform: uppercase; letter-spacing: 0.6px; color: #78716c; border-bottom: 1px solid #e7e5e4; padding: 8px 0; }
  td { padding: 12px 0; font-size: 13px; border-bottom: 1px solid #f5f5f4; }
  td.amt, th.amt { text-align: right; font-variant-numeric: tabular-nums; }
  .totals { margin-left: auto; width: 260px; }
  .totals .row { display: flex; justify-content: space-between; padding: 8px 0; font-size: 13px; }
  .totals .grand { border-top: 2px solid #1c1917; margin-top: 4px; padding-top: 12px; font-size: 16px; font-weight: 700; }
  .note { margin-top: 32px; font-size: 12px; color: #57534e; line-height: 1.6; border-top: 1px solid #e7e5e4; padding-top: 16px; }
  .pill { display: inline-block; font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 999px; background: #ecfdf5; color: #047857; }
</style>
</head>
<body>
<div class="doc">
  <div class="head">
    <div>
      <div class="seller-name">{{.SellerName}}</div>
      <div class="seller-addr">{{.SellerAddress}}</div>
    </div>
    <div class="title">
      <h1>CREDIT NOTE</h1>
      <div class="num">{{.CreditNoteNumber}}</div>
    </div>
  </div>

  <div class="meta">
    <div class="col">
      <div class="label">Credited to</div>
      <div class="addr">{{if .BuyerName}}{{.BuyerName}}
{{end}}{{.BuyerAddress}}{{if .BuyerEmail}}
{{.BuyerEmail}}{{end}}</div>
    </div>
    <div class="col">
      <div class="label">Issued</div>
      <div class="val">{{.IssueDate}}</div>
      <div class="label" style="margin-top:12px;">Type</div>
      <div class="val"><span class="pill">{{.TypeLabel}}</span></div>
      {{if .OriginalInvoice}}<div class="label" style="margin-top:12px;">Against invoice</div>
      <div class="val">{{.OriginalInvoice}}</div>{{end}}
    </div>
  </div>

  <table>
    <thead>
      <tr><th>Description</th><th class="amt">Amount</th></tr>
    </thead>
    <tbody>
      <tr>
        <td>{{if .Reason}}{{.Reason}}{{else}}Credit adjustment{{end}}</td>
        <td class="amt">{{.Amount}}</td>
      </tr>
    </tbody>
  </table>

  <div class="totals">
    <div class="row"><span>Credit amount</span><span>{{.Amount}}</span></div>
    <div class="row grand"><span>Balance remaining</span><span>{{.Balance}}</span></div>
  </div>

  <div class="note">
    {{if .IsRefund}}This credit note records a refund of {{.Amount}}{{if .OriginalInvoice}} against invoice {{.OriginalInvoice}}{{end}}. Refund status: {{.RefundStatus}}.{{else}}This is an account credit of {{.Amount}}, applied automatically to future invoices until the balance is exhausted{{if .OriginalInvoice}}, originating from invoice {{.OriginalInvoice}}{{end}}.{{end}}
  </div>
</div>
</body>
</html>`
