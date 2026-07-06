package domain

import (
	"time"

	"github.com/google/uuid"
)

// AccountingEntityMapping links an internal entity (customer, invoice,
// product) to its provider-side ID on a specific accounting connection
// (e.g. a QuickBooks Customer.Id or a Xero ContactID). It is the source of
// truth for "has this entity already been created on the provider's books",
// which prevents duplicate creates and supplies the provider references that
// dependent syncs (invoices referencing customers) require.
type AccountingEntityMapping struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ConnectionID uuid.UUID `json:"connection_id" db:"connection_id"`
	EntityType   string    `json:"entity_type" db:"entity_type"`
	EntityID     uuid.UUID `json:"entity_id" db:"entity_id"`
	ExternalID   string    `json:"external_id" db:"external_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	// UpdatedAt is written exclusively by the sync upsert (every successful
	// push refreshes it), so it doubles as the last-synced timestamp: the
	// bulk sync skips entities whose source updated_at is not after it.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
