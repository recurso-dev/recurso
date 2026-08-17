package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

type fakeUsageRepo struct {
	agg      []*domain.CloudTenantUsage
	upserted []*domain.CloudTenantUsage
	gotStart time.Time
	gotEnd   time.Time
}

func (f *fakeUsageRepo) AggregateUsage(_ context.Context, start, end time.Time) ([]*domain.CloudTenantUsage, error) {
	f.gotStart, f.gotEnd = start, end
	return f.agg, nil
}

func (f *fakeUsageRepo) Upsert(_ context.Context, rows []*domain.CloudTenantUsage) error {
	f.upserted = rows
	return nil
}

func TestMonthBounds(t *testing.T) {
	// mid-February (leap year 2024) → [Feb 1, Mar 1)
	got := time.Date(2024, 2, 14, 9, 30, 0, 0, time.UTC)
	start, end := MonthBounds(got)
	if !start.Equal(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("start = %v", start)
	}
	if !end.Equal(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("end = %v", end)
	}
	// December wraps the year → [Dec 1, Jan 1 next year)
	_, decEnd := MonthBounds(time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC))
	if !decEnd.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("december end = %v", decEnd)
	}
}

func TestMeasurePeriod_StampsAndUpserts(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	repo := &fakeUsageRepo{agg: []*domain.CloudTenantUsage{
		{TenantID: tenantA, Currency: "USD", TrackedRevenueMinor: 1_800_000, CollectedVolumeMinor: 1_500_000},
		{TenantID: tenantB, Currency: "INR", TrackedRevenueMinor: 500_000, CollectedVolumeMinor: 0},
	}}
	svc := NewCloudUsageService(repo, nil)

	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	n, err := svc.MeasurePeriod(context.Background(), start, end)
	if err != nil {
		t.Fatalf("MeasurePeriod: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 readings written, got %d", n)
	}
	if !repo.gotStart.Equal(start) || !repo.gotEnd.Equal(end) {
		t.Fatalf("aggregate called with wrong window: %v..%v", repo.gotStart, repo.gotEnd)
	}
	if len(repo.upserted) != 2 {
		t.Fatalf("expected 2 rows upserted, got %d", len(repo.upserted))
	}
	for _, r := range repo.upserted {
		if r.ID == uuid.Nil {
			t.Fatalf("reading id not stamped")
		}
		if !r.PeriodStart.Equal(start) || !r.PeriodEnd.Equal(end) {
			t.Fatalf("period not stamped on reading: %+v", r)
		}
		if r.ComputedAt.IsZero() {
			t.Fatalf("computed_at not stamped")
		}
	}
	// measurement values pass through untouched
	if repo.upserted[0].TrackedRevenueMinor != 1_800_000 || repo.upserted[0].CollectedVolumeMinor != 1_500_000 {
		t.Fatalf("measurement mutated: %+v", repo.upserted[0])
	}
}

func TestMeasureCurrentPeriod_UsesCurrentMonth(t *testing.T) {
	repo := &fakeUsageRepo{}
	svc := NewCloudUsageService(repo, nil)
	if _, err := svc.MeasureCurrentPeriod(context.Background()); err != nil {
		t.Fatalf("MeasureCurrentPeriod: %v", err)
	}
	wantStart, wantEnd := MonthBounds(time.Now().UTC())
	if !repo.gotStart.Equal(wantStart) || !repo.gotEnd.Equal(wantEnd) {
		t.Fatalf("current period window = %v..%v, want %v..%v", repo.gotStart, repo.gotEnd, wantStart, wantEnd)
	}
}
