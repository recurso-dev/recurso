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
