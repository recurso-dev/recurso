package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	DataRegion     string     `json:"data_region" db:"data_region"`
	BaseCurrency   string     `json:"base_currency" db:"base_currency"` // Default: "USD"
	OrganizationID *uuid.UUID `json:"organization_id,omitempty" db:"organization_id"`
	// Managed-cloud billing lifecycle (Phase B). BillingStatus:
	// 'trialing' | 'active' | 'past_due' | 'canceled'. PlanTier: 'trial' |
	// 'free' | a paid tier. TrialEndsAt is nil for non-trial tenants.
	TrialEndsAt   *time.Time `json:"trial_ends_at,omitempty" db:"trial_ends_at"`
	BillingStatus string     `json:"billing_status" db:"billing_status"`
	PlanTier      string     `json:"plan_tier" db:"plan_tier"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Billing status + plan-tier constants for the managed-cloud lifecycle.
const (
	BillingStatusTrialing = "trialing"
	BillingStatusActive   = "active"
	BillingStatusPastDue  = "past_due"
	BillingStatusCanceled = "canceled"

	PlanTierTrial = "trial"
	PlanTierFree  = "free"
)

// IsTrialing reports whether the tenant is currently in a trial.
func (t *Tenant) IsTrialing() bool { return t.BillingStatus == BillingStatusTrialing }

// TrialDaysLeft returns whole days remaining in the trial (0 if none/expired,
// rounded up so "18 hours left" reads as 1 day).
func (t *Tenant) TrialDaysLeft(now time.Time) int {
	if t.TrialEndsAt == nil {
		return 0
	}
	d := t.TrialEndsAt.Sub(now)
	if d <= 0 {
		return 0
	}
	days := int(d / (24 * time.Hour))
	if d%(24*time.Hour) > 0 {
		days++
	}
	return days
}

// IsTrialExpired reports whether a trialing tenant's trial window has passed.
func (t *Tenant) IsTrialExpired(now time.Time) bool {
	return t.IsTrialing() && t.TrialEndsAt != nil && !t.TrialEndsAt.After(now)
}

// IRPConfig holds per-tenant IRP (Invoice Registration Portal) credentials
type IRPConfig struct {
	ID           string `json:"id" db:"id"`
	TenantID     string `json:"tenant_id" db:"tenant_id"`
	Environment  string `json:"environment" db:"environment"` // "sandbox" or "production"
	ClientID     string `json:"client_id" db:"client_id"`
	ClientSecret string `json:"client_secret" db:"client_secret"`
	Username     string `json:"username" db:"username"`
	Password     string `json:"password" db:"password"`
	GSTIN        string `json:"gstin" db:"gstin"`
	IsEnabled    bool   `json:"is_enabled" db:"is_enabled"`
}

type APIKey struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	KeyValue  string    `json:"key_value,omitempty"`  // Original key: populated ONLY at creation; reads leave it empty (hash is stored)
	KeyHash   string    `json:"-"`                    // bcrypt hash (stored in DB)
	KeyPrefix string    `json:"key_prefix,omitempty"` // First 8 chars (for lookup + display)
	Type      string    `json:"type"`                 // "secret"
	IsActive  bool      `json:"is_active"`
	Livemode  bool      `json:"livemode"` // true = rsk_live_ (real money), false = rsk_test_
	CreatedAt time.Time `json:"created_at"`
}

// NewAPIKeyValue builds a fresh secret key string for the given mode. Live keys
// are prefixed rsk_live_, test keys rsk_test_ — the prefix is what the auth
// layer gates against, so a test key can never run on a live-money server.
func NewAPIKeyValue(livemode bool, randomPart string) string {
	if livemode {
		return "rsk_live_" + randomPart
	}
	return "rsk_test_" + randomPart
}
