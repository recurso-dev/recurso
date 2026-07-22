package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestWalletClose_RefundForfeitAndLedger_Postgres proves wallet closure: paid
// residue is refunded to the customer, promotional residue is forfeited, the
// wallet is zeroed + marked closed (and excluded from the drainable lookup), the
// ledger legs post (DR Customer Credit / CR Cash for the refund; DR Customer
// Credit / CR Credits for the forfeit), and a second close is rejected.
func TestWalletClose_RefundForfeitAndLedger_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed wallet-close test")
	}
	if err := db.RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	dbx, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = dbx.Close() }()
	conn := dbx.DB
	ctx := context.Background()
	run := uuid.NewString()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "WC-"+run, "wc-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
		customerID, tenantID, "WC Cust", "wc-"+run+"@c.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	var entityID uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT id FROM entities WHERE tenant_id=$1 AND is_primary`, tenantID).Scan(&entityID); err != nil {
		t.Fatalf("primary entity: %v", err)
	}

	walletRepo := db.NewWalletRepository(conn)
	entityRepo := db.NewEntityRepository(conn)
	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(entityRepo)
	svc := NewWalletService(walletRepo, nil, ledger)

	now := time.Now().UTC()
	wallet := &domain.Wallet{
		ID: uuid.New(), TenantID: tenantID, EntityID: entityID, CustomerID: customerID,
		Currency: "INR", CreatedAt: now, UpdatedAt: now,
	}
	if err := walletRepo.Create(ctx, wallet); err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	// 50000 paid (refundable) + 20000 promotional (forfeit).
	if err := walletRepo.TopUp(ctx, &domain.WalletTransaction{
		ID: uuid.New(), TenantID: tenantID, WalletID: wallet.ID, Type: domain.WalletTxTopUp,
		Source: domain.WalletSourceManual, Amount: 50000, CreatedAt: now,
	}); err != nil {
		t.Fatalf("manual top-up: %v", err)
	}
	if err := walletRepo.TopUp(ctx, &domain.WalletTransaction{
		ID: uuid.New(), TenantID: tenantID, WalletID: wallet.ID, Type: domain.WalletTxTopUp,
		Source: domain.WalletSourcePromotional, Amount: 20000, CreatedAt: now,
	}); err != nil {
		t.Fatalf("promo top-up: %v", err)
	}

	res, err := svc.CloseWallet(ctx, tenantID, wallet.ID)
	if err != nil {
		t.Fatalf("CloseWallet: %v", err)
	}
	if res.Refunded != 50000 {
		t.Errorf("refunded = %d, want 50000", res.Refunded)
	}
	if res.Forfeited != 20000 {
		t.Errorf("forfeited = %d, want 20000", res.Forfeited)
	}

	// Wallet zeroed + closed, and no longer drainable.
	after, _ := walletRepo.GetByID(ctx, tenantID, wallet.ID)
	if after.Balance != 0 || after.ClosedAt == nil {
		t.Errorf("wallet after close: balance=%d closed_at=%v, want 0 / set", after.Balance, after.ClosedAt)
	}
	if drainable, _ := walletRepo.GetByCustomerEntityAndCurrency(ctx, tenantID, customerID, entityID, "INR"); drainable != nil {
		t.Errorf("closed wallet still returned by the drainable lookup")
	}

	// Ledger legs: refund (code 13) = 50000, forfeit (code 14) = 20000.
	var refundAmt, forfeitAmt int64
	_ = conn.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM ledger_transactions WHERE code=$1`, domain.LedgerCodeWalletRefund).Scan(&refundAmt)
	_ = conn.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM ledger_transactions WHERE code=$1`, domain.LedgerCodeWalletForfeit).Scan(&forfeitAmt)
	if refundAmt != 50000 {
		t.Errorf("ledger refund total = %d, want 50000", refundAmt)
	}
	if forfeitAmt != 20000 {
		t.Errorf("ledger forfeit total = %d, want 20000", forfeitAmt)
	}

	// Double close is rejected.
	if _, err := svc.CloseWallet(ctx, tenantID, wallet.ID); err == nil {
		t.Errorf("second close should fail, got nil")
	} else if _, ok := err.(WalletValidationError); !ok && !errors.Is(err, ErrWalletNotFound) {
		t.Errorf("second close error = %T %v, want a validation error", err, err)
	}
}
