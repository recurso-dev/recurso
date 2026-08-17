package domain

import "time"

// PlatformMetrics is the operator-facing (founder) cross-tenant snapshot of the
// managed-cloud funnel: signups, activation, and billing state across ALL
// tenants. It deliberately lives outside the tenant-scoped API — a tenant login
// can never see it — and is served only to a FOUNDER_TOKEN holder. Read-only.
type PlatformMetrics struct {
	TotalTenants     int              `json:"total_tenants"`
	SignupsLast7d    int              `json:"signups_last_7d"`
	SignupsLast30d   int              `json:"signups_last_30d"`
	ActivatedTenants int              `json:"activated_tenants"` // tenants with >=1 customer
	TrialsExpiring7d int              `json:"trials_expiring_7d"`
	ByBillingStatus  map[string]int   `json:"by_billing_status"`
	ByPlanTier       map[string]int   `json:"by_plan_tier"`
	RecentSignups    []PlatformSignup `json:"recent_signups"`
	GeneratedAt      time.Time        `json:"generated_at"`

	// Recurso Cloud self-billing dry-run (Increment 3a): what each tenant WOULD
	// be charged this period. A preview only — no invoice, no money. Empty until
	// the usage meter + preview run (PLATFORM_TENANT_ID set).
	CloudCharges          []PlatformCloudCharge `json:"cloud_charges"`
	CloudChargeTotalMinor int64                 `json:"cloud_charge_total_minor"`
	CloudChargeCurrency   string                `json:"cloud_charge_currency"`
}

// PlatformCloudCharge is one tenant's dry-run Recurso Cloud charge for the
// current period, with the tenant's identity, in the reporting currency.
type PlatformCloudCharge struct {
	TenantID             string `json:"tenant_id"`
	Name                 string `json:"name"`
	Email                string `json:"email"`
	TrackedRevenueMinor  int64  `json:"tracked_revenue_minor"`
	CollectedVolumeMinor int64  `json:"collected_volume_minor"`
	WouldChargeMinor     int64  `json:"would_charge_minor"`
	Reason               string `json:"reason"`
}

// PlatformSignup is one recent tenant in the founder feed.
type PlatformSignup struct {
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	PlanTier      string     `json:"plan_tier"`
	BillingStatus string     `json:"billing_status"`
	TrialEndsAt   *time.Time `json:"trial_ends_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	Activated     bool       `json:"activated"` // has created >=1 customer
}
