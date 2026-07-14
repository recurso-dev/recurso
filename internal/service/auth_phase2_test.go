package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// --- Phase 2 fakes ---------------------------------------------------------

type fakeResetRepo struct {
	byHash map[string]*domain.PasswordResetToken
	byID   map[uuid.UUID]*domain.PasswordResetToken
}

func newFakeResetRepo() *fakeResetRepo {
	return &fakeResetRepo{
		byHash: map[string]*domain.PasswordResetToken{},
		byID:   map[uuid.UUID]*domain.PasswordResetToken{},
	}
}
func (r *fakeResetRepo) Create(_ context.Context, t *domain.PasswordResetToken) error {
	cp := *t
	r.byHash[t.TokenHash] = &cp
	r.byID[t.ID] = &cp
	return nil
}
func (r *fakeResetRepo) GetByTokenHash(_ context.Context, h string) (*domain.PasswordResetToken, error) {
	t, ok := r.byHash[h]
	if !ok {
		return nil, domain.ErrInvalidResetToken
	}
	cp := *t
	return &cp, nil
}
func (r *fakeResetRepo) MarkUsed(_ context.Context, id uuid.UUID) (bool, error) {
	t, ok := r.byID[id]
	if !ok || t.UsedAt != nil {
		return false, nil
	}
	now := time.Now().UTC()
	t.UsedAt = &now
	r.byHash[t.TokenHash].UsedAt = &now
	return true, nil
}

type fakeBackupRepo struct {
	codes []*domain.MFABackupCode
}

func newFakeBackupRepo() *fakeBackupRepo { return &fakeBackupRepo{} }
func (r *fakeBackupRepo) CreateMany(_ context.Context, codes []*domain.MFABackupCode) error {
	for _, c := range codes {
		cp := *c
		r.codes = append(r.codes, &cp)
	}
	return nil
}
func (r *fakeBackupRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*domain.MFABackupCode, error) {
	var out []*domain.MFABackupCode
	for _, c := range r.codes {
		if c.UserID == userID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *fakeBackupRepo) MarkUsed(_ context.Context, id uuid.UUID) (bool, error) {
	for _, c := range r.codes {
		if c.ID == id {
			if c.UsedAt != nil {
				return false, nil
			}
			now := time.Now().UTC()
			c.UsedAt = &now
			return true, nil
		}
	}
	return false, nil
}
func (r *fakeBackupRepo) DeleteByUser(_ context.Context, userID uuid.UUID) error {
	var kept []*domain.MFABackupCode
	for _, c := range r.codes {
		if c.UserID != userID {
			kept = append(kept, c)
		}
	}
	r.codes = kept
	return nil
}

type fakeMFATokenRepo struct {
	byHash map[string]*domain.MFALoginToken
	byID   map[uuid.UUID]*domain.MFALoginToken
}

func newFakeMFATokenRepo() *fakeMFATokenRepo {
	return &fakeMFATokenRepo{
		byHash: map[string]*domain.MFALoginToken{},
		byID:   map[uuid.UUID]*domain.MFALoginToken{},
	}
}
func (r *fakeMFATokenRepo) Create(_ context.Context, t *domain.MFALoginToken) error {
	cp := *t
	r.byHash[t.TokenHash] = &cp
	r.byID[t.ID] = &cp
	return nil
}
func (r *fakeMFATokenRepo) GetByTokenHash(_ context.Context, h string) (*domain.MFALoginToken, error) {
	t, ok := r.byHash[h]
	if !ok {
		return nil, domain.ErrInvalidMFAToken
	}
	cp := *t
	return &cp, nil
}
func (r *fakeMFATokenRepo) MarkUsed(_ context.Context, id uuid.UUID) (bool, error) {
	t, ok := r.byID[id]
	if !ok || t.UsedAt != nil {
		return false, nil
	}
	now := time.Now().UTC()
	t.UsedAt = &now
	r.byHash[t.TokenHash].UsedAt = &now
	return true, nil
}

type fakeMailer struct {
	sends         int
	lastTo        string
	lastURL       string
	failNext      bool
	lastInviteTo  string
	lastInviteURL string
}

func (m *fakeMailer) SendInvite(_ context.Context, to, _, inviteURL string) error {
	m.lastInviteTo = to
	m.lastInviteURL = inviteURL
	return nil
}

func (m *fakeMailer) SendPasswordReset(_ context.Context, to, resetURL string) error {
	if m.failNext {
		m.failNext = false
		return errors.New("smtp boom")
	}
	m.sends++
	m.lastTo = to
	m.lastURL = resetURL
	return nil
}

// newPhase2Auth builds an AuthService with all Phase 2 deps wired.
func newPhase2Auth() (*AuthService, *fakeResetRepo, *fakeBackupRepo, *fakeMFATokenRepo, *fakeMailer, *fakeSessionRepo) {
	ur := newFakeUserRepo()
	sr := newFakeSessionRepo()
	svc := NewAuthService(ur, sr, newFakeTenants(), time.Hour)
	reset := newFakeResetRepo()
	backup := newFakeBackupRepo()
	mfaTok := newFakeMFATokenRepo()
	mailer := &fakeMailer{}
	svc.ConfigurePasswordReset(reset, mailer, "https://dash.recurso.test/")
	svc.ConfigureMFA(backup, mfaTok)
	return svc, reset, backup, mfaTok, mailer, sr
}

func tokenFromURL(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("bad reset url %q: %v", raw, err)
	}
	tok := u.Query().Get("token")
	if tok == "" {
		t.Fatalf("no token in reset url %q", raw)
	}
	return tok
}

