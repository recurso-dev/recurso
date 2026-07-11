package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"golang.org/x/crypto/bcrypt"
)

// TOTP validation parameters — must match the codes generated at setup
// (pquerna/otp defaults: 30s period, ±1 step skew, 6 digits, SHA1).
const (
	totpPeriod = 30
	totpSkew   = 1
)

// matchTOTPTimestep returns the timestep (Unix-time / period) at which code is a
// valid TOTP for secret within the skew window, or (0, false). Knowing the
// matched timestep is what lets us enforce single-use (ENG-151).
func matchTOTPTimestep(code, secret string, now time.Time) (int64, bool) {
	opts := totp.ValidateOpts{Period: totpPeriod, Skew: totpSkew, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1}
	counter := now.Unix() / totpPeriod
	for d := int64(-totpSkew); d <= totpSkew; d++ {
		c := counter + d
		if c < 0 {
			continue
		}
		gen, err := totp.GenerateCodeCustom(secret, time.Unix(c*totpPeriod, 0), opts)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(gen), []byte(code)) == 1 {
			return c, true
		}
	}
	return 0, false
}

// validateTOTPSingleUse validates a TOTP code AND enforces that its timestep has
// not been consumed before (ENG-151): a captured code can no longer be replayed
// within the 30-90s validity window. On success the consumed timestep is
// persisted. Fails closed if it can't be persisted — a replay-protection control
// that silently no-ops on a write error is worse than a retryable login failure.
func (s *AuthService) validateTOTPSingleUse(ctx context.Context, user *domain.User, code string) bool {
	if user.MFASecret == "" {
		return false
	}
	ts, ok := matchTOTPTimestep(code, user.MFASecret, time.Now())
	if !ok {
		return false
	}
	if ts <= user.MFALastTimestep {
		return false // replay of an already-consumed (or older) code
	}
	if err := s.users.SetMFALastTimestep(ctx, user.TenantID, user.ID, ts); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to persist consumed TOTP timestep; rejecting to prevent replay", "user_id", user.ID, "error", err)
		}
		return false
	}
	user.MFALastTimestep = ts
	return true
}

// passwordResetTTL is how long an emailed reset link stays valid.
const passwordResetTTL = time.Hour

// backupCodeCount is how many one-time recovery codes are issued when MFA is
// enabled.
const backupCodeCount = 10

// mfaIssuer labels the account in authenticator apps.
const mfaIssuer = "Recurso"

// --- Password reset --------------------------------------------------------

// RequestPasswordReset creates a single-use reset token for the account with
// the given email (if one exists) and emails a reset link. To prevent account
// enumeration it returns nil whether or not the account exists; only genuine
// infrastructure failures return an error (which the handler logs but still
// answers generically).
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	if s.resetTokens == nil {
		return fmt.Errorf("password reset is not configured")
	}
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// No such account: silently succeed (no enumeration).
		return nil
	}

	raw, tokenHash, err := newSessionToken()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := s.resetTokens.Create(ctx, &domain.PasswordResetToken{
		ID:        uuid.New(),
		TokenHash: tokenHash,
		UserID:    user.ID,
		ExpiresAt: now.Add(passwordResetTTL),
		CreatedAt: now,
	}); err != nil {
		return err
	}

	link := fmt.Sprintf("%s/reset-password?token=%s", s.appBaseURL, raw)
	if s.mailer != nil {
		if err := s.mailer.SendPasswordReset(ctx, user.Email, link); err != nil {
			// Best-effort: the token exists, so surface the failure to logs but
			// do not leak it to the caller.
			s.logger.Error("failed to send password reset email", "error", err)
		}
	}
	return nil
}

// ResetPassword validates a reset token (unused and unexpired), sets the new
// bcrypt password, marks the token used, and deletes ALL of the user's sessions
// so every device is forced to re-login. Any bad/expired/used token yields the
// same generic ErrInvalidResetToken.
func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if s.resetTokens == nil {
		return fmt.Errorf("password reset is not configured")
	}
	if len(newPassword) < minPasswordLen {
		return domain.ErrWeakPassword
	}
	if rawToken == "" {
		return domain.ErrInvalidResetToken
	}

	tok, err := s.resetTokens.GetByTokenHash(ctx, hashToken(rawToken))
	if err != nil {
		return domain.ErrInvalidResetToken
	}
	if tok.UsedAt != nil || time.Now().UTC().After(tok.ExpiresAt) {
		return domain.ErrInvalidResetToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, tok.UserID, string(hash)); err != nil {
		return err
	}
	if err := s.resetTokens.MarkUsed(ctx, tok.ID); err != nil {
		return err
	}
	// Force re-login everywhere: a password change invalidates existing sessions.
	return s.sessions.DeleteByUser(ctx, tok.UserID)
}

