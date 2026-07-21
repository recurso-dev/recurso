package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// MeteringService owns usage-based billing configuration
// (spec_usage_billing.md): billable-metric CRUD, plan charges, and the live
// usage-amount preview. Rating onto invoices lives in InvoiceService.

// MeteringValidationError marks invalid caller input (maps to HTTP 400).
type MeteringValidationError string

func (e MeteringValidationError) Error() string { return string(e) }

// Sentinel errors for HTTP mapping.
var (
	ErrMetricNotFound   = errors.New("billable metric not found")
	ErrMetricCodeExists = errors.New("a metric with this code already exists")
	// ErrMetricInUse maps to HTTP 409: the metric is referenced by a plan charge.
	ErrMetricInUse          = errors.New("metric is referenced by a plan charge")
	ErrMeteringPlanNotFound = errors.New("plan not found")
)

// maxMetricCodeLen matches usage_events.dimension VARCHAR(50) — the code IS
// the dimension it aggregates.
const maxMetricCodeLen = 50

const maxMetricFieldLen = 100

type MeteringService struct {
	metrics       port.BillableMetricRepository
	charges       port.ChargeRepository
	plans         port.PlanRepository
	subscriptions port.SubscriptionRepository
	usage         port.UsageRepository
	now           func() time.Time // injectable for tests
}

func NewMeteringService(
	metrics port.BillableMetricRepository,
	charges port.ChargeRepository,
	plans port.PlanRepository,
	subscriptions port.SubscriptionRepository,
	usage port.UsageRepository,
) *MeteringService {
	return &MeteringService{
		metrics:       metrics,
		charges:       charges,
		plans:         plans,
		subscriptions: subscriptions,
		usage:         usage,
		now:           func() time.Time { return time.Now().UTC() },
	}
}

// MetricInput is the caller-facing shape for creating/updating a metric.
type MetricInput struct {
	Name            string `json:"name" binding:"required"`
	Code            string `json:"code" binding:"required"`
	AggregationType string `json:"aggregation_type" binding:"required"`
	FieldName       string `json:"field_name"`
	// Expression is the per-event formula for the custom aggregation (required
	// for it, rejected for every other type).
	Expression string `json:"expression"`
}

// maxExpressionLen bounds a custom-aggregation expression. Generous for real
// formulas, tight enough to reject a pasted program.
const maxExpressionLen = 1000

func (s *MeteringService) validateMetricInput(in MetricInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return MeteringValidationError("name is required")
	}
	if in.Code == "" || len(in.Code) > maxMetricCodeLen || !featureKeyRe.MatchString(in.Code) {
		return MeteringValidationError(fmt.Sprintf(
			"code must be 1-%d chars of letters, digits, '.', '_', ':' or '-' (it doubles as the usage event dimension)", maxMetricCodeLen))
	}
	agg := domain.AggregationType(in.AggregationType)
	if !domain.ValidAggregationType(agg) {
		return MeteringValidationError("aggregation_type must be one of: count, sum, max, unique, latest, percentile, custom, weighted_sum")
	}
	field := strings.TrimSpace(in.FieldName)
	if len(field) > maxMetricFieldLen {
		return MeteringValidationError("field_name is too long")
	}
	expr := strings.TrimSpace(in.Expression)
	// expression is only for custom; reject it everywhere else so the DB
	// field_check constraint can never be violated (and config stays clean).
	if agg != domain.AggregationCustom && expr != "" {
		return MeteringValidationError("expression is only valid for the custom aggregation")
	}
	switch agg {
	case domain.AggregationUnique:
		// field_name is the event property whose distinct values are counted.
		if field == "" {
			return MeteringValidationError("field_name is required for the unique aggregation")
		}
	case domain.AggregationPercentile:
		// field_name carries the percentile as an integer 1-99 (e.g. "95").
		p, err := strconv.Atoi(field)
		if err != nil || p < 1 || p > 99 {
			return MeteringValidationError(`field_name must be the percentile 1-99 for the percentile aggregation (e.g. "95")`)
		}
	case domain.AggregationCustom:
		// expression is the sandboxed per-event formula; field_name is unused.
		if field != "" {
			return MeteringValidationError("field_name is not used by the custom aggregation; put the formula in expression")
		}
		if expr == "" {
			return MeteringValidationError("expression is required for the custom aggregation")
		}
		if len(expr) > maxExpressionLen {
			return MeteringValidationError(fmt.Sprintf("expression is too long (max %d chars)", maxExpressionLen))
		}
		// Compile now so an invalid or unsafe expression is a 400 at config time,
		// not a failed invoice later.
		if _, err := CompileCustomExpression(expr); err != nil {
			return MeteringValidationError(err.Error())
		}
	default:
		// count / sum / max / latest take no field_name.
		if field != "" {
			return MeteringValidationError("field_name is only valid for the unique and percentile aggregations")
		}
	}
	return nil
}

