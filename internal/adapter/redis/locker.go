package redis

import (
	"context"
	"time"

	"github.com/bsm/redislock"
	"github.com/redis/go-redis/v9"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// RedisLocker implements Locker using Redis
type RedisLocker struct {
	client *redislock.Client
}

// NewRedisLocker creates a new Redis-backed locker
func NewRedisLocker(client *redis.Client) port.Locker {
	return &RedisLocker{
		client: redislock.New(client),
	}
}

func (l *RedisLocker) Obtain(ctx context.Context, key string, ttl time.Duration) (func(context.Context) error, bool, error) {
	// Try to obtain lock
	lock, err := l.client.Obtain(ctx, key, ttl, nil)

	if err == redislock.ErrNotObtained {
		return nil, false, nil // Could not acquire
	} else if err != nil {
		return nil, false, err // Connection error
	}

	// Return release function
	return lock.Release, true, nil
}