// --- TOTP MFA --------------------------------------------------------------

// MFASetupResult carries the provisioning data for a new TOTP secret. MFA is
// NOT yet enabled — the user must confirm a code via VerifyAndEnableMFA.
type MFASetupResult struct {
	Secret     string // base32 secret, for manual entry
	OtpauthURL string // otpauth:// URI, for QR display
}

// SetupMFA generates a fresh TOTP secret for the user and stores it as pending
// (mfa_enabled stays false until a code is verified).
func (s *AuthService) SetupMFA(ctx context.Context, tenantID, userID uuid.UUID) (*MFASetupResult, error) {
	user, err := s.users.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if user.MFAEnabled {
		return nil, domain.ErrMFAAlreadyEnabled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      mfaIssuer,
		AccountName: user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}
	if err := s.users.SetMFASecret(ctx, tenantID, userID, key.Secret()); err != nil {
		return nil, err
	}
	return &MFASetupResult{Secret: key.Secret(), OtpauthURL: key.URL()}, nil
}

// VerifyAndEnableMFA validates a TOTP code against the pending secret and, on
// success, enables MFA and issues a fresh set of one-time backup codes. The raw
// backup codes are returned ONCE; only their hashes are stored.
func (s *AuthService) VerifyAndEnableMFA(ctx context.Context, tenantID, userID uuid.UUID, code string) ([]string, error) {
	if s.backupCodes == nil {
		return nil, fmt.Errorf("mfa is not configured")
	}
	user, err := s.users.GetByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if user.MFAEnabled {
		return nil, domain.ErrMFAAlreadyEnabled
	}
	if user.MFASecret == "" {
		return nil, domain.ErrMFANotConfigured
	}
	// Enrollment proof-of-possession only — this runs inside an already
	// authenticated session, so it does NOT consume a login timestep (that would
	// just block the user's first real login with their current code). The
	// single-use replay guard is enforced on the login/step-up path below.
	if !totp.Validate(strings.TrimSpace(code), user.MFASecret) {
		return nil, domain.ErrInvalidMFACode
	}

	if err := s.users.SetMFAEnabled(ctx, tenantID, userID, true); err != nil {
		return nil, err
	}

	plain, models, err := newBackupCodes(userID, backupCodeCount)
	if err != nil {
		return nil, err
	}
	// Replace any codes from a previous enablement, then store the fresh set.
	if err := s.backupCodes.DeleteByUser(ctx, userID); err != nil {
		return nil, err
	}
	if err := s.backupCodes.CreateMany(ctx, models); err != nil {
		return nil, err
	}
	return plain, nil
}

// DisableMFA verifies a TOTP or unused backup code and then disables MFA,
// wiping the secret and all backup codes.
func (s *AuthService) DisableMFA(ctx context.Context, tenantID, userID uuid.UUID, code string) error {
	user, err := s.users.GetByID(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if !user.MFAEnabled {
		return domain.ErrMFANotEnabled
	}
	if !s.verifyMFACode(ctx, user, code) {
		return domain.ErrInvalidMFACode
	}
	if err := s.users.ClearMFA(ctx, tenantID, userID); err != nil {
		return err
	}
	if s.backupCodes != nil {
		return s.backupCodes.DeleteByUser(ctx, userID)
	}
	return nil
}

// LoginMFA completes a two-step login: it validates the challenge token issued
// by Login, verifies a TOTP or unused backup code, consumes the challenge (and
// the backup code, if used), and opens a full session.
func (s *AuthService) LoginMFA(ctx context.Context, rawMFAToken, code, userAgent string) (*LoginResult, error) {
	if s.mfaTokens == nil {
		return nil, fmt.Errorf("mfa is not configured")
	}
	if rawMFAToken == "" {
		return nil, domain.ErrInvalidMFAToken
	}

	tok, err := s.mfaTokens.GetByTokenHash(ctx, hashToken(rawMFAToken))
	if err != nil {
		return nil, domain.ErrInvalidMFAToken
	}
	if tok.UsedAt != nil || time.Now().UTC().After(tok.ExpiresAt) {
		return nil, domain.ErrInvalidMFAToken
	}

	user, err := s.users.GetByIDGlobal(ctx, tok.UserID)
	if err != nil || !user.MFAEnabled {
		return nil, domain.ErrInvalidMFAToken
	}

	// Per-account lockout also covers the MFA step, so a stolen password can't be
	// paired with unlimited TOTP guesses (ENG-151).
	if user.IsLocked(time.Now()) {
		return nil, domain.ErrAccountLocked
	}

	if !s.verifyMFACode(ctx, user, code) {
		s.registerFailedLogin(ctx, user)
		return nil, domain.ErrInvalidMFACode
	}

	// Full login success — reset the lockout counter.
	_ = s.users.ClearFailedLogins(ctx, user.ID)

	// Consume the single-use challenge token before minting a session.
	if err := s.mfaTokens.MarkUsed(ctx, tok.ID); err != nil {
		return nil, err
	}

	tenant, err := s.tenants.GetAccount(ctx, user.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tenant: %w", err)
	}
	session, err := s.openSession(ctx, user, userAgent)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: user, Tenant: tenant, SessionToken: session}, nil
}

