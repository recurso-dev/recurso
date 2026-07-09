package domain

import (
	"time"

	"github.com/google/uuid"
)

// NexusType is why a tenant has sales-tax nexus in a US state.
type NexusType string

const (
	// NexusPhysical — an office, employees, or inventory in the state.
	NexusPhysical NexusType = "physical"
	// NexusVoluntary — the seller registered voluntarily.
	NexusVoluntary NexusType = "voluntary"
	// NexusEconomic — an economic-nexus threshold was crossed (Phase 2 sets this).
	NexusEconomic NexusType = "economic"
)

// TaxNexus is one US state where a tenant has declared sales-tax nexus. In
// Phase 1 these are tenant-declared (physical/voluntary); Phase 2 will add
// economic-nexus rows automatically when a state threshold is crossed.
type TaxNexus struct {
	ID            uuid.UUID  `json:"-" db:"id"`
	TenantID      uuid.UUID  `json:"-" db:"tenant_id"`
	StateCode     string     `json:"state_code" db:"state_code"`
	NexusType     NexusType  `json:"nexus_type" db:"nexus_type"`
	EstablishedAt *time.Time `json:"established_at,omitempty" db:"established_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}
