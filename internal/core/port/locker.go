package port

import (
	"context"
	"time"
)

// Locker provides distributed locking capabilities
type Locker interface {
	// Obtain tries to obtain a lock with the given key and TTL.
	// Returns a Release function (to unlock) and boolean (true if acquired).
	// Or returns an error if connection fails.
	Obtain(ctx context.Context, key string, ttl time.Duration) (func(context.Context) error, bool, error)
}
