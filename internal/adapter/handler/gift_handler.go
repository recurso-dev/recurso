package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/middleware"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type GiftHandler struct {
	giftService *service.GiftService
}

func NewGiftHandler(giftService *service.GiftService) *GiftHandler {
	return &GiftHandler{giftService: giftService}
}

type PurchaseGiftRequest struct {
	BuyerCustomerID uuid.UUID `json:"buyer_customer_id" binding:"required"`
	PlanID          uuid.UUID `json:"plan_id" binding:"required"`
	RecipientEmail  string    `json:"recipient_email"`
	DurationMonths  int       `json:"duration_months" binding:"required"`
}

func (h *GiftHandler) PurchaseGift(c *gin.Context) {
	var req PurchaseGiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID.(uuid.UUID))
	gift, err := h.giftService.PurchaseGift(ctx, tenantID.(uuid.UUID), req.BuyerCustomerID, req.PlanID, req.RecipientEmail, req.DurationMonths)
	if err != nil {
		if errors.Is(err, service.ErrInvalidGiftDuration) {
			respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
			return
		}
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gift)
}

type RedeemGiftRequest struct {
	Code                string    `json:"code" binding:"required"`
	RecipientCustomerID uuid.UUID `json:"recipient_customer_id" binding:"required"`
}

func (h *GiftHandler) RedeemGift(c *gin.Context) {
	var req RedeemGiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID.(uuid.UUID))
	sub, err := h.giftService.RedeemGift(ctx, tenantID.(uuid.UUID), req.RecipientCustomerID, req.Code)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sub})
}

func (h *GiftHandler) ListGifts(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	pagination := ParsePagination(c)

	gifts, err := h.giftService.ListGifts(c.Request.Context(), tenantID.(uuid.UUID), pagination.Limit, pagination.Offset)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gifts,
		"meta": gin.H{"page": pagination.Page, "per_page": pagination.PerPage},
	})
}

// CancelGift cancels an unredeemed gift (policy: account credit). The buyer of
// a PAID purchase receives a spendable adjustment credit note; a still-open
// purchase invoice is voided instead. POST /v1/gifts/:id/cancel
func (h *GiftHandler) CancelGift(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	giftID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid gift id")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID.(uuid.UUID))
	actorID := middleware.GetUserID(c)
	actorRole, _ := middleware.GetUserRole(c)

	res, err := h.giftService.CancelGift(ctx, tenantID.(uuid.UUID), giftID, actorID, actorRole)
	switch {
	case errors.Is(err, service.ErrGiftNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, err.Error())
		return
	case errors.Is(err, service.ErrGiftAlreadyRedeemed), errors.Is(err, service.ErrGiftAlreadyCanceled):
		respondError(c, http.StatusConflict, codeConflict, err.Error())
		return
	case errors.Is(err, service.ErrGiftCreditUnwired):
		respondError(c, http.StatusServiceUnavailable, codeInternalError, err.Error())
		return
	case errors.Is(err, service.ErrGiftCanceledCreditFailed):
		// Partial success: the cancel took effect but the buyer's credit needs
		// a manual step. Explicit message, and 424 NOT 502 — Cloudflare fronts
		// the API and replaces origin 502s with its own HTML page, which would
		// hide the "manual credit owed" outcome (the exact thing this error
		// exists to surface).
		respondError(c, http.StatusFailedDependency, codeInternalError, err.Error())
		return
	case err != nil:
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}
