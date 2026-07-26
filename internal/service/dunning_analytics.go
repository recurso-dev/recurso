package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// DunningAnalyticsRepository defines the read methods needed for analytics
type DunningAnalyticsRepository interface {
	// GetAllWeights returns the shared bandit model's weights (keyed by
	// context/action, not per-tenant) — intentionally global.
	GetAllWeights(ctx context.Context) ([]domain.DunningWeight, error)
	// GetRecentHistory / GetHistoryStats read per-tenant dunning_history and
	// are tenant-scoped.
	GetRecentHistory(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.DunningHistory, error)
	GetHistoryStats(ctx context.Context, tenantID uuid.UUID) (totalRetries int, totalSuccesses int, err error)
	// GetTimingRates tallies historical retry outcomes by hour-of-day and
	// day-of-week (Collections Intelligence Inc 4). Read-only; tenant-scoped.
	GetTimingRates(ctx context.Context, tenantID uuid.UUID) ([]domain.DunningTimingBucket, error)
}

type DunningAnalyticsService struct {
	repo DunningAnalyticsRepository
}

func NewDunningAnalyticsService(repo DunningAnalyticsRepository) *DunningAnalyticsService {
	return &DunningAnalyticsService{repo: repo}
}

type DunningOverview struct {
	TotalRetries   int     `json:"total_retries"`
	TotalSuccesses int     `json:"total_successes"`
	SuccessRate    float64 `json:"success_rate"`
}

func (s *DunningAnalyticsService) GetOverview(ctx context.Context, tenantID uuid.UUID) (*DunningOverview, error) {
	totalRetries, totalSuccesses, err := s.repo.GetHistoryStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	successRate := 0.0
	if totalRetries > 0 {
		successRate = float64(totalSuccesses) / float64(totalRetries)
	}

	return &DunningOverview{
		TotalRetries:   totalRetries,
		TotalSuccesses: totalSuccesses,
		SuccessRate:    successRate,
	}, nil
}

func (s *DunningAnalyticsService) GetWeightsByContext(ctx context.Context) ([]domain.DunningWeight, error) {
	return s.repo.GetAllWeights(ctx)
}

func (s *DunningAnalyticsService) GetRecentHistory(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.DunningHistory, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.GetRecentHistory(ctx, tenantID, limit)
}

// minTimingSample is the fewest retries a bucket needs before its success rate
// is trustworthy enough to be crowned "best" — below it, one lucky success would
// dominate the recommendation.
const minTimingSample = 5

// DunningTimingRate is one time bucket's success rate.
type DunningTimingRate struct {
	Bucket      int     `json:"bucket"`
	Total       int     `json:"total"`
	Successes   int     `json:"successes"`
	SuccessRate float64 `json:"success_rate"`
}

// DunningTimingInsights answers "when should we retry?" from historical outcomes
// (Collections Intelligence Inc 4). ByHour is 0-23, ByDayOfWeek is 0-6
// (Sunday=0), both in UTC. BestHour/BestDay are the highest-success-rate buckets
// with enough samples to trust; nil when there isn't enough history. Read-only —
// this does NOT change the live retry bandit.
type DunningTimingInsights struct {
	ByHour      []DunningTimingRate `json:"by_hour"`
	ByDayOfWeek []DunningTimingRate `json:"by_day_of_week"`
	BestHour    *int                `json:"best_hour,omitempty"`
	BestDay     *int                `json:"best_day,omitempty"`
	SampleSize  int                 `json:"sample_size"`
}

// GetTimingInsights builds the timing insights: it fills every bucket (so the
// chart has no gaps), computes rates, and picks the best-performing hour and day
// among buckets that clear the minimum sample size.
func (s *DunningAnalyticsService) GetTimingInsights(ctx context.Context, tenantID uuid.UUID) (*DunningTimingInsights, error) {
	buckets, err := s.repo.GetTimingRates(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	hours := make([]DunningTimingRate, 24)
	for i := range hours {
		hours[i] = DunningTimingRate{Bucket: i}
	}
	days := make([]DunningTimingRate, 7)
	for i := range days {
		days[i] = DunningTimingRate{Bucket: i}
	}

	sample := 0
	for _, b := range buckets {
		var target *DunningTimingRate
		switch b.Unit {
		case "hour":
			if b.Bucket >= 0 && b.Bucket < 24 {
				target = &hours[b.Bucket]
			}
		case "dow":
			if b.Bucket >= 0 && b.Bucket < 7 {
				target = &days[b.Bucket]
			}
		}
		if target == nil {
			continue
		}
		target.Total = b.Total
		target.Successes = b.Successes
		if b.Total > 0 {
			target.SuccessRate = float64(b.Successes) / float64(b.Total)
		}
		if b.Unit == "hour" {
			sample += b.Total // total retries counted once (hour buckets partition all rows)
		}
	}

	return &DunningTimingInsights{
		ByHour:      hours,
		ByDayOfWeek: days,
		BestHour:    bestBucket(hours),
		BestDay:     bestBucket(days),
		SampleSize:  sample,
	}, nil
}

// bestBucket returns the bucket index with the highest success rate among those
// meeting minTimingSample, or nil if none qualify.
func bestBucket(rates []DunningTimingRate) *int {
	best := -1
	bestRate := -1.0
	for _, r := range rates {
		if r.Total < minTimingSample {
			continue
		}
		if r.SuccessRate > bestRate {
			bestRate = r.SuccessRate
			best = r.Bucket
		}
	}
	if best < 0 {
		return nil
	}
	return &best
}
