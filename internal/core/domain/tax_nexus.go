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

// NexusThreshold is one US state's economic-nexus threshold. Seeded
// UNCERTIFIED — DatasetCertified on the status response tells callers whether
// this data has passed a professional review (see docs/design-us-nexus.md).
type NexusThreshold struct {
	StateCode         string `json:"state_code" db:"state_code"`
	SalesThreshold    *int64 `json:"sales_threshold,omitempty" db:"sales_threshold"` // USD cents
	TxnThreshold      *int   `json:"txn_threshold,omitempty" db:"txn_threshold"`
	Combinator        string `json:"combinator" db:"combinator"` // "or" | "and"
	MeasurementPeriod string `json:"measurement_period" db:"measurement_period"`
	Certified         bool   `json:"certified" db:"certified"`
}

// NexusStateSales is the cumulative taxable sales + transaction count for one
// (tenant, state, calendar-year) — computed from posted invoices.
type NexusStateSales struct {
	StateCode    string `json:"state_code"`
	TaxableSales int64  `json:"taxable_sales"` // USD cents (invoice subtotals)
	TxnCount     int    `json:"txn_count"`
}

// NexusStateStatus is the per-state view returned by the nexus-status
// endpoint: declared/economic nexus, year-to-date activity, the threshold,
// and how close the tenant is to crossing it.
type NexusStateStatus struct {
	StateCode     string          `json:"state_code"`
	NexusType     NexusType       `json:"nexus_type,omitempty"` // empty = no nexus
	EstablishedAt *time.Time      `json:"established_at,omitempty"`
	TaxableSales  int64           `json:"taxable_sales"`
	TxnCount      int             `json:"txn_count"`
	Threshold     *NexusThreshold `json:"threshold,omitempty"` // nil = no state sales tax / unknown state
	// ProximityPct is how close the tenant is to the threshold (0–100+,
	// capped at 999): the max of sales/salesThreshold and txns/txnThreshold
	// for "or" states, the min for "and" states.
	ProximityPct int  `json:"proximity_pct"`
	Crossed      bool `json:"crossed"`
}
