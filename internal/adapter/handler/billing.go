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

// PlatformPlan is one managed-cloud plan tenants can be on. Values MIRROR the
// public pricing at recurso.dev/pricing — the founder-authored source of truth —
// so the dashboard and the marketing site never disagree. Cloud is usage-metered
// (free below the threshold, then a % of tracked volume).
type PlatformPlan struct {
	Tier        string   `json:"tier"`
	Name        string   `json:"name"`
	Price       string   `json:"price"`     // display string, e.g. "$0" / "0.4% of volume" / "Custom"
	Period      string   `json:"period"`    // e.g. "forever" / "to start" / ""
	FreeNote    string   `json:"free_note"` // e.g. "Free to $10k tracked revenue/mo"
	Features    []string `json:"features"`
	CTA         string   `json:"cta"`
	Recommended bool     `json:"recommended"`
}

// platformPlans mirrors recurso.dev/pricing. Kept as data so pricing changes are
// a one-line edit; the checkout/metering that ENFORCE these land once the
// managed-cloud gateway credentials are provided (business/infra dependency).
var platformPlans = []PlatformPlan{
	{
		Tier: "self_hosted", Name: "Self-Hosted", Price: "Free", Period: "forever",
		FreeNote: "Unlimited — run it yourself",
		Features: []string{"Every feature, no paywalled add-ons", "All gateways + GST/EU/US tax", "Community support", "MIT licensed"},
		CTA:      "Get started on GitHub",
	},
	{
		Tier: "cloud", Name: "Cloud", Price: "0.4% of volume", Period: "usage-based",
		FreeNote:    "Free to $10k tracked revenue / mo",
		Features:    []string{"Managed hosting + auto-scaling", "Daily backups", "99.9% uptime SLA", "Email support"},
		CTA:         "Start free",
		Recommended: true,
	},
	{
		Tier: "enterprise", Name: "Enterprise", Price: "Custom", Period: "",
		FreeNote: "Volume pricing + SOC 2",
		Features: []string{"Priority support + SLA", "99.99% uptime", "SOC 2", "On-prem option"},
		CTA:      "Talk to us",
	},
}

// Plans returns the managed-cloud plan catalog.
//
// GET /v1/billing/plans
func (h *BillingHandler) Plans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"plans": platformPlans})
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
