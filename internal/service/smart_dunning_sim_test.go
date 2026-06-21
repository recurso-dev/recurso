package service

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

// In-memory mock for simulation
type mockDunningRepo struct {
	weights map[string]domain.DunningWeight
}

func (m *mockDunningRepo) GetWeights(ctx context.Context, contextKey string) ([]domain.DunningWeight, error) {
	var results []domain.DunningWeight
	for k, v := range m.weights {
		if k[:len(contextKey)] == contextKey {
			results = append(results, v)
		}
	}
	return results, nil
}

func (m *mockDunningRepo) UpdateWeight(ctx context.Context, weight domain.DunningWeight) error {
	m.weights[weight.ContextKey+":"+weight.ActionID] = weight
	return nil
}

func (m *mockDunningRepo) RecordHistory(ctx context.Context, history domain.DunningHistory) error {
	return nil
}

func TestSmartDunningSimulation(t *testing.T) {
	repo := &mockDunningRepo{weights: make(map[string]domain.DunningWeight)}
	svc := NewSmartRetryService(repo)
	svc.epsilon = 0.2 // Higher exploration for faster convergence in sim

	invoice := &domain.Invoice{
		Currency: "USD",
	}
	errorCode := "insufficient_funds"
	contextKey := "USD:insufficient_funds"

	// THE HIDDEN TRUTH (What we want the AI to learn):
	// 1h: 10% success
	// 24h: 60% success (The Optimal Arm)
	// 3d: 30% success
	// 7d: 5% success
	trueProbabilities := map[string]float64{
		"1h":  0.1,
		"24h": 0.6,
		"3d":  0.3,
		"7d":  0.05,
	}

	iterations := 2000
	successes := make(map[string]int)
	choices := make(map[string]int)

	logInterval := 500

	for i := 0; i < iterations; i++ {
		// 1. Agent selects action
		action := svc.SelectAction(context.Background(), invoice, errorCode)
		choices[action.ID]++

		// 2. Simulate environment response
		reward := 0.0
		if rand.Float64() < trueProbabilities[action.ID] {
			reward = 1.0
			successes[action.ID]++
		}

		// 3. Update Agent
		err := svc.RecordOutcome(context.Background(), domain.DunningHistory{
			ContextKey: contextKey,
			ActionID:   action.ID,
			Reward:     reward,
			ID:         uuid.New(),
		})
		if err != nil {
			t.Fatalf("failed to record outcome: %v", err)
		}

		if (i+1)%logInterval == 0 {
			fmt.Printf("Iteration %d: Choices so far: %v\n", i+1, choices)
		}
	}

	// Final Weights Check
	fmt.Println("--- Final Learned Success Rates ---")
	for id, prob := range trueProbabilities {
		w, _ := repo.GetWeights(context.Background(), contextKey)
		learned := 0.0
		for _, weight := range w {
			if weight.ActionID == id {
				learned = weight.AverageReward
			}
		}
		fmt.Printf("Action %s: True=%.2f, Learned=%.2f, Total Selected=%d\n", id, prob, learned, choices[id])
	}

	// ASSERTION: The "24h" arm should be the most selected one (Exploitation)
	if choices["24h"] <= choices["1h"] || choices["24h"] <= choices["3d"] {
		t.Errorf("Agent failed to converge on the optimal arm (24h). Choices: %v", choices)
	} else {
		fmt.Println("✅ Agent successfully converged on the optimal retry window!")
	}
}
