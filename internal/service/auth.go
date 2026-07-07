package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"golang.org/x/crypto/bcrypt"
)

// minPasswordLen is the minimum acceptable password length.
const minPasswordLen = 8

// bcryptCost matches the cost used for API keys (bcrypt.DefaultCost) so the two
// credential types share a consistent work factor.
const bcryptCost = bcrypt.DefaultCost

// dummyPasswordHash is compared against on login when no user matches the
// email, so an unknown-email attempt costs the same bcrypt time as a
// wrong-password attempt. Prevents timing-based user enumeration.
var dummyPasswordHash = mustDummyHash()

func mustDummyHash() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("recurso-timing-equalizer"), bcryptCost)
	if err != nil {
		panic(err)
	}
	return h
}

// tenantRegistrar is the slice of tenant behavior AuthService needs: creating a
// tenant (+ first API key) at signup, and loading a tenant for login/me.
// *TenantService satisfies it; tests supply a fake.
type tenantRegistrar interface {
	Register(ctx context.Context, name, email string) (*domain.Tenant, *domain.APIKey, error)
	GetAccount(ctx context.Context, tenantID uuid.UUID) (*domain.Tenant, error)
}

// AuthService owns dashboard user accounts, sessions, and team management.
type AuthService struct {
	users      port.UserRepository
	sessions   port.SessionRepository
	tenants    tenantRegistrar
	sessionTTL time.Duration
}

func NewAuthService(users port.UserRepository, sessions port.SessionRepository, tenants tenantRegistrar, sessionTTL time.Duration) *AuthService {
	if sessionTTL <= 0 {
		sessionTTL = 7 * 24 * time.Hour
	}
	return &AuthService{users: users, sessions: sessions, tenants: tenants, sessionTTL: sessionTTL}
}

// SessionTTL exposes the configured session lifetime (for cookie max-age).
func (s *AuthService) SessionTTL() time.Duration { return s.sessionTTL }

// hashToken returns the hex SHA-256 of a raw session token. Only this hash is
// ever stored server-side.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// newSessionToken returns a URL-safe opaque token from 32 bytes of CSPRNG
// entropy along with its storage hash.
func newSessionToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate session token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, hashToken(raw), nil
}

// openSession creates and persists a session for the user, returning the raw
// token (to be placed in the cookie).
func (s *AuthService) openSession(ctx context.Context, user *domain.User, userAgent string) (string, error) {
	raw, tokenHash, err := newSessionToken()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	sess := &domain.Session{
		ID:        uuid.New(),
		TokenHash: tokenHash,
		UserID:    user.ID,
		TenantID:  user.TenantID,
		ExpiresAt: now.Add(s.sessionTTL),
		CreatedAt: now,
		UserAgent: truncate(userAgent, 512),
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return "", fmt.Errorf("failed to open session: %w", err)
	}
	return raw, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// RegisterResult is returned from Register: a fresh tenant, its first API key,
// the owner user, and an open session token.
type RegisterResult struct {
	Tenant       *domain.Tenant
	APIKey       *domain.APIKey
	User         *domain.User
	SessionToken string
}

// Register creates a tenant (reusing the tenant-creation path), its first user
// as the owner, and opens a session. The tenant API key is still returned so
// CLI/dev flows keep a key.
func (s *AuthService) Register(ctx context.Context, companyName, name, email, password, userAgent string) (*RegisterResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(password) < minPasswordLen {
		return nil, domain.ErrWeakPassword
	}
	if email == "" || name == "" || companyName == "" {
		return nil, fmt.Errorf("company_name, name, and email are required")
	}

	// Reject duplicate email up front (the DB unique index is the race-safe
	// backstop; Create returns ErrDuplicateEmail too).
	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, domain.ErrDuplicateEmail
	}

	tenant, apiKey, err := s.tenants.Register(ctx, companyName, email)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         domain.RoleOwner,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.openSession(ctx, user, userAgent)
	if err != nil {
		return nil, err
	}
	return &RegisterResult{Tenant: tenant, APIKey: apiKey, User: user, SessionToken: token}, nil
}

// LoginResult is returned from a successful login.
type LoginResult struct {
	User         *domain.User
	Tenant       *domain.Tenant
	SessionToken string
}

