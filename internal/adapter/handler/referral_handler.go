package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/service"
)

type ReferralHandler struct {
	referralService *service.ReferralService
}

func NewReferralHandler(referralService *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{referralService: referralService}
}

func (h *ReferralHandler) ListReferrals(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	pagination := ParsePagination(c)

	referrals, err := h.referralService.ListReferrals(c.Request.Context(), tenantID.(uuid.UUID), pagination.Limit, pagination.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": referrals,
		"meta": gin.H{"page": pagination.Page, "per_page": pagination.PerPage},
	})
}

type CreateReferralRequest struct {
	ReferrerID   uuid.UUID `json:"referrer_id" binding:"required"`
	ReferredID   uuid.UUID `json:"referred_id" binding:"required"`
	RewardAmount int64     `json:"reward_amount"`
	Currency     string    `json:"currency"`
}

func (h *ReferralHandler) CreateReferral(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	var req CreateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.RewardAmount == 0 {
		req.RewardAmount = 500 // Default $5.00
	}

	referral, err := h.referralService.CreateReferral(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		req.ReferrerID,
		req.ReferredID,
		req.RewardAmount,
		req.Currency,
	)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrSelfReferral || err == service.ErrAlreadyReferred {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": referral})
}

type GenerateCodeRequest struct {
	CustomerID uuid.UUID `json:"customer_id" binding:"required"`
}

func (h *ReferralHandler) QualifyReferral(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid referral ID"})
		return
	}

	referral, err := h.referralService.QualifyReferral(c.Request.Context(), tenantID.(uuid.UUID), referralID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": referral})
}

func (h *ReferralHandler) GenerateCode(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	var req GenerateCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	code, err := h.referralService.GenerateCode(c.Request.Context(), tenantID.(uuid.UUID), req.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"code": code}})
}
