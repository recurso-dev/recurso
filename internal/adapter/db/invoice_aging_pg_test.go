package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func openAgingTestDB(t *testing.T) (*InvoiceRepository, *sql.DB) {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed invoice aging test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	repo, ok := NewInvoiceRepository(conn).(*InvoiceRepository)
	if !ok {
		t.Fatalf("NewInvoiceRepository did not return *InvoiceRepository")
	}
	return repo, conn
}

func seedAgingInvoice(t *testing.T, conn *sql.DB, tenantID, customerID uuid.UUID, total, paid int64, status string, due time.Time) {
	t.Helper()
	id := uuid.New()
	if _, err := conn.ExecContext(context.Background(),
		`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date)
		 VALUES ($1, $2, $3, 'USD', $4, $4, $5, 0, $6, $7, NOW(), $8)`,
		id, tenantID, customerID, total, paid, status, "INV-"+id.String()[:8], due); err != nil {
		t.Fatalf("seed aging invoice: %v", err)
	}
}

func TestInvoiceAging_Buckets(t *testing.T) {
	repo, conn := openAgingTestDB(t)
	ctx := context.Background()
	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)
	now := time.Now()

	// One invoice per bucket (+ excluded rows), outstanding = total - amount_paid.
	seedAgingInvoice(t, conn, tenantID, customerID, 1000, 0, "open", now.Add(10*24*time.Hour))       // current (not due)
	seedAgingInvoice(t, conn, tenantID, customerID, 2000, 0, "past_due", now.Add(-10*24*time.Hour))  // 1-30
	seedAgingInvoice(t, conn, tenantID, customerID, 3000, 500, "open", now.Add(-45*24*time.Hour))    // 31-60, outstanding 2500
	seedAgingInvoice(t, conn, tenantID, customerID, 4000, 0, "past_due", now.Add(-120*24*time.Hour)) // 90+
	// Excluded: fully paid (amount_remaining 0) and a draft.
	seedAgingInvoice(t, conn, tenantID, customerID, 5000, 5000, "paid", now.Add(-200*24*time.Hour))
	seedAgingInvoice(t, conn, tenantID, customerID, 6000, 0, "draft", now.Add(-5*24*time.Hour))

	rows, err := repo.GetInvoiceAgingRows(ctx, tenantID, nil)
	if err != nil {
		t.Fatalf("GetInvoiceAgingRows: %v", err)
	}

	got := map[string]struct {
		count int
		amt   int64
	}{}
	for _, r := range rows {
		got[r.Bucket] = struct {
			count int
			amt   int64
		}{r.Count, r.Amount}
	}
	want := map[string]int64{"current": 1000, "1-30": 2000, "31-60": 2500, "90+": 4000}
	for bucket, amt := range want {
		if got[bucket].amt != amt || got[bucket].count != 1 {
			t.Errorf("bucket %s = count %d / amt %d, want count 1 / amt %d", bucket, got[bucket].count, got[bucket].amt, amt)
		}
	}
	if _, ok := got["61-90"]; ok {
		t.Errorf("61-90 should be empty, got %+v", got["61-90"])
	}
	// Paid + draft excluded → exactly 4 buckets populated.
	if len(rows) != 4 {
		t.Errorf("got %d bucket rows, want 4 (paid + draft excluded): %+v", len(rows), rows)
	}
}

