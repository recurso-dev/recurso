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

// TestWalletDrainFIFOAndExpiry_Postgres proves the B1 residue semantics:
// drains consume the oldest-EXPIRING residue first (so dated promotional
// credit is spent before it can lapse), expired residue is never drained,
// and the expiry sweep writes it off with a transaction + balance update.
func TestWalletDrainFIFOAndExpiry_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed wallet test")
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
		tenantID, "Wal-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com")
	customerID := uuid.New()
	mustExec(t, conn, `INSERT INTO customers (id, tenant_id, name, email, ledger_account_id, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`,
		customerID, tenantID, "Wal Cust", "w-"+customerID.String()[:8]+"@example.com", uuid.New())

	repo := db.NewWalletRepository(conn)
	now := time.Now().UTC()
	wallet := &domain.Wallet{
		ID: uuid.New(), TenantID: tenantID, CustomerID: customerID,
		Currency: "INR", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctx, wallet); err != nil {
		t.Fatalf("create wallet: %v", err)
	}

	topUp := func(amount int64, expires *time.Time) uuid.UUID {
		wtx := &domain.WalletTransaction{
			ID: uuid.New(), TenantID: tenantID, WalletID: wallet.ID,
			Type: domain.WalletTxTopUp, Source: domain.WalletSourcePromotional,
			Amount: amount, ExpiresAt: expires, CreatedAt: time.Now().UTC(),
		}
		if err := repo.TopUp(ctx, wtx); err != nil {
			t.Fatalf("top-up: %v", err)
		}
		return wtx.ID
	}
	soon := now.Add(24 * time.Hour)
	gone := now.Add(-time.Hour)
	expiredID := topUp(10000, &gone) // already expired: never drainable
	datedID := topUp(20000, &soon)   // expires tomorrow: drained FIRST
	openID := topUp(30000, nil)      // no expiry: drained last

	// Balance counts every residue until the sweep writes off the expired one.
	w, _ := repo.GetByID(ctx, tenantID, wallet.ID)
	if w.Balance != 60000 {
		t.Fatalf("balance = %d, want 60000", w.Balance)
	}

	// Drain 25000: all 20000 of the dated residue + 5000 of the open one;
	// the expired residue is untouched.
	drained, err := repo.Drain(ctx, tenantID, wallet.ID, 25000, uuid.New(), time.Now().UTC())
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if drained != 25000 {
		t.Fatalf("drained = %d, want 25000", drained)
	}
	remaining := func(id uuid.UUID) int64 {
		var rem int64
		if err := conn.QueryRowContext(ctx, `SELECT remaining FROM wallet_transactions WHERE id = $1`, id).Scan(&rem); err != nil {
			t.Fatalf("read residue: %v", err)
		}
		return rem
	}
	if remaining(datedID) != 0 || remaining(openID) != 25000 || remaining(expiredID) != 10000 {
		t.Fatalf("residues dated/open/expired = %d/%d/%d, want 0/25000/10000",
			remaining(datedID), remaining(openID), remaining(expiredID))
	}

	// The expiry sweep writes off the expired residue.
	touched, err := repo.ExpireOverdue(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if touched != 1 {
		t.Fatalf("expiry touched %d wallets, want 1", touched)
	}
	if remaining(expiredID) != 0 {
		t.Fatal("expired residue must be zeroed by the sweep")
	}
	w, _ = repo.GetByID(ctx, tenantID, wallet.ID)
	if w.Balance != 25000 {
		t.Fatalf("post-sweep balance = %d, want 25000 (60000 - 25000 drained - 10000 expired)", w.Balance)
	}

	// Draining more than remains empties the wallet exactly.
	drained, err = repo.Drain(ctx, tenantID, wallet.ID, 999999, uuid.New(), time.Now().UTC())
	if err != nil || drained != 25000 {
		t.Fatalf("final drain = %d (err %v), want 25000", drained, err)
	}
	w, _ = repo.GetByID(ctx, tenantID, wallet.ID)
	if w.Balance != 0 {
		t.Fatalf("final balance = %d, want 0", w.Balance)
	}
}
