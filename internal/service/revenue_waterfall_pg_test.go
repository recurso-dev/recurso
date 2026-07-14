package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// seedScheduleWithEvents inserts an active schedule for (tenant, invoice, sub)
// and the given events verbatim (fixed dates/status), so the waterfall
// assertions are deterministic regardless of run date.
func seedScheduleWithEvents(t *testing.T, conn *sql.DB, tenantID, invoiceID, subID uuid.UUID, events []struct {
	Date   string
	Amount int64
	Status string
}) {
	t.Helper()
	ctx := context.Background()
	schedID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO revenue_schedules (id, tenant_id, invoice_id, subscription_id, total_amount, currency, start_date, end_date, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,'USD', '2026-01-01', '2026-04-01', 'active', NOW(), NOW())`,
		schedID, tenantID, invoiceID, subID, int64(100000)); err != nil {
		t.Fatalf("seed schedule: %v", err)
	}
	for i, e := range events {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO recognition_events (id, revenue_schedule_id, tenant_id, amount, recognition_date, status, created_at)
			 VALUES ($1,$2,$3,$4,$5::timestamptz,$6, NOW())`,
			uuid.New(), schedID, tenantID, e.Amount, e.Date, e.Status); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}
}

// TestRevenueWaterfall_Postgres proves the monthly curve: recognized events
// roll into the recognized column, pending into scheduled, both bucketed by the
// month of recognition_date; canceled/failed are excluded; and it is
// tenant-scoped.
func TestRevenueWaterfall_Postgres(t *testing.T) {
	conn := openRevRecTestDB(t)
	defer func() { _ = conn.Close() }()
	tenantID := seedRevRecTenant(t, conn)

	subID, invID, _ := seedSubAndInvoice(t, conn, tenantID, 100000)
	seedScheduleWithEvents(t, conn, tenantID, invID, subID, []struct {
		Date   string
		Amount int64
		Status string
	}{
		{"2026-01-15", 5000, domain.RecognitionStatusRecognized},
		{"2026-01-20", 3000, domain.RecognitionStatusRecognized}, // same month -> 8000 recognized
		{"2026-02-10", 10000, domain.RecognitionStatusPending},
		{"2026-03-10", 12000, domain.RecognitionStatusPending},
		{"2026-02-01", 9999, domain.RecognitionStatusCanceled}, // must be excluded
	})

	// A second tenant with its own events must not leak into tenant 1's curve.
	other := seedRevRecTenant(t, conn)
	oSub, oInv, _ := seedSubAndInvoice(t, conn, other, 50000)
	seedScheduleWithEvents(t, conn, other, oInv, oSub, []struct {
		Date   string
		Amount int64
		Status string
	}{
		{"2026-01-15", 77777, domain.RecognitionStatusRecognized},
	})

	svc := NewRevRecService(db.NewRevRecRepository(conn), nil, nil)
	w, err := svc.GetWaterfall(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetWaterfall: %v", err)
	}

	if w.TotalRecognized != 8000 || w.TotalScheduled != 22000 {
		t.Fatalf("totals = R%d/S%d, want R8000/S22000 (canceled excluded, other tenant excluded)", w.TotalRecognized, w.TotalScheduled)
	}
	if len(w.Buckets) != 3 {
		t.Fatalf("got %d monthly buckets, want 3 (Jan/Feb/Mar 2026)", len(w.Buckets))
	}
	jan := w.Buckets[0]
	if jan.Year != 2026 || jan.Month != 1 || jan.Recognized != 8000 || jan.Scheduled != 0 {
		t.Errorf("Jan bucket = %+v, want {2026 1 recognized=8000 scheduled=0}", jan)
	}
	feb := w.Buckets[1]
	if feb.Month != 2 || feb.Scheduled != 10000 || feb.Recognized != 0 {
		t.Errorf("Feb bucket = %+v, want {month=2 scheduled=10000 recognized=0}", feb)
	}
}
