package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/service"
)

// ManualInvoiceHandler serves POST /v1/invoices/manual.
type ManualInvoiceHandler struct {
	svc *service.ManualInvoiceService
}

func NewManualInvoiceHandler(svc *service.ManualInvoiceService) *ManualInvoiceHandler {
	return &ManualInvoiceHandler{svc: svc}
}

type manualInvoiceLineRequest struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitAmount  int64  `json:"unit_amount"` // minor units
	Amount      int64  `json:"amount"`      // minor units; negative = waiver/discount
}

type createManualInvoiceRequest struct {
	CustomerID   string                     `json:"customer_id" binding:"required"`
	EntityID     string                     `json:"entity_id"`
	Currency     string                     `json:"currency" binding:"required"`
	InvoiceDate  string                     `json:"invoice_date"` // YYYY-MM-DD or RFC 3339; default now
	DueDate      string                     `json:"due_date"`     // YYYY-MM-DD or RFC 3339; wins over payment_terms
	PaymentTerms string                     `json:"payment_terms"`
	ExternalRef  string                     `json:"external_ref"`
	LineItems    []manualInvoiceLineRequest `json:"line_items" binding:"required,min=1"`
}

// parseManualDate accepts a calendar date or a full timestamp. A bare date is
// midnight UTC, so "2026-08-01" prints as 1 Aug regardless of server zone.
func parseManualDate(field, s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%s must be YYYY-MM-DD or an RFC 3339 timestamp", field)
}

// CreateManualInvoice handles POST /v1/invoices/manual.
//
// Creates a one-off invoice from explicit line items (negative lines allowed),
// for a chosen issuing entity and dates. Re-sending the same external_ref
// returns the original invoice (200, replayed=true) instead of a duplicate.
func (h *ManualInvoiceHandler) CreateManualInvoice(c *gin.Context) {
	// Creating a receivable books revenue and AR — owner/admin only (API keys bypass).
	if !requireManagerRole(c) {
		return
	}
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	var req createManualInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer_id")
		return
	}
	in := service.CreateManualInvoiceInput{
		TenantID:     tenantID,
		CustomerID:   customerID,
		Currency:     req.Currency,
		PaymentTerms: req.PaymentTerms,
		ExternalRef:  strings.TrimSpace(req.ExternalRef),
	}
	if req.EntityID != "" {
		entityID, perr := uuid.Parse(req.EntityID)
		if perr != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid entity_id")
			return
		}
		in.EntityID = &entityID
	}
	if req.InvoiceDate != "" {
		t, perr := parseManualDate("invoice_date", req.InvoiceDate)
		if perr != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, perr.Error())
			return
		}
		in.InvoiceDate = t
	}
	if req.DueDate != "" {
		t, perr := parseManualDate("due_date", req.DueDate)
		if perr != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, perr.Error())
			return
		}
		in.DueDate = &t
	}
	in.Lines = make([]service.ManualInvoiceLine, len(req.LineItems))
	for i, l := range req.LineItems {
		in.Lines[i] = service.ManualInvoiceLine{
			Description: l.Description,
			Quantity:    l.Quantity,
			UnitAmount:  l.UnitAmount,
			Amount:      l.Amount,
		}
	}

	inv, created, err := h.svc.CreateManualInvoice(c.Request.Context(), in)
	if err != nil {
		if errors.Is(err, service.ErrInvalidManualInvoice) {
			respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
			return
		}
		if errors.Is(err, service.ErrManualInvoiceRefConflict) {
			respondError(c, http.StatusConflict, codeConflict, err.Error())
			return
		}
		respondInternalError(c, err)
		return
	}

	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"data": inv, "replayed": !created})
}
