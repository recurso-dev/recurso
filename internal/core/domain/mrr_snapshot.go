package domain

import (
	"time"

	"github.com/google/uuid"
)

// MRRSnapshot is one active subscription's monthly-normalized MRR on a given
// date, in the subscription's own currency. It is the history the MRR waterfall
// diffs across two dates.
type MRRSnapshot struct {
	TenantID       uuid.UUID  `json:"tenant_id"`
	SubscriptionID uuid.UUID  `json:"subscription_id"`
	SnapshotDate   time.Time  `json:"snapshot_date"`
	MRRAmount      int64      `json:"mrr_amount"` // monthly-normalized, minor units, native currency
	Currency       string     `json:"currency"`
	CustomerID     *uuid.UUID `json:"customer_id,omitempty"`
	PlanID         *uuid.UUID `json:"plan_id,omitempty"`
	// EntityID is the concrete legal entity this MRR belongs to (the primary
	// entity's id when the subscription is on the primary). Stored concrete — not
	// NULL-for-primary — so per-entity reporting is a plain equality filter.
	EntityID *uuid.UUID `json:"entity_id,omitempty"`
}

// MRRWaterfall breaks the change in MRR between two dates into its movement
// components, all in the reporting currency. Contraction and Churned are
// reported as positive magnitudes. The identity holds:
//
//	EndingMRR = StartingMRR + New + Expansion + Reactivation - Contraction - Churned
type MRRWaterfall struct {
	StartDate         time.Time `json:"start_date"`
	EndDate           time.Time `json:"end_date"`
	StartingMRR       int64     `json:"starting_mrr"`
	New               int64     `json:"new"`
	Expansion         int64     `json:"expansion"`
	Contraction       int64     `json:"contraction"` // positive magnitude
	Churned           int64     `json:"churned"`     // positive magnitude
	Reactivation      int64     `json:"reactivation"`
	EndingMRR         int64     `json:"ending_mrr"`
	ReportingCurrency string    `json:"reporting_currency"`
	// NetDollarRetention and GrossDollarRetention are percentages of the starting
	// MRR retained from the existing customer base over the period (New and
	// Reactivation are excluded — they are not part of the starting cohort):
	//
	//	GDR = (Starting - Contraction - Churned) / Starting * 100   (≤ 100)
	//	NDR = (Starting + Expansion - Contraction - Churned) / Starting * 100
	//
	// Both are 0 when StartingMRR is 0 (no base to retain).
	NetDollarRetention   float64 `json:"net_dollar_retention"`
	GrossDollarRetention float64 `json:"gross_dollar_retention"`
	// HasStartHistory is false when no snapshot exists on or before StartDate —
	// then StartingMRR is 0 and every ending subscription counts as New, so the
	// UI can warn that history only begins partway through the range.
	HasStartHistory bool `json:"has_start_history"`
}
