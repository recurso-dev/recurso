package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/service"
)

// InvoicePDFHandler renders a printable HTML/PDF for a real invoice, choosing
// the jurisdiction layout (India GST vs a plain sales-tax/VAT invoice).
type InvoicePDFHandler struct {
	pdfService   *service.InvoicePDFService
	invoiceRepo  port.InvoiceRepository
	customerRepo port.CustomerRepository
}

// NewInvoicePDFHandler creates a new PDF handler.
func NewInvoicePDFHandler(pdfService *service.InvoicePDFService, invoiceRepo port.InvoiceRepository, customerRepo port.CustomerRepository) *InvoicePDFHandler {
	return &InvoicePDFHandler{
		pdfService:   pdfService,
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
	}
}

// DownloadPDF renders the invoice as printable HTML.
// GET /v1/invoices/:id/pdf (session or API key; tenant-scoped)
func (h *InvoicePDFHandler) DownloadPDF(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice id")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	inv, err := h.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice")
		return
	}
	if inv == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return
	}

	// A tax invoice without its buyer block is legally non-compliant, so a
	// failed customer lookup is an error, not a blank Bill To.
	var customer *domain.Customer
	if h.customerRepo != nil {
		customer, err = h.customerRepo.GetByID(ctx, inv.CustomerID)
		if err != nil || customer == nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice customer")
			return
		}
	}

	data := h.pdfService.BuildInvoiceData(inv, customer)

	// The e-invoice QR is GST-only — the IRN is set only on e-invoiced invoices.
	if data.IRN != "" {
		if qr, qerr := service.GenerateQRCode("SignedQRCode:" + data.IRN); qerr == nil {
			data.QRCodeData = qr
		}
	}

	html, err := h.pdfService.GenerateInvoiceHTML(data)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to generate invoice")
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Content-Disposition", "inline; filename=\"invoice-"+data.InvoiceNumber+".html\"")
	c.String(http.StatusOK, html)
}

// PreviewHTML returns the same rendered invoice.
// GET /v1/invoices/:id/preview
func (h *InvoicePDFHandler) PreviewHTML(c *gin.Context) {
	h.DownloadPDF(c)
}
