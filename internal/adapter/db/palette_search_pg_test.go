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
	"github.com/recurso-dev/recurso/internal/core/port"
)

// openSearchTestDB is the shared skip/migrate/open for the command-palette search
// tests (Batch F2). Returns nil when TEST_DATABASE_URL is unset (test skips).
func openSearchTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed search test")
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

func mkInvoice(t *testing.T, repo port.InvoiceRepository, tenantID, customerID uuid.UUID, number string, when time.Time) uuid.UUID {
	t.Helper()
	inv := &domain.Invoice{
		ID:            uuid.New(),
		TenantID:      tenantID,
		CustomerID:    customerID,
		InvoiceNumber: number,
		Currency:      "USD",
		Subtotal:      1000,
		Total:         1000,
		Status:        domain.InvoiceStatusOpen,
		CreatedAt:     when,
		DueDate:       when,
	}
	if err := repo.Create(context.Background(), inv); err != nil {
		t.Fatalf("create invoice %s: %v", number, err)
	}
	return inv.ID
}

// TestInvoiceRepository_SearchPaginated — command-palette invoice lookup:
// case-insensitive invoice_number match, tenant-isolated, paginated, empty-q safe.
func TestInvoiceRepository_SearchPaginated(t *testing.T) {
	conn := openSearchTestDB(t)
	repo := NewInvoiceRepository(conn)
	ctx := context.Background()

	t1, c1 := seedCreditAppTenantCustomer(t, conn)
	t2, c2 := seedCreditAppTenantCustomer(t, conn)
	base := time.Now().Add(-time.Hour)

	// Tenant 1: two matching invoices + one non-matching. Distinct suffixes so a
	// reused test DB can't collide across runs.
	run := t1.String()[:8]
	mkInvoice(t, repo, t1, c1, "INV-FIND-"+run+"-A", base.Add(1*time.Minute))
	mkInvoice(t, repo, t1, c1, "INV-FIND-"+run+"-B", base.Add(2*time.Minute))
	mkInvoice(t, repo, t1, c1, "MISC-"+run+"-Z", base)
	// Tenant 2: a same-prefix invoice that must NEVER leak into tenant 1's search.
	mkInvoice(t, repo, t2, c2, "INV-FIND-"+t2.String()[:8]+"-T2", base.Add(3*time.Minute))

	q := "inv-find-" + run // lowercase → proves case-insensitivity (ILIKE)

	got, err := repo.SearchPaginated(ctx, t1, q, 10, 0)
	if err != nil {
		t.Fatalf("SearchPaginated: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("search returned %d invoices, want 2 (case-insensitive, own-tenant only)", len(got))
	}
	for _, inv := range got {
		if inv.TenantID != t1 {
			t.Errorf("search leaked a foreign tenant's invoice: %s", inv.ID)
		}
	}
	// newest-first
	if got[0].InvoiceNumber != "INV-FIND-"+run+"-B" {
		t.Errorf("search not newest-first: got %s first", got[0].InvoiceNumber)
	}

	count, err := repo.CountSearch(ctx, t1, q)
	if err != nil || count != 2 {
		t.Fatalf("CountSearch = %d (err %v), want 2", count, err)
	}

	// Tenant isolation: tenant 2's identical-prefix search sees only its own.
	got2, _ := repo.SearchPaginated(ctx, t2, "inv-find", 10, 0)
	for _, inv := range got2 {
		if inv.TenantID != t2 {
			t.Errorf("tenant 2 search leaked tenant %s", inv.TenantID)
		}
	}

	// Pagination: limit 1 returns 1; offset advances.
	page1, _ := repo.SearchPaginated(ctx, t1, q, 1, 0)
	page2, _ := repo.SearchPaginated(ctx, t1, q, 1, 1)
	if len(page1) != 1 || len(page2) != 1 || page1[0].ID == page2[0].ID {
		t.Errorf("pagination broken: page1=%d page2=%d overlap=%v", len(page1), len(page2), page1[0].ID == page2[0].ID)
	}

	// Empty query returns nothing (never an unbounded scan).
	empty, _ := repo.SearchPaginated(ctx, t1, "", 10, 0)
	if len(empty) != 0 {
		t.Errorf("empty search returned %d rows, want 0", len(empty))
	}
	if n, _ := repo.CountSearch(ctx, t1, ""); n != 0 {
		t.Errorf("empty CountSearch = %d, want 0", n)
	}
}

// TestPaymentAttemptRepository_SearchList — command-palette payment lookup:
// matches invoice_number OR gateway reference, tenant-isolated, empty-q safe.
func TestPaymentAttemptRepository_SearchList(t *testing.T) {
	conn := openSearchTestDB(t)
	repo := NewPaymentAttemptRepository(conn)
	ctx := context.Background()

	mkPay := func(tenantID uuid.UUID, invoiceNumber, pi string) {
		t.Helper()
		invoiceID := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO invoices (id, tenant_id, currency, subtotal, total, invoice_number, status, created_at)
			 VALUES ($1,$2,'USD',1000,1000,$3,'open',NOW())`,
			invoiceID, tenantID, invoiceNumber); err != nil {
			t.Fatalf("seed invoice %s: %v", invoiceNumber, err)
		}
		att := &domain.PaymentAttempt{
			TenantID:               tenantID,
			InvoiceID:              invoiceID,
			Gateway:                "stripe",
			Method:                 "card",
			GatewayPaymentIntentID: pi,
			Status:                 domain.PaymentAttemptProcessing,
			Amount:                 1000,
		}
		if err := repo.Create(ctx, att); err != nil {
			t.Fatalf("create attempt %s: %v", pi, err)
		}
	}

	t1, _ := seedCreditAppTenantCustomer(t, conn)
	t2, _ := seedCreditAppTenantCustomer(t, conn)
	r1, r2 := t1.String()[:8], t2.String()[:8]

	mkPay(t1, "INV-PAY-"+r1, "pi_findme_"+r1)      // matches by invoice number AND by gateway ref
	mkPay(t1, "INV-OTHER-"+r1, "pi_unrelated_"+r1) // non-matching
	mkPay(t2, "INV-PAY-"+r2, "pi_t2_"+r2)          // tenant 2 — must not leak

	// Match by invoice number (case-insensitive), own tenant only.
	byInv, _, err := repo.SearchList(ctx, t1, "inv-pay-"+r1, 10, 0)
	if err != nil {
		t.Fatalf("SearchList by invoice: %v", err)
	}
	if len(byInv) != 1 || byInv[0].TenantID != t1 || byInv[0].InvoiceNumber != "INV-PAY-"+r1 {
		t.Fatalf("invoice-number search = %+v, want the single own-tenant match", byInv)
	}

	// Match by gateway reference.
	byRef, _, _ := repo.SearchList(ctx, t1, "findme_"+r1, 10, 0)
	if len(byRef) != 1 || byRef[0].GatewayPaymentIntentID != "pi_findme_"+r1 {
		t.Fatalf("gateway-ref search = %+v, want the pi_findme match", byRef)
	}

	// Tenant isolation: tenant 1 must not see tenant 2's INV-PAY.
	for _, it := range byInv {
		if it.TenantID == t2 {
			t.Errorf("payment search leaked tenant 2")
		}
	}

	// Empty query returns nothing.
	empty, total, _ := repo.SearchList(ctx, t1, "", 10, 0)
	if len(empty) != 0 || total != 0 {
		t.Errorf("empty payment search = %d rows / total %d, want 0/0", len(empty), total)
	}
}
