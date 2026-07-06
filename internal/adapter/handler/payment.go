package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/port"
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

	invoice, err := h.invoiceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Database error")
		return
	}
	if invoice == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "Invoice not found")
		return
	}

	// Create Order on Gateway
	order, err := h.gateway.CreateOrder(
		c.Request.Context(),
		invoice.Total,
		invoice.Currency,
		invoice.InvoiceNumber,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Gateway error: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, order)
}