// TestInvoiceAging_PerEntity proves the aging query filters by legal entity: a
// concrete entity returns only its receivables; nil returns the whole tenant.
func TestInvoiceAging_PerEntity(t *testing.T) {
	repo, conn := openAgingTestDB(t)
	ctx := context.Background()
	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)
	now := time.Now()

	var entA, entB uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT id FROM entities WHERE tenant_id=$1 AND is_primary=TRUE`, tenantID).Scan(&entA); err != nil {
		t.Fatalf("load primary entity: %v", err)
	}
	if err := conn.QueryRowContext(ctx,
		`INSERT INTO entities (tenant_id, name, is_primary, tb_ledger_id, invoice_prefix) VALUES ($1,'Branch',FALSE,2,$2) RETURNING id`,
		tenantID, "AGBR"+uuid.New().String()[:4]).Scan(&entB); err != nil {
		t.Fatalf("seed entity B: %v", err)
	}

	// stampAging inserts an open invoice and assigns it to an entity.
	stampAging := func(total int64, ent uuid.UUID) {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, entity_id, created_at, due_date)
			 VALUES ($1,$2,$3,'USD',$4,$4,0,0,'past_due',$5,$6,NOW(),$7)`,
			id, tenantID, customerID, total, "AG-"+id.String()[:8], ent, now.Add(-10*24*time.Hour)); err != nil {
			t.Fatalf("stamp aging invoice: %v", err)
		}
	}
	stampAging(1000, entA)
	stampAging(2000, entA)
	stampAging(4000, entB)

	total := func(entity *uuid.UUID) (int64, int) {
		rows, err := repo.GetInvoiceAgingRows(ctx, tenantID, entity)
		if err != nil {
			t.Fatalf("GetInvoiceAgingRows: %v", err)
		}
		var amt int64
		var cnt int
		for _, r := range rows {
			amt += r.Amount
			cnt += r.Count
		}
		return amt, cnt
	}

	if amt, cnt := total(&entA); amt != 3000 || cnt != 2 {
		t.Errorf("entity A aging = %d/%d, want 3000/2", amt, cnt)
	}
	if amt, cnt := total(&entB); amt != 4000 || cnt != 1 {
		t.Errorf("entity B aging = %d/%d, want 4000/1", amt, cnt)
	}
	if amt, cnt := total(nil); amt != 7000 || cnt != 3 {
		t.Errorf("consolidated aging = %d/%d, want 7000/3", amt, cnt)
	}
}

// TestOutstandingByEntity_Postgres proves the multi-entity AR aggregate: open AR
// summed per entity + currency, NULL entity_id returned as-is, paid/covered
// excluded.
func TestOutstandingByEntity_Postgres(t *testing.T) {
	repo, conn := openAgingTestDB(t)
	ctx := context.Background()
	tenantID, customerID := seedCreditAppTenantCustomer(t, conn)
	now := time.Now()

	var entB uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`INSERT INTO entities (tenant_id, name, is_primary, tb_ledger_id, invoice_prefix) VALUES ($1,'Branch',FALSE,2,$2) RETURNING id`,
		tenantID, "OBE"+uuid.New().String()[:4]).Scan(&entB); err != nil {
		t.Fatalf("seed entity B: %v", err)
	}

	mk := func(total int64, ent *uuid.UUID, status string) {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, credit_applied, status, invoice_number, entity_id, created_at, due_date)
			 VALUES ($1,$2,$3,'USD',$4,$4,0,0,$5,$6,$7,NOW(),$8)`,
			id, tenantID, customerID, total, status, "OBE-"+id.String()[:8], ent, now.Add(-5*24*time.Hour)); err != nil {
			t.Fatalf("seed invoice: %v", err)
		}
	}
	mk(10000, &entB, "past_due")
	mk(5000, &entB, "open")
	mk(3000, nil, "past_due") // NULL entity_id
	mk(9000, &entB, "paid")   // excluded (paid, amount_remaining 0 since paid sets... actually status paid)

	rows, err := repo.GetOutstandingByEntity(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetOutstandingByEntity: %v", err)
	}
	var entBAmt, nullAmt int64
	for _, r := range rows {
		if r.EntityID != nil && *r.EntityID == entB {
			entBAmt += r.Amount
		}
		if r.EntityID == nil {
			nullAmt += r.Amount
		}
	}
	if entBAmt != 15000 {
		t.Errorf("entity B open AR = %d, want 15000", entBAmt)
	}
	if nullAmt != 3000 {
		t.Errorf("NULL-entity open AR = %d, want 3000", nullAmt)
	}
}
