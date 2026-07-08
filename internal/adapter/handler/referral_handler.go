package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	pagination := ParsePagination(c)

	referrals, err := h.referralService.ListReferrals(c.Request.Context(), tenantID.(uuid.UUID), pagination.Limit, pagination.Offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	var req CreateReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
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
		respondErrorStatus(c, status, err.Error())
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
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	referralID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid referral ID")
		return
	}

	referral, err := h.referralService.QualifyReferral(c.Request.Context(), tenantID.(uuid.UUID), referralID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": referral})
}

func (h *ReferralHandler) GenerateCode(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	var req GenerateCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID.(uuid.UUID))
	code, err := h.referralService.GenerateCode(ctx, tenantID.(uuid.UUID), req.CustomerID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"code": code}})
}
