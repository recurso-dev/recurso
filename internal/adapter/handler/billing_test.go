package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type fakeTenantGetter struct{ t *domain.Tenant }

func (f *fakeTenantGetter) GetAccount(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
	return f.t, nil
}

func TestBillingStatus_TrialingTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	end := time.Now().Add(72 * time.Hour)
	h := NewBillingHandler(&fakeTenantGetter{t: &domain.Tenant{
		BillingStatus: domain.BillingStatusTrialing,
		PlanTier:      domain.PlanTierTrial,
		TrialEndsAt:   &end,
	}})

	c, w := jsonCtx(http.MethodGet, "/v1/billing/status", "")
	c.Set("tenant_id", uuid.New())
	h.Status(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", w.Code, w.Body.String())
	}
	var v struct {
		BillingStatus string `json:"billing_status"`
		PlanTier      string `json:"plan_tier"`
		TrialDaysLeft int    `json:"trial_days_left"`
		TrialExpired  bool   `json:"trial_expired"`
		TrialEndsAt   string `json:"trial_ends_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if v.BillingStatus != "trialing" || v.PlanTier != "trial" {
		t.Errorf("wrong status/tier: %+v", v)
	}
	if v.TrialDaysLeft != 3 || v.TrialExpired {
		t.Errorf("want 3 days left, not expired; got days=%d expired=%v", v.TrialDaysLeft, v.TrialExpired)
	}
	if v.TrialEndsAt == "" {
		t.Error("trial_ends_at should be present for a trialing tenant")
	}
}

func TestBillingStatus_ActiveTenantHasNoTrial(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewBillingHandler(&fakeTenantGetter{t: &domain.Tenant{
		BillingStatus: domain.BillingStatusActive,
		PlanTier:      domain.PlanTierFree,
	}})

	c, w := jsonCtx(http.MethodGet, "/v1/billing/status", "")
	c.Set("tenant_id", uuid.New())
	h.Status(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var v struct {
		BillingStatus string `json:"billing_status"`
		TrialDaysLeft int    `json:"trial_days_left"`
		TrialEndsAt   string `json:"trial_ends_at"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &v)
	if v.BillingStatus != "active" || v.TrialDaysLeft != 0 || v.TrialEndsAt != "" {
		t.Errorf("active tenant should have no trial: %+v", v)
	}
}
