package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Fakes for usage-alert tests (Lago-parity B3) ---

type fakeAlertRepo struct {
	port.UsageAlertRepository
	alerts []domain.UsageAlert
	fired  map[uuid.UUID]time.Time // alertID -> claimed period
}

func (f *fakeAlertRepo) ListAll(ctx context.Context, limit int) ([]domain.UsageAlert, error) {
	return f.alerts, nil
}

func (f *fakeAlertRepo) MarkFired(ctx context.Context, id uuid.UUID, periodStart time.Time) (bool, error) {
	if f.fired == nil {
		f.fired = map[uuid.UUID]time.Time{}
	}
	if prev, ok := f.fired[id]; ok && prev.Equal(periodStart) {
		return false, nil // already fired this period
	}
	f.fired[id] = periodStart
	return true, nil
}

type fakeAlertMetricRepo struct {
	port.BillableMetricRepository
	metric *domain.BillableMetric
}

func (f *fakeAlertMetricRepo) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.BillableMetric, error) {
	if f.metric != nil && f.metric.Code == code {
		return f.metric, nil
	}
	return nil, nil
}

type fakeAlertEntitlements struct{ limit *int64 }

func (f *fakeAlertEntitlements) CheckFeature(ctx context.Context, tenantID, customerID uuid.UUID, featureKey string) (*domain.EntitlementCheck, error) {
	return &domain.EntitlementCheck{FeatureKey: featureKey, Granted: f.limit != nil, LimitValue: f.limit}, nil
}

type fakeAlertPublisher struct{ events []PublishEventInput }

func (f *fakeAlertPublisher) PublishEvent(ctx context.Context, input PublishEventInput) (*domain.Event, error) {
	f.events = append(f.events, input)
	return &domain.Event{ID: uuid.New()}, nil
}

func alertFixture(qty int64, threshold int64, tt domain.UsageAlertThresholdType) (*UsageAlertService, *fakeAlertRepo, *fakeAlertPublisher, *domain.UsageAlert) {
	tenantID := uuid.New()
	sub := &domain.Subscription{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(), PlanID: uuid.New(),
		Status:             domain.SubscriptionStatusActive,
		CurrentPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:   time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	alert := domain.UsageAlert{
		ID: uuid.New(), TenantID: tenantID, SubscriptionID: sub.ID,
		MetricCode: "api_calls", ThresholdType: tt, Threshold: threshold,
	}
	repo := &fakeAlertRepo{alerts: []domain.UsageAlert{alert}}
	metricRepo := &fakeAlertMetricRepo{metric: &domain.BillableMetric{
		ID: uuid.New(), Code: "api_calls", AggregationType: domain.AggregationSum,
	}}
	limit := int64(10000)
	svc := NewUsageAlertService(
		repo,
		&mandateMeteredSubRepo{sub: sub},
		metricRepo,
		&mockUsageRepoForMeter{qtyByMetricCode: map[string]int64{"api_calls": qty}},
		&fakeWalletCustomerRepo{customer: &domain.Customer{ID: sub.CustomerID, Email: "c@x.com"}},
		&fakeAlertEntitlements{limit: &limit},
	)
	pub := &fakeAlertPublisher{}
	svc.SetEventPublisher(pub)
	svc.now = func() time.Time { return time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC) }
	return svc, repo, pub, &repo.alerts[0]
}

func TestUsageAlertFiresOnceAtQuantityThreshold(t *testing.T) {
	svc, repo, pub, alert := alertFixture(8500, 8000, domain.AlertThresholdQuantity)

	fired, err := svc.EvaluateAlerts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fired != 1 || len(pub.events) != 1 {
		t.Fatalf("fired/events = %d/%d, want 1/1", fired, len(pub.events))
	}
	ev := pub.events[0]
	if ev.Type != domain.EventUsageAlertTriggered || ev.Data["quantity"].(int64) != 8500 {
		t.Fatalf("event = %+v, want usage.alert.triggered with quantity 8500", ev)
	}

	// Second sweep in the same period: dedup keeps it silent. The fake repo
	// claims per (alert, period); refresh the in-memory alert like a reload.
	repo.alerts[0].LastFiredPeriodStart = func() *time.Time { ts := repo.fired[alert.ID]; return &ts }()
	fired, err = svc.EvaluateAlerts(context.Background())
	if err != nil || fired != 0 || len(pub.events) != 1 {
		t.Fatalf("re-sweep fired/events = %d/%d (err %v), want 0/1", fired, len(pub.events), err)
	}
}

func TestUsageAlertBelowThresholdSilent(t *testing.T) {
	svc, _, pub, _ := alertFixture(7999, 8000, domain.AlertThresholdQuantity)
	fired, err := svc.EvaluateAlerts(context.Background())
	if err != nil || fired != 0 || len(pub.events) != 0 {
		t.Fatalf("fired/events = %d/%d (err %v), want silence below threshold", fired, len(pub.events), err)
	}
}

func TestUsageAlertPercentOfLimit(t *testing.T) {
	// limit 10000, threshold 80% -> fires at 8000.
	svc, _, pub, _ := alertFixture(8000, 80, domain.AlertThresholdPercentOfLimit)
	fired, err := svc.EvaluateAlerts(context.Background())
	if err != nil || fired != 1 || len(pub.events) != 1 {
		t.Fatalf("fired/events = %d/%d (err %v), want 1/1 at 80%% of limit", fired, len(pub.events), err)
	}
}

func TestUsageAlertPercentWithoutLimitSilent(t *testing.T) {
	svc, _, pub, _ := alertFixture(999999, 80, domain.AlertThresholdPercentOfLimit)
	// Remove the limit.
	svc.entitlements = &fakeAlertEntitlements{limit: nil}
	fired, err := svc.EvaluateAlerts(context.Background())
	if err != nil || fired != 0 || len(pub.events) != 0 {
		t.Fatalf("fired/events = %d/%d (err %v), want silence without an entitlement limit", fired, len(pub.events), err)
	}
}

func TestUsageAlertNewPeriodFiresAgain(t *testing.T) {
	svc, repo, pub, alert := alertFixture(8500, 8000, domain.AlertThresholdQuantity)
	// Fired last period; the new period start differs, so it fires again.
	last := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo.alerts[0].LastFiredPeriodStart = &last

	fired, err := svc.EvaluateAlerts(context.Background())
	if err != nil || fired != 1 || len(pub.events) != 1 {
		t.Fatalf("fired/events = %d/%d (err %v), want 1/1 in a new period", fired, len(pub.events), err)
	}
	if !repo.fired[alert.ID].Equal(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("claimed period = %v, want the current period start", repo.fired[alert.ID])
	}
}
