package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Role is a dashboard user's authorization level within a tenant.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// Valid reports whether r is a recognized role.
func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

// CanManageTeam reports whether the role may create/modify/remove teammates.
// Members are read-only for team management.
func (r Role) CanManageTeam() bool {
	return r == RoleOwner || r == RoleAdmin
}

// User is a human account that logs into the admin dashboard. Every user
// belongs to exactly one tenant.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // never serialized
	Name         string    `json:"name" db:"name"`
	Role         Role      `json:"role" db:"role"`
	// MFAEnabled reports whether TOTP two-factor auth is active for this user.
	MFAEnabled bool `json:"mfa_enabled" db:"mfa_enabled"`
	// MFASecret is the base32 TOTP secret. Populated at setup time (before the
	// user proves possession) and only trusted once MFAEnabled is true. Never
	// serialized to clients.
	MFASecret string `json:"-" db:"mfa_secret"`
	// MFALastTimestep is the last consumed TOTP timestep (Unix-time / 30). A code
	// whose timestep is <= this value is a replay and is rejected (ENG-151).
	MFALastTimestep int64 `json:"-" db:"mfa_last_timestep"`
	// FailedLoginAttempts / LockedUntil back the per-account lockout (ENG-151).
	FailedLoginAttempts int        `json:"-" db:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"-" db:"locked_until"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// IsLocked reports whether the account is currently within a lockout window.
func (u *User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && u.LockedUntil.After(now)
}

// Session is an opaque, server-side login session. Only the SHA-256 hash of the
// token is stored; the raw token lives solely in the client's httpOnly cookie.
type Session struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TokenHash string    `json:"-" db:"token_hash"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UserAgent string    `json:"user_agent,omitempty" db:"user_agent"`
}

// SessionCookieName is the httpOnly cookie that carries the opaque session
// token for dashboard users. Shared by the handler (which sets it) and the
// dual-auth middleware (which reads it).
const SessionCookieName = "recurso_session"

// PasswordResetToken is a single-use, short-lived credential emailed to a user
// to authorize a password change. Only the SHA-256 hash is stored server-side.
type PasswordResetToken struct {
	ID        uuid.UUID  `db:"id"`
	TokenHash string     `db:"token_hash"`
	UserID    uuid.UUID  `db:"user_id"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}

// MFABackupCode is one hashed single-use recovery code for a user's TOTP MFA.
type MFABackupCode struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	CodeHash  string     `db:"code_hash"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}

// MFALoginToken is the short-lived, single-use challenge issued after a correct
// password when MFA is enabled. It is exchanged (with a TOTP or backup code) for
// a real session. Only the SHA-256 hash is stored server-side.
type MFALoginToken struct {
	ID        uuid.UUID  `db:"id"`
	TokenHash string     `db:"token_hash"`
	UserID    uuid.UUID  `db:"user_id"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}

// Auth domain errors. Login-facing errors are deliberately coarse so callers
// cannot use them to enumerate accounts.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrDuplicateEmail     = errors.New("a user with that email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrAccountLocked is returned when an account is temporarily locked after
	// too many failed login/MFA attempts (ENG-151).
	ErrAccountLocked   = errors.New("account temporarily locked due to too many failed attempts")
	ErrSessionNotFound = errors.New("session not found or expired")
	ErrLastOwner       = errors.New("cannot remove or demote the last owner")
	ErrSelfLockout     = errors.New("you cannot remove your own account")
	ErrWeakPassword    = errors.New("password must be at least 8 characters")
	ErrInvalidRole     = errors.New("role must be one of owner, admin, member")
	// ErrOwnerRoleRequired guards the owner boundary: only an owner may grant the
	// owner role or remove/demote an existing owner. Without it an admin (who can
	// manage the team) could self-promote to owner (privilege escalation).
	ErrOwnerRoleRequired = errors.New("only an owner can grant or remove the owner role")

	// Password reset. Deliberately coarse so a caller cannot tell a bad token
	// from an expired or already-used one.
	ErrInvalidResetToken = errors.New("invalid or expired reset token")

	// MFA. Login/challenge errors are coarse to avoid oracles.
	ErrMFANotConfigured  = errors.New("mfa setup has not been started")
	ErrMFAAlreadyEnabled = errors.New("mfa is already enabled")
	ErrMFANotEnabled     = errors.New("mfa is not enabled")
	ErrInvalidMFACode    = errors.New("invalid code")
	ErrInvalidMFAToken   = errors.New("invalid or expired authentication token")
	ErrUserRequired      = errors.New("this endpoint requires a logged-in user")
)
