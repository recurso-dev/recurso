package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
)

// --- Phase 2 handler fakes -------------------------------------------------

type memResetRepo struct {
	byHash map[string]*domain.PasswordResetToken
	byID   map[uuid.UUID]*domain.PasswordResetToken
}

func newMemResetRepo() *memResetRepo {
	return &memResetRepo{byHash: map[string]*domain.PasswordResetToken{}, byID: map[uuid.UUID]*domain.PasswordResetToken{}}
}
func (r *memResetRepo) Create(_ context.Context, t *domain.PasswordResetToken) error {
	cp := *t
	r.byHash[t.TokenHash] = &cp
	r.byID[t.ID] = &cp
	return nil
}
func (r *memResetRepo) GetByTokenHash(_ context.Context, h string) (*domain.PasswordResetToken, error) {
	t, ok := r.byHash[h]
	if !ok {
		return nil, domain.ErrInvalidResetToken
	}
	cp := *t
	return &cp, nil
}
func (r *memResetRepo) MarkUsed(_ context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	if t, ok := r.byID[id]; ok {
		t.UsedAt = &now
		r.byHash[t.TokenHash].UsedAt = &now
	}
	return nil
}

type memBackupRepo struct{ codes []*domain.MFABackupCode }

func (r *memBackupRepo) CreateMany(_ context.Context, codes []*domain.MFABackupCode) error {
	for _, c := range codes {
		cp := *c
		r.codes = append(r.codes, &cp)
	}
	return nil
}
func (r *memBackupRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*domain.MFABackupCode, error) {
	var out []*domain.MFABackupCode
	for _, c := range r.codes {
		if c.UserID == userID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *memBackupRepo) MarkUsed(_ context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	for _, c := range r.codes {
		if c.ID == id {
			c.UsedAt = &now
		}
	}
	return nil
}
func (r *memBackupRepo) DeleteByUser(_ context.Context, userID uuid.UUID) error {
	var kept []*domain.MFABackupCode
	for _, c := range r.codes {
		if c.UserID != userID {
			kept = append(kept, c)
		}
	}
	r.codes = kept
	return nil
}

type memMFATokenRepo struct {
	byHash map[string]*domain.MFALoginToken
	byID   map[uuid.UUID]*domain.MFALoginToken
}

func newMemMFATokenRepo() *memMFATokenRepo {
	return &memMFATokenRepo{byHash: map[string]*domain.MFALoginToken{}, byID: map[uuid.UUID]*domain.MFALoginToken{}}
}
func (r *memMFATokenRepo) Create(_ context.Context, t *domain.MFALoginToken) error {
	cp := *t
	r.byHash[t.TokenHash] = &cp
	r.byID[t.ID] = &cp
	return nil
}
func (r *memMFATokenRepo) GetByTokenHash(_ context.Context, h string) (*domain.MFALoginToken, error) {
	t, ok := r.byHash[h]
	if !ok {
		return nil, domain.ErrInvalidMFAToken
	}
	cp := *t
	return &cp, nil
}
func (r *memMFATokenRepo) MarkUsed(_ context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	if t, ok := r.byID[id]; ok {
		t.UsedAt = &now
		r.byHash[t.TokenHash].UsedAt = &now
	}
	return nil
}

type memMailer struct {
	sends   int
	lastURL string
}

func (m *memMailer) SendPasswordReset(_ context.Context, _, resetURL string) error {
	m.sends++
	m.lastURL = resetURL
	return nil
}

// phase2Fixture wires an AuthService with shared in-memory repos so a test can
// both drive the handler and inspect state (e.g. read a session's real ID or a
// MFA secret) through the same backing store.
type phase2Fixture struct {
	svc    *service.AuthService
	users  *memUserRepo
	sess   *memSessionRepo
	reset  *memResetRepo
	backup *memBackupRepo
	mfaTok *memMFATokenRepo
	mailer *memMailer
	h      *AuthHandler
}

func newPhase2Fixture() *phase2Fixture {
	users := newMemUserRepo()
	sess := newMemSessionRepo()
	reset := newMemResetRepo()
	backup := &memBackupRepo{}
	mfaTok := newMemMFATokenRepo()
	mailer := &memMailer{}
	svc := service.NewAuthService(users, sess, &memTenants{tenants: map[uuid.UUID]*domain.Tenant{}}, time.Hour)
	svc.ConfigurePasswordReset(reset, mailer, "https://dash.test")
	svc.ConfigureMFA(backup, mfaTok)
	return &phase2Fixture{svc: svc, users: users, sess: sess, reset: reset, backup: backup, mfaTok: mfaTok, mailer: mailer, h: NewAuthHandler(svc, false)}
}

func (f *phase2Fixture) register(t *testing.T, email string) *service.RegisterResult {
	t.Helper()
	res, err := f.svc.Register(context.Background(), "Acme", "Alice", email, "supersecret", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return res
}

// --- forgot / reset password -----------------------------------------------

func TestForgotPassword_AlwaysGeneric200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newPhase2Fixture()
	f.register(t, "a@b.com")

	// Known account.
	c1, w1 := jsonCtx(http.MethodPost, "/auth/forgot-password", `{"email":"a@b.com"}`)
	f.h.ForgotPassword(c1)
	// Unknown account.
	c2, w2 := jsonCtx(http.MethodPost, "/auth/forgot-password", `{"email":"nobody@x.com"}`)
	f.h.ForgotPassword(c2)

	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Fatalf("codes = %d/%d, want 200/200", w1.Code, w2.Code)
	}
	if w1.Body.String() != w2.Body.String() {
		t.Fatalf("responses differ (enumeration): %q vs %q", w1.Body.String(), w2.Body.String())
	}
	if f.mailer.sends != 1 {
		t.Fatalf("email sends = %d, want 1 (only for the real account)", f.mailer.sends)
	}
}

