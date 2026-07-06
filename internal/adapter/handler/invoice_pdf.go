package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	// For demo, create sample data
	// In production, fetch from invoice service
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
		IRN:     service.GenerateIRN(), // Use helper
		AckNo:   "123456789012345",
		AckDate: "2024-01-15 10:00:00",
		LineItems: []service.PDFLineItem{
			{
				SNo:         1,
				Description: "SaaS Subscription - Pro Plan",
				SACCode:     "998314",
				Quantity:    "1",
				UnitPrice:   "₹10,000.00",
				TaxRate:     "18%",
				Amount:      "₹10,000.00",
			},
		},
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
