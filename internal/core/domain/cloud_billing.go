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
