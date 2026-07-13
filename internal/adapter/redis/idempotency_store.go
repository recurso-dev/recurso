package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/redis/go-redis/v9"
)

// inProgressStatus is the sentinel Status of an in-progress reservation placed
// by Claim. Real HTTP statuses are >= 100, so 0 never collides with one.
const inProgressStatus = 0

// claimTTL bounds how long a reservation is held if the request dies without
// releasing it (crash mid-flight), after which the key is reclaimable.
const claimTTL = 5 * time.Minute

// RedisIdempotencyStore implements IdempotencyStore using Redis
type RedisIdempotencyStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisIdempotencyStore creates a new Redis-backed store
func NewRedisIdempotencyStore(client *redis.Client, ttl time.Duration) port.IdempotencyStore {
	return &RedisIdempotencyStore{
		client: client,
		ttl:    ttl,
	}
}

func (s *RedisIdempotencyStore) Get(ctx context.Context, key string) (*domain.StoredResponse, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get idempotency key from redis: %w", err)
	}

	var response domain.StoredResponse
	if err := json.Unmarshal([]byte(val), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal idempotency response: %w", err)
	}

	if response.Status == inProgressStatus {
		return nil, nil // in-progress reservation, not a completed response
	}

	return &response, nil
}

// Claim atomically reserves key via SetNX. See port.IdempotencyStore.Claim.
func (s *RedisIdempotencyStore) Claim(ctx context.Context, key string) (bool, *domain.StoredResponse, error) {
	marker, err := json.Marshal(domain.StoredResponse{Status: inProgressStatus})
	if err != nil {
		return false, nil, fmt.Errorf("failed to marshal idempotency marker: %w", err)
	}

	ok, err := s.client.SetNX(ctx, key, marker, claimTTL).Result()
	if err != nil {
		return false, nil, fmt.Errorf("failed to claim idempotency key in redis: %w", err)
	}
	if ok {
		return true, nil, nil // reservation acquired
	}

	// Already taken — Get returns the completed response, or nil if the holder is
	// still in flight (the marker).
	existing, err := s.Get(ctx, key)
	if err != nil {
		return false, nil, err
	}
	return false, existing, nil
}

func (s *RedisIdempotencyStore) Delete(ctx context.Context, key string) error {
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete idempotency key in redis: %w", err)
	}
	return nil
}

func (s *RedisIdempotencyStore) Set(ctx context.Context, key string, response *domain.StoredResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal idempotency response: %w", err)
	}

	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set idempotency key in redis: %w", err)
	}

	return nil
}
