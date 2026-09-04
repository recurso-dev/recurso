package db_test

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

// TestClaimDueForRetry_ConcurrentClaimsDisjoint_Postgres proves the dunning retry
// worker is safe to run on many instances (Cloud Run scales out, and the Locker
// is a no-op without Redis): several ClaimDueForRetry calls racing over the SAME
// due invoices must take DISJOINT sets — an invoice claimed by two workers in one
// cycle would be re-collected (charged) twice. Runs under `-race`.
//
// The exclusivity is guaranteed by the next_retry_at LEASE (ADR-003): the claim
// atomically pushes each row's next_retry_at forward, so under READ COMMITTED a
// concurrent claimer's EvalPlanQual re-check sees the row as no-longer-due and
// drops it. FOR UPDATE SKIP LOCKED is a non-blocking optimization on top, NOT the
// correctness mechanism — verified: removing SKIP LOCKED keeps claims disjoint,
// while removing the lease bump makes concurrent workers double-claim (this test
// then fails reliably). A start barrier releases all workers at once so the claim
// genuinely contends.
func TestClaimDueForRetry_ConcurrentClaimsDisjoint_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed retry claim test")
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

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := conn.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("exec %q: %v", q, err)
		}
	}

	tenantID := uuid.New()
	exec(`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Retry-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	exec(`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New())

	// insertInvoice creates one invoice with the given dunning knobs.
	insertInvoice := func(status string, retryDue bool, managedBy string, paused bool) uuid.UUID {
		id := uuid.New()
		retry := "NOW() - INTERVAL '1 minute'"
		if !retryDue {
			retry = "NOW() + INTERVAL '1 hour'"
		}
		exec(`INSERT INTO invoices (id, tenant_id, customer_id, currency, subtotal, total, amount_paid, status, invoice_number, next_retry_at, dunning_managed_by, dunning_paused, created_at, due_date)
			VALUES ($1,$2,$3,'USD',10000,10000,0,$4,$5, `+retry+`, $6, $7, NOW(), NOW())`,
			id, tenantID, customerID, status, "RTRY-"+id.String()[:8], managedBy, paused)
		return id
	}

	const n = 12
	due := map[uuid.UUID]bool{}
	for i := 0; i < n; i++ {
		due[insertInvoice("past_due", true, "worker", false)] = true
	}
	// Excluded rows: paused dunning, gateway-managed cycle, not-yet-due, paid.
	insertInvoice("past_due", true, "worker", true)   // paused
	insertInvoice("past_due", true, "gateway", false) // not worker-managed
	insertInvoice("past_due", false, "worker", false) // retry in the future
	insertInvoice("paid", true, "worker", false)      // wrong status

	repo := db.NewInvoiceRepository(conn)

	// A start barrier releases both workers at the same instant so their claims
	// genuinely contend (otherwise one finishes and leases the rows before the
	// other starts, and the race is never exercised).
	const workers = 4
	var ready, wg sync.WaitGroup
	var mu sync.Mutex
	claimCount := map[uuid.UUID]int{}
	claimTenant := map[uuid.UUID]uuid.UUID{}
	start := make(chan struct{})
	ready.Add(workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ready.Done()
			<-start
			// Long lease so a claimed row can't come back due within the test, and
			// a limit no other tenant's leftovers in a shared database can crowd.
			invs, err := repo.ClaimDueForRetry(ctx, time.Hour, 1000)
			if err != nil {
				t.Errorf("ClaimDueForRetry: %v", err)
				return
			}
			mu.Lock()
			for _, inv := range invs {
				claimCount[inv.ID]++
				claimTenant[inv.ID] = inv.TenantID
			}
			mu.Unlock()
		}()
	}
	ready.Wait()
	close(start)
	wg.Wait()

	// No invoice was claimed by both workers (the double-charge invariant), and
	// only due worker-managed invoices were claimed. The sweep is deliberately
	// tenant-agnostic, so rows other tests left due in a shared database may
	// come back too; eligibility is judged for this tenant's rows only, while
	// the double-claim check stays global.
	claimedDue := 0
	for id, c := range claimCount {
		if c > 1 {
			t.Fatalf("invoice %s was claimed %d times — concurrent workers would double-charge it", id, c)
		}
		if claimTenant[id] != tenantID {
			continue
		}
		if !due[id] {
			t.Errorf("a non-eligible invoice %s was claimed", id)
		} else {
			claimedDue++
		}
	}
	// Together the two workers should have claimed every eligible invoice exactly
	// once (SKIP LOCKED partitions, it doesn't drop work).
	if claimedDue != n {
		t.Errorf("claimed %d of %d eligible due invoices across both workers, want all", claimedDue, n)
	}
}
