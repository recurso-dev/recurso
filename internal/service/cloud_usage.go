package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// cloudUsageRepo is the narrow persistence the usage meter needs — accepted as
// an interface so the orchestration is unit-testable with a fake.
type cloudUsageRepo interface {
	AggregateUsage(ctx context.Context, start, end time.Time) ([]*domain.CloudTenantUsage, error)
	Upsert(ctx context.Context, rows []*domain.CloudTenantUsage) error
}

// CloudUsageService measures each tenant's per-period activity for Recurso Cloud
// self-billing (tracked revenue + collected volume, per currency). It is
// money-free: it reads the tenants' own billing data and writes only usage
// readings. No plan, subscription, invoice, or ledger leg is touched here —
// that is a later increment.
type CloudUsageService struct {
	repo   cloudUsageRepo
	logger *slog.Logger
}

func NewCloudUsageService(repo cloudUsageRepo, logger *slog.Logger) *CloudUsageService {
	if logger == nil {
		logger = slog.Default()
	}
	return &CloudUsageService{repo: repo, logger: logger}
}

// MeasurePeriod aggregates usage for [start, end) and upserts one reading per
// (tenant, currency). Idempotent — re-running the same period refreshes the
// rows. Returns the number of readings written.
func (s *CloudUsageService) MeasurePeriod(ctx context.Context, start, end time.Time) (int, error) {
	readings, err := s.repo.AggregateUsage(ctx, start, end)
	if err != nil {
		return 0, fmt.Errorf("aggregate cloud usage: %w", err)
	}
	now := time.Now().UTC()
	for _, r := range readings {
		r.ID = uuid.New()
		r.PeriodStart = start
		r.PeriodEnd = end
		r.ComputedAt = now
	}
	if err := s.repo.Upsert(ctx, readings); err != nil {
		return 0, fmt.Errorf("upsert cloud usage: %w", err)
	}
	return len(readings), nil
}

// MeasureCurrentPeriod measures the current calendar month (UTC) — the natural
// Recurso Cloud billing window. Safe to run repeatedly as the month accrues.
func (s *CloudUsageService) MeasureCurrentPeriod(ctx context.Context) (int, error) {
	start, end := MonthBounds(time.Now().UTC())
	return s.MeasurePeriod(ctx, start, end)
}

// MonthBounds returns [first instant of t's month, first instant of next month)
// in t's location. Exported for tests and the scheduler.
func MonthBounds(t time.Time) (start, end time.Time) {
	y, m, _ := t.Date()
	start = time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(0, 1, 0)
	return start, end
}