func (s *MeteringService) CreateMetric(ctx context.Context, tenantID uuid.UUID, in MetricInput) (*domain.BillableMetric, error) {
	if err := s.validateMetricInput(in); err != nil {
		return nil, err
	}
	now := s.now()
	m := &domain.BillableMetric{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            strings.TrimSpace(in.Name),
		Code:            in.Code,
		AggregationType: domain.AggregationType(in.AggregationType),
		FieldName:       strings.TrimSpace(in.FieldName),
		Expression:      strings.TrimSpace(in.Expression),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.metrics.Create(ctx, m); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrMetricCodeExists
		}
		return nil, err
	}
	return m, nil
}

// UpdateMetric changes a metric's name/aggregation/field. Code is immutable —
// it is the metric's identity link to already-recorded events; changing it
// would silently re-point history.
func (s *MeteringService) UpdateMetric(ctx context.Context, tenantID, id uuid.UUID, in MetricInput) (*domain.BillableMetric, error) {
	existing, err := s.metrics.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrMetricNotFound
	}
	if in.Code != "" && in.Code != existing.Code {
		return nil, MeteringValidationError("code is immutable; create a new metric instead")
	}
	in.Code = existing.Code
	if err := s.validateMetricInput(in); err != nil {
		return nil, err
	}
	existing.Name = strings.TrimSpace(in.Name)
	existing.AggregationType = domain.AggregationType(in.AggregationType)
	existing.FieldName = strings.TrimSpace(in.FieldName)
	existing.Expression = strings.TrimSpace(in.Expression)
	existing.UpdatedAt = s.now()
	if err := s.metrics.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *MeteringService) GetMetric(ctx context.Context, tenantID, id uuid.UUID) (*domain.BillableMetric, error) {
	m, err := s.metrics.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMetricNotFound
	}
	return m, nil
}

func (s *MeteringService) ListMetrics(ctx context.Context, tenantID uuid.UUID) ([]domain.BillableMetric, error) {
	metrics, err := s.metrics.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if metrics == nil {
		metrics = []domain.BillableMetric{}
	}
	return metrics, nil
}

func (s *MeteringService) DeleteMetric(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.metrics.Delete(ctx, tenantID, id)
	switch {
	case err == nil:
		return nil
	case db.IsForeignKeyViolation(err):
		return ErrMetricInUse
	case errors.Is(err, sql.ErrNoRows):
		return ErrMetricNotFound
	default:
		return err
	}
}

// ChargeInput is the caller-facing shape for one plan charge (PUT replace
// semantics over the full set, like entitlements).
type ChargeInput struct {
	MetricID    string                          `json:"metric_id" binding:"required"`
	ChargeModel string                          `json:"charge_model" binding:"required"`
	Amounts     map[string]domain.ChargeAmounts `json:"amounts" binding:"required"`
	// FilterKey/Filters (A4) price distinct values of one event property
	// differently. Empty FilterKey = an ordinary (unfiltered) charge.
	FilterKey    string              `json:"filter_key"`
	Filters      []ChargeFilterInput `json:"filters"`
	PayInAdvance bool                `json:"pay_in_advance"`
	HSNCode      string              `json:"hsn_code"`
}

// ChargeFilterInput is one dimensional-pricing band: events whose FilterKey
// property equals Value bill at these per-currency amounts.
type ChargeFilterInput struct {
	Value   string                          `json:"value"`
	Amounts map[string]domain.ChargeAmounts `json:"amounts"`
}

