package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/service"
	"golang.org/x/oauth2"
)

// --- fake oauth identity repo (handler package) ---

type memOAuthIdentityRepo struct {
	byKey map[string]*domain.OAuthIdentity
}

func newMemOAuthIdentityRepo() *memOAuthIdentityRepo {
	return &memOAuthIdentityRepo{byKey: map[string]*domain.OAuthIdentity{}}
}
func (r *memOAuthIdentityRepo) Create(_ context.Context, i *domain.OAuthIdentity) error {
	k := i.Provider + "|" + i.ProviderUserID
	if _, ok := r.byKey[k]; ok {
		return domain.ErrDuplicateEmail
	}
	cp := *i
	r.byKey[k] = &cp
	return nil
}
func (r *memOAuthIdentityRepo) GetByProviderUserID(_ context.Context, provider, uid string) (*domain.OAuthIdentity, error) {
	if i, ok := r.byKey[provider+"|"+uid]; ok {
		cp := *i
		return &cp, nil
	}
	return nil, domain.ErrUserNotFound
}
func (r *memOAuthIdentityRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*domain.OAuthIdentity, error) {
	var out []*domain.OAuthIdentity
	for _, i := range r.byKey {
		if i.UserID == userID {
			cp := *i
			out = append(out, &cp)
		}
	}
	return out, nil
}

func oauthMockProvider(t *testing.T, userInfoJSON string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"Bearer"}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(userInfoJSON))
	})
	return httptest.NewServer(mux)
}

func newOAuthHandler(t *testing.T, srv *httptest.Server) (*OAuthHandler, *service.AuthService) {
	t.Helper()
	auth := service.NewAuthService(newMemUserRepo(), newMemSessionRepo(), &memTenants{tenants: map[uuid.UUID]*domain.Tenant{}}, time.Hour)
	auth.ConfigureOAuth(newMemOAuthIdentityRepo())
	provider := &service.OAuthProvider{
		Name:         service.ProviderGoogle,
		ClientID:     "cid",
		ClientSecret: "secret",
		Endpoint:     oauth2.Endpoint{AuthURL: srv.URL + "/authorize", TokenURL: srv.URL + "/token"},
		UserInfoURL:  srv.URL + "/userinfo",
		RedirectURL:  "http://localhost:8080/auth/oauth/google/callback",
		HTTPClient:   srv.Client(),
	}
	reg := service.NewOAuthRegistryWithProviders(provider)
	h := NewOAuthHandler(auth, reg, "http://dash.local", []byte("test-state-secret"), false)
	return h, auth
}

// runStart drives /start and returns the state cookie + the state param bound
// into the provider redirect.
func runStart(t *testing.T, h *OAuthHandler, provider string) (*http.Cookie, string) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/oauth/"+provider+"/start", nil)
	c.Params = gin.Params{{Key: "provider", Value: provider}}
	h.Start(c)
	if w.Code != http.StatusFound {
		t.Fatalf("start status = %d, want 302", w.Code)
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("bad redirect: %v", err)
	}
	state := loc.Query().Get("state")
	var cookie *http.Cookie
	for _, ck := range w.Result().Cookies() {
		if ck.Name == oauthStateCookieName {
			cookie = ck
		}
	}
	if cookie == nil || state == "" {
		t.Fatal("start did not set a state cookie + state param")
	}
	return cookie, state
}

func runCallback(t *testing.T, h *OAuthHandler, provider, code, state string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/oauth/"+provider+"/callback?code="+code+"&state="+url.QueryEscape(state), nil)
	if cookie != nil {
		c.Request.AddCookie(cookie)
	}
	c.Params = gin.Params{{Key: "provider", Value: provider}}
	h.Callback(c)
	return w
}

func TestOAuthCallback_BrandNewCreatesSessionAndRedirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := oauthMockProvider(t, `{"sub":"new-1","email":"founder@startup.io","email_verified":true,"name":"Founder"}`)
	defer srv.Close()
	h, _ := newOAuthHandler(t, srv)

	cookie, state := runStart(t, h, "google")
	w := runCallback(t, h, "google", "the-code", state, cookie)

	if w.Code != http.StatusFound {
		t.Fatalf("callback status = %d body=%s, want 302", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "http://dash.local/" {
		t.Fatalf("redirect = %q, want dashboard root", loc)
	}
	if !hasSessionCookie(w) {
		t.Fatal("expected recurso_session cookie after oauth login")
	}
}

func TestOAuthCallback_StateMismatchForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := oauthMockProvider(t, `{"sub":"x","email":"x@y.com","email_verified":true}`)
	defer srv.Close()
	h, _ := newOAuthHandler(t, srv)

	cookie, _ := runStart(t, h, "google")
	// Wrong state value in the callback query.
	w := runCallback(t, h, "google", "code", "totally-wrong-state", cookie)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for state mismatch", w.Code)
	}
	if hasSessionCookie(w) {
		t.Fatal("no session should be issued on state mismatch")
	}
}

func TestOAuthCallback_UnverifiedGoogleEmailRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := oauthMockProvider(t, `{"sub":"u-1","email":"unverified@x.com","email_verified":false}`)
	defer srv.Close()
	h, _ := newOAuthHandler(t, srv)

	cookie, state := runStart(t, h, "google")
	w := runCallback(t, h, "google", "code", state, cookie)
	// Soft failure → redirect to the login error page, no session.
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 to error page", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "http://dash.local/login?error=oauth" {
		t.Fatalf("redirect = %q, want login error page", loc)
	}
	if hasSessionCookie(w) {
		t.Fatal("no session should be issued for unverified email")
	}
}

func TestOAuthStart_DisabledProvider404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := oauthMockProvider(t, `{}`)
	defer srv.Close()
	h, _ := newOAuthHandler(t, srv) // only google registered

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/oauth/github/start", nil)
	c.Params = gin.Params{{Key: "provider", Value: "github"}}
	h.Start(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for disabled provider", w.Code)
	}
}

func TestOAuthProviders_ReflectsEnv(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := oauthMockProvider(t, `{}`)
	defer srv.Close()
	h, _ := newOAuthHandler(t, srv) // only google enabled

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/oauth/providers", nil)
	h.Providers(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Providers []service.ProviderStatus `json:"providers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	got := map[string]bool{}
	for _, p := range resp.Providers {
		got[p.Name] = p.Enabled
	}
	if !got["google"] {
		t.Error("google should be enabled")
	}
	if got["github"] {
		t.Error("github should be disabled")
	}
}