// --- password reset --------------------------------------------------------

func TestRequestPasswordReset_RealAccountEmailsAndCreatesToken(t *testing.T) {
	svc, reset, _, _, mailer, _ := newPhase2Auth()
	if _, err := svc.Register(context.Background(), "Acme", "Alice", "alice@acme.com", "supersecret", ""); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.RequestPasswordReset(context.Background(), "Alice@Acme.com"); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if mailer.sends != 1 {
		t.Fatalf("email sends = %d, want 1", mailer.sends)
	}
	if len(reset.byHash) != 1 {
		t.Fatalf("tokens created = %d, want 1", len(reset.byHash))
	}
	if !strings.HasPrefix(mailer.lastURL, "https://dash.recurso.test/reset-password?token=") {
		t.Fatalf("reset url = %q, want dashboard reset link", mailer.lastURL)
	}
}

func TestRequestPasswordReset_UnknownAccountNoEnumeration(t *testing.T) {
	svc, reset, _, _, mailer, _ := newPhase2Auth()
	if err := svc.RequestPasswordReset(context.Background(), "nobody@nowhere.com"); err != nil {
		t.Fatalf("request reset (unknown) should be a silent success, got %v", err)
	}
	if mailer.sends != 0 || len(reset.byHash) != 0 {
		t.Fatalf("unknown account must not email (%d) or create a token (%d)", mailer.sends, len(reset.byHash))
	}
}

func TestRequestPasswordReset_EmailFailureStillSucceeds(t *testing.T) {
	svc, _, _, _, mailer, _ := newPhase2Auth()
	if _, err := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", ""); err != nil {
		t.Fatalf("register: %v", err)
	}
	mailer.failNext = true
	if err := svc.RequestPasswordReset(context.Background(), "a@b.com"); err != nil {
		t.Fatalf("email failure must not surface to caller, got %v", err)
	}
}

func TestResetPassword_ChangesPasswordAndKillsAllSessions(t *testing.T) {
	svc, _, _, _, mailer, sr := newPhase2Auth()
	if _, err := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", ""); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Register already opened one session; open two more for the user.
	l1, _ := svc.Login(context.Background(), "a@b.com", "supersecret", "ua1")
	l2, _ := svc.Login(context.Background(), "a@b.com", "supersecret", "ua2")
	if len(sr.sessions) != 3 {
		t.Fatalf("sessions = %d, want 3", len(sr.sessions))
	}

	if err := svc.RequestPasswordReset(context.Background(), "a@b.com"); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	rawTok := tokenFromURL(t, mailer.lastURL)

	if err := svc.ResetPassword(context.Background(), rawTok, "brandnewpass"); err != nil {
		t.Fatalf("reset password: %v", err)
	}

	// All prior sessions invalidated.
	if len(sr.sessions) != 0 {
		t.Fatalf("sessions after reset = %d, want 0", len(sr.sessions))
	}
	if _, err := svc.ResolveSession(context.Background(), l1.SessionToken); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("old session 1 should be dead, got %v", err)
	}
	if _, err := svc.ResolveSession(context.Background(), l2.SessionToken); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("old session 2 should be dead, got %v", err)
	}

	// New password works, old does not.
	if _, err := svc.Login(context.Background(), "a@b.com", "brandnewpass", ""); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	if _, err := svc.Login(context.Background(), "a@b.com", "supersecret", ""); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("old password should fail, got %v", err)
	}
}

