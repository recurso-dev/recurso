package memory

import (
	"context"
	"time"

	"github.com/recur-so/recurso/internal/core/port"
)

// NoOpLocker is a dummy locker that always succeeds (for dev mode single instance)
type NoOpLocker struct{}

func NewNoOpLocker() port.Locker {
	return &NoOpLocker{}
}

func (l *NoOpLocker) Obtain(ctx context.Context, key string, ttl time.Duration) (func(context.Context) error, bool, error) {
	// Always succeed, do nothing on release
	return func(context.Context) error { return nil }, true, nil
}
