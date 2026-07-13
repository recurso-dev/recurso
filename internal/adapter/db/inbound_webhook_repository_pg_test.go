package db

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// TestInboundWebhookRepository_Dedup proves the ENG-162 webhook idempotency
// primitive: an event id is "not processed" until recorded, recording is
// idempotent, and a burst of concurrent MarkProcessed for the same id leaves
// exactly one row — so a redelivered (or concurrently-delivered) webhook is
// recognized as a duplicate.
func TestInboundWebhookRepository_Dedup(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed inbound-webhook test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	repo := NewInboundWebhookRepository(conn)
	eventID := "evt_" + uuid.New().String()

	// Unseen id → not processed.
	if seen, err := repo.WasProcessed(ctx, "stripe", eventID); err != nil || seen {
		t.Fatalf("WasProcessed before record: seen=%v err=%v, want false", seen, err)
	}

	if err := repo.MarkProcessed(ctx, "stripe", eventID, "payment_intent.succeeded"); err != nil {
		t.Fatalf("MarkProcessed: %v", err)
	}

	// Now processed.
	if seen, err := repo.WasProcessed(ctx, "stripe", eventID); err != nil || !seen {
		t.Fatalf("WasProcessed after record: seen=%v err=%v, want true", seen, err)
	}
	// Same id under a different gateway is independent.
	if seen, err := repo.WasProcessed(ctx, "razorpay", eventID); err != nil || seen {
		t.Fatalf("WasProcessed cross-gateway: seen=%v err=%v, want false", seen, err)
	}

	// Recording is idempotent, including under concurrency (ON CONFLICT).
	var wg sync.WaitGroup
	dupID := "evt_" + uuid.New().String()
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := repo.MarkProcessed(ctx, "stripe", dupID, "invoice.payment_failed"); err != nil {
				t.Errorf("concurrent MarkProcessed: %v", err)
			}
		}()
	}
	wg.Wait()

	var count int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inbound_webhook_events WHERE gateway = $1 AND event_id = $2`,
		"stripe", dupID).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("rows for one event id = %d, want exactly 1", count)
	}
}
