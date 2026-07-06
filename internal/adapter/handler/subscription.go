package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

type SubscriptionHandler struct {
	service *service.SubscriptionService
}

func NewSubscriptionHandler(s *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: s}
}

type createSubscriptionRequest struct {
	CustomerID        string    `json:"customer_id" binding:"required,uuid"`
	PlanID            string    `json:"plan_id" binding:"required,uuid"`
	CouponCode        string    `json:"coupon_code"`         // P7
	StartDate         time.Time `json:"start_date"`          // Optional
	BillingAnchorType string    `json:"billing_anchor_type"` // P26: "acquisition" or "first_of_month"
	PaymentTerms      string    `json:"payment_terms"`       // P26: "net0", "net15", "net30", "net60"
}

func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	var req createSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	customerID, _ := uuid.Parse(req.CustomerID)
	planID, _ := uuid.Parse(req.PlanID)

	input := service.CreateSubscriptionInput{
		TenantID:          tenantID,
		CustomerID:        customerID,
		PlanID:            planID,
		CouponCode:        req.CouponCode,
		StartDate:         req.StartDate,
		BillingAnchorType: req.BillingAnchorType,
		PaymentTerms:      req.PaymentTerms,
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	sub, err := h.service.CreateSubscription(ctx, input)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, sub)
}

func (h *SubscriptionHandler) ListSubscriptions(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)

	// Parse query params
	status := c.Query("status")
	search := c.Query("q")

	limit := 10
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	offset := 0
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			offset = (v - 1) * limit
		}
	}

	filter := domain.SubscriptionFilter{
		Status: status,
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	subs, err := h.service.ListSubscriptions(ctx, tenantID, filter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if subs == nil {
		subs = []*domain.Subscription{}
	}
	c.JSON(http.StatusOK, gin.H{"data": subs})
}

func (h *SubscriptionHandler) ListInvoices(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	invs, err := h.service.ListInvoices(ctx, tenantID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}
	if invs == nil {
		invs = []*domain.Invoice{}
	}
	c.JSON(http.StatusOK, gin.H{"data": invs})
}

type updateSubscriptionRequest struct {
	PlanID string `json:"plan_id" binding:"required,uuid"`
}

func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	subscriptionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid subscription ID")
		return
	}

	var req updateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	newPlanID, _ := uuid.Parse(req.PlanID)

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	sub, err := h.service.UpdateSubscription(ctx, tenantID, subscriptionID, newPlanID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, sub)
}

// PauseSubscription handles POST /subscriptions/:id/pause
func (h *SubscriptionHandler) PauseSubscription(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	subID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid subscription ID")
		return
	}

	sub, err := h.service.PauseSubscription(c.Request.Context(), tenantID, subID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sub})
}

// ResumeSubscription handles POST /subscriptions/:id/resume
func (h *SubscriptionHandler) ResumeSubscription(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	subID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid subscription ID")
		return
	}

	sub, err := h.service.ResumeSubscription(c.Request.Context(), tenantID, subID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sub})
}