// SetPlanCharges validates and fully replaces a plan's charge set.
func (s *MeteringService) SetPlanCharges(ctx context.Context, tenantID, planID uuid.UUID, inputs []ChargeInput) ([]domain.Charge, error) {
	plan, err := s.plans.GetByID(ctx, planID)
	if err != nil || plan == nil || plan.TenantID != tenantID {
		return nil, ErrMeteringPlanNotFound
	}

	now := s.now()
	seen := map[uuid.UUID]bool{}
	charges := make([]domain.Charge, 0, len(inputs))
	for i, in := range inputs {
		metric, model, normalized, err := s.resolveChargeInput(ctx, tenantID, i, in)
		if err != nil {
			return nil, err
		}
		if seen[metric.ID] {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: duplicate charge for metric %s", i, metric.Code))
		}
		seen[metric.ID] = true

		filterKey, filters, err := resolveChargeFilters(model, i, in)
		if err != nil {
			return nil, err
		}

		charges = append(charges, domain.Charge{
			ID:           uuid.New(),
			TenantID:     tenantID,
			PlanID:       planID,
			MetricID:     metric.ID,
			ChargeModel:  model,
			Amounts:      normalized,
			FilterKey:    filterKey,
			Filters:      filters,
			PayInAdvance: in.PayInAdvance,
			HSNCode:      strings.TrimSpace(in.HSNCode),
			CreatedAt:    now,
			UpdatedAt:    now,
			Metric:       metric,
		})
	}

	if err := s.charges.ReplaceForPlan(ctx, tenantID, planID, charges); err != nil {
		return nil, err
	}
	return charges, nil
}

// resolveChargeInput validates one proposed charge — metric exists for the
// tenant, charge model is supported, and every currency's amounts rate cleanly
// against a probe quantity (the same path invoice generation uses, so a config
// that validates here cannot fail at rating time). It returns the loaded
// metric, parsed model, and normalized (upper-cased currency) amounts. It does
// NOT check for duplicate metrics — the caller owns that. Shared by
// SetPlanCharges and SimulateCharges so a config that simulates cleanly also
// saves cleanly. idx is only for error messages.
func (s *MeteringService) resolveChargeInput(ctx context.Context, tenantID uuid.UUID, idx int, in ChargeInput) (*domain.BillableMetric, domain.ChargeModel, map[string]domain.ChargeAmounts, error) {
	metricID, err := uuid.Parse(in.MetricID)
	if err != nil {
		return nil, "", nil, MeteringValidationError(fmt.Sprintf("charges[%d]: invalid metric_id", idx))
	}
	metric, err := s.metrics.GetByID(ctx, tenantID, metricID)
	if err != nil {
		return nil, "", nil, err
	}
	if metric == nil {
		return nil, "", nil, MeteringValidationError(fmt.Sprintf("charges[%d]: metric not found", idx))
	}
	model := domain.ChargeModel(in.ChargeModel)
	if !domain.ValidChargeModel(model) {
		return nil, "", nil, MeteringValidationError(fmt.Sprintf("charges[%d]: charge_model must be one of: per_unit, graduated, volume, package, percentage, graduated_percentage, dynamic", idx))
	}
	if in.PayInAdvance && !domain.PayInAdvanceEligible(model) {
		return nil, "", nil, MeteringValidationError(fmt.Sprintf(
			"charges[%d]: pay_in_advance requires a non-cumulative model (per_unit, percentage, or dynamic), not %q", idx, model))
	}
	// The custom and weighted_sum aggregations compute the quantity in Go (a
	// per-event expression sum, a time-weighted average), so they are
	// incompatible with two features that compute the quantity a different way:
	// dimensional filters (which aggregate subsets in SQL) and the dynamic charge
	// model (which sums per-event dynamic_amount and ignores the aggregation).
	// Reject here rather than fail at invoice time.
	if domain.FractionalAggregation(metric.AggregationType) {
		if strings.TrimSpace(in.FilterKey) != "" {
			return nil, "", nil, MeteringValidationError(fmt.Sprintf(
				"charges[%d]: charge filters are not supported with the %s aggregation", idx, metric.AggregationType))
		}
		if model == domain.ChargeDynamic {
			return nil, "", nil, MeteringValidationError(fmt.Sprintf(
				"charges[%d]: the dynamic charge model is incompatible with the %s aggregation", idx, metric.AggregationType))
		}
	}
	// weighted_sum is a time-weighted average over the whole period, so it can
	// only be rated at period close — pay-in-advance (per-event capture) is
	// meaningless for it.
	if metric.AggregationType == domain.AggregationWeightedSum && in.PayInAdvance {
		return nil, "", nil, MeteringValidationError(fmt.Sprintf(
			"charges[%d]: pay_in_advance is not supported with the weighted_sum aggregation (period-close only)", idx))
	}
	normalized, err := normalizeChargeAmounts(model, in.Amounts, idx, "amounts")
	if err != nil {
		return nil, "", nil, err
	}
	return metric, model, normalized, nil
}

