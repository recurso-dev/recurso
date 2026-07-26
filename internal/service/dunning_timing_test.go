package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// stubTimingRepo satisfies DunningAnalyticsRepository; only GetTimingRates matters.
type stubTimingRepo struct {
	buckets []domain.DunningTimingBucket
}

func (s *stubTimingRepo) GetAllWeights(_ context.Context) ([]domain.DunningWeight, error) {
	return nil, nil
}
func (s *stubTimingRepo) GetRecentHistory(_ context.Context, _ uuid.UUID, _ int) ([]domain.DunningHistory, error) {
	return nil, nil
}
func (s *stubTimingRepo) GetHistoryStats(_ context.Context, _ uuid.UUID) (int, int, error) {
	return 0, 0, nil
}
func (s *stubTimingRepo) GetTimingRates(_ context.Context, _ uuid.UUID) ([]domain.DunningTimingBucket, error) {
	return s.buckets, nil
}

// Insights fill every bucket, compute rates, and crown the highest-rate hour/day
// that clears the minimum sample size.
func TestGetTimingInsights_PicksBestWithSampleFloor(t *testing.T) {
	repo := &stubTimingRepo{buckets: []domain.DunningTimingBucket{
		{Unit: "hour", Bucket: 9, Total: 10, Successes: 8},  // 0.80, enough samples → candidate
		{Unit: "hour", Bucket: 3, Total: 2, Successes: 2},   // 1.00 but only 2 samples → ignored
		{Unit: "hour", Bucket: 14, Total: 20, Successes: 5}, // 0.25
		{Unit: "dow", Bucket: 2, Total: 12, Successes: 9},   // Tuesday 0.75
		{Unit: "dow", Bucket: 6, Total: 6, Successes: 1},    // 0.167
	}}
	svc := NewDunningAnalyticsService(repo)

	ins, err := svc.GetTimingInsights(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetTimingInsights: %v", err)
	}
	if len(ins.ByHour) != 24 || len(ins.ByDayOfWeek) != 7 {
		t.Fatalf("expected filled buckets 24/7, got %d/%d", len(ins.ByHour), len(ins.ByDayOfWeek))
	}
	if ins.BestHour == nil || *ins.BestHour != 9 {
		t.Errorf("best hour = %v, want 9 (0.80, cleared sample floor; the 1.00 hour had too few)", ins.BestHour)
	}
	if ins.BestDay == nil || *ins.BestDay != 2 {
		t.Errorf("best day = %v, want 2 (Tuesday)", ins.BestDay)
	}
	// Sample size = total hour-bucket rows (10+2+20 = 32).
	if ins.SampleSize != 32 {
		t.Errorf("sample size = %d, want 32", ins.SampleSize)
	}
	// Rate is computed for a filled bucket.
	if ins.ByHour[14].SuccessRate < 0.24 || ins.ByHour[14].SuccessRate > 0.26 {
		t.Errorf("hour 14 rate = %f, want ~0.25", ins.ByHour[14].SuccessRate)
	}
	// An empty bucket stays zero, not NaN.
	if ins.ByHour[0].Total != 0 || ins.ByHour[0].SuccessRate != 0 {
		t.Errorf("empty bucket should be zero, got %+v", ins.ByHour[0])
	}
}

// With no history that clears the sample floor, best hour/day are nil (no
// misleading recommendation from thin data).
func TestGetTimingInsights_InsufficientData(t *testing.T) {
	repo := &stubTimingRepo{buckets: []domain.DunningTimingBucket{
		{Unit: "hour", Bucket: 9, Total: 2, Successes: 2},
	}}
	svc := NewDunningAnalyticsService(repo)
	ins, err := svc.GetTimingInsights(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetTimingInsights: %v", err)
	}
	if ins.BestHour != nil || ins.BestDay != nil {
		t.Errorf("expected no recommendation from thin data, got hour=%v day=%v", ins.BestHour, ins.BestDay)
	}
}
