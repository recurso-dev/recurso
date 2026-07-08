package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"golang.org/x/oauth2"
)

// --- fake oauth identity repo ---

type fakeOAuthIdentityRepo struct {
	byProviderUID map[string]*domain.OAuthIdentity // key: provider|providerUserID
}

func newFakeOAuthIdentityRepo() *fakeOAuthIdentityRepo {
	return &fakeOAuthIdentityRepo{byProviderUID: map[string]*domain.OAuthIdentity{}}
}

func oauthKey(provider, uid string) string { return provider + "|" + uid }

func (r *fakeOAuthIdentityRepo) Create(_ context.Context, i *domain.OAuthIdentity) error {
	k := oauthKey(i.Provider, i.ProviderUserID)
	if _, ok := r.byProviderUID[k]; ok {
		return domain.ErrDuplicateEmail
	}
	cp := *i
	r.byProviderUID[k] = &cp
	return nil
}

func (r *fakeOAuthIdentityRepo) GetByProviderUserID(_ context.Context, provider, uid string) (*domain.OAuthIdentity, error) {
	if i, ok := r.byProviderUID[oauthKey(provider, uid)]; ok {
		cp := *i
		return &cp, nil
	}
	return nil, domain.ErrUserNotFound
}

func (r *fakeOAuthIdentityRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*domain.OAuthIdentity, error) {
	var out []*domain.OAuthIdentity
	for _, i := range r.byProviderUID {
		if i.UserID == userID {
			cp := *i
			out = append(out, &cp)
		}
	}
	return out, nil
}

func newOAuthTestAuth() (*AuthService, *fakeUserRepo, *fakeOAuthIdentityRepo) {
	ur := newFakeUserRepo()
	sr := newFakeSessionRepo()
	ir := newFakeOAuthIdentityRepo()
	svc := NewAuthService(ur, sr, newFakeTenants(), time.Hour)
	svc.ConfigureOAuth(ir)
	return svc, ur, ir
}

// --- find-or-create rules ---

func TestLoginWithOAuth_IdentityExists_LogsIn(t *testing.T) {
	svc, ur, ir := newOAuthTestAuth()
	// Seed an existing user + identity.
	reg, err := svc.Register(context.Background(), "Acme", "Alice", "alice@acme.com", "supersecret", "")
	if err != nil {
		t.Fatalf("seed register: %v", err)
	}
	_ = ir.Create(context.Background(), &domain.OAuthIdentity{
		ID: uuid.New(), UserID: reg.User.ID, Provider: "google", ProviderUserID: "sub-123",
		Email: "alice@acme.com", CreatedAt: time.Now(),
	})

	user, token, err := svc.LoginWithOAuth(context.Background(), "google",
		&OAuthUserInfo{ProviderUserID: "sub-123", Email: "different@acme.com", EmailVerified: false}, "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if user.ID != reg.User.ID {
		t.Fatalf("resolved wrong user")
	}
	if token == "" {
		t.Fatal("expected a session token")
	}
	// No new user created.
	if len(ur.users) != 1 {
		t.Fatalf("user count = %d, want 1", len(ur.users))
	}
}

func TestLoginWithOAuth_EmailMatch_LinksAndLogsIn(t *testing.T) {
	svc, ur, ir := newOAuthTestAuth()
	reg, _ := svc.Register(context.Background(), "Acme", "Alice", "alice@acme.com", "supersecret", "")

	user, token, err := svc.LoginWithOAuth(context.Background(), "google",
		&OAuthUserInfo{ProviderUserID: "sub-999", Email: "Alice@Acme.com", EmailVerified: true}, "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if user.ID != reg.User.ID {
		t.Fatalf("expected to link to existing user")
	}
	if token == "" {
		t.Fatal("expected session token")
	}
	if len(ur.users) != 1 {
		t.Fatalf("user count = %d, want 1 (no new tenant)", len(ur.users))
	}
	// Identity was linked.
	if _, err := ir.GetByProviderUserID(context.Background(), "google", "sub-999"); err != nil {
		t.Fatalf("identity not linked: %v", err)
	}
}

