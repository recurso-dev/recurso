package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/service"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	gift, err := h.giftService.PurchaseGift(c.Request.Context(), tenantID.(uuid.UUID), req.BuyerCustomerID, req.PlanID, req.RecipientEmail, req.DurationMonths)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	sub, err := h.giftService.RedeemGift(c.Request.Context(), tenantID.(uuid.UUID), req.RecipientCustomerID, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sub)
}

func (h *GiftHandler) ListGifts(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	pagination := ParsePagination(c)

	gifts, err := h.giftService.ListGifts(c.Request.Context(), tenantID.(uuid.UUID), pagination.Limit, pagination.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gifts,
		"meta": gin.H{"page": pagination.Page, "per_page": pagination.PerPage},
	})
}
