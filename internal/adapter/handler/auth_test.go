package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// --- in-memory fakes (mirror the ports; kept local to the handler package) ---

type memUserRepo struct{ users map[uuid.UUID]*domain.User }

func newMemUserRepo() *memUserRepo { return &memUserRepo{users: map[uuid.UUID]*domain.User{}} }

func (r *memUserRepo) Create(_ context.Context, u *domain.User) error {
	for _, e := range r.users {
		if strings.EqualFold(e.Email, u.Email) {
			return domain.ErrDuplicateEmail
		}
	}
	cp := *u
	r.users[u.ID] = &cp
	return nil
}
func (r *memUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if strings.EqualFold(u.Email, email) {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
func (r *memUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, err := r.GetByEmail(context.Background(), email)
	return err == nil, nil
}
func (r *memUserRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.User, error) {
	if u, ok := r.users[id]; ok && u.TenantID == tenantID {
		cp := *u
		return &cp, nil
	}
	return nil, domain.ErrUserNotFound
}
func (r *memUserRepo) GetByIDGlobal(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if u, ok := r.users[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, domain.ErrUserNotFound
}
func (r *memUserRepo) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	if u, ok := r.users[id]; ok {
		u.PasswordHash = hash
		return nil
	}
	return domain.ErrUserNotFound
}
func (r *memUserRepo) SetMFASecret(_ context.Context, tenantID, id uuid.UUID, secret string) error {
	if u, ok := r.users[id]; ok && u.TenantID == tenantID {
		u.MFASecret = secret
		return nil
	}
	return domain.ErrUserNotFound
}
func (r *memUserRepo) SetMFAEnabled(_ context.Context, tenantID, id uuid.UUID, enabled bool) error {
	if u, ok := r.users[id]; ok && u.TenantID == tenantID {
		u.MFAEnabled = enabled
		return nil
	}
	return domain.ErrUserNotFound
}
func (r *memUserRepo) SetMFALastTimestep(_ context.Context, tenantID, id uuid.UUID, timestep int64) error {
	if u, ok := r.users[id]; ok && u.TenantID == tenantID {
		if timestep > u.MFALastTimestep {
			u.MFALastTimestep = timestep
		}
		return nil
	}
	return domain.ErrUserNotFound
}
func (r *memUserRepo) RegisterFailedLogin(_ context.Context, id uuid.UUID, lockThreshold int, lockFor time.Duration) error {
	if u, ok := r.users[id]; ok {
		u.FailedLoginAttempts++
		if u.FailedLoginAttempts >= lockThreshold {
			t := time.Now().Add(lockFor)
			u.LockedUntil = &t
		}
	}
	return nil
}
func (r *memUserRepo) ClearFailedLogins(_ context.Context, id uuid.UUID) error {
	if u, ok := r.users[id]; ok {
		u.FailedLoginAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}
func (r *memUserRepo) ClearMFA(_ context.Context, tenantID, id uuid.UUID) error {
	if u, ok := r.users[id]; ok && u.TenantID == tenantID {
		u.MFAEnabled = false
		u.MFASecret = ""
		return nil
	}
	return domain.ErrUserNotFound
}
func (r *memUserRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*domain.User, error) {
	var out []*domain.User
	for _, u := range r.users {
		if u.TenantID == tenantID {
			cp := *u
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *memUserRepo) UpdateRole(_ context.Context, tenantID, id uuid.UUID, role domain.Role) error {
	if u, ok := r.users[id]; ok && u.TenantID == tenantID {
		u.Role = role
		return nil
	}
	return domain.ErrUserNotFound
}
func (r *memUserRepo) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	if u, ok := r.users[id]; ok && u.TenantID == tenantID {
		delete(r.users, id)
		return nil
	}
	return domain.ErrUserNotFound
}
func (r *memUserRepo) CountOwners(_ context.Context, tenantID uuid.UUID) (int, error) {
	n := 0
	for _, u := range r.users {
		if u.TenantID == tenantID && u.Role == domain.RoleOwner {
			n++
		}
	}
	return n, nil
}

type memSessionRepo struct{ sessions map[string]*domain.Session }

func newMemSessionRepo() *memSessionRepo {
	return &memSessionRepo{sessions: map[string]*domain.Session{}}
}
func (r *memSessionRepo) Create(_ context.Context, s *domain.Session) error {
	cp := *s
	r.sessions[s.TokenHash] = &cp
	return nil
}
func (r *memSessionRepo) GetByTokenHash(_ context.Context, h string) (*domain.Session, error) {
	if s, ok := r.sessions[h]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, domain.ErrSessionNotFound
}
func (r *memSessionRepo) DeleteByTokenHash(_ context.Context, h string) error {
	delete(r.sessions, h)
	return nil
}
func (r *memSessionRepo) DeleteByUser(_ context.Context, userID uuid.UUID) error {
	for h, s := range r.sessions {
		if s.UserID == userID {
			delete(r.sessions, h)
		}
	}
	return nil
}
func (r *memSessionRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Session, error) {
	for _, s := range r.sessions {
		if s.ID == id {
			cp := *s
			return &cp, nil
		}
	}
	return nil, domain.ErrSessionNotFound
}
func (r *memSessionRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*domain.Session, error) {
	var out []*domain.Session
	for _, s := range r.sessions {
		if s.UserID == userID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *memSessionRepo) DeleteByID(_ context.Context, id uuid.UUID) error {
	for h, s := range r.sessions {
		if s.ID == id {
			delete(r.sessions, h)
		}
	}
	return nil
}
func (r *memSessionRepo) DeleteByUserExcept(_ context.Context, userID uuid.UUID, exceptHash string) error {
	for h, s := range r.sessions {
		if s.UserID == userID && h != exceptHash {
			delete(r.sessions, h)
		}
	}
	return nil
}

type memTenants struct{ tenants map[uuid.UUID]*domain.Tenant }

func (m *memTenants) Register(_ context.Context, name, email string) (*domain.Tenant, *domain.APIKey, error) {
	t := &domain.Tenant{ID: uuid.New(), Name: name, Email: email, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.tenants[t.ID] = t
	return t, &domain.APIKey{ID: uuid.New(), TenantID: t.ID, KeyValue: "sk_live_" + uuid.NewString(), Type: "secret", IsActive: true}, nil
}
func (m *memTenants) GetAccount(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("not found")
}

func newTestAuthService() *service.AuthService {
	return service.NewAuthService(newMemUserRepo(), newMemSessionRepo(), &memTenants{tenants: map[uuid.UUID]*domain.Tenant{}}, time.Hour)
}

func jsonCtx(method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

// --- auth endpoint tests ---

func TestRegisterHandler_SetsCookieAndReturnsKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(newTestAuthService(), false)

	c, w := jsonCtx(http.MethodPost, "/auth/register", `{"company_name":"Acme","name":"Alice","email":"alice@acme.com","password":"supersecret"}`)
	h.Register(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s, want 201", w.Code, w.Body.String())
	}
	var resp struct {
		APIKey string   `json:"api_key"`
		User   userView `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if resp.APIKey == "" {
		t.Error("expected api_key in body")
	}
	if resp.User.Role != "owner" {
		t.Errorf("role = %q, want owner", resp.User.Role)
	}
	if !hasSessionCookie(w) {
		t.Error("expected recurso_session cookie to be set")
	}
}

func TestRegisterHandler_DuplicateEmailConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestAuthService()
	h := NewAuthHandler(svc, false)

	body := `{"company_name":"Acme","name":"Alice","email":"a@b.com","password":"supersecret"}`
	c1, _ := jsonCtx(http.MethodPost, "/auth/register", body)
	h.Register(c1)

	c2, w2 := jsonCtx(http.MethodPost, "/auth/register", body)
	h.Register(c2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 for duplicate email", w2.Code)
	}
}

func TestLoginHandler_GenericErrorNoCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestAuthService()
	h := NewAuthHandler(svc, false)
	reg, _ := jsonCtx(http.MethodPost, "/auth/register", `{"company_name":"Acme","name":"Alice","email":"a@b.com","password":"supersecret"}`)
	h.Register(reg)

	// Wrong password.
	c, w := jsonCtx(http.MethodPost, "/auth/login", `{"email":"a@b.com","password":"WRONGWRONG"}`)
	h.Login(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if strings.Contains(strings.ToLower(w.Body.String()), "password") || strings.Contains(strings.ToLower(w.Body.String()), "email") {
		t.Errorf("login error leaks which field was wrong: %s", w.Body.String())
	}
	if hasSessionCookie(w) {
		t.Error("no cookie should be set on failed login")
	}

	// Unknown email → identical generic response.
	c2, w2 := jsonCtx(http.MethodPost, "/auth/login", `{"email":"nobody@x.com","password":"supersecret"}`)
	h.Login(c2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("unknown-email status = %d, want 401", w2.Code)
	}
}

func TestMeAndLogout_Flow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestAuthService()
	h := NewAuthHandler(svc, false)

	reg, wReg := jsonCtx(http.MethodPost, "/auth/register", `{"company_name":"Acme","name":"Alice","email":"a@b.com","password":"supersecret"}`)
	h.Register(reg)
	token := sessionCookieValue(wReg)
	if token == "" {
		t.Fatal("no session cookie from register")
	}

	// /auth/me with the cookie resolves the user.
	cMe, wMe := jsonCtx(http.MethodGet, "/auth/me", "")
	cMe.Request.AddCookie(&http.Cookie{Name: domain.SessionCookieName, Value: token})
	h.Me(cMe)
	if wMe.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s, want 200", wMe.Code, wMe.Body.String())
	}

	// Logout invalidates the session.
	cOut, _ := jsonCtx(http.MethodPost, "/auth/logout", "")
	cOut.Request.AddCookie(&http.Cookie{Name: domain.SessionCookieName, Value: token})
	h.Logout(cOut)

	cMe2, wMe2 := jsonCtx(http.MethodGet, "/auth/me", "")
	cMe2.Request.AddCookie(&http.Cookie{Name: domain.SessionCookieName, Value: token})
	h.Me(cMe2)
	if wMe2.Code != http.StatusUnauthorized {
		t.Fatalf("me-after-logout status = %d, want 401", wMe2.Code)
	}
}

// --- team role-gate tests ---

func TestTeamCreate_MemberForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	th := NewTeamHandler(newTestAuthService())

	c, w := jsonCtx(http.MethodPost, "/v1/users", `{"email":"x@y.com","name":"X","role":"member","password":"supersecret"}`)
	c.Set("tenant_id", uuid.New())
	c.Set("user_role", "member") // session user who is a member
	th.CreateUser(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for member", w.Code)
	}
}

func TestTeamCreate_AdminAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	th := NewTeamHandler(newTestAuthService())
	tenantID := uuid.New()

	c, w := jsonCtx(http.MethodPost, "/v1/users", `{"email":"x@y.com","name":"X","role":"member","password":"supersecret"}`)
	c.Set("tenant_id", tenantID)
	c.Set("user_role", "admin")
	th.CreateUser(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s, want 201 for admin", w.Code, w.Body.String())
	}
}

func TestTeamCreate_APIKeyCallerAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	th := NewTeamHandler(newTestAuthService())

	// No user_role set == API-key (machine) caller: allowed (full tenant access).
	c, w := jsonCtx(http.MethodPost, "/v1/users", `{"email":"x@y.com","name":"X","role":"member","password":"supersecret"}`)
	c.Set("tenant_id", uuid.New())
	th.CreateUser(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s, want 201 for API-key caller", w.Code, w.Body.String())
	}
}

// --- helpers ---

func hasSessionCookie(w *httptest.ResponseRecorder) bool {
	return sessionCookieValue(w) != ""
}

func sessionCookieValue(w *httptest.ResponseRecorder) string {
	for _, c := range w.Result().Cookies() {
		if c.Name == domain.SessionCookieName && c.Value != "" && c.MaxAge >= 0 {
			return c.Value
		}
	}
	return ""
}