func TestLoginWithOAuth_BrandNewEmail_CreatesTenantOwner(t *testing.T) {
	svc, ur, ir := newOAuthTestAuth()

	user, token, err := svc.LoginWithOAuth(context.Background(), "github",
		&OAuthUserInfo{ProviderUserID: "gh-1", Email: "new@startup.io", EmailVerified: true, Name: "New Person"}, "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" {
		t.Fatal("expected session token")
	}
	if user.Role != domain.RoleOwner {
		t.Fatalf("role = %q, want owner", user.Role)
	}
	if user.Email != "new@startup.io" {
		t.Fatalf("email = %q", user.Email)
	}
	if len(ur.users) != 1 {
		t.Fatalf("user count = %d, want 1", len(ur.users))
	}
	if _, err := ir.GetByProviderUserID(context.Background(), "github", "gh-1"); err != nil {
		t.Fatalf("identity not created: %v", err)
	}
}

func TestLoginWithOAuth_UnverifiedEmailNoIdentity_Rejected(t *testing.T) {
	svc, _, _ := newOAuthTestAuth()
	_, _, err := svc.LoginWithOAuth(context.Background(), "google",
		&OAuthUserInfo{ProviderUserID: "sub-x", Email: "x@y.com", EmailVerified: false}, "ua")
	if err == nil {
		t.Fatal("expected rejection for unverified email with no existing identity")
	}
}

func TestCompanyNameFromEmail(t *testing.T) {
	cases := map[string]string{
		"alice@acme.com":     "Acme",
		"bob@startup.io":     "Startup",
		"x@":                 "My Workspace",
		"noatsign":           "My Workspace",
		"user@sub.domain.co": "Sub",
	}
	for email, want := range cases {
		if got := companyNameFromEmail(email); got != want {
			t.Errorf("companyNameFromEmail(%q) = %q, want %q", email, got, want)
		}
	}
}

// --- provider Exchange + FetchUserInfo against an httptest server ---

// mockProviderServer serves a token endpoint and a userinfo endpoint with no
// real network, so the full exchange→userinfo path is exercised.
func mockProviderServer(t *testing.T, userInfoJSON string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer"}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token" {
			t.Errorf("userinfo Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(userInfoJSON))
	})
	return httptest.NewServer(mux)
}

func TestProvider_ExchangeAndFetchUserInfo_Google(t *testing.T) {
	srv := mockProviderServer(t, `{"sub":"g-42","email":"dev@example.com","email_verified":true,"name":"Dev"}`)
	defer srv.Close()

	p := &OAuthProvider{
		Name:         ProviderGoogle,
		ClientID:     "cid",
		ClientSecret: "secret",
		Endpoint:     oauth2.Endpoint{AuthURL: srv.URL + "/authorize", TokenURL: srv.URL + "/token"},
		UserInfoURL:  srv.URL + "/userinfo",
		RedirectURL:  "http://localhost/cb",
		HTTPClient:   srv.Client(),
	}

	verifier := oauth2.GenerateVerifier()
	tok, err := p.Exchange(context.Background(), "the-code", verifier)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	info, err := p.FetchUserInfo(context.Background(), tok)
	if err != nil {
		t.Fatalf("userinfo: %v", err)
	}
	if info.ProviderUserID != "g-42" || info.Email != "dev@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected info: %+v", info)
	}
	// AuthCodeURL must carry the PKCE S256 challenge and state.
	authURL := p.AuthCodeURL("the-state", verifier)
	if !strings.Contains(authURL, "code_challenge=") || !strings.Contains(authURL, "code_challenge_method=S256") {
		t.Fatalf("auth url missing PKCE: %s", authURL)
	}
	if !strings.Contains(authURL, "state=the-state") {
		t.Fatalf("auth url missing state: %s", authURL)
	}
}

func TestRegistry_ReflectsConfiguredProviders(t *testing.T) {
	// Only Google configured.
	reg := NewOAuthRegistry(OAuthConfig{
		GoogleClientID:     "id",
		GoogleClientSecret: "secret",
		RedirectBaseURL:    "http://localhost:8080",
	})
	statuses := map[string]bool{}
	for _, s := range reg.Statuses() {
		statuses[s.Name] = s.Enabled
	}
	if !statuses[ProviderGoogle] {
		t.Error("google should be enabled")
	}
	if statuses[ProviderGitHub] {
		t.Error("github should be disabled (no creds)")
	}
	if _, ok := reg.Get(ProviderGitHub); ok {
		t.Error("Get(github) should be false when unconfigured")
	}
	if _, ok := reg.Get(ProviderGoogle); !ok {
		t.Error("Get(google) should be true")
	}
}
