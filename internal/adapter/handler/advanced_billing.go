package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type AdvancedBillingHandler struct {
	Service        *service.AdvancedBillingService
	InvoiceService *service.InvoiceService
}

func NewAdvancedBillingHandler(svc *service.AdvancedBillingService, invSvc *service.InvoiceService) *AdvancedBillingHandler {
	return &AdvancedBillingHandler{Service: svc, InvoiceService: invSvc}
}

// tenantCtx wraps the authenticated tenant into the request context so the
// repositories (which read domain.TenantIDKey) can scope their queries.
func (h *AdvancedBillingHandler) tenantCtx(c *gin.Context) (context.Context, bool) {
	v, exists := c.Get("tenant_id")
	tenantID, ok := v.(uuid.UUID)
	if !exists || !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return nil, false
	}
	return context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID), true
}

type AddUnbilledChargeRequest struct {
	Amount      int64  `json:"amount" binding:"required"`
	Currency    string `json:"currency" binding:"required"`
	Description string `json:"description" binding:"required"`
	// HSNCode is optional. When set, the charge is taxed at this HSN/SAC code's
	// rate on the invoice; empty falls back to the tenant SAC.
	HSNCode string `json:"hsn_code"`
}

func (h *AdvancedBillingHandler) AddUnbilledCharge(c *gin.Context) {
	subIDStr := c.Param("id")
	subID, err := uuid.Parse(subIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid subscription ID")
		return
	}

	var req AddUnbilledChargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	ctx, ok := h.tenantCtx(c)
	if !ok {
		return
	}
	charge, err := h.Service.AddUnbilledCharge(ctx, subID, req.Amount, req.Currency, req.Description, req.HSNCode)
	if err != nil {
		if errors.Is(err, service.ErrInvalidChargeAmount) {
			respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, charge)
}

func (h *AdvancedBillingHandler) ListUnbilledCharges(c *gin.Context) {
	subIDStr := c.Param("id")
	subID, err := uuid.Parse(subIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid subscription ID")
		return
	}

	ctx, ok := h.tenantCtx(c)
	if !ok {
		return
	}
	charges, err := h.Service.ListUnbilledCharges(ctx, subID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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
		respondError(c, http.StatusBadRequest, codeValidationFailed, "Invalid subscription ID")
		return
	}

	var req AdvanceInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	ctx, ok := h.tenantCtx(c)
	if !ok {
		return
	}
	inv, err := h.InvoiceService.GenerateAdvanceInvoice(ctx, subID, req.Periods)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, inv)
}
