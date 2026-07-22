package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/recurso-dev/recurso/internal/adapter/db"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestWalletExpiry_PostsLedgerLeg_Postgres proves the expiry sweep now discharges
// the Customer-Credit liability in the GL: when promotional residue lapses, the
// wallet is written off AND a ledger leg (code 15, DR Customer Credit / CR
// Credits) is posted — previously the wallet shrank but the GL liability was
// left overstated.
func TestWalletExpiry_PostsLedgerLeg_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed wallet-expiry-ledger test")
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
		tenantID, "WE-"+run, "we-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
		customerID, tenantID, "WE Cust", "we-"+run+"@c.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	var entityID uuid.UUID
	if err := conn.QueryRowContext(ctx,
		`SELECT id FROM entities WHERE tenant_id=$1 AND is_primary`, tenantID).Scan(&entityID); err != nil {
		t.Fatalf("primary entity: %v", err)
	}

	walletRepo := db.NewWalletRepository(conn)
	ledger := NewLedgerService(nil, db.NewLedgerRepository(conn))
	ledger.SetEntityReader(db.NewEntityRepository(conn))
	svc := NewWalletService(walletRepo, nil, ledger)
	svc.now = func() time.Time { return time.Now().UTC() }

	now := time.Now().UTC()
	wallet := &domain.Wallet{
		ID: uuid.New(), TenantID: tenantID, EntityID: entityID, CustomerID: customerID,
		Currency: "INR", CreatedAt: now, UpdatedAt: now,
	}
	if err := walletRepo.Create(ctx, wallet); err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	// Promotional top-up that already expired.
	past := now.Add(-time.Hour)
	if err := walletRepo.TopUp(ctx, &domain.WalletTransaction{
		ID: uuid.New(), TenantID: tenantID, WalletID: wallet.ID, Type: domain.WalletTxTopUp,
		Source: domain.WalletSourcePromotional, Amount: 15000, ExpiresAt: &past, CreatedAt: now,
	}); err != nil {
		t.Fatalf("promo top-up: %v", err)
	}

	touched, err := svc.ExpireOverdueCredits(ctx)
	if err != nil {
		t.Fatalf("ExpireOverdueCredits: %v", err)
	}
	if touched != 1 {
		t.Errorf("touched = %d, want 1", touched)
	}
	if after, _ := walletRepo.GetByID(ctx, tenantID, wallet.ID); after.Balance != 0 {
		t.Errorf("wallet balance after expiry = %d, want 0", after.Balance)
	}

	// The GL now carries the discharging leg (code 15) for the written-off amount.
	var expiryPosted int64
	_ = conn.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount),0) FROM ledger_transactions WHERE code=$1`, domain.LedgerCodeWalletExpiry).Scan(&expiryPosted)
	if expiryPosted != 15000 {
		t.Errorf("ledger expiry total = %d, want 15000 (liability discharged)", expiryPosted)
	}
}
