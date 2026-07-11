package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func openReplayTestDB(t *testing.T) *SSOAssertionReplayRepository {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed sso replay test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return NewSSOAssertionReplayRepository(conn)
}

func TestSSOAssertionReplay_MarkConsumed(t *testing.T) {
	repo := openReplayTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	// Unique per run so repeated CI runs against a shared DB don't collide.
	assertionID := "_assn-" + uuid.NewString()
	future := time.Now().Add(5 * time.Minute)

	// First consume wins.
	if err := repo.MarkConsumed(ctx, tenantID, assertionID, future); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	// Second consume of the same ID is a replay.
	if err := repo.MarkConsumed(ctx, tenantID, assertionID, future); !errors.Is(err, domain.ErrSSOAssertionReplay) {
		t.Fatalf("replay err = %v, want ErrSSOAssertionReplay", err)
	}
	// A different ID is still accepted.
	if err := repo.MarkConsumed(ctx, tenantID, "_assn-"+uuid.NewString(), future); err != nil {
		t.Fatalf("distinct consume: %v", err)
	}
}

func TestSSOAssertionReplay_PrunesExpired(t *testing.T) {
	repo := openReplayTestDB(t)
	ctx := context.Background()
	tenantID := uuid.New()
	expiredID := "_assn-expired-" + uuid.NewString()

	// Record an already-expired assertion, then a fresh consume triggers the prune.
	if err := repo.MarkConsumed(ctx, tenantID, expiredID, time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("consume expired: %v", err)
	}
	if err := repo.MarkConsumed(ctx, tenantID, "_assn-"+uuid.NewString(), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("consume fresh (triggers prune): %v", err)
	}

	// The expired row was pruned, so re-consuming its ID is NOT flagged as replay.
	if err := repo.MarkConsumed(ctx, tenantID, expiredID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("re-consume after prune should succeed, got %v", err)
	}
}
