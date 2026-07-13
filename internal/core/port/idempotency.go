package port

import (
	"context"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// IdempotencyStore persists API responses for idempotent requests
type IdempotencyStore interface {
	// Get retrieves a stored (completed) response for a given key. An in-progress
	// reservation (see Claim) is not a completed response and returns (nil, nil).
	Get(ctx context.Context, key string) (*domain.StoredResponse, error)

	// Set stores a completed response for a given key with expiration, replacing
	// any in-progress reservation.
	Set(ctx context.Context, key string, response *domain.StoredResponse) error

	// Claim atomically reserves a key for processing. It is the concurrency-safe
	// gate that replaces a naive Get-then-process check:
	//   - acquired=true: the caller won the reservation and must process the
	//     request (then call Set on success, or Delete on failure).
	//   - acquired=false, existing!=nil: a completed response already exists;
	//     the caller should replay it.
	//   - acquired=false, existing=nil: another request holds the reservation and
	//     is still in flight; the caller should reject with 409.
	Claim(ctx context.Context, key string) (acquired bool, existing *domain.StoredResponse, err error)

	// Delete releases a key. Used to drop an in-progress reservation after a 5xx
	// or panic so the request can be safely retried with the same key.
	Delete(ctx context.Context, key string) error
}
