package service

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// BanditStrategy defines which multi-armed bandit algorithm to use
type BanditStrategy string

const (
	StrategyEpsilonGreedy    BanditStrategy = "epsilon_greedy"
	StrategyThompsonSampling BanditStrategy = "thompson_sampling"
	StrategyUCB1             BanditStrategy = "ucb1"
)

type DunningRepository interface {
	GetWeights(ctx context.Context, contextKey string) ([]domain.DunningWeight, error)
	// ApplyOutcome atomically folds one observed reward into the (context_key,
	// action_id) weight's running average in SQL, so concurrent outcome
	// recordings can't lose an update (the old read-average-write did).
	ApplyOutcome(ctx context.Context, contextKey, actionID string, reward float64) error
	RecordHistory(ctx context.Context, history domain.DunningHistory) error
}

// RetryDecision contains the full details of a retry decision for RL attribution
type RetryDecision struct {
	NextRetryAt time.Time
	Action      domain.DunningAction
	ContextKey  string
	ErrorCode   string
}

// weightCacheEntry stores cached weights with a TTL
type weightCacheEntry struct {
	weights   []domain.DunningWeight
	expiresAt time.Time
}

type SmartRetryService struct {
	repo           DunningRepository
	epsilon        float64 // Exploration rate (e.g., 0.1)
	strategy       BanditStrategy
	totalDecisions int64

	// In-memory weight cache
	cacheMu  sync.RWMutex
	cache    map[string]weightCacheEntry
	cacheTTL time.Duration
}

func NewSmartRetryService(repo DunningRepository) *SmartRetryService {
	return &SmartRetryService{
		repo:     repo,
		epsilon:  0.1, // 10% exploration
		strategy: StrategyEpsilonGreedy,
		cache:    make(map[string]weightCacheEntry),
		cacheTTL: 5 * time.Minute,
	}
}

// SetStrategy sets the bandit algorithm to use
func (s *SmartRetryService) SetStrategy(strategy BanditStrategy) {
	s.strategy = strategy
}

// GetNextRetryTime calculates the absolute time for the next retry attempt (backward-compatible wrapper)
func (s *SmartRetryService) GetNextRetryTime(invoice *domain.Invoice) time.Time {
	decision := s.DecideRetry(context.Background(), invoice, "GENERIC_FAILURE")
	if decision == nil {
		return time.Time{}
	}
	return decision.NextRetryAt
}

// DecideRetry selects the next retry action and returns full decision details for RL attribution
func (s *SmartRetryService) DecideRetry(ctx context.Context, invoice *domain.Invoice, errorCode string) *RetryDecision {
	if invoice.RetryCount >= 10 {
		return nil
	}

	action := s.SelectAction(ctx, invoice, errorCode)
	dContext := domain.DunningContext{
		Currency:     invoice.Currency,
		ErrorCode:    errorCode,
		AttemptCount: invoice.RetryCount,
		AmountBucket: domain.AmountToBucket(invoice.Total),
		DayOfWeek:    int(time.Now().Weekday()),
	}

	return &RetryDecision{
		NextRetryAt: time.Now().Add(action.Interval),
		Action:      action,
		ContextKey:  dContext.Key(),
		ErrorCode:   errorCode,
	}
}

// SelectAction chooses the optimal retry interval based on the configured strategy
func (s *SmartRetryService) SelectAction(ctx context.Context, invoice *domain.Invoice, errorCode string) domain.DunningAction {
	atomic.AddInt64(&s.totalDecisions, 1)

	dContext := domain.DunningContext{
		Currency:     invoice.Currency,
		ErrorCode:    errorCode,
		AttemptCount: invoice.RetryCount,
		AmountBucket: domain.AmountToBucket(invoice.Total),
		DayOfWeek:    int(time.Now().Weekday()),
	}

	switch s.strategy {
	case StrategyThompsonSampling:
		return s.selectThompsonSampling(ctx, dContext)
	case StrategyUCB1:
		return s.selectUCB1(ctx, dContext)
	default:
		return s.selectEpsilonGreedy(ctx, dContext)
	}
}