// normalizeChargeAmounts upper-cases each currency, checks it is a 3-letter ISO
// code, and validates the pricing by rating a probe quantity (the same path
// invoice generation uses). label names the field for error messages.
func normalizeChargeAmounts(model domain.ChargeModel, amounts map[string]domain.ChargeAmounts, idx int, label string) (map[string]domain.ChargeAmounts, error) {
	if len(amounts) == 0 {
		return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: %s must define at least one currency", idx, label))
	}
	normalized := make(map[string]domain.ChargeAmounts, len(amounts))
	for currency, a := range amounts {
		cur := strings.ToUpper(strings.TrimSpace(currency))
		if len(cur) != 3 {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: %q is not an ISO currency code", idx, currency))
		}
		if _, err := RateCharge(model, a, 1); err != nil {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d].%s[%s]: %v", idx, label, cur, err))
		}
		normalized[cur] = a
	}
	return normalized, nil
}

// resolveChargeFilters validates a charge's dimensional-pricing filters (A4):
// filters require a filter_key, each value must be unique and non-empty, and
// each value's amounts must rate cleanly for the model. Returns the trimmed
// key and normalized filters (empty when the charge is unfiltered).
func resolveChargeFilters(model domain.ChargeModel, idx int, in ChargeInput) (string, []domain.ChargeFilterValue, error) {
	filterKey := strings.TrimSpace(in.FilterKey)
	if filterKey == "" {
		if len(in.Filters) > 0 {
			return "", nil, MeteringValidationError(fmt.Sprintf("charges[%d]: filters require a filter_key", idx))
		}
		return "", nil, nil
	}
	if len(in.Filters) == 0 {
		return "", nil, MeteringValidationError(fmt.Sprintf("charges[%d]: filter_key %q set but no filters given", idx, filterKey))
	}
	seen := map[string]bool{}
	filters := make([]domain.ChargeFilterValue, 0, len(in.Filters))
	for _, f := range in.Filters {
		v := strings.TrimSpace(f.Value)
		if v == "" {
			return "", nil, MeteringValidationError(fmt.Sprintf("charges[%d]: a filter value must not be empty", idx))
		}
		if seen[v] {
			return "", nil, MeteringValidationError(fmt.Sprintf("charges[%d]: duplicate filter value %q", idx, v))
		}
		seen[v] = true
		norm, err := normalizeChargeAmounts(model, f.Amounts, idx, fmt.Sprintf("filters[%s].amounts", v))
		if err != nil {
			return "", nil, err
		}
		filters = append(filters, domain.ChargeFilterValue{Value: v, Amounts: norm})
	}
	return filterKey, filters, nil
}

// meteredQuantity picks the aggregate a charge prices for [start, end), as an
// exact rational so a fractional aggregation is priced without pre-rounding:
//   - a `custom` metric evaluates its expression per event and sums the results
//     (may be fractional);
//   - a `dynamic` charge bills the Σ per-event dynamic_amount;
//   - every other charge uses the metric's configured integer aggregation.
//
// The integer paths return a whole-number rational (SetInt64), so callers can
// treat every case uniformly and rate via RateChargeRat. Both the live usage
// preview and invoice generation route through this so they agree. The caller
// guarantees ch.Metric is non-nil.
func meteredQuantity(ctx context.Context, repo port.UsageRepository, subID uuid.UUID, ch domain.Charge, start, end time.Time) (*big.Rat, error) {
	if ch.Metric.AggregationType == domain.AggregationCustom {
		ev, err := CompileCustomExpression(ch.Metric.Expression)
		if err != nil {
			return nil, err
		}
		return AggregateCustom(ctx, repo, ev, subID, ch.Metric.Code, start, end)
	}
	if ch.Metric.AggregationType == domain.AggregationWeightedSum {
		return AggregateWeightedSum(ctx, repo, subID, ch.Metric.Code, start, end)
	}
	if ch.ChargeModel == domain.ChargeDynamic {
		n, err := repo.SumDynamicAmount(ctx, subID, ch.Metric.Code, start, end)
		if err != nil {
			return nil, err
		}
		return new(big.Rat).SetInt64(n), nil
	}
	n, err := repo.AggregateForMetric(ctx, subID, *ch.Metric, start, end)
	if err != nil {
		return nil, err
	}
	return new(big.Rat).SetInt64(n), nil
}

