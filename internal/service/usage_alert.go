package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// UsageAlertService owns usage threshold alerts (Lago-parity B3): CRUD and
// the periodic evaluation sweep. An alert fires at most once per billing
// period per threshold — the repository's conditional MarkFired is the
// dedup, so concurrent sweeps cannot double-fire.

var (
	ErrAlertNotFound = errors.New("usage alert not found")
	ErrAlertExists   = errors.New("an identical alert already exists")
)

// alertEventPublisher is the slice of WebhookService alerts emit through.
type alertEventPublisher interface {
	PublishEvent(ctx context.Context, input PublishEventInput) (*domain.Event, error)
}

type UsageAlertService struct {
	alerts        port.UsageAlertRepository
	subscriptions port.SubscriptionRepository
	metrics       port.BillableMetricRepository
	usage         port.UsageRepository
	customers     port.CustomerRepository
	entitlements  usageEntitlementChecker // percent_of_limit resolution
	events        alertEventPublisher     // nil-safe
	notifier      port.Notifier           // nil-safe

	now func() time.Time
}

func NewUsageAlertService(
	alerts port.UsageAlertRepository,
	subscriptions port.SubscriptionRepository,
	metrics port.BillableMetricRepository,
	usage port.UsageRepository,
	customers port.CustomerRepository,
	entitlements usageEntitlementChecker,
) *UsageAlertService {
	return &UsageAlertService{
		alerts:        alerts,
		subscriptions: subscriptions,
		metrics:       metrics,
		usage:         usage,
		customers:     customers,
		entitlements:  entitlements,
		now:           func() time.Time { return time.Now().UTC() },
	}
}

// SetEventPublisher wires webhook emission (nil-safe).
func (s *UsageAlertService) SetEventPublisher(p alertEventPublisher) { s.events = p }

// SetNotifier wires the email notification (nil-safe).
func (s *UsageAlertService) SetNotifier(n port.Notifier) { s.notifier = n }

// UsageAlertInput creates one alert.
type UsageAlertInput struct {
	SubscriptionID string `json:"subscription_id" binding:"required"`
	MetricCode     string `json:"metric_code" binding:"required"`
	ThresholdType  string `json:"threshold_type" binding:"required"`
	Threshold      int64  `json:"threshold" binding:"required,gt=0"`
}

func (s *UsageAlertService) CreateAlert(ctx context.Context, tenantID uuid.UUID, in UsageAlertInput) (*domain.UsageAlert, error) {
	subID, err := uuid.Parse(in.SubscriptionID)
	if err != nil {
		return nil, MeteringValidationError("invalid subscription_id")
	}
	tt := domain.UsageAlertThresholdType(in.ThresholdType)
	if tt != domain.AlertThresholdQuantity && tt != domain.AlertThresholdPercentOfLimit {
		return nil, MeteringValidationError("threshold_type must be quantity or percent_of_limit")
	}
	if in.Threshold <= 0 || (tt == domain.AlertThresholdPercentOfLimit && in.Threshold > 1000) {
		return nil, MeteringValidationError("threshold must be positive (percent thresholds at most 1000)")
	}

	sub, err := s.subscriptions.GetByID(ctx, subID)
	if err != nil || sub == nil || sub.TenantID != tenantID {
		return nil, ErrUsageSubscriptionNotFound
	}
	metric, err := s.metrics.GetByCode(ctx, tenantID, in.MetricCode)
	if err != nil {
		return nil, err
	}
	if metric == nil {
		return nil, MeteringValidationError(fmt.Sprintf("no billable metric with code %q", in.MetricCode))
	}

	now := s.now()
	a := &domain.UsageAlert{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SubscriptionID: subID,
		MetricCode:     in.MetricCode,
		ThresholdType:  tt,
		Threshold:      in.Threshold,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.alerts.Create(ctx, a); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, ErrAlertExists
		}
		return nil, err
	}
	return a, nil
}

func (s *UsageAlertService) ListAlerts(ctx context.Context, tenantID uuid.UUID, subscriptionID *uuid.UUID) ([]domain.UsageAlert, error) {
	var (
		alerts []domain.UsageAlert
		err    error
	)
	if subscriptionID != nil {
		alerts, err = s.alerts.ListBySubscription(ctx, tenantID, *subscriptionID)
	} else {
		alerts, err = s.alerts.ListByTenant(ctx, tenantID)
	}
	if err != nil {
		return nil, err
	}
	if alerts == nil {
		alerts = []domain.UsageAlert{}
	}
	return alerts, nil
}

