package service

import (
	"context"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// DunningAnalyticsRepository defines the read methods needed for analytics
type DunningAnalyticsRepository interface {
	GetAllWeights(ctx context.Context) ([]domain.DunningWeight, error)
	GetRecentHistory(ctx context.Context, limit int) ([]domain.DunningHistory, error)
	GetHistoryStats(ctx context.Context) (totalRetries int, totalSuccesses int, err error)
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

func (s *DunningAnalyticsService) GetOverview(ctx context.Context) (*DunningOverview, error) {
	totalRetries, totalSuccesses, err := s.repo.GetHistoryStats(ctx)
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

func (s *DunningAnalyticsService) GetRecentHistory(ctx context.Context, limit int) ([]domain.DunningHistory, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.GetRecentHistory(ctx, limit)
}
