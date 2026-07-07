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
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
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

// Auth domain errors. Login-facing errors are deliberately coarse so callers
// cannot use them to enumerate accounts.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrDuplicateEmail     = errors.New("a user with that email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found or expired")
	ErrLastOwner          = errors.New("cannot remove or demote the last owner")
	ErrSelfLockout        = errors.New("you cannot remove your own account")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
	ErrInvalidRole        = errors.New("role must be one of owner, admin, member")
)