func (s *UsageAlertService) DeleteAlert(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.alerts.Delete(ctx, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAlertNotFound
	}
	return err
}

// alertSweepLimit bounds one evaluation pass.
const alertSweepLimit = 1000

// EvaluateAlerts runs one sweep: for every configured alert, aggregate the
// subscription's current-period usage for the metric and fire when the
// threshold is crossed. Firing = conditional claim → webhook event →
// email; per-alert failures log and continue.
func (s *UsageAlertService) EvaluateAlerts(ctx context.Context) (int, error) {
	alerts, err := s.alerts.ListAll(ctx, alertSweepLimit)
	if err != nil {
		return 0, err
	}
	fired := 0
	for i := range alerts {
		if s.evaluateOne(ctx, &alerts[i]) {
			fired++
		}
	}
	return fired, nil
}

func (s *UsageAlertService) evaluateOne(ctx context.Context, a *domain.UsageAlert) bool {
	tctx := context.WithValue(ctx, domain.TenantIDKey, a.TenantID)
	sub, err := s.subscriptions.GetByID(tctx, a.SubscriptionID)
	if err != nil || sub == nil || sub.Status != domain.SubscriptionStatusActive {
		return false
	}
	// Already fired this period? Cheap local skip before any queries.
	if a.LastFiredPeriodStart != nil && a.LastFiredPeriodStart.Equal(sub.CurrentPeriodStart) {
		return false
	}

	metric, err := s.metrics.GetByCode(tctx, a.TenantID, a.MetricCode)
	if err != nil || metric == nil {
		return false
	}
	qty, err := s.usage.AggregateForMetric(tctx, sub.ID, *metric, sub.CurrentPeriodStart, s.now())
	if err != nil {
		slog.Warn("usage alert aggregation failed", "alert_id", a.ID, "error", err)
		return false
	}

	threshold := a.Threshold
	if a.ThresholdType == domain.AlertThresholdPercentOfLimit {
		check, err := s.entitlements.CheckFeature(tctx, a.TenantID, sub.CustomerID, a.MetricCode)
		if err != nil || check == nil || check.LimitValue == nil || *check.LimitValue <= 0 {
			return false // no limit to be a percentage of
		}
		threshold = *check.LimitValue * a.Threshold / 100
	}
	if qty < threshold {
		return false
	}

	// Claim the firing for this period; a losing concurrent sweep stops here.
	claimed, err := s.alerts.MarkFired(ctx, a.ID, sub.CurrentPeriodStart)
	if err != nil || !claimed {
		return false
	}

	if s.events != nil {
		if _, err := s.events.PublishEvent(tctx, PublishEventInput{
			TenantID:   a.TenantID,
			Type:       domain.EventUsageAlertTriggered,
			ObjectType: "usage_alert",
			ObjectID:   a.ID,
			Data: map[string]interface{}{
				"subscription_id": sub.ID.String(),
				"customer_id":     sub.CustomerID.String(),
				"metric_code":     a.MetricCode,
				"threshold_type":  string(a.ThresholdType),
				"threshold":       a.Threshold,
				"quantity":        qty,
				"period_start":    sub.CurrentPeriodStart,
			},
		}); err != nil {
			slog.Error("usage alert webhook emission failed", "alert_id", a.ID, "error", err)
		}
	}
	if s.notifier != nil {
		if customer, err := s.customers.GetByID(tctx, sub.CustomerID); err == nil && customer != nil {
			subject := fmt.Sprintf("Usage alert: %s", a.MetricCode)
			body := fmt.Sprintf("Your %s usage this period (%d) has crossed the configured threshold.", a.MetricCode, qty)
			if err := s.notifier.SendEmail(tctx, customer.Email, subject, body); err != nil {
				slog.Warn("usage alert email failed", "alert_id", a.ID, "error", err)
			}
		}
	}
	slog.Info("usage alert fired", "alert_id", a.ID, "metric", a.MetricCode, "quantity", qty)
	return true
}
