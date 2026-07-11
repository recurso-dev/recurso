package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func openMRRSnapshotTestDB(t *testing.T) *MRRSnapshotRepository {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed mrr snapshot test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return NewMRRSnapshotRepository(conn)
}

func mrrSnap(tenant, sub uuid.UUID, date time.Time, amount int64) domain.MRRSnapshot {
	return domain.MRRSnapshot{TenantID: tenant, SubscriptionID: sub, SnapshotDate: date, MRRAmount: amount, Currency: "USD"}
}

func TestMRRSnapshotRepository_RoundTrip(t *testing.T) {
	repo := openMRRSnapshotTestDB(t)
	ctx := context.Background()

	tenant := uuid.New() // fresh tenant → assertions isolated in a shared DB
	subA, subB := uuid.New(), uuid.New()
	d0 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	if err := repo.UpsertSnapshots(ctx, []domain.MRRSnapshot{
		mrrSnap(tenant, subA, d1, 1000),
		mrrSnap(tenant, subA, d2, 1500),
		mrrSnap(tenant, subB, d0, 800),
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// ResolveSnapshotDate: mid-June resolves to d1 (nearest on-or-before).
	got, ok, err := repo.ResolveSnapshotDate(ctx, tenant, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))
	if err != nil || !ok {
		t.Fatalf("resolve: ok=%v err=%v", ok, err)
	}
	if !got.Equal(d1) {
		t.Errorf("resolved date = %v, want %v", got.Format("2006-01-02"), d1.Format("2006-01-02"))
	}

	// GetSnapshotsOn d1 → just subA at 1000.
	snaps, err := repo.GetSnapshotsOn(ctx, tenant, d1)
	if err != nil {
		t.Fatalf("get on d1: %v", err)
	}
	if len(snaps) != 1 || snaps[0].SubscriptionID != subA || snaps[0].MRRAmount != 1000 {
		t.Fatalf("d1 snapshots = %+v, want [subA=1000]", snaps)
	}

	// Idempotent upsert: re-write subA@d1 with a new amount.
	if err := repo.UpsertSnapshots(ctx, []domain.MRRSnapshot{mrrSnap(tenant, subA, d1, 1200)}); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}
	snaps, _ = repo.GetSnapshotsOn(ctx, tenant, d1)
	if len(snaps) != 1 || snaps[0].MRRAmount != 1200 {
		t.Fatalf("after re-upsert d1 = %+v, want [subA=1200]", snaps)
	}

	// SubscriptionIDsSeenBefore(d1): subB (d0 < d1) yes, subA (earliest d1) no.
	seen, err := repo.SubscriptionIDsSeenBefore(ctx, tenant, d1)
	if err != nil {
		t.Fatalf("seen-before: %v", err)
	}
	if !seen[subB] || seen[subA] {
		t.Errorf("seen-before d1 = %v, want {subB:true, subA:false}", seen)
	}
}
