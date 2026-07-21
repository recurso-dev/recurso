package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
}

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
		return MeteringValidationError("aggregation_type must be one of: count, sum, max, unique")
	}
	switch {
	case agg == domain.AggregationUnique && strings.TrimSpace(in.FieldName) == "":
		return MeteringValidationError("field_name is required for the unique aggregation")
	case agg != domain.AggregationUnique && in.FieldName != "":
		return MeteringValidationError("field_name is only valid for the unique aggregation")
	case len(in.FieldName) > maxMetricFieldLen:
		return MeteringValidationError("field_name is too long")
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
	HSNCode     string                          `json:"hsn_code"`
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
		metricID, err := uuid.Parse(in.MetricID)
		if err != nil {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: invalid metric_id", i))
		}
		metric, err := s.metrics.GetByID(ctx, tenantID, metricID)
		if err != nil {
			return nil, err
		}
		if metric == nil {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: metric not found", i))
		}
		if seen[metricID] {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: duplicate charge for metric %s", i, metric.Code))
		}
		seen[metricID] = true

		model := domain.ChargeModel(in.ChargeModel)
		if !domain.ValidChargeModel(model) {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: charge_model must be one of: per_unit, graduated, volume, package, percentage", i))
		}
		if len(in.Amounts) == 0 {
			return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: amounts must define at least one currency", i))
		}

		// Validate every currency's pricing by rating a probe quantity —
		// the same code path invoice generation uses, so a config that
		// passes here cannot fail at rating time.
		normalized := make(map[string]domain.ChargeAmounts, len(in.Amounts))
		for currency, amounts := range in.Amounts {
			cur := strings.ToUpper(strings.TrimSpace(currency))
			if len(cur) != 3 {
				return nil, MeteringValidationError(fmt.Sprintf("charges[%d]: %q is not an ISO currency code", i, currency))
			}
			if _, err := RateCharge(model, amounts, 1); err != nil {
				return nil, MeteringValidationError(fmt.Sprintf("charges[%d].amounts[%s]: %v", i, cur, err))
			}
			normalized[cur] = amounts
		}

		charges = append(charges, domain.Charge{
			ID:          uuid.New(),
			TenantID:    tenantID,
			PlanID:      planID,
			MetricID:    metricID,
			ChargeModel: model,
			Amounts:     normalized,
			HSNCode:     strings.TrimSpace(in.HSNCode),
			CreatedAt:   now,
			UpdatedAt:   now,
			Metric:      metric,
		})
	}

	if err := s.charges.ReplaceForPlan(ctx, tenantID, planID, charges); err != nil {
		return nil, err
	}
	return charges, nil
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
		qty, err := s.usage.AggregateForMetric(ctx, sub.ID, *ch.Metric, sub.CurrentPeriodStart, asOf)
		if err != nil {
			return nil, err
		}
		amount, err := RateCharge(ch.ChargeModel, amounts, qty)
		if err != nil {
			return nil, err
		}
		out.Charges = append(out.Charges, UsageAmountItem{
			MetricCode:      ch.Metric.Code,
			MetricName:      ch.Metric.Name,
			AggregationType: string(ch.Metric.AggregationType),
			ChargeModel:     string(ch.ChargeModel),
			Quantity:        qty,
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
