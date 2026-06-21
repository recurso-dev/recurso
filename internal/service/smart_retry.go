package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/recur-so/recurso/internal/core/domain"
)

type DunningRepository interface {
	GetWeights(ctx context.Context, contextKey string) ([]domain.DunningWeight, error)
	UpdateWeight(ctx context.Context, weight domain.DunningWeight) error
	RecordHistory(ctx context.Context, history domain.DunningHistory) error
}

type SmartRetryService struct {
	repo    DunningRepository
	epsilon float64 // Exploration rate (e.g., 0.1)
}

func NewSmartRetryService(repo DunningRepository) *SmartRetryService {
	return &SmartRetryService{
		repo:    repo,
		epsilon: 0.1, // 10% exploration
	}
}

// GetNextRetryTime calculates the absolute time for the next retry attempt
func (s *SmartRetryService) GetNextRetryTime(invoice *domain.Invoice) time.Time {
	// Max retries safety
	if invoice.RetryCount >= 10 {
		return time.Time{}
	}

	// For MVP, we pass "GENERIC_FAILURE" as errorCode if not provided
	action := s.SelectAction(context.Background(), invoice, "GENERIC_FAILURE")
	return time.Now().Add(action.Interval)
}

// SelectAction chooses the optimal retry interval based on epsilon-greedy strategy
func (s *SmartRetryService) SelectAction(ctx context.Context, invoice *domain.Invoice, errorCode string) domain.DunningAction {
	dContext := domain.DunningContext{
		Currency:  invoice.Currency,
		ErrorCode: errorCode,
	}

	// 1. Exploration: Randomly pick an action with probability epsilon
	if rand.Float64() < s.epsilon {
		return domain.DefaultDunningActions[rand.Intn(len(domain.DefaultDunningActions))]
	}

	// 2. Exploitation: Pick the action with the highest average reward
	weights, err := s.repo.GetWeights(ctx, dContext.Key())
	if err != nil || len(weights) == 0 {
		// Fallback if no data yet or error
		return domain.Action24Hour
	}

	bestActionID := weights[0].ActionID
	maxReward := weights[0].AverageReward

	for _, w := range weights {
		if w.AverageReward > maxReward {
			maxReward = w.AverageReward
			bestActionID = w.ActionID
		}
	}

	// Map ActionID back to DunningAction
	for _, a := range domain.DefaultDunningActions {
		if a.ID == bestActionID {
			return a
		}
	}

	return domain.Action24Hour
}

// RecordOutcome updates the bandit weights based on success (reward=1.0) or failure (reward=0.0)
func (s *SmartRetryService) RecordOutcome(ctx context.Context, history domain.DunningHistory) error {
	// 1. Persist History
	if err := s.repo.RecordHistory(ctx, history); err != nil {
		return err
	}

	// 2. Update Weights (Incremental Average)
	// NewAverage = OldAverage + (Reward - OldAverage) / NewSampleCount
	weights, err := s.repo.GetWeights(ctx, history.ContextKey)
	if err != nil {
		return err
	}

	var weight domain.DunningWeight
	found := false
	for _, w := range weights {
		if w.ActionID == history.ActionID {
			weight = w
			found = true
			break
		}
	}

	if !found {
		weight = domain.DunningWeight{
			ContextKey: history.ContextKey,
			ActionID:   history.ActionID,
		}
	}

	weight.SampleCount++
	weight.AverageReward += (history.Reward - weight.AverageReward) / float64(weight.SampleCount)
	weight.UpdatedAt = time.Now()

	return s.repo.UpdateWeight(ctx, weight)
}
