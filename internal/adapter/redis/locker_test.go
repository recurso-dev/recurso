package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/redis/go-redis/v9"
)

// newTestLocker connects to the throwaway Redis named by TEST_REDIS_URL, or
// skips. Mirrors the TEST_DATABASE_URL pattern used by the postgres tests.
func newTestLocker(t *testing.T) port.Locker {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set; skipping redis locker test")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse TEST_REDIS_URL: %v", err)
	}
	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return NewRedisLocker(client)
}

// TestRedisLocker_MutualExclusion is the property the whole reliability floor
// rests on: while one instance holds a scheduler lock, a second instance's
// Obtain must return acquired=false (no error) so it skips the run instead of
// double-processing. After release, the lock is free again.
func TestRedisLocker_MutualExclusion(t *testing.T) {
	locker := newTestLocker(t)
	ctx := context.Background()
	key := "test:lock:" + uuid.NewString()

	release, acquired, err := locker.Obtain(ctx, key, 30*time.Second)
	if err != nil || !acquired {
		t.Fatalf("first Obtain: acquired=%v err=%v, want true/nil", acquired, err)
	}

	// A second instance must NOT acquire the same key while it is held.
	if _, acquired2, err2 := locker.Obtain(ctx, key, 30*time.Second); acquired2 || err2 != nil {
		t.Fatalf("second Obtain while held: acquired=%v err=%v, want false/nil (double-processing guard)", acquired2, err2)
	}

	// Release, then the key is obtainable again.
	if err := release(ctx); err != nil {
		t.Fatalf("release: %v", err)
	}
	release3, acquired3, err3 := locker.Obtain(ctx, key, 30*time.Second)
	if err3 != nil || !acquired3 {
		t.Fatalf("Obtain after release: acquired=%v err=%v, want true/nil", acquired3, err3)
	}
	_ = release3(ctx)
}

// TestRedisLocker_DifferentKeysIndependent: distinct scheduler locks never
// block each other (e.g. dunning vs mandate-debit run concurrently).
func TestRedisLocker_DifferentKeysIndependent(t *testing.T) {
	locker := newTestLocker(t)
	ctx := context.Background()
	keyA := "test:lock:" + uuid.NewString()
	keyB := "test:lock:" + uuid.NewString()

	relA, okA, errA := locker.Obtain(ctx, keyA, 30*time.Second)
	if errA != nil || !okA {
		t.Fatalf("Obtain A: acquired=%v err=%v", okA, errA)
	}
	defer func() { _ = relA(ctx) }()

	relB, okB, errB := locker.Obtain(ctx, keyB, 30*time.Second)
	if errB != nil || !okB {
		t.Fatalf("Obtain B blocked by A: acquired=%v err=%v", okB, errB)
	}
	_ = relB(ctx)
}

// TestRedisLocker_TTLExpiry: a holder that dies without releasing (no Release
// call) must not wedge the lock forever — the TTL expires and another instance
// can acquire. This is what keeps a crashed scheduler from freezing billing.
func TestRedisLocker_TTLExpiry(t *testing.T) {
	locker := newTestLocker(t)
	ctx := context.Background()
	key := "test:lock:" + uuid.NewString()

	if _, acquired, err := locker.Obtain(ctx, key, 1*time.Second); err != nil || !acquired {
		t.Fatalf("Obtain with short TTL: acquired=%v err=%v", acquired, err)
	}
	// Deliberately do not release. After the TTL lapses, the key frees itself.
	time.Sleep(1500 * time.Millisecond)

	release, acquired, err := locker.Obtain(ctx, key, 5*time.Second)
	if err != nil || !acquired {
		t.Fatalf("Obtain after TTL expiry: acquired=%v err=%v, want true/nil", acquired, err)
	}
	_ = release(ctx)
}
