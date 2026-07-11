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

func openRevRecReportTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed revrec report test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func seedRevSchedule(t *testing.T, conn *sql.DB, tenantID, invoiceID uuid.UUID, currency string, total int64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO revenue_schedules (id, tenant_id, invoice_id, total_amount, currency, start_date, end_date, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), 'active', NOW(), NOW())`,
		id, tenantID, invoiceID, total, currency); err != nil {
		t.Fatalf("seed revenue_schedule: %v", err)
	}
	return id
}

func seedRecEvent(t *testing.T, conn *sql.DB, scheduleID, tenantID uuid.UUID, amount int64, date time.Time, status string) {
	t.Helper()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		uuid.New(), scheduleID, tenantID, amount, date, status); err != nil {
		t.Fatalf("seed recognition_event: %v", err)
	}
}

// TestRevRecReport_Rollforward verifies the deferred-revenue report: recognized
// in the requested period, the total deferred balance, the release schedule by
// month, and the per-currency split. All assertions are tenant-scoped, so the
// test is isolated from other data in a shared test database.
func TestRevRecReport_Rollforward(t *testing.T) {
	conn := openRevRecReportTestDB(t)
	repo := NewRevRecRepository(conn)
	ctx := context.Background()

	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)

	// USD schedule: 1000 recognized in Mar 2026, 1000 pending Apr, 1000 pending May.
	invUSD := seedInvoiceRow(t, conn, tenantID, customerID, 12000)
	schedUSD := seedRevSchedule(t, conn, tenantID, invUSD, "USD", 12000)
	seedRecEvent(t, conn, schedUSD, tenantID, 1000, time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), domain.RecognitionStatusRecognized)
	seedRecEvent(t, conn, schedUSD, tenantID, 1000, time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), domain.RecognitionStatusPending)
	seedRecEvent(t, conn, schedUSD, tenantID, 1000, time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), domain.RecognitionStatusPending)

	// EUR schedule: 2500 pending in Apr 2026.
	invEUR := seedInvoiceRow(t, conn, tenantID, customerID, 5000)
	schedEUR := seedRevSchedule(t, conn, tenantID, invEUR, "EUR", 5000)
	seedRecEvent(t, conn, schedEUR, tenantID, 2500, time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC), domain.RecognitionStatusPending)

	rep, err := repo.GetReport(ctx, tenantID, 3, 2026)
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}

	if rep.RecognizedAmount != 1000 {
		t.Errorf("RecognizedAmount = %d, want 1000 (recognized in Mar 2026)", rep.RecognizedAmount)
	}
	if rep.DeferredBalance != 4500 {
		t.Errorf("DeferredBalance = %d, want 4500 (1000+1000 USD + 2500 EUR still pending)", rep.DeferredBalance)
	}

	// Release schedule: Apr 2026 = 1000 USD + 2500 EUR = 3500, May 2026 = 1000.
	wantBuckets := []domain.DeferredRecognitionBucket{
		{Year: 2026, Month: 4, Amount: 3500},
		{Year: 2026, Month: 5, Amount: 1000},
	}
	if len(rep.Upcoming) != len(wantBuckets) {
		t.Fatalf("Upcoming = %+v, want %+v", rep.Upcoming, wantBuckets)
	}
	for i, w := range wantBuckets {
		if rep.Upcoming[i] != w {
			t.Errorf("Upcoming[%d] = %+v, want %+v", i, rep.Upcoming[i], w)
		}
	}

	// Currency split (ordered by currency): EUR 2500, USD 2000.
	wantCur := map[string]int64{"EUR": 2500, "USD": 2000}
	got := map[string]int64{}
	for _, c := range rep.ByCurrency {
		got[c.Currency] = c.Deferred
	}
	for cur, amt := range wantCur {
		if got[cur] != amt {
			t.Errorf("ByCurrency[%s] = %d, want %d (full split %+v)", cur, got[cur], amt, rep.ByCurrency)
		}
	}
}