func TestResetPassword_HandlerFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newPhase2Fixture()
	f.register(t, "a@b.com")

	cReq, _ := jsonCtx(http.MethodPost, "/auth/forgot-password", `{"email":"a@b.com"}`)
	f.h.ForgotPassword(cReq)
	u, _ := url.Parse(f.mailer.lastURL)
	token := u.Query().Get("token")

	// Invalid token → generic 400.
	cBad, wBad := jsonCtx(http.MethodPost, "/auth/reset-password", `{"token":"nope","password":"newpassword"}`)
	f.h.ResetPassword(cBad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("invalid token status = %d, want 400", wBad.Code)
	}

	// Valid token → 200 and password changes.
	cOK, wOK := jsonCtx(http.MethodPost, "/auth/reset-password", `{"token":"`+token+`","password":"newpassword"}`)
	f.h.ResetPassword(cOK)
	if wOK.Code != http.StatusOK {
		t.Fatalf("valid reset status = %d body=%s, want 200", wOK.Code, wOK.Body.String())
	}
	if _, err := f.svc.Login(context.Background(), "a@b.com", "newpassword", ""); err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}
}

// --- two-step MFA login via handlers ---------------------------------------

// enableMFAViaHandler runs setup+verify through the HTTP handlers for the given
// authed user, returning the TOTP secret and backup codes.
func (f *phase2Fixture) enableMFAViaHandler(t *testing.T, tenantID, userID uuid.UUID) (string, []string) {
	t.Helper()
	// setup
	cSetup, wSetup := jsonCtx(http.MethodPost, "/v1/auth/mfa/setup", "")
	cSetup.Set("tenant_id", tenantID)
	cSetup.Set("user_id", userID)
	f.h.MFASetup(cSetup)
	if wSetup.Code != http.StatusOK {
		t.Fatalf("mfa setup status = %d body=%s", wSetup.Code, wSetup.Body.String())
	}
	var setupResp struct {
		Secret     string `json:"secret"`
		OtpauthURL string `json:"otpauth_url"`
	}
	_ = json.Unmarshal(wSetup.Body.Bytes(), &setupResp)

	code, _ := totp.GenerateCode(setupResp.Secret, time.Now())
	cVer, wVer := jsonCtx(http.MethodPost, "/v1/auth/mfa/verify", `{"code":"`+code+`"}`)
	cVer.Set("tenant_id", tenantID)
	cVer.Set("user_id", userID)
	f.h.MFAVerify(cVer)
	if wVer.Code != http.StatusOK {
		t.Fatalf("mfa verify status = %d body=%s", wVer.Code, wVer.Body.String())
	}
	var verResp struct {
		BackupCodes []string `json:"backup_codes"`
	}
	_ = json.Unmarshal(wVer.Body.Bytes(), &verResp)
	return setupResp.Secret, verResp.BackupCodes
}

func TestMFALoginFlow_ChallengeThenTOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newPhase2Fixture()
	reg := f.register(t, "a@b.com")
	secret, _ := f.enableMFAViaHandler(t, reg.Tenant.ID, reg.User.ID)

	// Login now returns an MFA challenge, no session cookie.
	cLogin, wLogin := jsonCtx(http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"supersecret"}`)
	f.h.Login(cLogin)
	if wLogin.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", wLogin.Code)
	}
	if hasSessionCookie(wLogin) {
		t.Fatal("no session cookie should be set at the challenge step")
	}
	var loginResp struct {
		MFARequired bool   `json:"mfa_required"`
		MFAToken    string `json:"mfa_token"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	if !loginResp.MFARequired || loginResp.MFAToken == "" {
		t.Fatalf("expected mfa challenge, got %s", wLogin.Body.String())
	}

	// Exchange the challenge + TOTP for a session.
	code, _ := totp.GenerateCode(secret, time.Now())
	cMFA, wMFA := jsonCtx(http.MethodPost, "/auth/login/mfa", `{"mfa_token":"`+loginResp.MFAToken+`","code":"`+code+`"}`)
	f.h.LoginMFA(cMFA)
	if wMFA.Code != http.StatusOK {
		t.Fatalf("login/mfa status = %d body=%s, want 200", wMFA.Code, wMFA.Body.String())
	}
	if !hasSessionCookie(wMFA) {
		t.Fatal("expected a session cookie after successful MFA")
	}
}

