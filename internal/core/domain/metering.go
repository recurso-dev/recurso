package domain

import (
	"time"

	"github.com/google/uuid"
)

// Metering & usage-based billing v1 (spec_usage_billing.md).
//
// A BillableMetric is a tenant-defined meter over usage events: its Code
// doubles as the event Dimension it aggregates, so existing events (and the
// entitlement feature_key linkage) keep working unchanged. A Charge attaches
// a non-flat price for a metric to a plan; flat subscription fees stay on
// Price — a plan holding both is "hybrid" (flat fee in advance + usage in
// arrears).

// AggregationType is how a metric reduces a period's events to one quantity.
type AggregationType string

const (
	// AggregationCount counts events (quantity ignored).
	AggregationCount AggregationType = "count"
	// AggregationSum sums event quantities.
	AggregationSum AggregationType = "sum"
	// AggregationMax takes the largest single-event quantity.
	AggregationMax AggregationType = "max"
	// AggregationUnique counts distinct values of properties[FieldName].
	AggregationUnique AggregationType = "unique"
	// AggregationLatest takes the most recent event's quantity in the period
	// (last by timestamp). Uses Quantity; FieldName is not used.
	AggregationLatest AggregationType = "latest"
	// AggregationPercentile takes the p-th percentile of event quantities in
	// the period (e.g. p95/p99 latency SLO billing). FieldName carries the
	// percentile as an integer 1-99 (e.g. "95").
	AggregationPercentile AggregationType = "percentile"
)

// ValidAggregationType reports whether t is a supported aggregation.
func ValidAggregationType(t AggregationType) bool {
	switch t {
	case AggregationCount, AggregationSum, AggregationMax, AggregationUnique,
		AggregationLatest, AggregationPercentile:
		return true
	}
	return false
}

// BillableMetric is a tenant-defined meter. Code is unique per tenant and
// equals the UsageEvent.Dimension it aggregates.
type BillableMetric struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	Name            string          `json:"name"`
	Code            string          `json:"code"`
	AggregationType AggregationType `json:"aggregation_type"`
	// FieldName is the event property whose distinct values are counted for
	// the unique aggregation. Empty for count/sum/max (they use Quantity).
	FieldName string    `json:"field_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChargeModel is how an aggregated quantity is priced.
type ChargeModel string

const (
	// ChargePerUnit prices every unit at UnitAmount.
	ChargePerUnit ChargeModel = "per_unit"
	// ChargeGraduated prices each tier's units at that tier's rate
	// (0-100 @ 1.00, 101+ @ 0.50 -> 150 units = 100x1.00 + 50x0.50).
	ChargeGraduated ChargeModel = "graduated"
	// ChargeVolume prices the WHOLE quantity at the single tier it reaches.
	ChargeVolume ChargeModel = "volume"
	// ChargePackage prices ceil(quantity/PackageSize) bundles at PackageAmount.
	ChargePackage ChargeModel = "package"
)

// ValidChargeModel reports whether m is a supported charge model.
func ValidChargeModel(m ChargeModel) bool {
	switch m {
	case ChargePerUnit, ChargeGraduated, ChargeVolume, ChargePackage:
		return true
	}
	return false
}

// ChargeTier is one band of a graduated or volume charge. Tiers are ordered
// by UpTo ascending; the last tier's UpTo is nil (infinity).
type ChargeTier struct {
	// UpTo is the inclusive upper unit bound of the tier; nil = no bound.
	UpTo *int64 `json:"up_to"`
	// UnitAmount is the per-unit rate as a decimal string in MAJOR currency
	// units (e.g. "0.0035") — D1: sub-minor-unit rates are first-class.
	UnitAmount string `json:"unit_amount"`
	// FlatAmount (minor units) is added once when any unit lands in the tier.
	FlatAmount int64 `json:"flat_amount,omitempty"`
}

// ChargeAmounts is a charge's pricing properties for ONE currency.
type ChargeAmounts struct {
	// UnitAmount (per_unit): per-unit rate as a decimal string in MAJOR
	// currency units, e.g. "0.0035" rupees per call.
	UnitAmount string `json:"unit_amount,omitempty"`
	// PackageAmount (package): price in minor units per bundle.
	PackageAmount int64 `json:"package_amount,omitempty"`
	// PackageSize (package): units per bundle; partial bundles round UP.
	PackageSize int64 `json:"package_size,omitempty"`
	// Tiers (graduated/volume).
	Tiers []ChargeTier `json:"tiers,omitempty"`
}

// Charge attaches usage pricing for a metric to a plan. Amounts is keyed by
// ISO currency code (mirroring per-currency Price rows); the invoice
// currency selects the entry at rating time.
type Charge struct {
	ID          uuid.UUID                `json:"id"`
	TenantID    uuid.UUID                `json:"tenant_id"`
	PlanID      uuid.UUID                `json:"plan_id"`
	MetricID    uuid.UUID                `json:"metric_id"`
	ChargeModel ChargeModel              `json:"charge_model"`
	Amounts     map[string]ChargeAmounts `json:"amounts"`
	// HSNCode taxes this charge's invoice lines; empty falls back to the
	// plan HSN, then the tenant SAC (the existing resolution chain).
	HSNCode   string    `json:"hsn_code,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Metric is the joined billable metric (read paths), when loaded.
	Metric *BillableMetric `json:"metric,omitempty"`
}

// UsageRating is the double-billing guard: one row per (subscription,
// charge, period_start) window ever rated onto an invoice. The unique
// constraint makes rating idempotent — a retried invoice generation for an
// already-rated window produces no metered lines.
type UsageRating struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	SubscriptionID uuid.UUID `json:"subscription_id"`
	ChargeID       uuid.UUID `json:"charge_id"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	InvoiceID      uuid.UUID `json:"invoice_id"`
	// Quantity is the aggregated units billed; Amount the priced result
	// (minor units) — kept for audit even though the invoice line carries them.
	Quantity  int64     `json:"quantity"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
