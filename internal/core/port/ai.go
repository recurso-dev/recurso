package port

import (
	"context"
)

type LLMProvider interface {
	GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}
