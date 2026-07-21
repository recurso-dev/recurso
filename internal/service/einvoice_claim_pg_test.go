package service

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/adapter/db"
)

// TestClaimFailedEInvoices_ExclusiveAndLeased_Postgres proves the ENG-200 guard:
// two concurrent retry runners never both claim the same failed e-invoice (no
// duplicate government IRN submission), and a claimed row is leased forward so
// an immediate re-claim sees nothing.
func TestClaimFailedEInvoices_ExclusiveAndLeased_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed e-invoice claim test")
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
	mustExec(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "EIClaim-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	mustExec(t, conn, `INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
		customerID, tenantID, "EI Cust", "ei-"+customerID.String()[:8]+"@example.com", uuid.New())

	const n = 8
	for i := 0; i < n; i++ {
		invID := uuid.New()
		mustExec(t, conn, `INSERT INTO invoices
			(id, tenant_id, customer_id, currency, subtotal, tax_amount, total, amount_paid, credit_applied, status, invoice_number, created_at, due_date, e_invoice_status, e_invoice_next_retry_at)
			VALUES ($1,$2,$3,'INR',10000,1800,11800,11800,0,'paid',$4, NOW(), NOW(), 'FAILED', (NOW() AT TIME ZONE 'UTC') - INTERVAL '1 minute')`,
			invID, tenantID, customerID, "INV-EI-"+invID.String()[:8])
	}

	repo := db.NewInvoiceRepository(conn)
	now := time.Now().UTC()
	lease := now.Add(15 * time.Minute)

	var wg sync.WaitGroup
	var mu sync.Mutex
	claimed := map[uuid.UUID]int{}
	for r := 0; r < 2; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			invs, err := repo.ClaimFailedEInvoices(ctx, now, lease, n)
			if err != nil {
				t.Errorf("ClaimFailedEInvoices: %v", err)
				return
			}
			mu.Lock()
			for _, inv := range invs {
				claimed[inv.ID]++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(claimed) != n {
		t.Fatalf("claimed %d distinct e-invoices, want %d (some lost)", len(claimed), n)
	}
	for id, c := range claimed {
		if c != 1 {
			t.Errorf("e-invoice %s claimed %d times, want exactly 1 (duplicate IRN submission)", id, c)
		}
	}

	again, err := repo.ClaimFailedEInvoices(ctx, now, lease, n)
	if err != nil {
		t.Fatalf("re-claim: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("re-claim returned %d rows, want 0 (lease should hide claimed rows)", len(again))
	}
}