// selectEpsilonGreedy implements epsilon-greedy with epsilon decay
func (s *SmartRetryService) selectEpsilonGreedy(ctx context.Context, dContext domain.DunningContext) domain.DunningAction {
	// Epsilon decay: epsilon / (1 + 0.001 * totalDecisions)
	decayedEpsilon := s.epsilon / (1.0 + 0.001*float64(atomic.LoadInt64(&s.totalDecisions)))

	// Exploration: randomly pick an action
	if rand.Float64() < decayedEpsilon {
		return domain.DefaultDunningActions[rand.Intn(len(domain.DefaultDunningActions))]
	}

	// Exploitation: pick the action with the highest average reward
	weights := s.getCachedWeights(ctx, dContext.Key())
	if len(weights) == 0 {
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

	return actionByID(bestActionID)
}

// selectThompsonSampling uses Beta distribution approximation for action selection
func (s *SmartRetryService) selectThompsonSampling(ctx context.Context, dContext domain.DunningContext) domain.DunningAction {
	weights := s.getCachedWeights(ctx, dContext.Key())

	bestActionID := ""
	bestSample := -1.0

	for _, action := range domain.DefaultDunningActions {
		// Default prior: Beta(1, 1) = uniform
		alpha := 1.0
		beta := 1.0

		for _, w := range weights {
			if w.ActionID == action.ID {
				// alpha = successes + 1, beta = failures + 1
				successes := w.AverageReward * float64(w.SampleCount)
				failures := float64(w.SampleCount) - successes
				alpha = successes + 1.0
				beta = failures + 1.0
				break
			}
		}

		// Sample from Beta distribution using approximation
		sample := betaSample(alpha, beta)
		if sample > bestSample {
			bestSample = sample
			bestActionID = action.ID
		}
	}

	if bestActionID == "" {
		return domain.Action24Hour
	}
	return actionByID(bestActionID)
}

// selectUCB1 uses Upper Confidence Bound for action selection
func (s *SmartRetryService) selectUCB1(ctx context.Context, dContext domain.DunningContext) domain.DunningAction {
	weights := s.getCachedWeights(ctx, dContext.Key())

	// Calculate total trials across all arms
	var totalTrials int64
	weightMap := make(map[string]domain.DunningWeight)
	for _, w := range weights {
		totalTrials += w.SampleCount
		weightMap[w.ActionID] = w
	}

	if totalTrials == 0 {
		// No data yet — pick randomly
		return domain.DefaultDunningActions[rand.Intn(len(domain.DefaultDunningActions))]
	}

	bestActionID := ""
	bestUCB := -1.0

	for _, action := range domain.DefaultDunningActions {
		w, exists := weightMap[action.ID]
		if !exists || w.SampleCount == 0 {
			// Never tried — infinite UCB, pick this
			return action
		}

		// UCB1 = average_reward + sqrt(2 * ln(totalTrials) / sampleCount)
		explorationBonus := math.Sqrt(2.0 * math.Log(float64(totalTrials)) / float64(w.SampleCount))
		ucb := w.AverageReward + explorationBonus

		if ucb > bestUCB {
			bestUCB = ucb
			bestActionID = action.ID
		}
	}

	if bestActionID == "" {
		return domain.Action24Hour
	}
	return actionByID(bestActionID)
}

// getCachedWeights retrieves weights from cache or DB
func (s *SmartRetryService) getCachedWeights(ctx context.Context, contextKey string) []domain.DunningWeight {
	// Check cache first
	s.cacheMu.RLock()
	entry, ok := s.cache[contextKey]
	s.cacheMu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.weights
	}

	// Cache miss or expired — fetch from DB
	weights, err := s.repo.GetWeights(ctx, contextKey)
	if err != nil || len(weights) == 0 {
		return nil
	}

	// Store in cache
	s.cacheMu.Lock()
	s.cache[contextKey] = weightCacheEntry{
		weights:   weights,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
	s.cacheMu.Unlock()

	return weights
}

// invalidateCache removes a context key from the cache
func (s *SmartRetryService) invalidateCache(contextKey string) {
	s.cacheMu.Lock()
	delete(s.cache, contextKey)
	s.cacheMu.Unlock()
}

// RecordOutcome updates the bandit weights based on success (reward=1.0) or failure (reward=0.0)
func (s *SmartRetryService) RecordOutcome(ctx context.Context, history domain.DunningHistory) error {
	// 1. Persist History
	if err := s.repo.RecordHistory(ctx, history); err != nil {
		return err
	}

	// 2. Fold the reward into the (context_key, action_id) running average
	// ATOMICALLY in SQL — NewAverage = OldAverage + (Reward - OldAverage) /
	// NewSampleCount computed on the current DB row. The old read-average-then-
	// write lost updates when two outcomes for the same arm raced (e.g. two
	// retry-worker instances, or a retry outcome racing a webhook success).
	s.invalidateCache(history.ContextKey)
	return s.repo.ApplyOutcome(ctx, history.ContextKey, history.ActionID, history.Reward)
}

// betaSample generates a sample from a Beta(alpha, beta) distribution
// using the Gamma distribution method
func betaSample(alpha, beta float64) float64 {
	// Use the relationship: if X ~ Gamma(alpha, 1) and Y ~ Gamma(beta, 1),
	// then X / (X + Y) ~ Beta(alpha, beta)
	x := gammaSample(alpha)
	y := gammaSample(beta)
	if x+y == 0 {
		return 0.5
	}
	return x / (x + y)
}

// gammaSample generates a sample from Gamma(alpha, 1) using Marsaglia and Tsang's method
func gammaSample(alpha float64) float64 {
	if alpha < 1 {
		// Boost alpha for the algorithm, then adjust
		return gammaSample(alpha+1) * math.Pow(rand.Float64(), 1.0/alpha)
	}

	d := alpha - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)

	for {
		var x, v float64
		for {
			x = rand.NormFloat64()
			v = 1.0 + c*x
			if v > 0 {
				break
			}
		}
		v = v * v * v
		u := rand.Float64()

		if u < 1.0-0.0331*(x*x)*(x*x) {
			return d * v
		}
		if math.Log(u) < 0.5*x*x+d*(1.0-v+math.Log(v)) {
			return d * v
		}
	}
}

// actionByID maps an action ID back to a DunningAction
func actionByID(id string) domain.DunningAction {
	for _, a := range domain.DefaultDunningActions {
		if a.ID == id {
			return a
		}
	}
	return domain.Action24Hour
}
