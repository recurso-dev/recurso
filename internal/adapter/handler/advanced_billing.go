package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/service"
)

type AdvancedBillingHandler struct {
	Service *service.AdvancedBillingService
    InvoiceService *service.InvoiceService
}

func NewAdvancedBillingHandler(svc *service.AdvancedBillingService, invSvc *service.InvoiceService) *AdvancedBillingHandler {
	return &AdvancedBillingHandler{Service: svc, InvoiceService: invSvc}
}

type AddUnbilledChargeRequest struct {
	Amount      int64  `json:"amount" binding:"required"`
	Currency    string `json:"currency" binding:"required"`
	Description string `json:"description" binding:"required"`
}

func (h *AdvancedBillingHandler) AddUnbilledCharge(c *gin.Context) {
	subIDStr := c.Param("id")
	subID, err := uuid.Parse(subIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	var req AddUnbilledChargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	charge, err := h.Service.AddUnbilledCharge(c.Request.Context(), subID, req.Amount, req.Currency, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, charge)
}

func (h *AdvancedBillingHandler) ListUnbilledCharges(c *gin.Context) {
	subIDStr := c.Param("id")
	subID, err := uuid.Parse(subIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	charges, err := h.Service.ListUnbilledCharges(subID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": charges})
}

type AdvanceInvoiceRequest struct {
    Periods int `json:"periods" binding:"required,min=1"`
}

func (h *AdvancedBillingHandler) GenerateAdvanceInvoice(c *gin.Context) {
    subIDStr := c.Param("id")
    subID, err := uuid.Parse(subIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
        return
    }

    var req AdvanceInvoiceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    inv, err := h.InvoiceService.GenerateAdvanceInvoice(c.Request.Context(), subID, req.Periods)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, inv)
}
