package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type CheckoutHandler struct {
	invoiceRepo    port.InvoiceRepository
	paymentGateway port.PaymentGateway
}

func NewCheckoutHandler(repo port.InvoiceRepository, gw port.PaymentGateway) *CheckoutHandler {
	return &CheckoutHandler{
		invoiceRepo:    repo,
		paymentGateway: gw,
	}
}

// ShowCheckout returns invoice data as JSON for the frontend checkout page.
func (h *CheckoutHandler) ShowCheckout(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice ID")
		return
	}

	invoice, err := h.invoiceRepo.GetByIDPublic(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice")
		return
	}
	if invoice == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":             invoice.ID,
			"invoice_number": invoice.InvoiceNumber,
			"status":         string(invoice.Status),
			"currency":       invoice.Currency,
			"subtotal":       invoice.Subtotal,
			"tax_amount":     invoice.TaxAmount,
			"total":          invoice.Total,
			"display_amount": fmt.Sprintf("%.2f", float64(invoice.Total)/100.0),
			"due_date":       invoice.DueDate.Format("2006-01-02"),
			"customer_id":    invoice.CustomerID,
		},
	})
}

// InitiatePayment creates a payment order via the gateway and returns the order details.
func (h *CheckoutHandler) InitiatePayment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice ID")
		return
	}

	invoice, err := h.invoiceRepo.GetByIDPublic(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice")
		return
	}
	if invoice == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return
	}

	if invoice.Status == domain.InvoiceStatusPaid {
		respondError(c, http.StatusBadRequest, codeInvoiceAlreadyPaid, "invoice already paid")
		return
	}

	order, err := h.paymentGateway.CreateOrder(
		c.Request.Context(),
		invoice.Total,
		invoice.Currency,
		invoice.InvoiceNumber,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to create payment order")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"order_id":       order.ID,
			"amount":         order.Amount,
			"currency":       order.Currency,
			"invoice_id":     invoice.ID,
			"invoice_number": invoice.InvoiceNumber,
		},
	})
}

// CheckoutSuccess marks the invoice as paid and returns a success response.
func (h *CheckoutHandler) CheckoutSuccess(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice ID")
		return
	}

	invoice, err := h.invoiceRepo.GetByIDPublic(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to fetch invoice")
		return
	}
	if invoice == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return
	}

	if invoice.Status != domain.InvoiceStatusPaid {
		now := time.Now()
		invoice.Status = domain.InvoiceStatusPaid
		invoice.PaidAt = &now
		invoice.AmountPaid = invoice.Total

		if err := h.invoiceRepo.Update(c.Request.Context(), invoice); err != nil {
			respondError(c, http.StatusInternalServerError, codeInternalError, "failed to update invoice")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status":         "paid",
			"invoice_id":     invoice.ID,
			"invoice_number": invoice.InvoiceNumber,
		},
	})
}
