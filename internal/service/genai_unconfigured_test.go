package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// TestGenAIAskUnconfigured proves Ask fails with the typed sentinel — not a
// nil-interface panic — when no LLM provider is wired (OPENAI_API_KEY unset).
// Before the guard, every /v1/analytics/ask call on an unconfigured server
// panicked and surfaced as a bodyless 500.
func TestGenAIAskUnconfigured(t *testing.T) {
	svc := NewGenAIService(nil, nil)

	_, _, err := svc.Ask(context.Background(), uuid.New(), "how much MRR?")
	if !errors.Is(err, ErrGenAINotConfigured) {
		t.Fatalf("err = %v, want ErrGenAINotConfigured", err)
	}
}
