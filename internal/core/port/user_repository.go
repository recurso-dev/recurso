package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
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
	// GetByIDGlobal resolves a user by id with no tenant scope. Used only by
	// flows that already hold a proof-of-identity token (MFA challenge), where
	// the tenant is not otherwise available.
	GetByIDGlobal(ctx context.Context, id uuid.UUID) (*domain.User, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.User, error)
	UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.Role) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// CountOwners returns how many owner-role users the tenant has.
	CountOwners(ctx context.Context, tenantID uuid.UUID) (int, error)
	// UpdatePassword sets a new bcrypt hash for the user, keyed by id alone
	// (password reset carries no tenant context).
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	// SetMFASecret stores a pending TOTP secret without enabling MFA.
	SetMFASecret(ctx context.Context, tenantID, id uuid.UUID, secret string) error
	// SetMFAEnabled toggles the mfa_enabled flag.
	SetMFAEnabled(ctx context.Context, tenantID, id uuid.UUID, enabled bool) error
	// SetMFALastTimestep records the last consumed TOTP timestep for replay
	// protection; the write is monotonic (never lowers the stored value).
	SetMFALastTimestep(ctx context.Context, tenantID, id uuid.UUID, timestep int64) error
	// RegisterFailedLogin increments the failed-attempt counter and locks the
	// account for lockFor once it reaches lockThreshold (per-account lockout).
	RegisterFailedLogin(ctx context.Context, id uuid.UUID, lockThreshold int, lockFor time.Duration) error
	// ClearFailedLogins resets the counter and lock on a successful login.
	ClearFailedLogins(ctx context.Context, id uuid.UUID) error
	// ClearMFA disables MFA and wipes the stored secret.
	ClearMFA(ctx context.Context, tenantID, id uuid.UUID) error
}

// SessionRepository persists opaque login sessions keyed by the SHA-256 hash
// of the token.
type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	// GetByTokenHash returns the session for the given token hash, or
	// ErrSessionNotFound if it does not exist.
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error)
	// GetByID returns a session by its primary key, or ErrSessionNotFound.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error)
	// ListByUser returns every session (including expired ones) belonging to a
	// user; callers filter on expiry.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	// DeleteByID removes a single session by primary key.
	DeleteByID(ctx context.Context, id uuid.UUID) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
	// DeleteByUserExcept removes all of a user's sessions except the one whose
	// token hashes to exceptTokenHash ("log out everywhere else").
	DeleteByUserExcept(ctx context.Context, userID uuid.UUID, exceptTokenHash string) error
}

// PasswordResetRepository persists single-use password reset tokens keyed by
// the SHA-256 hash of the raw token.
type PasswordResetRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	// GetByTokenHash returns the token for the given hash, or
	// ErrInvalidResetToken if none exists.
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error)
	// MarkUsed atomically stamps used_at only if it was NULL, returning true when
	// this call is the one that consumed the token. Callers use the bool as the
	// single-use gate so two concurrent requests can't both spend one token.
	MarkUsed(ctx context.Context, id uuid.UUID) (bool, error)
}

// MFABackupCodeRepository persists hashed one-time MFA recovery codes.
type MFABackupCodeRepository interface {
	CreateMany(ctx context.Context, codes []*domain.MFABackupCode) error
	// ListByUser returns all backup codes for a user (used and unused).
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.MFABackupCode, error)
	// MarkUsed atomically consumes the code, returning true only for the caller
	// that won the claim (see PasswordResetRepository.MarkUsed).
	MarkUsed(ctx context.Context, id uuid.UUID) (bool, error)
	// DeleteByUser removes every backup code for a user (on disable / re-issue).
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}

// MFALoginTokenRepository persists short-lived MFA challenge tokens keyed by the
// SHA-256 hash of the raw token.
type MFALoginTokenRepository interface {
	Create(ctx context.Context, token *domain.MFALoginToken) error
	// GetByTokenHash returns the token for the given hash, or
	// ErrInvalidMFAToken if none exists.
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.MFALoginToken, error)
	// MarkUsed atomically consumes the challenge token, returning true only for
	// the caller that won the claim (see PasswordResetRepository.MarkUsed).
	MarkUsed(ctx context.Context, id uuid.UUID) (bool, error)
}
