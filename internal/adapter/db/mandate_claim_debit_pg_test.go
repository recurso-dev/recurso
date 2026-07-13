package db

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func openMandateClaimTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed mandate-claim test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return dbx
}

// seedDueMandate inserts an active, pre-notified mandate whose next_debit_at is
// already in the past, i.e. eligible for ClaimDueForDebit.
func seedDueMandate(t *testing.T, conn *sql.DB) (tenantID, mandateID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	tenantID = uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "MD-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	mandateID = uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO mandates (id, tenant_id, customer_id, mandate_type, payment_method, vpa,
			razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status, pre_debit_notified, next_debit_at, created_at, updated_at)
		 VALUES ($1, $2, $3, 'upi', 'upi', 'test@upi', $4, '', '', 50000, 'monthly', 'active', TRUE, NOW() - INTERVAL '1 minute', NOW(), NOW())`,
		mandateID, tenantID, customerID, "token_"+mandateID.String()[:8]); err != nil {
		t.Fatalf("seed mandate: %v", err)
	}
	return tenantID, mandateID
}

// TestClaimDueForDebit_ConcurrentClaimsAreDisjoint proves the ENG-161 fix: when
// many runners claim the same due mandate at once (as multi-instance schedulers
// do when the distributed lock is a no-op), the atomic UPDATE ... RETURNING
// hands the mandate to EXACTLY ONE runner. Before the fix, every runner's
// unscoped GetReadyForDebit returned the mandate and each charged it.
func TestClaimDueForDebit_ConcurrentClaimsAreDisjoint(t *testing.T) {
	dbx := openMandateClaimTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	repo := NewMandateRepository(conn)
	ctx := context.Background()

	_, mandateID := seedDueMandate(t, conn)

	const runners = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	var mu sync.Mutex
	claimedBy := 0    // how many runners got the mandate
	totalClaimed := 0 // total mandate rows returned across all runners

	for i := 0; i < runners; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release all goroutines at once to maximize contention
			got, err := repo.ClaimDueForDebit(ctx, 15*time.Minute)
			if err != nil {
				t.Errorf("ClaimDueForDebit: %v", err)
				return
			}
			mu.Lock()
			totalClaimed += len(got)
			for _, m := range got {
				if m.ID == mandateID {
					claimedBy++
				}
			}
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	if claimedBy != 1 {
		t.Fatalf("mandate was claimed by %d runners, want exactly 1 (double-charge race)", claimedBy)
	}
	if totalClaimed != 1 {
		t.Fatalf("total claimed rows = %d, want exactly 1", totalClaimed)
	}

	// The winning claim must have pushed next_debit_at into the future, so a
	// later tick (or a concurrent runner) no longer sees it as due.
	var nextDebit time.Time
	if err := conn.QueryRowContext(ctx, `SELECT next_debit_at FROM mandates WHERE id = $1`, mandateID).Scan(&nextDebit); err != nil {
		t.Fatalf("read next_debit_at: %v", err)
	}
	if !nextDebit.After(time.Now()) {
		t.Errorf("next_debit_at = %v, want a future time (claim lease not applied)", nextDebit)
	}

	// A follow-up claim finds nothing due — the row is leased, not re-charged.
	again, err := repo.ClaimDueForDebit(ctx, 15*time.Minute)
	if err != nil {
		t.Fatalf("second ClaimDueForDebit: %v", err)
	}
	for _, m := range again {
		if m.ID == mandateID {
			t.Error("mandate was re-claimed while leased — would double-charge")
		}
	}
}

// TestClaimDueForDebit_SkipsIneligible confirms the claim honors the same
// eligibility as the read path: not-yet-due, not pre-notified, and inactive
// mandates are never claimed.
func TestClaimDueForDebit_SkipsIneligible(t *testing.T) {
	dbx := openMandateClaimTestDB(t)
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	repo := NewMandateRepository(conn)
	ctx := context.Background()

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "MDI-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, customerID.String()[:8]+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	insert := func(status string, preNotified bool, nextDebit string) uuid.UUID {
		id := uuid.New()
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO mandates (id, tenant_id, customer_id, mandate_type, payment_method, vpa,
				razorpay_token_id, razorpay_subscription_id, razorpay_customer_id, max_amount, frequency, status, pre_debit_notified, next_debit_at, created_at, updated_at)
			 VALUES ($1,$2,$3,'upi','upi','x@upi',$4,'','',50000,'monthly',$5,$6,`+nextDebit+`, NOW(), NOW())`,
			id, tenantID, customerID, "tok_"+id.String()[:8], status, preNotified); err != nil {
			t.Fatalf("seed mandate: %v", err)
		}
		return id
	}

	notDue := insert("active", true, "NOW() + INTERVAL '1 day'")
	notNotified := insert("active", false, "NOW() - INTERVAL '1 minute'")
	inactive := insert("revoked", true, "NOW() - INTERVAL '1 minute'")

	claimed, err := repo.ClaimDueForDebit(ctx, 15*time.Minute)
	if err != nil {
		t.Fatalf("ClaimDueForDebit: %v", err)
	}
	for _, m := range claimed {
		switch m.ID {
		case notDue:
			t.Error("claimed a mandate whose next_debit_at is in the future")
		case notNotified:
			t.Error("claimed a mandate that was not pre-notified")
		case inactive:
			t.Error("claimed a non-active mandate")
		}
	}
}
