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

// NexusAlertLevel is the stage of an economic-nexus threshold alert (Track D · D1).
type NexusAlertLevel string

const (
	// NexusAlertApproaching — activity reached the approaching band (default 80%)
	// of a state's economic-nexus threshold but has not crossed it.
	NexusAlertApproaching NexusAlertLevel = "approaching"
	// NexusAlertCrossed — the threshold was crossed and economic nexus established.
	NexusAlertCrossed NexusAlertLevel = "crossed"
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

// USLiabilityStateLine is one state's US sales-tax liability for a period
// (Track D · D3): what was sold and how much tax was collected, so the tenant
// can file. All money is USD cents.
type USLiabilityStateLine struct {
	StateCode string `json:"state_code"`
	// GrossSales is the sum of invoice subtotals into the state.
	GrossSales int64 `json:"gross_sales"`
	// TaxableSales is the subtotal of invoices that collected tax (tax_amount > 0).
	// ExemptSales is the subtotal invoiced under a customer exemption
	// (tax_type "sales_tax_exempt"). NonTaxableSales is the remaining zero-tax
	// sales — no-nexus or below-threshold. The three partition GrossSales.
	TaxableSales    int64 `json:"taxable_sales"`
	ExemptSales     int64 `json:"exempt_sales"`
	NonTaxableSales int64 `json:"non_taxable_sales"`
	// TaxCollected is the sum of invoice tax amounts into the state.
	TaxCollected int64 `json:"tax_collected"`
	InvoiceCount int   `json:"invoice_count"`
	// HasNexus reflects whether the tenant has declared or economic nexus in
	// this state — a state with tax collected but no nexus (or vice-versa) is
	// worth the tenant's attention.
	HasNexus  bool      `json:"has_nexus"`
	NexusType NexusType `json:"nexus_type,omitempty"`
}

// USLiabilityReport is the per-state US sales-tax liability for a filing period.
type USLiabilityReport struct {
	FromDate          string                 `json:"from_date"` // inclusive, YYYY-MM-DD (UTC)
	ToDate            string                 `json:"to_date"`   // exclusive, YYYY-MM-DD (UTC)
	Currency          string                 `json:"currency"`  // always "USD"
	States            []USLiabilityStateLine `json:"states"`
	TotalGrossSales   int64                  `json:"total_gross_sales"`
	TotalTaxCollected int64                  `json:"total_tax_collected"`
}
