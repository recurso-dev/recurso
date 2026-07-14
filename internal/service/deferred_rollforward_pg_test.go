package service

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestDeferredRollforward_Postgres proves the Deferred Revenue rollforward from
// the ledger: a subscription invoice credits Deferred (added), a recognition
// debits it (released), and the period buckets resolve correctly — including
// the opening balance when the window starts after all activity.
func TestDeferredRollforward_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed deferred-rollforward test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "DRF-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	svc := NewLedgerService(nil, db.NewLedgerRepository(conn))

	subID := uuid.New()
	subInv := &domain.Invoice{
		ID: uuid.New(), TenantID: tenantID, CustomerID: uuid.New(),
		SubscriptionID: &subID, InvoiceNumber: "DRF-1", Total: 12000, Currency: "USD",
	}
	if err := svc.RecordInvoice(ctx, subInv); err != nil { // DR AR / CR Deferred 12000
		t.Fatalf("RecordInvoice: %v", err)
	}
	if _, err := svc.RecordRecognition(ctx, tenantID, 2000, uuid.New()); err != nil { // DR Deferred / CR Recognized 2000
		t.Fatalf("RecordRecognition: %v", err)
	}

	now := time.Now()

	// Window bracketing the just-posted activity: nothing before start, both
	// postings inside -> opening 0, added 12000, released 2000, closing 10000.
	inPeriod, err := svc.GetDeferredRollforward(ctx, tenantID, now.Add(-24*time.Hour), now.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("GetDeferredRollforward(in-period): %v", err)
	}
	if inPeriod.Opening != 0 || inPeriod.Added != 12000 || inPeriod.Released != 2000 || inPeriod.Closing != 10000 {
		t.Fatalf("in-period rollforward = %+v, want opening=0 added=12000 released=2000 closing=10000", inPeriod)
	}

	// Window entirely in the future: all activity is now "opening" -> opening
	// 10000 (12000 credit - 2000 debit), no in-period movement, closing 10000.
	future, err := svc.GetDeferredRollforward(ctx, tenantID, now.Add(time.Hour), now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("GetDeferredRollforward(future): %v", err)
	}
	if future.Opening != 10000 || future.Added != 0 || future.Released != 0 || future.Closing != 10000 {
		t.Fatalf("future-window rollforward = %+v, want opening=10000 added=0 released=0 closing=10000", future)
	}
}
