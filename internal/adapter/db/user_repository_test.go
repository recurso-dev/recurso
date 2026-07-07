package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TestUserSessionRepositories_Postgres exercises the real SQL for the users and
// sessions tables against a throwaway database. It applies the embedded
// migrations first, which also validates migration 000064.
//
// Skipped unless TEST_DATABASE_URL points at a scratch database, e.g.:
//
//	createdb recurso_repo_test
//	TEST_DATABASE_URL='postgres://localhost:5432/recurso_repo_test?sslmode=disable' go test ./internal/adapter/db/
func TestUserSessionRepositories_Postgres(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres-backed repository test")
	}
	if err := RunMigrations(dbURL); err != nil {
		t.Fatalf("failed to run migrations (000064 users/sessions): %v", err)
	}
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ctx := context.Background()

	// Two tenants for isolation checks.
	tenantA, tenantB := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{tenantA, tenantB} {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`,
			id, "T-"+id.String()[:8], id.String()[:8]+"@t.com"); err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
	}

	users := NewUserRepository(conn)
	sessions := NewSessionRepository(conn)

	owner := &domain.User{
		ID: uuid.New(), TenantID: tenantA, Email: "Owner@Acme.com", PasswordHash: "hash",
		Name: "Owner", Role: domain.RoleOwner, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := users.Create(ctx, owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}

	// Email stored lower-cased; global uniqueness enforced.
	dup := &domain.User{ID: uuid.New(), TenantID: tenantB, Email: "owner@acme.com", PasswordHash: "h", Name: "Dup", Role: domain.RoleMember, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := users.Create(ctx, dup); !errors.Is(err, domain.ErrDuplicateEmail) {
		t.Fatalf("duplicate email err = %v, want ErrDuplicateEmail", err)
	}

	got, err := users.GetByEmail(ctx, "OWNER@acme.com")
	if err != nil || got.ID != owner.ID {
		t.Fatalf("GetByEmail = %v, %v", got, err)
	}
	if got.Email != "owner@acme.com" {
		t.Fatalf("email not normalized on read: %q", got.Email)
	}

	// Cross-tenant GetByID must not leak.
	if _, err := users.GetByID(ctx, tenantB, owner.ID); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("cross-tenant GetByID err = %v, want ErrUserNotFound", err)
	}

	if n, _ := users.CountOwners(ctx, tenantA); n != 1 {
		t.Fatalf("owners = %d, want 1", n)
	}

	// Sessions round-trip.
	sess := &domain.Session{ID: uuid.New(), TokenHash: "abc123hash", UserID: owner.ID, TenantID: tenantA, ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UserAgent: "go-test"}
	if err := sessions.Create(ctx, sess); err != nil {
		t.Fatalf("create session: %v", err)
	}
	rs, err := sessions.GetByTokenHash(ctx, "abc123hash")
	if err != nil || rs.UserID != owner.ID {
		t.Fatalf("GetByTokenHash = %v, %v", rs, err)
	}
	if err := sessions.DeleteByTokenHash(ctx, "abc123hash"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, err := sessions.GetByTokenHash(ctx, "abc123hash"); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("after delete err = %v, want ErrSessionNotFound", err)
	}

	// ON DELETE CASCADE: deleting the user removes their sessions.
	sess2 := &domain.Session{ID: uuid.New(), TokenHash: "cascade", UserID: owner.ID, TenantID: tenantA, ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	_ = sessions.Create(ctx, sess2)
	if err := users.Delete(ctx, tenantA, owner.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	if _, err := sessions.GetByTokenHash(ctx, "cascade"); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("session should be cascade-deleted, err = %v", err)
	}
}