// verifyMFACode returns true if code is a valid current TOTP for the user OR an
// unused backup code. A matching backup code is consumed as a side effect. The
// comparison of backup-code hashes is constant-time.
func (s *AuthService) verifyMFACode(ctx context.Context, user *domain.User, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	if s.validateTOTPSingleUse(ctx, user, code) {
		return true
	}
	if s.backupCodes == nil {
		return false
	}
	codes, err := s.backupCodes.ListByUser(ctx, user.ID)
	if err != nil {
		return false
	}
	want := hashToken(normalizeBackupCode(code))
	for _, bc := range codes {
		if bc.UsedAt != nil {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(bc.CodeHash), []byte(want)) == 1 {
			_ = s.backupCodes.MarkUsed(ctx, bc.ID)
			return true
		}
	}
	return false
}

// --- Session management ----------------------------------------------------

// ActiveSession is a session enriched with a "current" flag for the request's
// own session.
type ActiveSession struct {
	Session *domain.Session
	Current bool
}

// ListActiveSessions returns the user's unexpired sessions, flagging the one
// that matches currentRawToken (the caller's own cookie).
func (s *AuthService) ListActiveSessions(ctx context.Context, userID uuid.UUID, currentRawToken string) ([]ActiveSession, error) {
	all, err := s.sessions.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	currentHash := ""
	if currentRawToken != "" {
		currentHash = hashToken(currentRawToken)
	}
	now := time.Now().UTC()
	out := make([]ActiveSession, 0, len(all))
	for _, sess := range all {
		if now.After(sess.ExpiresAt) {
			continue
		}
		out = append(out, ActiveSession{Session: sess, Current: sess.TokenHash == currentHash})
	}
	return out, nil
}

// RevokeSession deletes one of the user's OWN sessions. A session that does not
// exist or belongs to another user yields ErrSessionNotFound (404).
func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	sess, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return domain.ErrSessionNotFound
	}
	if sess.UserID != userID {
		return domain.ErrSessionNotFound
	}
	return s.sessions.DeleteByID(ctx, sessionID)
}

// RevokeOtherSessions deletes all of the user's sessions except the current one
// (identified by currentRawToken) — "log out everywhere else".
func (s *AuthService) RevokeOtherSessions(ctx context.Context, userID uuid.UUID, currentRawToken string) error {
	return s.sessions.DeleteByUserExcept(ctx, userID, hashToken(currentRawToken))
}

// --- helpers ---------------------------------------------------------------

// newBackupCodes returns n human-friendly one-time codes (plaintext) plus the
// corresponding storage models holding only SHA-256 hashes.
func newBackupCodes(userID uuid.UUID, n int) (plain []string, models []*domain.MFABackupCode, err error) {
	now := time.Now().UTC()
	plain = make([]string, 0, n)
	models = make([]*domain.MFABackupCode, 0, n)
	for i := 0; i < n; i++ {
		code, genErr := generateBackupCode()
		if genErr != nil {
			return nil, nil, genErr
		}
		plain = append(plain, code)
		models = append(models, &domain.MFABackupCode{
			ID:        uuid.New(),
			UserID:    userID,
			CodeHash:  hashToken(normalizeBackupCode(code)),
			CreatedAt: now,
		})
	}
	return plain, models, nil
}

// backupCodeEncoding is unpadded, lowercase-insensitive base32 without the
// easily-confused characters handled by normalizeBackupCode.
var backupCodeEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// generateBackupCode returns a 10-character base32 code (~50 bits of entropy)
// formatted as XXXXX-XXXXX for readability.
func generateBackupCode() (string, error) {
	b := make([]byte, 7) // 56 bits -> 11 base32 chars; we take 10
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate backup code: %w", err)
	}
	enc := backupCodeEncoding.EncodeToString(b)
	enc = enc[:10]
	return enc[:5] + "-" + enc[5:], nil
}

// normalizeBackupCode canonicalizes a code for hashing/comparison: upper-cased
// with separators stripped, so "abcde-fghij" and "ABCDEFGHIJ" match.
func normalizeBackupCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return code
}
