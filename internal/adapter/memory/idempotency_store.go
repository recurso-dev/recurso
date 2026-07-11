package memory

import (
	"context"
	"sync"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

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

	return it.response, nil
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
