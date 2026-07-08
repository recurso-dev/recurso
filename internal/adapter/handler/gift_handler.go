package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
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
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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

	c.JSON(http.StatusOK, sub)
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
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gifts,
		"meta": gin.H{"page": pagination.Page, "per_page": pagination.PerPage},
	})
}
