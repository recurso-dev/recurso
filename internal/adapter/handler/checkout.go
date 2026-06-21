package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/port"
)

type CheckoutHandler struct {
	invoiceRepo port.InvoiceRepository
}

func NewCheckoutHandler(repo port.InvoiceRepository) *CheckoutHandler {
	return &CheckoutHandler{invoiceRepo: repo}
}

func (h *CheckoutHandler) ShowCheckout(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid Invoice ID")
		return
	}

	invoice, err := h.invoiceRepo.GetByIDPublic(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching invoice")
		return
	}
	if invoice == nil {
		c.String(http.StatusNotFound, "Invoice not found")
		return
	}

	// View Model
	data := gin.H{
		"InvoiceNumber": invoice.InvoiceNumber,
		"Status":        string(invoice.Status),
		"Currency":      invoice.Currency,
		"DisplayAmount": fmt.Sprintf("%.2f", float64(invoice.Total)/100.0), // Assuming usually 2 decimals
		"DueDate":       invoice.DueDate.Format("2006-01-02"),
	}

	c.HTML(http.StatusOK, "checkout.html", data)
}
