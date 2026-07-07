package handler

import (
	"context"
	"errors"
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
	TrialDays         int       `json:"trial_days"`          // >0 starts the subscription in "trialing"
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
		TrialDays:         req.TrialDays,
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

// PreviewPlanChange handles GET /subscriptions/:id/preview-change?plan_id=<uuid>.
// It returns the proration breakdown for switching plans WITHOUT applying it,
// using the same math UpdateSubscription applies.
func (h *SubscriptionHandler) PreviewPlanChange(c *gin.Context) {
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

	planIDStr := c.Query("plan_id")
	if planIDStr == "" {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "plan_id query parameter is required")
		return
	}
	newPlanID, err := uuid.Parse(planIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid plan_id")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	preview, err := h.service.PreviewPlanChange(ctx, tenantID, subscriptionID, newPlanID)
	if err != nil {
		if errors.Is(err, service.ErrSubscriptionNotFound) || errors.Is(err, service.ErrPlanNotFound) {
			respondError(c, http.StatusNotFound, codeNotFound, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, preview)
}

type addAddonRequest struct {
	PlanID   string `json:"plan_id" binding:"required,uuid"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

// AddAddon handles POST /subscriptions/:id/addons. It attaches an add-on plan
// to the subscription; the add-on is billed from the next recurring invoice.
func (h *SubscriptionHandler) AddAddon(c *gin.Context) {
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

	var req addAddonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	planID, _ := uuid.Parse(req.PlanID)

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	addon, err := h.service.AddAddon(ctx, tenantID, subID, planID, req.Quantity)
	if err != nil {
		h.respondAddonError(c, err)
		return
	}

	c.JSON(http.StatusCreated, addon)
}

// ListAddons handles GET /subscriptions/:id/addons.
func (h *SubscriptionHandler) ListAddons(c *gin.Context) {
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

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	addons, err := h.service.ListAddons(ctx, tenantID, subID)
	if err != nil {
		h.respondAddonError(c, err)
		return
	}
	if addons == nil {
		addons = []*domain.SubscriptionAddon{}
	}
	c.JSON(http.StatusOK, gin.H{"data": addons})
}

// RemoveAddon handles DELETE /subscriptions/:id/addons/:addonId.
func (h *SubscriptionHandler) RemoveAddon(c *gin.Context) {
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
	addonID, err := uuid.Parse(c.Param("addonId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid add-on ID")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	if err := h.service.RemoveAddon(ctx, tenantID, subID, addonID); err != nil {
		h.respondAddonError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// respondAddonError maps add-on service errors to the canonical HTTP envelope.
func (h *SubscriptionHandler) respondAddonError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSubscriptionNotFound),
		errors.Is(err, service.ErrPlanNotFound),
		errors.Is(err, service.ErrAddonNotFound):
		respondError(c, http.StatusNotFound, codeNotFound, err.Error())
	case errors.Is(err, service.ErrAddonCurrencyMismatch),
		errors.Is(err, service.ErrInvalidQuantity):
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
	default:
		respondError(c, http.StatusInternalServerError, codeInternalError, err.Error())
	}
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