func TestResetPassword_TokenSingleUse(t *testing.T) {
	svc, _, _, _, mailer, _ := newPhase2Auth()
	svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "") //nolint:errcheck
	svc.RequestPasswordReset(context.Background(), "a@b.com")                         //nolint:errcheck
	rawTok := tokenFromURL(t, mailer.lastURL)

	if err := svc.ResetPassword(context.Background(), rawTok, "brandnewpass"); err != nil {
		t.Fatalf("first reset: %v", err)
	}
	// Reusing the same (now used) token is rejected generically.
	if err := svc.ResetPassword(context.Background(), rawTok, "anotherpass"); !errors.Is(err, domain.ErrInvalidResetToken) {
		t.Fatalf("reused token err = %v, want ErrInvalidResetToken", err)
	}
}

func TestResetPassword_ExpiredAndInvalidRejected(t *testing.T) {
	svc, reset, _, _, mailer, _ := newPhase2Auth()
	svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "") //nolint:errcheck

	// Invalid/unknown token.
	if err := svc.ResetPassword(context.Background(), "totally-bogus", "brandnewpass"); !errors.Is(err, domain.ErrInvalidResetToken) {
		t.Fatalf("bogus token err = %v, want ErrInvalidResetToken", err)
	}

	// Expired token.
	svc.RequestPasswordReset(context.Background(), "a@b.com") //nolint:errcheck
	rawTok := tokenFromURL(t, mailer.lastURL)
	for _, tk := range reset.byHash {
		tk.ExpiresAt = time.Now().Add(-time.Minute)
	}
	if err := svc.ResetPassword(context.Background(), rawTok, "brandnewpass"); !errors.Is(err, domain.ErrInvalidResetToken) {
		t.Fatalf("expired token err = %v, want ErrInvalidResetToken", err)
	}
}

func TestResetPassword_WeakPasswordRejected(t *testing.T) {
	svc, _, _, _, mailer, _ := newPhase2Auth()
	svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "") //nolint:errcheck
	svc.RequestPasswordReset(context.Background(), "a@b.com")                         //nolint:errcheck
	rawTok := tokenFromURL(t, mailer.lastURL)
	if err := svc.ResetPassword(context.Background(), rawTok, "short"); !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("weak password err = %v, want ErrWeakPassword", err)
	}
}

// --- MFA setup / verify / disable ------------------------------------------

// enableMFA is a helper that runs setup+verify and returns the secret + backup codes.
func enableMFA(t *testing.T, svc *AuthService, tenantID, userID uuid.UUID) (secret string, backupCodes []string) {
	t.Helper()
	setup, err := svc.SetupMFA(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("setup mfa: %v", err)
	}
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("totp gen: %v", err)
	}
	codes, err := svc.VerifyAndEnableMFA(context.Background(), tenantID, userID, code)
	if err != nil {
		t.Fatalf("verify mfa: %v", err)
	}
	return setup.Secret, codes
}

func TestMFA_SetupVerifyEnablesAndIssuesBackupCodes(t *testing.T) {
	svc, _, _, _, _, _ := newPhase2Auth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")

	setup, err := svc.SetupMFA(context.Background(), reg.Tenant.ID, reg.User.ID)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if setup.Secret == "" || !strings.HasPrefix(setup.OtpauthURL, "otpauth://") {
		t.Fatalf("bad setup result: %+v", setup)
	}

	// Wrong code rejected; MFA stays off.
	if _, err := svc.VerifyAndEnableMFA(context.Background(), reg.Tenant.ID, reg.User.ID, "000000"); !errors.Is(err, domain.ErrInvalidMFACode) {
		t.Fatalf("wrong code err = %v, want ErrInvalidMFACode", err)
	}

	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	codes, err := svc.VerifyAndEnableMFA(context.Background(), reg.Tenant.ID, reg.User.ID, code)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(codes) != backupCodeCount {
		t.Fatalf("backup codes = %d, want %d", len(codes), backupCodeCount)
	}
	// Now enabled.
	u, _ := svc.users.GetByID(context.Background(), reg.Tenant.ID, reg.User.ID)
	if !u.MFAEnabled {
		t.Fatal("MFA should be enabled after verify")
	}
}

