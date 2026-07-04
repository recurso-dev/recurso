package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

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

	return &response, nil
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
