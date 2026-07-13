package memory

import (
	"context"
	"sync"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// claimTTL bounds how long an in-progress reservation is held if the request
// dies without releasing it (crash mid-flight). After it lapses the key is
// reclaimable. Requests these keys guard (billing POSTs) complete well inside
// this window.
const claimTTL = 5 * time.Minute

// item holds either a completed response (response != nil) or an in-progress
// reservation (response == nil) placed by Claim.
type item struct {
	response *domain.StoredResponse
	expires  time.Time
}

// InMemoryIdempotencyStore is a simple thread-safe map storage
type InMemoryIdempotencyStore struct {
	mu    sync.RWMutex
	store map[string]item
	ttl   time.Duration
}

// NewInMemoryIdempotencyStore creates a new store with specified TTL
func NewInMemoryIdempotencyStore(ttl time.Duration) port.IdempotencyStore {
	store := &InMemoryIdempotencyStore{
		store: make(map[string]item),
		ttl:   ttl,
	}

	// Background cleanup
	go func() {
		for range time.Tick(1 * time.Hour) {
			store.cleanup()
		}
	}()

	return store
}

func (s *InMemoryIdempotencyStore) Get(ctx context.Context, key string) (*domain.StoredResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	it, found := s.store[key]
	if !found {
		return nil, nil // Not found
	}

	if time.Now().After(it.expires) {
		return nil, nil // Expired
	}

	return it.response, nil // response == nil for an in-progress reservation
}

// Claim atomically reserves key if it is free, otherwise reports the existing
// state. See port.IdempotencyStore.Claim.
func (s *InMemoryIdempotencyStore) Claim(ctx context.Context, key string) (bool, *domain.StoredResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if it, found := s.store[key]; found && now.Before(it.expires) {
		// Taken: a completed response (replay) or an in-progress reservation (409).
		return false, it.response, nil
	}

	// Free (absent or lapsed) — reserve it.
	s.store[key] = item{response: nil, expires: now.Add(claimTTL)}
	return true, nil, nil
}

func (s *InMemoryIdempotencyStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.store, key)
	return nil
}

func (s *InMemoryIdempotencyStore) Set(ctx context.Context, key string, response *domain.StoredResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[key] = item{
		response: response,
		expires:  time.Now().Add(s.ttl),
	}
	return nil
}

func (s *InMemoryIdempotencyStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for k, v := range s.store {
		if now.After(v.expires) {
			delete(s.store, k)
		}
	}
}
