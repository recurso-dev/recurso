package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func openTokenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed single-use token test")
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

func seedTokenUser(t *testing.T, conn *sql.DB) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	tenantID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
		tenantID, "TK-"+tenantID.String()[:8], tenantID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	userID := uuid.New()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO users (id, tenant_id, email, password_hash, name, role) VALUES ($1, $2, $3, 'x', 'U', 'owner')`,
		userID, tenantID, userID.String()[:8]+"@t.com"); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}

// TestSingleUseTokens_MarkUsedIsAtomic proves the ENG-176 fix: MarkUsed on each
// single-use auth token repo consumes the token on the FIRST call (returns true)
// and refuses every subsequent call (returns false). This is the atomic gate two
// concurrent requests race for, so only one can ever spend a given token.
func TestSingleUseTokens_MarkUsedIsAtomic(t *testing.T) {
	conn := openTokenTestDB(t)
	ctx := context.Background()
	userID := seedTokenUser(t, conn)
	future := time.Now().Add(time.Hour)

	t.Run("password reset", func(t *testing.T) {
		repo := NewPasswordResetRepository(conn)
		tok := &domain.PasswordResetToken{ID: uuid.New(), TokenHash: "reset-" + uuid.NewString(), UserID: userID, ExpiresAt: future, CreatedAt: time.Now()}
		if err := repo.Create(ctx, tok); err != nil {
			t.Fatalf("create: %v", err)
		}
		assertConsumedOnce(t, func() (bool, error) { return repo.MarkUsed(ctx, tok.ID) })
	})

	t.Run("mfa login token", func(t *testing.T) {
		repo := NewMFALoginTokenRepository(conn)
		tok := &domain.MFALoginToken{ID: uuid.New(), TokenHash: "mfa-" + uuid.NewString(), UserID: userID, ExpiresAt: future, CreatedAt: time.Now()}
		if err := repo.Create(ctx, tok); err != nil {
			t.Fatalf("create: %v", err)
		}
		assertConsumedOnce(t, func() (bool, error) { return repo.MarkUsed(ctx, tok.ID) })
	})

	t.Run("mfa backup code", func(t *testing.T) {
		repo := NewMFABackupCodeRepository(conn)
		code := &domain.MFABackupCode{ID: uuid.New(), UserID: userID, CodeHash: "bc-" + uuid.NewString(), CreatedAt: time.Now()}
		if err := repo.CreateMany(ctx, []*domain.MFABackupCode{code}); err != nil {
			t.Fatalf("create: %v", err)
		}
		assertConsumedOnce(t, func() (bool, error) { return repo.MarkUsed(ctx, code.ID) })
	})
}

func assertConsumedOnce(t *testing.T, markUsed func() (bool, error)) {
	t.Helper()
	first, err := markUsed()
	if err != nil {
		t.Fatalf("first MarkUsed: %v", err)
	}
	if !first {
		t.Fatal("first MarkUsed returned false, want true (token should be consumed)")
	}
	second, err := markUsed()
	if err != nil {
		t.Fatalf("second MarkUsed: %v", err)
	}
	if second {
		t.Fatal("second MarkUsed returned true — token was consumed twice (single-use race)")
	}
}
