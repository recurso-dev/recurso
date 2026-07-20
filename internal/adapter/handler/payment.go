package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type PaymentHandler struct {
	gateway     port.PaymentGateway
	invoiceRepo port.InvoiceRepository
}

func NewPaymentHandler(gateway port.PaymentGateway, repo port.InvoiceRepository) *PaymentHandler {
	return &PaymentHandler{gateway: gateway, invoiceRepo: repo}
}

type createOrderRequest struct {
	InvoiceID string `json:"invoice_id" binding:"required"`
}

func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	id, err := uuid.Parse(req.InvoiceID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid Invoice ID")
		return
	}

	// /payments/order is a public (unauthenticated) endpoint, so there is no
	// tenant in the request context. Use the tenant-agnostic public lookup.
	invoice, err := h.invoiceRepo.GetByIDPublic(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Database error")
		return
	}
	if invoice == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "Invoice not found")
		return
	}

	// Don't create a payment order for an already-settled invoice — the buyer
	// would pay real money for nothing, with no automatic refund path (the
	// authenticated InitiatePayment path short-circuits the same way).
	if invoice.Status == domain.InvoiceStatusPaid {
		respondError(c, http.StatusBadRequest, codeInvoiceAlreadyPaid, "invoice is already paid")
		return
	}

	// Create Order on Gateway. Public endpoint: carry the invoice's tenant so
	// BYO routing picks the seller's own gateway (env fallback when unset).
	payCtx := context.WithValue(c.Request.Context(), domain.TenantIDKey, invoice.TenantID)
	order, err := h.gateway.CreateOrder(
		payCtx,
		invoice.Total,
		invoice.Currency,
		invoice.InvoiceNumber,
		invoice.ID.String(),
	)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": order})
}
