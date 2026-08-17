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