// Login verifies credentials in constant time and, on success, opens a session.
// It never reveals whether the email or the password was wrong.
func (s *AuthService) Login(ctx context.Context, email, password, userAgent string) (*LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Unknown email: still pay the bcrypt cost against a dummy hash so the
		// response time matches a wrong-password attempt (no enumeration).
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	tenant, err := s.tenants.GetAccount(ctx, user.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tenant: %w", err)
	}

	token, err := s.openSession(ctx, user, userAgent)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: user, Tenant: tenant, SessionToken: token}, nil
}

// Logout deletes the session identified by the raw token and is a no-op if it
// does not exist.
func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.sessions.DeleteByTokenHash(ctx, hashToken(rawToken))
}

// ResolveSession validates a raw session token and returns the owning user.
// Expiry is enforced on every call; an expired session is deleted and rejected.
func (s *AuthService) ResolveSession(ctx context.Context, rawToken string) (*domain.User, error) {
	if rawToken == "" {
		return nil, domain.ErrSessionNotFound
	}
	sess, err := s.sessions.GetByTokenHash(ctx, hashToken(rawToken))
	if err != nil {
		return nil, domain.ErrSessionNotFound
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		_ = s.sessions.DeleteByTokenHash(ctx, sess.TokenHash)
		return nil, domain.ErrSessionNotFound
	}
	user, err := s.users.GetByID(ctx, sess.TenantID, sess.UserID)
	if err != nil {
		return nil, domain.ErrSessionNotFound
	}
	return user, nil
}

// Me returns the user + tenant for a session cookie.
func (s *AuthService) Me(ctx context.Context, rawToken string) (*domain.User, *domain.Tenant, error) {
	user, err := s.ResolveSession(ctx, rawToken)
	if err != nil {
		return nil, nil, err
	}
	tenant, err := s.tenants.GetAccount(ctx, user.TenantID)
	if err != nil {
		return nil, nil, err
	}
	return user, tenant, nil
}

// --- Team management (tenant-scoped) ---

// ListUsers returns all users in the tenant.
func (s *AuthService) ListUsers(ctx context.Context, tenantID uuid.UUID) ([]*domain.User, error) {
	return s.users.ListByTenant(ctx, tenantID)
}

// CreateUser adds a teammate to the tenant.
func (s *AuthService) CreateUser(ctx context.Context, tenantID uuid.UUID, email, name string, role domain.Role, password string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !role.Valid() {
		return nil, domain.ErrInvalidRole
	}
	if len(password) < minPasswordLen {
		return nil, domain.ErrWeakPassword
	}
	if email == "" || name == "" {
		return nil, fmt.Errorf("name and email are required")
	}
	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, domain.ErrDuplicateEmail
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUserRole changes a teammate's role. It refuses to demote the last owner
// of the tenant. targetID must belong to tenantID (else ErrUserNotFound).
func (s *AuthService) UpdateUserRole(ctx context.Context, tenantID, targetID uuid.UUID, role domain.Role) (*domain.User, error) {
	if !role.Valid() {
		return nil, domain.ErrInvalidRole
	}
	target, err := s.users.GetByID(ctx, tenantID, targetID)
	if err != nil {
		return nil, err
	}
	// Demoting the last owner would lock the tenant out of ownership.
	if target.Role == domain.RoleOwner && role != domain.RoleOwner {
		owners, err := s.users.CountOwners(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		if owners <= 1 {
			return nil, domain.ErrLastOwner
		}
	}
	if err := s.users.UpdateRole(ctx, tenantID, targetID, role); err != nil {
		return nil, err
	}
	target.Role = role
	return target, nil
}

// DeleteUser removes a teammate. It refuses to delete the last owner and refuses
// to let the acting user delete their own account (self-lockout). actingUserID
// may be uuid.Nil for machine (API-key) callers, which have no self to lock out.
func (s *AuthService) DeleteUser(ctx context.Context, tenantID, actingUserID, targetID uuid.UUID) error {
	target, err := s.users.GetByID(ctx, tenantID, targetID)
	if err != nil {
		return err
	}
	if actingUserID != uuid.Nil && actingUserID == targetID {
		return domain.ErrSelfLockout
	}
	if target.Role == domain.RoleOwner {
		owners, err := s.users.CountOwners(ctx, tenantID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return domain.ErrLastOwner
		}
	}
	return s.users.Delete(ctx, tenantID, targetID)
}
