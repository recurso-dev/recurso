package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// sellerJurisdictionResolver reports a tenant's seller ISO-2 country using the
// same logic as tax resolution, so invoice PRESENTATION matches taxation.
// Optional and nil-safe; *service.TaxResolver satisfies it.
type sellerJurisdictionResolver interface {
	SellerCountry(ctx context.Context, tenantID uuid.UUID) string
}

type SubscriptionHandler struct {
	service *service.SubscriptionService
	seller  sellerJurisdictionResolver // optional; stamps the per-tenant invoice regime
}

func NewSubscriptionHandler(s *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: s}
}

// SetSellerResolver wires the seller-jurisdiction resolver so listed invoices
// carry a tax_regime matching the seller's jurisdiction. Nil-safe: without it,
// each invoice falls back to RegimeFallback() (its own currency / GST split).
func (h *SubscriptionHandler) SetSellerResolver(r sellerJurisdictionResolver) { h.seller = r }

type createSubscriptionRequest struct {
	CustomerID        string    `json:"customer_id" binding:"required,uuid"`
	PlanID            string    `json:"plan_id" binding:"required,uuid"`
	EntityID          string    `json:"entity_id" binding:"omitempty,uuid"` // Multi-Entity: issuing entity; empty ⇒ primary
	CouponCode        string    `json:"coupon_code"`                        // P7
	StartDate         time.Time `json:"start_date"`                         // Optional
	BillingAnchorType string    `json:"billing_anchor_type"`                // P26: "acquisition" or "first_of_month"
	PaymentTerms      string    `json:"payment_terms"`                      // P26: "net0", "net15", "net30", "net60"
	TrialDays         int       `json:"trial_days"`                         // >0 starts the subscription in "trialing"
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

	var entityID *uuid.UUID
	if req.EntityID != "" {
		if id, err := uuid.Parse(req.EntityID); err == nil {
			entityID = &id
		}
	}

	input := service.CreateSubscriptionInput{
		TenantID:          tenantID,
		EntityID:          entityID,
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
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": sub})
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

	var planID uuid.UUID
	if s := c.Query("plan_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid plan_id")
			return
		}
		planID = id
	}

	var customerID uuid.UUID
	if s := c.Query("customer_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer_id")
			return
		}
		customerID = id
	}

	var startedAfter *time.Time
	if s := c.Query("started_after"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "started_after must be RFC 3339")
			return
		}
		startedAfter = &t
	}

	limit, offset := parsePageLimit(c)

	filter := domain.SubscriptionFilter{
		Status:       status,
		Search:       search,
		PlanID:       planID,
		CustomerID:   customerID,
		StartedAfter: startedAfter,
		Limit:        limit,
		Offset:       offset,
	}

	subs, err := h.service.ListSubscriptions(ctx, tenantID, filter)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if subs == nil {
		subs = []*domain.Subscription{}
	}
	c.JSON(http.StatusOK, gin.H{"data": subs})
}

// GetInvoice returns one invoice by id, scoped to the caller's tenant (the
// repository enforces the tenant from context). Foreign or missing invoices
// are a flat 404 — never leak existence across tenants. Stamps the same
// presentation regime the list endpoint stamps so the dashboard renders GST
// artifacts consistently.
// GET /v1/invoices/:id
func (h *SubscriptionHandler) GetInvoice(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	invID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid invoice id")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	inv, err := h.service.GetInvoice(ctx, invID)
	if err != nil || inv == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "invoice not found")
		return
	}
	if h.seller != nil {
		if regime := service.RegimeForCountry(h.seller.SellerCountry(ctx, tenantID)); regime != "" {
			inv.TaxRegime = regime
		} else {
			inv.TaxRegime = inv.RegimeFallback()
		}
	} else {
		inv.TaxRegime = inv.RegimeFallback()
	}
	c.JSON(http.StatusOK, gin.H{"data": inv})
}

// GetSubscription returns one subscription by id, scoped to the caller's
// tenant. A subscription owned by another tenant (or a missing one) is a flat
// 404 — never leak existence across tenants.
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}
	subID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid subscription id")
		return
	}
	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	sub, err := h.service.GetByID(ctx, tenantID, subID)
	if err != nil || sub == nil {
		respondError(c, http.StatusNotFound, codeNotFound, "subscription not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sub})
}

func (h *SubscriptionHandler) ListInvoices(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "tenant_id missing")
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	// Server-side pagination: a large account must not return every invoice in
	// one response. Defaults page=1/per_page=50, capped at 250 (ParsePagination).
	p := ParsePagination(c)

	var (
		invs  []*domain.Invoice
		total int
		err   error
	)
	if s := c.Query("customer_id"); s != "" {
		// Customer-scoped list for the customer object page. The repo query
		// carries the tenant predicate, so a foreign customer id yields an
		// empty page, never another tenant's invoices.
		customerID, perr := uuid.Parse(s)
		if perr != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid customer_id")
			return
		}
		invs, total, err = h.service.ListInvoicesByCustomerPaginated(ctx, tenantID, customerID, p.PerPage, p.Offset)
	} else if s := c.Query("subscription_id"); s != "" {
		// Subscription-scoped list for the subscription object page; same
		// tenant-in-SQL guarantee as the customer filter.
		subscriptionID, perr := uuid.Parse(s)
		if perr != nil {
			respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid subscription_id")
			return
		}
		invs, total, err = h.service.ListInvoicesBySubscriptionPaginated(ctx, tenantID, subscriptionID, p.PerPage, p.Offset)
	} else {
		invs, total, err = h.service.ListInvoicesPaginated(ctx, tenantID, p.PerPage, p.Offset)
	}
	if err != nil {
		respondInternalError(c, err)
		return
	}
	if invs == nil {
		invs = []*domain.Invoice{}
	}

	// Stamp the presentation regime so the dashboard hides GST artifacts (HSN,
	// CGST/SGST/IGST, IRN) for non-Indian sellers. All invoices in this response
	// share one tenant, so the seller country is resolved once, not per row.
	regime := ""
	if h.seller != nil {
		regime = service.RegimeForCountry(h.seller.SellerCountry(ctx, tenantID))
	}
	for _, inv := range invs {
		if regime != "" {
			inv.TaxRegime = regime
		} else {
			inv.TaxRegime = inv.RegimeFallback()
		}
	}

	totalPages := (total + p.PerPage - 1) / p.PerPage
	c.JSON(http.StatusOK, gin.H{
		"data": invs,
		"pagination": gin.H{
			"page":        p.Page,
			"per_page":    p.PerPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
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
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sub})
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
		respondInternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": preview})
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

	c.JSON(http.StatusCreated, gin.H{"data": addon})
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
		respondInternalError(c, err)
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

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	// Manual pause is indefinite (nil resume) — the caller resumes via /resume.
	// Timed pauses come from the retention flow, which passes a resume date.
	sub, err := h.service.PauseSubscription(ctx, tenantID, subID, nil)
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

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	sub, err := h.service.ResumeSubscription(ctx, tenantID, subID)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sub})
}

// SetCommitment handles PUT /v1/subscriptions/:id/commitment — the
// per-period minimum in minor units (Lago-parity B2). Amount 0 clears it.
func (h *SubscriptionHandler) SetCommitment(c *gin.Context) {
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

	var req struct {
		Amount *int64 `json:"amount" binding:"required,gte=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}

	ctx := context.WithValue(c.Request.Context(), domain.TenantIDKey, tenantID)
	sub, err := h.service.SetCommitment(ctx, tenantID, subID, *req.Amount)
	if err != nil {
		respondError(c, http.StatusBadRequest, codeValidationFailed, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sub})
}