func (s *MeteringService) GetPlanCharges(ctx context.Context, tenantID, planID uuid.UUID) ([]domain.Charge, error) {
	plan, err := s.plans.GetByID(ctx, planID)
	if err != nil || plan == nil || plan.TenantID != tenantID {
		return nil, ErrMeteringPlanNotFound
	}
	charges, err := s.charges.ListByPlan(ctx, tenantID, planID)
	if err != nil {
		return nil, err
	}
	if charges == nil {
		charges = []domain.Charge{}
	}
	return charges, nil
}

// UsageAmountItem is one charge's live preview: the current period's
// aggregated quantity and what it would rate to if invoiced now.
type UsageAmountItem struct {
	MetricCode      string `json:"metric_code"`
	MetricName      string `json:"metric_name"`
	AggregationType string `json:"aggregation_type"`
	ChargeModel     string `json:"charge_model"`
	Quantity        int64  `json:"quantity"`
	Amount          int64  `json:"amount"`
}

// UsageAmount is the GET /v1/subscriptions/:id/usage-amount response.
type UsageAmount struct {
	SubscriptionID     uuid.UUID         `json:"subscription_id"`
	Currency           string            `json:"currency"`
	CurrentPeriodStart time.Time         `json:"current_period_start"`
	AsOf               time.Time         `json:"as_of"`
	Charges            []UsageAmountItem `json:"charges"`
	TotalAmount        int64             `json:"total_amount"`
	// Commitment projection (Lago-parity B2): what the true-up line would
	// be if the period closed now — commitment minus (flat fee + usage),
	// floored at zero. Both zero when no commitment is set.
	CommitmentAmount int64 `json:"commitment_amount,omitempty"`
	ProjectedTrueUp  int64 `json:"projected_true_up,omitempty"`
}

// GetUsageAmount previews what the subscription's current-period usage
// would rate to if invoiced now: per charge, aggregate [period_start, now)
// and price it. Charges without pricing in the subscription currency are
// skipped, mirroring invoice generation.
func (s *MeteringService) GetUsageAmount(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*UsageAmount, error) {
	sub, err := s.subscriptions.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil || sub.TenantID != tenantID {
		return nil, ErrUsageSubscriptionNotFound
	}

	plan, err := s.plans.GetByID(ctx, sub.PlanID)
	if err != nil || plan == nil || len(plan.Prices) == 0 {
		return nil, fmt.Errorf("plan unavailable for subscription %s", subscriptionID)
	}
	currency := strings.ToUpper(plan.Prices[0].Currency)

	charges, err := s.charges.ListByPlan(ctx, tenantID, sub.PlanID)
	if err != nil {
		return nil, err
	}

	asOf := s.now()
	out := &UsageAmount{
		SubscriptionID:     sub.ID,
		Currency:           currency,
		CurrentPeriodStart: sub.CurrentPeriodStart,
		AsOf:               asOf,
		Charges:            []UsageAmountItem{},
	}
	for _, ch := range charges {
		if ch.Metric == nil {
			continue
		}
		amounts, ok := ch.Amounts[currency]
		if !ok {
			continue
		}
		qtyRat, err := meteredQuantity(ctx, s.usage, sub.ID, ch, sub.CurrentPeriodStart, asOf)
		if err != nil {
			return nil, err
		}
		amount, err := RateChargeRat(ch.ChargeModel, amounts, qtyRat)
		if err != nil {
			return nil, err
		}
		out.Charges = append(out.Charges, UsageAmountItem{
			MetricCode:      ch.Metric.Code,
			MetricName:      ch.Metric.Name,
			AggregationType: string(ch.Metric.AggregationType),
			ChargeModel:     string(ch.ChargeModel),
			Quantity:        roundRatHalfUp(qtyRat),
			Amount:          amount,
		})
		out.TotalAmount += amount
	}

	// Commitment projection: flat fee + usage vs the committed floor (B2).
	if sub.CommitmentAmount > 0 {
		out.CommitmentAmount = sub.CommitmentAmount
		projected := plan.Prices[0].Amount + out.TotalAmount
		if projected < sub.CommitmentAmount {
			out.ProjectedTrueUp = sub.CommitmentAmount - projected
		}
	}
	return out, nil
}
