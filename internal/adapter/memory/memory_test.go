package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func TestInMemoryIdempotencyStore_ClaimAndSet(t *testing.T) {
	store := memory.NewInMemoryIdempotencyStore(1 * time.Hour)
	ctx := context.Background()

	key := "idemp-key-100"

	// Initial Get -> empty
	res, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatal("expected empty stored response")
	}

	// Claim key
	claimed, stored, err := store.Claim(ctx, key)
	if err != nil {
		t.Fatalf("unexpected claim error: %v", err)
	}
	if !claimed || stored != nil {
		t.Errorf("expected claimed=true and stored=nil, got claimed=%v, stored=%v", claimed, stored)
	}

	// Re-claim should fail
	claimed2, stored2, err := store.Claim(ctx, key)
	if err != nil {
		t.Fatalf("unexpected second claim error: %v", err)
	}
	if claimed2 {
		t.Error("expected claimed2=false")
	}
	if stored2 != nil {
		t.Error("expected stored2=nil for in-progress claim")
	}

	// Set response
	resp := &domain.StoredResponse{
		Status: 201,
		Body:   []byte(`{"id":"inv_123"}`),
	}
	err = store.Set(ctx, key, resp)
	if err != nil {
		t.Fatalf("unexpected set error: %v", err)
	}

	// Get after set
	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if got == nil || got.Status != 201 {
		t.Errorf("expected status code 201, got %v", got)
	}

	// Delete
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}

	afterDel, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("unexpected get after delete error: %v", err)
	}
	if afterDel != nil {
		t.Error("expected nil after delete")
	}
}

func TestNoOpLocker_Obtain(t *testing.T) {
	locker := memory.NewNoOpLocker()
	ctx := context.Background()

	release, acquired, err := locker.Obtain(ctx, "resource-key", 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected obtain error: %v", err)
	}
	if !acquired {
		t.Fatal("expected acquired=true from NoOpLocker")
	}

	if release == nil {
		t.Fatal("expected non-nil release function")
	}

	err = release(ctx)
	if err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
}
