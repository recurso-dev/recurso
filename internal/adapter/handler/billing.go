package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// tenantAccountGetter is the narrow slice of TenantService the billing handler
// needs. *service.TenantService satisfies it; tests supply a fake.
type tenantAccountGetter interface {
	GetAccount(ctx context.Context, tenantID uuid.UUID) (*domain.Tenant, error)
}

// BillingHandler serves the tenant's managed-cloud billing/trial status. This
// first increment is read-only: it surfaces the trial so the dashboard can nudge
// toward upgrading. Self-serve checkout + paywall enforcement land once pricing
// is decided.
type BillingHandler struct {
	tenants tenantAccountGetter
}

func NewBillingHandler(tenants tenantAccountGetter) *BillingHandler {
	return &BillingHandler{tenants: tenants}
}

type billingStatusView struct {
	BillingStatus string  `json:"billing_status"`
	PlanTier      string  `json:"plan_tier"`
	TrialEndsAt   *string `json:"trial_ends_at,omitempty"`
	TrialDaysLeft int     `json:"trial_days_left"`
	TrialExpired  bool    `json:"trial_expired"`
}

// Status returns the caller tenant's billing lifecycle state.
//
// GET /v1/billing/status
func (h *BillingHandler) Status(c *gin.Context) {
	tenantID, ok := c.MustGet("tenant_id").(uuid.UUID)
	if !ok {
		respondError(c, http.StatusUnauthorized, codeUnauthorized, "missing tenant")
		return
	}

	t, err := h.tenants.GetAccount(c.Request.Context(), tenantID)
	if err != nil {
		respondInternalError(c, err)
		return
	}

	now := time.Now()
	view := billingStatusView{
		BillingStatus: t.BillingStatus,
		PlanTier:      t.PlanTier,
		TrialDaysLeft: t.TrialDaysLeft(now),
		TrialExpired:  t.IsTrialExpired(now),
	}
	if t.TrialEndsAt != nil {
		s := t.TrialEndsAt.Format(time.RFC3339)
		view.TrialEndsAt = &s
	}
	c.JSON(http.StatusOK, view)
}
