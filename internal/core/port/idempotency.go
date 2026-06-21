package port

import (
	"context"

	"github.com/recur-so/recurso/internal/core/domain"
)

// IdempotencyStore persists API responses for idempotent requests
type IdempotencyStore interface {
	// Get retrieves a stored response for a given key
	Get(ctx context.Context, key string) (*domain.StoredResponse, error)

	// Set stores a response for a given key with expiration
	Set(ctx context.Context, key string, response *domain.StoredResponse) error
}
