package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
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

// passwordResetEmailer sends the account emails AuthService triggers — the
// password-reset link and the team-invite link. *NotificationService satisfies
// it; kept as a narrow interface so AuthService need not depend on the whole
// notification surface (and tests can supply a fake/no-op).
type passwordResetEmailer interface {
	SendPasswordReset(ctx context.Context, toEmail, resetURL string) error
	SendInvite(ctx context.Context, toEmail, name, inviteURL string) error
}

// AuthService owns dashboard user accounts, sessions, and team management.
type AuthService struct {
	users      port.UserRepository
	sessions   port.SessionRepository
	tenants    tenantRegistrar
	sessionTTL time.Duration

	// Phase 2 dependencies (password reset, TOTP MFA). Configured via the
	// setters below so the base constructor — and every existing caller/test —
	// stays unchanged. Methods that need them guard against nil.
	resetTokens port.PasswordResetRepository
	backupCodes port.MFABackupCodeRepository
	mfaTokens   port.MFALoginTokenRepository
	mailer      passwordResetEmailer
	appBaseURL  string
	logger      *slog.Logger

	// Phase 3 dependency (OAuth social login). Configured via ConfigureOAuth so
	// the base constructor and existing callers stay unchanged; LoginWithOAuth
	// guards against nil.
	oauthIdentities port.OAuthIdentityRepository
}

func NewAuthService(users port.UserRepository, sessions port.SessionRepository, tenants tenantRegistrar, sessionTTL time.Duration) *AuthService {
	if sessionTTL <= 0 {
		sessionTTL = 7 * 24 * time.Hour
	}
	return &AuthService{
		users:      users,
		sessions:   sessions,
		tenants:    tenants,
		sessionTTL: sessionTTL,
		logger:     slog.Default().With("service", "auth"),
	}
}

// ConfigurePasswordReset wires the password-reset dependencies. appBaseURL is
// the base of the admin dashboard that hosts the /reset-password page.
func (s *AuthService) ConfigurePasswordReset(resetTokens port.PasswordResetRepository, mailer passwordResetEmailer, appBaseURL string) {
	s.resetTokens = resetTokens
	s.mailer = mailer
	s.appBaseURL = strings.TrimRight(appBaseURL, "/")
}

// ConfigureMFA wires the TOTP MFA dependencies.
func (s *AuthService) ConfigureMFA(backupCodes port.MFABackupCodeRepository, mfaTokens port.MFALoginTokenRepository) {
	s.backupCodes = backupCodes
	s.mfaTokens = mfaTokens
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

// OpenSessionForUser creates and persists a session for user and returns the
// raw session token, using the exact Phase 1 session path. Exposed so the OAuth
// and SAML SSO login flows issue an identical recurso_session without
// duplicating session creation.
func (s *AuthService) OpenSessionForUser(ctx context.Context, user *domain.User, userAgent string) (string, error) {
	return s.openSession(ctx, user, userAgent)
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

// LoginResult is returned from a login attempt. When MFARequired is true no
// session was opened: SessionToken/Tenant are empty and the caller must exchange
// MFAToken (plus a TOTP/backup code) at LoginMFA to finish authenticating.
type LoginResult struct {
	User         *domain.User
	Tenant       *domain.Tenant
	SessionToken string
	MFARequired  bool
	MFAToken     string
}

// mfaLoginTokenTTL is the lifetime of the short-lived challenge token issued
// between the password step and the MFA-code step of a two-step login.
const mfaLoginTokenTTL = 5 * time.Minute

// Login verifies credentials in constant time. On success it either opens a
// session (no MFA) or, when the user has MFA enabled, returns a short-lived
// single-use challenge token WITHOUT opening a session. It never reveals whether
// the email or the password was wrong.
// Per-account lockout thresholds (ENG-151): after this many consecutive failed
// password/MFA attempts the account is locked for lockoutDuration, bounding
// credential-stuffing spread across IPs that evades the per-IP rate limit.
const (
	maxFailedLogins = 5
	lockoutDuration = 15 * time.Minute
)

// registerFailedLogin records a failed password/MFA attempt and logs when the
// attempt trips the lock. Best-effort — a bookkeeping write failure must not
// turn a wrong password into a 500.
func (s *AuthService) registerFailedLogin(ctx context.Context, user *domain.User) {
	if err := s.users.RegisterFailedLogin(ctx, user.ID, maxFailedLogins, lockoutDuration); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to record failed login attempt", "user_id", user.ID, "error", err)
		}
		return
	}
	if user.FailedLoginAttempts+1 >= maxFailedLogins && s.logger != nil {
		s.logger.Warn("account locked after repeated failed auth attempts",
			"user_id", user.ID, "email", user.Email, "lock_minutes", int(lockoutDuration.Minutes()))
	}
}

