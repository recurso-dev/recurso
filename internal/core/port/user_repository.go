package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// UserRepository persists dashboard user accounts. Email lookups are global
// (login supplies no tenant); all other reads/writes are tenant-scoped so a
// user in tenant A can never touch a user in tenant B.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	// GetByEmail resolves a user by (lower-cased) email across all tenants.
	// Used by login, which receives no tenant. Returns ErrUserNotFound if none.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	// ExistsByEmail reports whether any user already uses this (lower-cased) email.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	// GetByID resolves a user by id, scoped to the tenant.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.User, error)
	UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.Role) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// CountOwners returns how many owner-role users the tenant has.
	CountOwners(ctx context.Context, tenantID uuid.UUID) (int, error)
}

// SessionRepository persists opaque login sessions keyed by the SHA-256 hash
// of the token.
type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	// GetByTokenHash returns the session for the given token hash, or
	// ErrSessionNotFound if it does not exist.
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}
