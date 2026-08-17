package domain

import (
	"time"

	"github.com/google/uuid"
)

// CloudTenantCustomer links a signup tenant to the Customer that represents it
// inside the platform (founder) tenant's own Recurso account. It is the record
// behind "Recurso runs on Recurso": every tenant that signs up for Recurso
// Cloud becomes a customer of the founder's own Recurso business, billed by the
// same engine.
//
// Increment 1 populates the customer link only. SubscriptionID stays nil until
// a later increment adds the Recurso Cloud plan + usage metering.
type CloudTenantCustomer struct {
	ID               uuid.UUID  `json:"id"`
	PlatformTenantID uuid.UUID  `json:"platform_tenant_id" db:"platform_tenant_id"`
	TenantID         uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	CustomerID       uuid.UUID  `json:"customer_id" db:"customer_id"`
	SubscriptionID   *uuid.UUID `json:"subscription_id,omitempty" db:"subscription_id"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// CloudTenantUsage is one tenant's metered activity for a billing period, in a
// single currency — the reading the founder charges Recurso Cloud on. Increment
// 2 measures it; it is not yet turned into a charge.
//
//   - TrackedRevenueMinor: everything the tenant invoiced in the period (paid or
//     not) — the number the free-tier threshold is compared against.
//   - CollectedVolumeMinor: payments the tenant actually collected in the period
//     — the base a usage fee (e.g. the published 0.4%) would apply to.
type CloudTenantUsage struct {
	ID                   uuid.UUID `json:"id"`
	TenantID             uuid.UUID `json:"tenant_id" db:"tenant_id"`
	PeriodStart          time.Time `json:"period_start" db:"period_start"`
	PeriodEnd            time.Time `json:"period_end" db:"period_end"`
	Currency             string    `json:"currency" db:"currency"`
	TrackedRevenueMinor  int64     `json:"tracked_revenue_minor" db:"tracked_revenue_minor"`
	CollectedVolumeMinor int64     `json:"collected_volume_minor" db:"collected_volume_minor"`
	ComputedAt           time.Time `json:"computed_at" db:"computed_at"`
}

// Recurso Cloud pricing (the published model on recurso.dev). Amounts are minor
// units of the reporting currency (USD cents). Free up to $10,000 of tracked
// revenue a month; above that, the lower of 0.4% of collected volume or a flat
// $99 cap.
const (
	CloudFreeTierTrackedRevenueMinor int64 = 10_000_00 // $10,000
	CloudUsageRateBps                int64 = 40        // 0.4% = 40 basis points
	CloudMonthlyCapMinor             int64 = 99_00     // $99
)

// ComputeCloudCharge applies the Recurso Cloud pricing to one tenant's period
// totals (already normalized to the reporting currency) and returns the amount
// to charge in minor units plus a human reason. Pure function — the single
// source of truth for what a tenant owes, so it can be unit-tested exhaustively
// and reused by both the dry-run preview and (later) real invoicing.
func ComputeCloudCharge(trackedRevenueMinor, collectedVolumeMinor int64) (int64, string) {
	if trackedRevenueMinor <= CloudFreeTierTrackedRevenueMinor {
		return 0, "under $10,000 free tier"
	}
	pct := collectedVolumeMinor * CloudUsageRateBps / 10_000 // 0.4% of collected volume
	if pct <= CloudMonthlyCapMinor {
		return pct, "0.4% of collected volume"
	}
	return CloudMonthlyCapMinor, "$99 monthly cap"
}

// CloudChargePreview is one tenant's DRY-RUN charge for a period: what Recurso
// Cloud would bill, in the reporting currency, with the usage it was computed
// from. It is a preview only — no invoice, no ledger, no money. Used to review
// pricing before real charging is enabled.
type CloudChargePreview struct {
	ID                   uuid.UUID `json:"id"`
	PeriodStart          time.Time `json:"period_start" db:"period_start"`
	PeriodEnd            time.Time `json:"period_end" db:"period_end"`
	TenantID             uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Currency             string    `json:"currency" db:"currency"`
	TrackedRevenueMinor  int64     `json:"tracked_revenue_minor" db:"tracked_revenue_minor"`
	CollectedVolumeMinor int64     `json:"collected_volume_minor" db:"collected_volume_minor"`
	WouldChargeMinor     int64     `json:"would_charge_minor" db:"would_charge_minor"`
	Reason               string    `json:"reason" db:"reason"`
	ComputedAt           time.Time `json:"computed_at" db:"computed_at"`
}