func TestMFA_DisableRequiresValidCode(t *testing.T) {
	svc, _, _, _, _, _ := newPhase2Auth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")
	secret, backup := enableMFA(t, svc, reg.Tenant.ID, reg.User.ID)

	// Wrong code cannot disable.
	if err := svc.DisableMFA(context.Background(), reg.Tenant.ID, reg.User.ID, "000000"); !errors.Is(err, domain.ErrInvalidMFACode) {
		t.Fatalf("disable wrong code err = %v, want ErrInvalidMFACode", err)
	}

	// A valid TOTP disables and wipes the secret.
	_ = backup
	code, _ := totp.GenerateCode(secret, time.Now())
	if err := svc.DisableMFA(context.Background(), reg.Tenant.ID, reg.User.ID, code); err != nil {
		t.Fatalf("disable: %v", err)
	}
	u, _ := svc.users.GetByID(context.Background(), reg.Tenant.ID, reg.User.ID)
	if u.MFAEnabled || u.MFASecret != "" {
		t.Fatalf("MFA should be wiped: enabled=%v secret=%q", u.MFAEnabled, u.MFASecret)
	}
}

// --- two-step login --------------------------------------------------------

func TestLogin_MFAUserReturnsChallengeNoSession(t *testing.T) {
	svc, _, _, mfaTok, _, sr := newPhase2Auth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")
	enableMFA(t, svc, reg.Tenant.ID, reg.User.ID)

	before := len(sr.sessions)
	res, err := svc.Login(context.Background(), "a@b.com", "supersecret", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !res.MFARequired || res.MFAToken == "" {
		t.Fatalf("expected mfa challenge, got %+v", res)
	}
	if res.SessionToken != "" {
		t.Fatal("no session token should be issued at the challenge step")
	}
	if len(sr.sessions) != before {
		t.Fatal("no session should be opened at the challenge step")
	}
	if len(mfaTok.byHash) != 1 {
		t.Fatalf("mfa challenge tokens = %d, want 1", len(mfaTok.byHash))
	}
}

func TestLoginMFA_ValidTOTPOpensSession(t *testing.T) {
	svc, _, _, _, _, sr := newPhase2Auth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")
	secret, _ := enableMFA(t, svc, reg.Tenant.ID, reg.User.ID)

	ch, _ := svc.Login(context.Background(), "a@b.com", "supersecret", "ua")
	code, _ := totp.GenerateCode(secret, time.Now())

	res, err := svc.LoginMFA(context.Background(), ch.MFAToken, code, "ua")
	if err != nil {
		t.Fatalf("login mfa: %v", err)
	}
	if res.SessionToken == "" {
		t.Fatal("expected a session token after mfa")
	}
	// One session from Register + one from the completed MFA login.
	if len(sr.sessions) != 2 {
		t.Fatalf("sessions = %d, want 2", len(sr.sessions))
	}
	if _, err := svc.ResolveSession(context.Background(), res.SessionToken); err != nil {
		t.Fatalf("session should resolve: %v", err)
	}
	// The challenge token is single-use.
	if _, err := svc.LoginMFA(context.Background(), ch.MFAToken, code, "ua"); !errors.Is(err, domain.ErrInvalidMFAToken) {
		t.Fatalf("reused challenge err = %v, want ErrInvalidMFAToken", err)
	}
}

// TestLogin_PerAccountLockout proves ENG-151: after maxFailedLogins wrong
// passwords the account locks, and even the CORRECT password is then refused
// with ErrAccountLocked until the window passes.
func TestLogin_PerAccountLockout(t *testing.T) {
	svc, _, _, _, _, _ := newPhase2Auth()
	_, _ = svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")

	for i := 0; i < maxFailedLogins; i++ {
		if _, err := svc.Login(context.Background(), "a@b.com", "wrongpass", "ua"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("attempt %d err = %v, want ErrInvalidCredentials", i+1, err)
		}
	}
	// Correct password is now rejected — the account is locked.
	if _, err := svc.Login(context.Background(), "a@b.com", "supersecret", "ua"); !errors.Is(err, domain.ErrAccountLocked) {
		t.Fatalf("after %d failures, correct password err = %v, want ErrAccountLocked", maxFailedLogins, err)
	}
}

// TestLogin_FailuresResetOnSuccess proves the counter resets on a successful
// login, so failures don't accumulate forever across sessions.
func TestLogin_FailuresResetOnSuccess(t *testing.T) {
	svc, _, _, _, _, _ := newPhase2Auth()
	_, _ = svc.Register(context.Background(), "Acme", "Bob", "b@b.com", "supersecret", "")

	// Below-threshold failures, then a success clears the counter.
	for i := 0; i < maxFailedLogins-2; i++ {
		_, _ = svc.Login(context.Background(), "b@b.com", "wrongpass", "ua")
	}
	if _, err := svc.Login(context.Background(), "b@b.com", "supersecret", "ua"); err != nil {
		t.Fatalf("correct login should succeed: %v", err)
	}
	// After the reset, another (maxFailedLogins-1) failures still shouldn't lock
	// (would have if the earlier failures had persisted).
	for i := 0; i < maxFailedLogins-1; i++ {
		_, _ = svc.Login(context.Background(), "b@b.com", "wrongpass", "ua")
	}
	if _, err := svc.Login(context.Background(), "b@b.com", "supersecret", "ua"); err != nil {
		t.Fatalf("counter should have reset; correct login should still succeed: %v", err)
	}
}

// TestLoginMFA_TOTPSingleUse_ReplayRejected proves ENG-151: a TOTP code, once
// used to complete a login, cannot be replayed on a fresh challenge within its
// validity window — the consumed timestep is remembered and rejected.
func TestLoginMFA_TOTPSingleUse_ReplayRejected(t *testing.T) {
	svc, _, _, _, _, _ := newPhase2Auth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")
	secret, _ := enableMFA(t, svc, reg.Tenant.ID, reg.User.ID)

	code, _ := totp.GenerateCode(secret, time.Now())

	// First use: valid.
	ch1, _ := svc.Login(context.Background(), "a@b.com", "supersecret", "ua")
	if _, err := svc.LoginMFA(context.Background(), ch1.MFAToken, code, "ua"); err != nil {
		t.Fatalf("first LoginMFA should succeed: %v", err)
	}

	// Replay the SAME code on a brand-new challenge (a captured-code attack).
	ch2, _ := svc.Login(context.Background(), "a@b.com", "supersecret", "ua")
	if _, err := svc.LoginMFA(context.Background(), ch2.MFAToken, code, "ua"); !errors.Is(err, domain.ErrInvalidMFACode) {
		t.Fatalf("replayed TOTP err = %v, want ErrInvalidMFACode (single-use)", err)
	}
}

func TestLoginMFA_BackupCodeWorksOnceThenConsumed(t *testing.T) {
	svc, _, _, _, _, _ := newPhase2Auth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")
	_, backup := enableMFA(t, svc, reg.Tenant.ID, reg.User.ID)
	backupCode := backup[0]

	// First use: succeeds.
	ch1, _ := svc.Login(context.Background(), "a@b.com", "supersecret", "ua")
	if _, err := svc.LoginMFA(context.Background(), ch1.MFAToken, backupCode, "ua"); err != nil {
		t.Fatalf("backup code login: %v", err)
	}

	// Second use of the same backup code: rejected.
	ch2, _ := svc.Login(context.Background(), "a@b.com", "supersecret", "ua")
	if _, err := svc.LoginMFA(context.Background(), ch2.MFAToken, backupCode, "ua"); !errors.Is(err, domain.ErrInvalidMFACode) {
		t.Fatalf("consumed backup code err = %v, want ErrInvalidMFACode", err)
	}
}

func TestLoginMFA_ExpiredChallengeRejected(t *testing.T) {
	svc, _, _, mfaTok, _, _ := newPhase2Auth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")
	secret, _ := enableMFA(t, svc, reg.Tenant.ID, reg.User.ID)

	ch, _ := svc.Login(context.Background(), "a@b.com", "supersecret", "ua")
	// Force the challenge token to be expired.
	for _, tk := range mfaTok.byHash {
		tk.ExpiresAt = time.Now().Add(-time.Minute)
	}
	code, _ := totp.GenerateCode(secret, time.Now())
	if _, err := svc.LoginMFA(context.Background(), ch.MFAToken, code, "ua"); !errors.Is(err, domain.ErrInvalidMFAToken) {
		t.Fatalf("expired challenge err = %v, want ErrInvalidMFAToken", err)
	}
}

func TestLogin_NoMFAStillOneStep(t *testing.T) {
	svc, _, _, _, _, sr := newPhase2Auth()
	svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "") //nolint:errcheck

	res, err := svc.Login(context.Background(), "a@b.com", "supersecret", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.MFARequired {
		t.Fatal("no-MFA user must not be challenged")
	}
	// Register opened one session; this one-step login adds exactly one more.
	if res.SessionToken == "" || len(sr.sessions) != 2 {
		t.Fatalf("one-step login should open exactly one session; token=%q sessions=%d", res.SessionToken, len(sr.sessions))
	}
}

// --- session management ----------------------------------------------------

func TestSessions_ListShowsCurrentFlag(t *testing.T) {
	svc, _, _, _, _, _ := newPhase2Auth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")
	current := reg.SessionToken
	// A second, different session.
	svc.Login(context.Background(), "a@b.com", "supersecret", "other") //nolint:errcheck

	list, err := svc.ListActiveSessions(context.Background(), reg.User.ID, current)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("active sessions = %d, want 2", len(list))
	}
	currentCount := 0
	for _, s := range list {
		if s.Current {
			currentCount++
		}
	}
	if currentCount != 1 {
		t.Fatalf("exactly one session should be marked current, got %d", currentCount)
	}
}

