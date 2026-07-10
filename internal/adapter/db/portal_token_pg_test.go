package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TestPortalMagicLink_HashedAndSingleUse_Postgres proves the ENG-145 fix: the
// magic-link token is stored as its SHA-256 (a DB read yields no usable token),
// lookups work by hashing the presented token, and MarkUsed is a single-use
// guard — the second concurrent claim loses.
func TestPortalMagicLink_HashedAndSingleUse_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed portal-token test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()
	run := uuid.New().String()[:8]

	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "Portal-"+run, "portal-"+run+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	customerID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO customers (id, tenant_id, email, ledger_account_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		customerID, tenantID, "cust-"+run+"@t.com", uuid.New()); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	repo := NewMagicLinkRepository(conn)
	rawToken := GenerateSecureToken()
	link := &domain.MagicLink{
		ID: uuid.New(), CustomerID: customerID, Token: rawToken,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	if err := repo.Create(ctx, link); err != nil {
		t.Fatalf("Create magic link: %v", err)
	}

	// At rest it must be the SHA-256, never the raw token.
	var stored string
	if err := conn.QueryRowContext(ctx, `SELECT token FROM magic_links WHERE id = $1`, link.ID).Scan(&stored); err != nil {
		t.Fatalf("read stored token: %v", err)
	}
	sum := sha256.Sum256([]byte(rawToken))
	wantHash := hex.EncodeToString(sum[:])
	if stored == rawToken {
		t.Fatal("token stored in plaintext — a DB read yields a usable magic link")
	}
	if stored != wantHash {
		t.Fatalf("stored token = %q, want the SHA-256 %q", stored, wantHash)
	}

	// Lookup by the raw token resolves via the hash.
	got, err := repo.GetByToken(ctx, rawToken)
	if err != nil || got.ID != link.ID {
		t.Fatalf("GetByToken(raw) = (%v, %v), want the created link", got, err)
	}

	// Single-use: first claim wins, second loses (the AND used_at IS NULL guard).
	if marked, err := repo.MarkUsed(ctx, link.ID); err != nil || !marked {
		t.Fatalf("first MarkUsed = (%v, %v), want (true, nil)", marked, err)
	}
	if marked, err := repo.MarkUsed(ctx, link.ID); err != nil || marked {
		t.Fatalf("second MarkUsed = (%v, %v), want (false, nil) — link must be single-use", marked, err)
	}
}
