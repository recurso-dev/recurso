package handler

import (
	"context"
	"errors"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type CreditNoteHandler struct {
	service  *service.CreditNoteService
	pdf      *service.InvoicePDFService
	branding invoiceBrandingResolver // optional; tenant letterhead on credit notes
}

func NewCreditNoteHandler(service *service.CreditNoteService) *CreditNoteHandler {
	return &CreditNoteHandler{service: service}
}

// SetPDFService wires the shared PDF renderer so credit notes print with the
// same seller letterhead as invoices. Nil-safe: without it, DownloadPDF 404s.
func (h *CreditNoteHandler) SetPDFService(pdf *service.InvoicePDFService) { h.pdf = pdf }

// SetBranding wires the per-tenant invoice branding so credit notes carry the
// same letterhead (display name + logo) as invoices. Nil-safe.
func (h *CreditNoteHandler) SetBranding(r invoiceBrandingResolver) { h.branding = r }

// DownloadPDF renders the credit note as printable HTML (the browser's
// print-to-PDF produces the document), mirroring the invoice PDF flow.
// GET /v1/credit-notes/:id/pdf (session or API key; tenant-scoped)
func (h *CreditNoteHandler) DownloadPDF(c *gin.Context) {
	if h.pdf == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "credit note documents are not enabled")
		return
	}
	tenantID := middleware.GetTenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid credit note id")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	cn, cust, invoiceNumber, err := h.service.GetForDocument(ctx, tenantID, id)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if cn == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "credit note not found")
		return
	}

	data := h.pdf.BuildCreditNoteData(cn, cust, invoiceNumber)
	if h.branding != nil {
		if b, berr := h.branding.GetByTenantID(ctx, tenantID); berr == nil && b != nil {
			if b.CompanyName != "" {
				data.SellerName = b.CompanyName
			}
			if b.LogoDataURL != "" {
				// Safe as template.URL: validated to a strict image data-URL shape
				// at write time (see invoice_branding.go).
				data.LogoDataURL = template.URL(b.LogoDataURL)
			}
		}
	}
	html, err := h.pdf.GenerateCreditNoteHTML(data)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to generate credit note")
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Content-Disposition", "inline; filename=\"credit-note-"+data.CreditNoteNumber+".html\"")
	c.String(http.StatusOK, html)
}

// GetCreditStatement returns a customer's consolidated account-credit statement:
// spendable balance (per currency/entity), grants, draw-down history, and a
// per-currency rollup.
// GET /v1/customers/:id/credit-statement
func (h *CreditNoteHandler) GetCreditStatement(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer id")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	stmt, err := h.service.GetCreditStatement(ctx, tenantID, customerID)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stmt})
}

func (h *CreditNoteHandler) CreateCreditNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	var req domain.CreateCreditNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	userID := middleware.GetUserID(c)
	userRole, _ := middleware.GetUserRole(c)

	cn, err := h.service.Create(ctx, tenantID, userID, userRole, req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrCreditNoteValidation) {
			status = http.StatusBadRequest
		}
		respondErrorStatus(c, status, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": cn})
}

func (h *CreditNoteHandler) ListCreditNotes(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var filter domain.CreditNoteFilter
	if customerIDStr := c.Query("customer_id"); customerIDStr != "" {
		if id, err := uuid.Parse(customerIDStr); err == nil {
			filter.CustomerID = &id
		}
	}

	// Status filter logic can be added later
	filter.Limit, filter.Offset = parseLimitOffset(c, 1000, 1000)

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	cns, err := h.service.List(ctx, tenantID, filter)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": cns})
}

func (h *CreditNoteHandler) ApproveCreditNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	cnIDStr := c.Param("id")
	cnID, err := uuid.Parse(cnIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid credit note id")
		return
	}
	userID := middleware.GetUserID(c)
	userRole, _ := middleware.GetUserRole(c)

	if userRole != "" && userRole != string(domain.RoleAdmin) && userRole != string(domain.RoleOwner) {
		respondError(c, http.StatusForbidden, codeValidationFailed, "only admins and owners can approve credit notes")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	cn, err := h.service.Approve(ctx, tenantID, cnID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrCreditNoteValidation) {
			status = http.StatusBadRequest
		}
		respondErrorStatus(c, status, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cn})
}

func (h *CreditNoteHandler) RejectCreditNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	cnIDStr := c.Param("id")
	cnID, err := uuid.Parse(cnIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid credit note id")
		return
	}
	userID := middleware.GetUserID(c)
	userRole, _ := middleware.GetUserRole(c)

	if userRole != "" && userRole != string(domain.RoleAdmin) && userRole != string(domain.RoleOwner) {
		respondError(c, http.StatusForbidden, codeValidationFailed, "only admins and owners can reject credit notes")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	cn, err := h.service.Reject(ctx, tenantID, cnID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrCreditNoteValidation) {
			status = http.StatusBadRequest
		}
		respondErrorStatus(c, status, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cn})
}

// VoidCreditNote cancels an issued account-credit note and writes off its
// unspent balance. Restricted to admins/owners, mirroring approve/reject.
// POST /v1/credit-notes/:id/void
func (h *CreditNoteHandler) VoidCreditNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	cnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid credit note id")
		return
	}
	userID := middleware.GetUserID(c)
	userRole, _ := middleware.GetUserRole(c)

	if userRole != "" && userRole != string(domain.RoleAdmin) && userRole != string(domain.RoleOwner) {
		respondError(c, http.StatusForbidden, codeValidationFailed, "only admins and owners can void credit notes")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	cn, err := h.service.Void(ctx, tenantID, cnID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrCreditNoteValidation) {
			status = http.StatusBadRequest
		}
		respondErrorStatus(c, status, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cn})
}
