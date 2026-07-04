package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/adapter/db"
	"github.com/recur-so/recurso/internal/core/domain"
)

type CouponHandler struct {
	repo *db.CouponRepository
}

func NewCouponHandler(repo *db.CouponRepository) *CouponHandler {
	return &CouponHandler{repo: repo}
}

type CreateCouponRequest struct {
	Code           string `json:"code" binding:"required"`
	DiscountType   string `json:"discount_type" binding:"required,oneof=percent amount"`
	DiscountValue  int64  `json:"discount_value" binding:"required,gt=0"`
	Duration       string `json:"duration" binding:"required,oneof=forever once repeating"`
	DurationMonths *int   `json:"duration_months"`
}

func (h *CouponHandler) CreateCoupon(c *gin.Context) {
	var req CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}
	coupon := &domain.Coupon{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Code:           req.Code,
		DiscountType:   domain.DiscountType(req.DiscountType),
		DiscountValue:  req.DiscountValue,
		Duration:       domain.DurationType(req.Duration),
		DurationMonths: req.DurationMonths,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	if err := h.repo.Create(ctx, coupon); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create coupon"})
		return
	}

	c.JSON(http.StatusCreated, coupon)
}

func (h *CouponHandler) ListCoupons(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id missing"})
		return
	}

	coupons, err := h.repo.List(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list coupons"})
		return
	}

	// Helper for JSON array always being [] not null
	if coupons == nil {
		coupons = []*domain.Coupon{}
	}

	c.JSON(http.StatusOK, gin.H{"data": coupons})
}
