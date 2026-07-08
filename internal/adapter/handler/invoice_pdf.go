package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

// InvoicePDFHandler handles PDF invoice downloads
type InvoicePDFHandler struct {
	pdfService *service.InvoicePDFService
}

// NewInvoicePDFHandler creates a new PDF handler
func NewInvoicePDFHandler(pdfService *service.InvoicePDFService) *InvoicePDFHandler {
	return &InvoicePDFHandler{pdfService: pdfService}
}

// DownloadPDF generates and returns a PDF invoice
// GET /invoices/:id/pdf
func (h *InvoicePDFHandler) DownloadPDF(c *gin.Context) {
	invoiceIDStr := c.Param("id")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice id")
		return
	}

	// For demo, create sample data.
	// In production, fetch the invoice (with its persisted line items) from the
	// invoice service. The line items below are built from the invoice's real
	// LineItems via service.BuildPDFLineItems, so each row reflects its own
	// HSN/SAC code, rate, and per-line CGST/SGST/IGST; legacy invoices with no
	// line items fall back to a single synthetic line (see BuildPDFLineItems).
	sampleInvoice := &domain.Invoice{
		Currency:   "INR",
		Subtotal:   1000000,
		TaxAmount:  180000,
		CGSTAmount: 90000,
		SGSTAmount: 90000,
		Total:      1180000,
		HSNCode:    "998314",
		LineItems: []domain.InvoiceItem{
			{
				Description:   "SaaS Subscription - Pro Plan",
				HSNCode:       "998314",
				Quantity:      1,
				UnitAmount:    1000000,
				Amount:        1000000,
				TaxableAmount: 1000000,
				TaxRate:       18,
				CGSTAmount:    90000,
				SGSTAmount:    90000,
			},
		},
	}

	data := service.PDFInvoiceData{
		InvoiceNumber: "INV-2024-" + invoiceID.String()[:8],
		InvoiceDate:   "January 15, 2024",
		DueDate:       "January 30, 2024",
		BuyerName:     "Sample Customer",
		BuyerAddress:  "123 Main Street, Mumbai, MH 400001",
		Subtotal:      "₹10,000.00",
		CGSTRate:      "9%",
		CGSTAmount:    "₹900.00",
		SGSTRate:      "9%",
		SGSTAmount:    "₹900.00",
		GrandTotal:    "₹11,800.00",
		AmountInWords: "Rupees Eleven Thousand Eight Hundred Only",
		SACCode:       "998314",
		IsInterState:  false,
		PlaceOfSupply: "Maharashtra (27)",
		// GST Demo Data
		IRN:       service.GenerateIRN(), // Use helper
		AckNo:     "123456789012345",
		AckDate:   "2024-01-15 10:00:00",
		LineItems: service.BuildPDFLineItems(sampleInvoice),
	}

	// Generate QR Code
	qrContent := "SignedQRCode:" + data.IRN
	qrImage, err := service.GenerateQRCode(qrContent)
	if err == nil {
		data.QRCodeData = qrImage
	}

	html, err := h.pdfService.GenerateInvoiceHTML(data)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to generate invoice")
		return
	}

	// Return HTML that can be printed as PDF
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Content-Disposition", "inline; filename=\"invoice-"+data.InvoiceNumber+".html\"")
	c.String(http.StatusOK, html)
}

// PreviewHTML returns HTML preview of the invoice
// GET /invoices/:id/preview
func (h *InvoicePDFHandler) PreviewHTML(c *gin.Context) {
	h.DownloadPDF(c)
}