func (s *AuthService) Login(ctx context.Context, email, password, userAgent string) (*LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Unknown email: still pay the bcrypt cost against a dummy hash so the
		// response time matches a wrong-password attempt (no enumeration).
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
		return nil, domain.ErrInvalidCredentials
	}

	// Per-account lockout (ENG-151): while locked, reject before checking the
	// password so a credential-stuffing burst can't keep guessing.
	if user.IsLocked(time.Now()) {
		return nil, domain.ErrAccountLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.registerFailedLogin(ctx, user)
		return nil, domain.ErrInvalidCredentials
	}

	// MFA gate: password was correct, but a second factor is required. Issue a
	// challenge token and stop here — no session cookie is opened.
	if user.MFAEnabled && s.mfaTokens != nil {
		raw, tokenHash, err := newSessionToken()
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		if err := s.mfaTokens.Create(ctx, &domain.MFALoginToken{
			ID:        uuid.New(),
			TokenHash: tokenHash,
			UserID:    user.ID,
			ExpiresAt: now.Add(mfaLoginTokenTTL),
			CreatedAt: now,
		}); err != nil {
			return nil, err
		}
		return &LoginResult{User: user, MFARequired: true, MFAToken: raw}, nil
	}

	// Full login success (no MFA required) — reset the lockout counter.
	_ = s.users.ClearFailedLogins(ctx, user.ID)

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
func (s *AuthService) CreateUser(ctx context.Context, tenantID uuid.UUID, actorRole domain.Role, email, name string, role domain.Role, password string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !role.Valid() {
		return nil, domain.ErrInvalidRole
	}
	// Only an owner may create another owner (privilege-escalation guard).
	if role == domain.RoleOwner && actorRole != domain.RoleOwner {
		return nil, domain.ErrOwnerRoleRequired
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
func (s *AuthService) UpdateUserRole(ctx context.Context, tenantID uuid.UUID, actorRole domain.Role, targetID uuid.UUID, role domain.Role) (*domain.User, error) {
	if !role.Valid() {
		return nil, domain.ErrInvalidRole
	}
	target, err := s.users.GetByID(ctx, tenantID, targetID)
	if err != nil {
		return nil, err
	}
	// Only an owner may cross the owner boundary — grant the owner role or
	// demote an existing owner. Otherwise an admin could self-promote to owner.
	if (role == domain.RoleOwner || target.Role == domain.RoleOwner) && actorRole != domain.RoleOwner {
		return nil, domain.ErrOwnerRoleRequired
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
func (s *AuthService) DeleteUser(ctx context.Context, tenantID uuid.UUID, actorRole domain.Role, actingUserID, targetID uuid.UUID) error {
	target, err := s.users.GetByID(ctx, tenantID, targetID)
	if err != nil {
		return err
	}
	if actingUserID != uuid.Nil && actingUserID == targetID {
		return domain.ErrSelfLockout
	}
	// Only an owner may remove an existing owner.
	if target.Role == domain.RoleOwner && actorRole != domain.RoleOwner {
		return domain.ErrOwnerRoleRequired
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