func TestSessions_RevokeOneOnlyAffectsOwnSession(t *testing.T) {
	svc, _, _, _, _, sr := newPhase2Auth()
	a, _ := svc.Register(context.Background(), "TenantA", "Alice", "alice@a.com", "supersecret", "")
	b, _ := svc.Register(context.Background(), "TenantB", "Bob", "bob@b.com", "supersecret", "")

	// Find Bob's session ID.
	var bobSessionID uuid.UUID
	for _, s := range sr.sessions {
		if s.UserID == b.User.ID {
			bobSessionID = s.ID
		}
	}

	// Alice cannot revoke Bob's session → 404-style not found.
	if err := svc.RevokeSession(context.Background(), a.User.ID, bobSessionID); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("cross-user revoke err = %v, want ErrSessionNotFound", err)
	}
	// Bob's session is untouched.
	if _, err := svc.ResolveSession(context.Background(), b.SessionToken); err != nil {
		t.Fatalf("bob session should survive: %v", err)
	}

	// Alice revokes her own session.
	var aliceSessionID uuid.UUID
	for _, s := range sr.sessions {
		if s.UserID == a.User.ID {
			aliceSessionID = s.ID
		}
	}
	if err := svc.RevokeSession(context.Background(), a.User.ID, aliceSessionID); err != nil {
		t.Fatalf("self revoke: %v", err)
	}
	if _, err := svc.ResolveSession(context.Background(), a.SessionToken); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("alice session should be gone: %v", err)
	}
}

func TestSessions_RevokeOthersKeepsCurrent(t *testing.T) {
	svc, _, _, _, _, sr := newPhase2Auth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")
	current := reg.SessionToken
	// Two more sessions.
	svc.Login(context.Background(), "a@b.com", "supersecret", "phone")  //nolint:errcheck
	svc.Login(context.Background(), "a@b.com", "supersecret", "tablet") //nolint:errcheck
	if len(sr.sessions) != 3 {
		t.Fatalf("sessions = %d, want 3", len(sr.sessions))
	}

	if err := svc.RevokeOtherSessions(context.Background(), reg.User.ID, current); err != nil {
		t.Fatalf("revoke others: %v", err)
	}
	if len(sr.sessions) != 1 {
		t.Fatalf("sessions after revoke-others = %d, want 1", len(sr.sessions))
	}
	// The current session survives.
	if _, err := svc.ResolveSession(context.Background(), current); err != nil {
		t.Fatalf("current session should survive: %v", err)
	}
}