func TestMFALoginFlow_WrongCodeRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newPhase2Fixture()
	reg := f.register(t, "a@b.com")
	f.enableMFAViaHandler(t, reg.Tenant.ID, reg.User.ID)

	cLogin, wLogin := jsonCtx(http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"supersecret"}`)
	f.h.Login(cLogin)
	var loginResp struct {
		MFAToken string `json:"mfa_token"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)

	cMFA, wMFA := jsonCtx(http.MethodPost, "/auth/login/mfa", `{"mfa_token":"`+loginResp.MFAToken+`","code":"000000"}`)
	f.h.LoginMFA(cMFA)
	if wMFA.Code != http.StatusUnauthorized {
		t.Fatalf("wrong mfa code status = %d, want 401", wMFA.Code)
	}
	if hasSessionCookie(wMFA) {
		t.Fatal("no session on wrong mfa code")
	}
}

func TestMFAEndpoints_RejectAPIKeyCaller(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newPhase2Fixture()
	// No user_id in context == API-key (machine) caller.
	c, w := jsonCtx(http.MethodPost, "/v1/auth/mfa/setup", "")
	c.Set("tenant_id", uuid.New())
	f.h.MFASetup(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("api-key caller status = %d, want 401 (no user)", w.Code)
	}
}

// --- session management via handlers ---------------------------------------

func TestSessionsHandler_ListRevokeOneAndOthers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newPhase2Fixture()
	reg := f.register(t, "a@b.com")
	current := reg.SessionToken
	// Two more sessions on other devices.
	f.svc.Login(context.Background(), "a@b.com", "supersecret", "phone")  //nolint:errcheck
	f.svc.Login(context.Background(), "a@b.com", "supersecret", "tablet") //nolint:errcheck

	// List: 3 sessions, exactly one current.
	cList, wList := jsonCtx(http.MethodGet, "/v1/auth/sessions", "")
	cList.Set("user_id", reg.User.ID)
	cList.Set("tenant_id", reg.Tenant.ID)
	cList.Request.AddCookie(&http.Cookie{Name: domain.SessionCookieName, Value: current})
	f.h.ListSessions(cList)
	if wList.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", wList.Code, wList.Body.String())
	}
	var listResp struct {
		Data []sessionView `json:"data"`
	}
	_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
	if len(listResp.Data) != 3 {
		t.Fatalf("sessions listed = %d, want 3", len(listResp.Data))
	}
	currentCount := 0
	var otherID string
	for _, s := range listResp.Data {
		if s.Current {
			currentCount++
		} else {
			otherID = s.ID
		}
	}
	if currentCount != 1 {
		t.Fatalf("current flag count = %d, want 1", currentCount)
	}

	// Revoke one (a non-current session).
	cDel, wDel := jsonCtx(http.MethodDelete, "/v1/auth/sessions/"+otherID, "")
	cDel.Params = gin.Params{{Key: "id", Value: otherID}}
	cDel.Set("user_id", reg.User.ID)
	cDel.Set("tenant_id", reg.Tenant.ID)
	f.h.RevokeSession(cDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("revoke-one status = %d body=%s", wDel.Code, wDel.Body.String())
	}

	// Revoke others: keep only current.
	cOthers, wOthers := jsonCtx(http.MethodDelete, "/v1/auth/sessions", "")
	cOthers.Set("user_id", reg.User.ID)
	cOthers.Set("tenant_id", reg.Tenant.ID)
	cOthers.Request.AddCookie(&http.Cookie{Name: domain.SessionCookieName, Value: current})
	f.h.RevokeOtherSessions(cOthers)
	if wOthers.Code != http.StatusOK {
		t.Fatalf("revoke-others status = %d body=%s", wOthers.Code, wOthers.Body.String())
	}

	// Only the current session remains active.
	remaining, _ := f.svc.ListActiveSessions(context.Background(), reg.User.ID, current)
	if len(remaining) != 1 || !remaining[0].Current {
		t.Fatalf("after revoke-others, remaining = %d (want 1, current)", len(remaining))
	}
}

func TestSessionsHandler_RevokeOneNotOwnedIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newPhase2Fixture()
	f.register(t, "a@b.com")
	// A different user with a session.
	other := f.register(t, "b@c.com")
	var otherSessionID uuid.UUID
	for _, s := range f.sess.sessions {
		if s.UserID == other.User.ID {
			otherSessionID = s.ID
		}
	}

	attacker := uuid.New()
	c, w := jsonCtx(http.MethodDelete, "/v1/auth/sessions/"+otherSessionID.String(), "")
	c.Params = gin.Params{{Key: "id", Value: otherSessionID.String()}}
	c.Set("user_id", attacker)
	c.Set("tenant_id", uuid.New())
	f.h.RevokeSession(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("revoke of another user's session status = %d, want 404", w.Code)
	}
	if strings.Contains(strings.ToLower(w.Body.String()), "forbidden") {
		t.Fatal("should be a plain 404, not reveal ownership")
	}
}
