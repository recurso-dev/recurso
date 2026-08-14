package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

type mockMetricRepo struct {
	metrics map[string]*domain.BillableMetric
}

func newMockMetricRepo() *mockMetricRepo {
	return &mockMetricRepo{metrics: make(map[string]*domain.BillableMetric)}
}

func (m *mockMetricRepo) Create(_ context.Context, metric *domain.BillableMetric) error {
	m.metrics[metric.Code] = metric
	return nil
}

func (m *mockMetricRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.BillableMetric, error) {
	for _, v := range m.metrics {
		if v.ID == id && v.TenantID == tenantID {
			return v, nil
		}
	}
	return nil, service.ErrMetricNotFound
}

func (m *mockMetricRepo) GetByCode(_ context.Context, tenantID uuid.UUID, code string) (*domain.BillableMetric, error) {
	if v, ok := m.metrics[code]; ok && v.TenantID == tenantID {
		return v, nil
	}
	return nil, service.ErrMetricNotFound
}

func (m *mockMetricRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]domain.BillableMetric, error) {
	var list []domain.BillableMetric
	for _, v := range m.metrics {
		if v.TenantID == tenantID {
			list = append(list, *v)
		}
	}
	return list, nil
}

func (m *mockMetricRepo) Update(_ context.Context, _ *domain.BillableMetric) error { return nil }
func (m *mockMetricRepo) Delete(_ context.Context, _, _ uuid.UUID) error           { return nil }

// mockChargeRepoRL is a minimal ChargeRepository for the reverse-lookup test.
type mockChargeRepoRL struct {
	byMetric map[uuid.UUID][]domain.MetricPlanCharge
}

func (m *mockChargeRepoRL) ReplaceForPlan(_ context.Context, _, _ uuid.UUID, _ []domain.Charge) error {
	return nil
}
func (m *mockChargeRepoRL) ListByPlan(_ context.Context, _, _ uuid.UUID) ([]domain.Charge, error) {
	return nil, nil
}
func (m *mockChargeRepoRL) ListByMetric(_ context.Context, _, metricID uuid.UUID) ([]domain.MetricPlanCharge, error) {
	return m.byMetric[metricID], nil
}

func TestMeteringService_GetMetricCharges(t *testing.T) {
	repo := newMockMetricRepo()
	tenantID := uuid.New()
	metricID := uuid.New()
	repo.metrics["api_calls"] = &domain.BillableMetric{ID: metricID, TenantID: tenantID, Code: "api_calls", Name: "API calls"}
	planID := uuid.New()
	charges := &mockChargeRepoRL{byMetric: map[uuid.UUID][]domain.MetricPlanCharge{
		metricID: {{ChargeID: uuid.New(), PlanID: planID, PlanName: "Pro", PlanCode: "pro", PlanActive: true, ChargeModel: "per_unit"}},
	}}
	svc := service.NewMeteringService(repo, charges, nil, nil, nil)
	ctx := context.Background()

	// The metric's consuming charges come back with their plan.
	got, err := svc.GetMetricCharges(ctx, tenantID, metricID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].PlanName != "Pro" || got[0].ChargeModel != "per_unit" {
		t.Fatalf("want one charge on Pro (per_unit), got %+v", got)
	}

	// An unknown metric is a 404, not an empty list.
	if _, err := svc.GetMetricCharges(ctx, tenantID, uuid.New()); err != service.ErrMetricNotFound {
		t.Fatalf("unknown metric: want ErrMetricNotFound, got %v", err)
	}
}

func TestMeteringService_ValidateMetricInput(t *testing.T) {
	repo := newMockMetricRepo()
	svc := service.NewMeteringService(repo, nil, nil, nil, nil)
	ctx := context.Background()

	// Empty name
	_, err := svc.CreateMetric(ctx, uuid.New(), service.MetricInput{
		Name:            "",
		Code:            "valid_code",
		AggregationType: "count",
	})
	if err == nil {
		t.Error("expected error for empty metric name")
	}

	// Invalid aggregation type
	_, err = svc.CreateMetric(ctx, uuid.New(), service.MetricInput{
		Name:            "API Calls",
		Code:            "api_calls",
		AggregationType: "invalid_agg",
	})
	if err == nil {
		t.Error("expected error for invalid aggregation type")
	}

	// Valid count metric input
	created, err := svc.CreateMetric(ctx, uuid.New(), service.MetricInput{
		Name:            "API Calls",
		Code:            "api_calls",
		AggregationType: string(domain.AggregationCount),
	})
	if err != nil {
		t.Fatalf("unexpected error creating valid metric: %v", err)
	}

	if created.Code != "api_calls" || created.Name != "API Calls" {
		t.Errorf("expected created metric code api_calls, got %s", created.Code)
	}
}

func TestMeteringValidationError(t *testing.T) {
	err := service.MeteringValidationError("custom validation error")
	if err.Error() != "custom validation error" {
		t.Errorf("expected 'custom validation error', got %s", err.Error())
	}
}
