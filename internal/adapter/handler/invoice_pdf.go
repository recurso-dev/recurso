package handler

import (
	"context"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// InvoicePDFHandler renders a printable HTML/PDF for a real invoice, choosing
// the jurisdiction layout (India GST vs a plain sales-tax/VAT invoice).
type InvoicePDFHandler struct {
	pdfService   *service.InvoicePDFService
	invoiceRepo  port.InvoiceRepository
	customerRepo port.CustomerRepository
	seller       sellerJurisdictionResolver // optional; picks the per-tenant regime
	usTax        usTaxIdentityResolver      // optional; per-tenant US seller identity (W-9)
	branding     invoiceBrandingResolver    // optional; per-tenant invoice presentation (logo, signature, bank, terms)
}

// invoiceBrandingResolver returns a tenant's invoice branding. Optional and
// nil-safe; *db.InvoiceBrandingRepository satisfies it.
type invoiceBrandingResolver interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantInvoiceBranding, error)
}

// usTaxIdentityResolver returns a tenant's US tax identity (W-9). Optional and
// nil-safe; *db.TenantUSTaxConfigRepository satisfies it.
type usTaxIdentityResolver interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantUSTaxConfig, error)
}

// NewInvoicePDFHandler creates a new PDF handler.
func NewInvoicePDFHandler(pdfService *service.InvoicePDFService, invoiceRepo port.InvoiceRepository, customerRepo port.CustomerRepository) *InvoicePDFHandler {
	return &InvoicePDFHandler{
		pdfService:   pdfService,
		invoiceRepo:  invoiceRepo,
		customerRepo: customerRepo,
	}
}

// SetSellerResolver wires the seller-jurisdiction resolver so each invoice
// renders under its own tenant's regime instead of the PDF service's env-global
// seller country. Nil-safe: without it the service default (env) is used.
func (h *InvoicePDFHandler) SetSellerResolver(r sellerJurisdictionResolver) { h.seller = r }

// SetUSTaxIdentity wires the per-tenant US tax identity (W-9). Nil-safe: without
// it, or on a GST invoice, the env seller identity is used unchanged.
func (h *InvoicePDFHandler) SetUSTaxIdentity(r usTaxIdentityResolver) { h.usTax = r }

// SetBranding wires the per-tenant invoice branding. Nil-safe: without it the
// env presentation defaults render unchanged.
func (h *InvoicePDFHandler) SetBranding(r invoiceBrandingResolver) { h.branding = r }

// applyBranding overlays the tenant's presentation settings on the built
// invoice data. It runs BEFORE applyUSSellerIdentity so a statutory legal
// name (W-9) still wins on US tax invoices. The stored image values were
// validated to a strict data:image/(png|jpeg);base64 shape at write time,
// which is what makes the template.URL cast safe.
func (h *InvoicePDFHandler) applyBranding(ctx context.Context, tenantID uuid.UUID, data *service.PDFInvoiceData) {
	if h.branding == nil {
		return
	}
	b, err := h.branding.GetByTenantID(ctx, tenantID)
	if err != nil || b == nil {
		return
	}
	if b.CompanyName != "" {
		data.SellerName = b.CompanyName
	}
	if b.LogoDataURL != "" {
		data.LogoDataURL = template.URL(b.LogoDataURL)
	}
	if b.SignatureDataURL != "" {
		data.SignatureImageURL = template.URL(b.SignatureDataURL)
	}
	if b.SignatoryName != "" {
		data.SignedBy = b.SignatoryName
	}
	if b.BankDetails != "" {
		data.BankDetails = b.BankDetails
	}
	if b.Terms != "" {
		data.TermsAndConditions = b.Terms
	}
}

// applyUSSellerIdentity overrides the seller block of a US (non-GST) invoice
// with the tenant's own W-9 identity when one is set. GST invoices and the
// no-config case are left untouched (env identity).
func (h *InvoicePDFHandler) applyUSSellerIdentity(ctx context.Context, tenantID uuid.UUID, data *service.PDFInvoiceData) {
	if h.usTax == nil || data.ShowGST {
		return
	}
	cfg, err := h.usTax.GetByTenantID(ctx, tenantID)
	if err != nil || cfg == nil {
		return
	}
	if cfg.LegalName != "" {
		data.SellerName = cfg.LegalName
	}
	if cfg.Address != "" {
		data.SellerAddress = cfg.Address
	}
	if cfg.EIN != "" {
		data.SellerTaxLabel = "EIN"
		data.SellerTaxID = cfg.EIN
	}
}

// sellerCountryFor resolves a tenant's seller country, or "" when no resolver is
// wired (BuildInvoiceDataFor then falls back to the service's env default).
func (h *InvoicePDFHandler) sellerCountryFor(ctx context.Context, tenantID uuid.UUID) string {
	if h.seller == nil {
		return ""
	}
	return h.seller.SellerCountry(ctx, tenantID)
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

	data := h.pdfService.BuildInvoiceDataFor(inv, customer, h.sellerCountryFor(ctx, tenantID))
	h.applyBranding(ctx, tenantID, &data)
	h.applyUSSellerIdentity(ctx, tenantID, &data)

	// The e-invoice QR is GST-only — the IRN is set only on e-invoiced invoices.
	if data.IRN != "" {
		if qr, qerr := service.GenerateQRCode("SignedQRCode:" + data.IRN); qerr == nil {
			data.QRCodeData = template.URL(qr)
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

// PortalDownloadPDF renders an invoice PDF for the authenticated portal customer
// (ENG-152). It is scoped to the customer's OWN invoices — the ownership check
// (inv.CustomerID == portal_customer_id) is what makes a public, token-authed
// route safe to expose, since the tenant-scoped DownloadPDF can't run without a
// dashboard session/API key.
// GET /portal/api/invoices/:id/pdf
func (h *InvoicePDFHandler) PortalDownloadPDF(c *gin.Context) {
	cidVal, exists := c.Get("portal_customer_id")
	customerID, ok := cidVal.(uuid.UUID)
	if !exists || !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "not authenticated")
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice id")
		return
	}

	ctx := c.Request.Context()
	inv, err := h.invoiceRepo.GetByIDPublic(ctx, invoiceID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice")
		return
	}
	// A missing invoice and an invoice belonging to another customer return the
	// same 404 — never reveal that an id exists but isn't yours.
	if inv == nil || inv.CustomerID != customerID {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return
	}

	var customer *domain.Customer
	if h.customerRepo != nil {
		customer, err = h.customerRepo.GetByIDPublic(ctx, inv.CustomerID)
		if err != nil || customer == nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice customer")
			return
		}
	}

	data := h.pdfService.BuildInvoiceDataFor(inv, customer, h.sellerCountryFor(ctx, inv.TenantID))
	h.applyBranding(ctx, inv.TenantID, &data)
	h.applyUSSellerIdentity(ctx, inv.TenantID, &data)
	if data.IRN != "" {
		if qr, qerr := service.GenerateQRCode("SignedQRCode:" + data.IRN); qerr == nil {
			data.QRCodeData = template.URL(qr)
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
