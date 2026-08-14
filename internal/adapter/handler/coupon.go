package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
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
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	// A percent discount above 100% is nonsensical and would drive an invoice's
	// taxable base negative; reject it at creation (the application path also
	// clamps the discount to the subtotal as a backstop).
	if req.DiscountType == "percent" && req.DiscountValue > 100 {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "a percent discount_value cannot exceed 100")
		return
	}

	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
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
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	if err := h.repo.Create(ctx, coupon); err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to create coupon")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": coupon})
}

type updateCouponRequest struct {
	Active *bool `json:"active" binding:"required"`
}

// UpdateCoupon flips the redemption gate: PUT /v1/coupons/:id {"active": false}
// deactivates (new subscriptions can no longer redeem the code), true restores.
// Existing subscriptions keep their applied discount either way.
func (h *CouponHandler) UpdateCoupon(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid coupon id")
		return
	}

	var req updateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	if err := h.repo.SetActive(c.Request.Context(), tenantID, id, *req.Active); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(c, http.StatusNotFound, codeNotFound, "coupon not found")
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to update coupon")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": map[bool]string{true: "activated", false: "deactivated"}[*req.Active]})
}

// GetCoupon handles GET /v1/coupons/:id (tenant-scoped) — one coupon as an
// addressable object for the dashboard's coupon page. A missing or cross-tenant
// id returns 404.
func (h *CouponHandler) GetCoupon(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid coupon id")
		return
	}
	coupon, err := h.repo.GetByID(c.Request.Context(), tenantID, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to load coupon")
		return
	}
	if coupon == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "coupon not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": coupon})
}

func (h *CouponHandler) ListCoupons(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	limit, offset := parseLimitOffset(c, 1000, 1000)
	coupons, err := h.repo.List(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "Failed to list coupons")
		return
	}

	// Helper for JSON array always being [] not null
	if coupons == nil {
		coupons = []*domain.Coupon{}
	}

	c.JSON(http.StatusOK, gin.H{"data": coupons})
}
