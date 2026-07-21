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
	// AggregationCustom evaluates the metric's Expression against each event and
	// SUMS the results into the period quantity (which may be fractional). The
	// expression is authored per metric and reads `quantity` and `properties`;
	// it is sandboxed (see service.CompileCustomExpression). Expression is
	// required for this type and empty for all others.
	AggregationCustom AggregationType = "custom"
)

// ValidAggregationType reports whether t is a supported aggregation.
func ValidAggregationType(t AggregationType) bool {
	switch t {
	case AggregationCount, AggregationSum, AggregationMax, AggregationUnique,
		AggregationLatest, AggregationPercentile, AggregationCustom:
		return true
	}
	return false
}

// FractionalAggregation reports whether an aggregation can produce a
// non-integer period quantity, so billing must rate it through the exact-
// rational path (RateChargeRat) rather than pre-rounding. `custom` sums
// arbitrary per-event expression results; `weighted_sum` is a time-weighted
// average. Every other aggregation yields a whole count/sum/percentile.
func FractionalAggregation(t AggregationType) bool {
	return t == AggregationCustom
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
	// the unique aggregation, or the percentile 1-99 for the percentile
	// aggregation. Empty for count/sum/max/latest/custom.
	FieldName string `json:"field_name,omitempty"`
	// Expression is the sandboxed per-event formula for the custom aggregation
	// (e.g. "quantity * properties.multiplier"). Required for AggregationCustom,
	// empty for every other type. It reads `quantity` (the event quantity) and
	// `properties` (its numeric properties); results are summed over the period.
	Expression string    `json:"expression,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
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
	// ChargePercentage prices a percentage of the aggregated monetary base
	// (a sum in minor units — e.g. payment volume), plus an optional flat fee,
	// with an optional free-units allowance deducted before the rate applies
	// and optional min/max clamps on the line. Unlike the other models, its
	// quantity is a money amount in minor units, not a unit count.
	ChargePercentage ChargeModel = "percentage"
	// ChargeGraduatedPercentage prices each band of the monetary base at that
	// band's percentage rate (Tiers[i].Rate) plus that band's flat amount —
	// the percentage analogue of graduated. Like percentage, its quantity is a
	// money amount in minor units, and the tier UpTo bounds band that base.
	ChargeGraduatedPercentage ChargeModel = "graduated_percentage"
	// ChargeDynamic bills the sum of the per-event UsageEvent.DynamicAmount for
	// the period — the caller supplies the exact price with each event, so the
	// line is that sum with no rate applied. Its quantity is the aggregated
	// dynamic amount in minor units.
	ChargeDynamic ChargeModel = "dynamic"
)

// PayInAdvanceEligible reports whether a charge model can be billed
// pay-in-advance. Only non-cumulative models qualify: a single event has a
// well-defined price under per_unit / percentage / dynamic, but graduated /
// volume / graduated_percentage / package price depend on the whole period's
// cumulative quantity, so they can only be rated at period close.
func PayInAdvanceEligible(m ChargeModel) bool {
	switch m {
	case ChargePerUnit, ChargePercentage, ChargeDynamic:
		return true
	}
	return false
}

// ProgressiveBillingEligible reports whether a charge model can be billed
// progressively (interim invoices via a billed-amount watermark). The watermark
// requires the fee to be MONOTONIC non-decreasing in the cumulative quantity —
// every model qualifies EXCEPT `volume`, which re-prices the whole quantity at a
// (cheaper) tier as usage grows, so its fee can DROP. On a progressive
// subscription, a volume charge falls back to classic period-close billing.
func ProgressiveBillingEligible(m ChargeModel) bool {
	return m != ChargeVolume && ValidChargeModel(m)
}

// ValidChargeModel reports whether m is a supported charge model.
func ValidChargeModel(m ChargeModel) bool {
	switch m {
	case ChargePerUnit, ChargeGraduated, ChargeVolume, ChargePackage,
		ChargePercentage, ChargeGraduatedPercentage, ChargeDynamic:
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
	// Used by graduated/volume.
	UnitAmount string `json:"unit_amount"`
	// Rate is the percentage applied to this band of the monetary base, a
	// decimal string of PERCENT (e.g. "2.5"). Used by graduated_percentage.
	Rate string `json:"rate,omitempty"`
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

	// Rate (percentage): the percentage applied to the base, as a decimal
	// string of PERCENT, e.g. "2.5" = 2.5%. Exact big.Rat path like UnitAmount.
	Rate string `json:"rate,omitempty"`
	// FixedAmount (percentage): a flat fee in minor units added to the line
	// after the percentage — e.g. a per-invoice processing fee.
	FixedAmount int64 `json:"fixed_amount,omitempty"`
	// FreeUnits (percentage): units of the base exempt from the percentage,
	// deducted from the base before the rate applies.
	FreeUnits int64 `json:"free_units,omitempty"`
	// MinAmount (percentage): a floor on the line in minor units, applied only
	// when there is usage. 0 means no floor.
	MinAmount int64 `json:"min_amount,omitempty"`
	// MaxAmount (percentage): a cap on the line in minor units. 0 means no cap.
	MaxAmount int64 `json:"max_amount,omitempty"`
}

// ChargeFilterValue is one dimensional-pricing band: events whose FilterKey
// property equals Value are priced by these per-currency Amounts (A4).
type ChargeFilterValue struct {
	Value   string                   `json:"value"`
	Amounts map[string]ChargeAmounts `json:"amounts"`
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
	// FilterKey (A4) is the event property this charge prices dimensionally;
	// empty means the charge is not filtered (rated the classic way). Filters
	// lists each priced value; events matching none use Amounts (the default).
	FilterKey string              `json:"filter_key,omitempty"`
	Filters   []ChargeFilterValue `json:"filters,omitempty"`
	// PayInAdvance rates this charge per usage event at ingestion time
	// (captured as an unbilled charge) instead of aggregating at period close.
	// Only non-cumulative models may set it (see PayInAdvanceEligible).
	PayInAdvance bool `json:"pay_in_advance,omitempty"`
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
