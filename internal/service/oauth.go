package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// ProviderGoogle / ProviderGitHub are the canonical provider names used in
// URLs, the DB (user_oauth_identities.provider), the /providers endpoint AND to
// select how userinfo is parsed (see FetchUserInfo).
const (
	ProviderGoogle = "google"
	ProviderGitHub = "github"
)

// OAuthUserInfo is the normalized identity a provider returns after a successful
// code exchange. Providers differ in wire format; FetchUserInfo maps them here.
type OAuthUserInfo struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	Name           string
}

// OAuthProvider is one configured social-login provider. It is only present in
// the registry when both ClientID and ClientSecret are set. The endpoint URLs
// and HTTPClient are injectable so tests can point the flow at an httptest
// server with no real network.
type OAuthProvider struct {
	Name         string
	ClientID     string
	ClientSecret string
	Endpoint     oauth2.Endpoint // AuthURL + TokenURL
	Scopes       []string
	RedirectURL  string
	// UserInfoURL is the provider's userinfo endpoint (Google) or the user
	// profile endpoint (GitHub, api.github.com/user).
	UserInfoURL string
	// EmailsURL is GitHub-only: the /user/emails endpoint used to find the
	// primary verified email (GitHub's profile email may be null/unverified).
	EmailsURL string
	// HTTPClient, when non-nil, is used for the token exchange and userinfo
	// fetch. Tests inject an httptest client here; production leaves it nil.
	HTTPClient *http.Client
}

func (p *OAuthProvider) oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint:     p.Endpoint,
		RedirectURL:  p.RedirectURL,
		Scopes:       p.Scopes,
	}
}

// clientCtx binds the provider's (possibly test) HTTP client to ctx so both the
// oauth2 exchange and manual userinfo GETs share it.
func (p *OAuthProvider) clientCtx(ctx context.Context) context.Context {
	if p.HTTPClient != nil {
		return context.WithValue(ctx, oauth2.HTTPClient, p.HTTPClient)
	}
	return ctx
}

func (p *OAuthProvider) httpClient() *http.Client {
	if p.HTTPClient != nil {
		return p.HTTPClient
	}
	return http.DefaultClient
}

// AuthCodeURL builds the provider authorize URL for a login attempt, binding the
// CSRF state and a PKCE S256 challenge derived from verifier.
func (p *OAuthProvider) AuthCodeURL(state, verifier string) string {
	return p.oauthConfig().AuthCodeURL(state,
		oauth2.AccessTypeOnline,
		oauth2.S256ChallengeOption(verifier),
	)
}

// Exchange swaps an authorization code (plus the PKCE verifier) for a token.
func (p *OAuthProvider) Exchange(ctx context.Context, code, verifier string) (*oauth2.Token, error) {
	return p.oauthConfig().Exchange(p.clientCtx(ctx), code, oauth2.VerifierOption(verifier))
}

// FetchUserInfo calls the provider's userinfo endpoint(s) and normalizes the
// result. For Google it reads sub/email/email_verified; for GitHub it reads the
// numeric id and resolves the primary verified email from /user/emails.
func (p *OAuthProvider) FetchUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	switch strings.ToLower(p.Name) {
	case ProviderGoogle:
		return p.fetchGoogleUserInfo(ctx, token)
	case ProviderGitHub:
		return p.fetchGitHubUserInfo(ctx, token)
	default:
		return nil, fmt.Errorf("unknown provider %q", p.Name)
	}
}

func (p *OAuthProvider) getJSON(ctx context.Context, url string, token *oauth2.Token, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := p.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("userinfo request to %s failed: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (p *OAuthProvider) fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	var raw struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := p.getJSON(ctx, p.UserInfoURL, token, &raw); err != nil {
		return nil, err
	}
	if raw.Sub == "" {
		return nil, fmt.Errorf("google userinfo missing sub")
	}
	return &OAuthUserInfo{
		ProviderUserID: raw.Sub,
		Email:          raw.Email,
		EmailVerified:  raw.EmailVerified,
		Name:           raw.Name,
	}, nil
}

func (p *OAuthProvider) fetchGitHubUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error) {
	var profile struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := p.getJSON(ctx, p.UserInfoURL, token, &profile); err != nil {
		return nil, err
	}
	if profile.ID == 0 {
		return nil, fmt.Errorf("github userinfo missing id")
	}
	info := &OAuthUserInfo{
		ProviderUserID: fmt.Sprintf("%d", profile.ID),
		Name:           profile.Name,
	}
	// GitHub's profile email may be null or unverified; the primary verified
	// email lives on /user/emails. We only trust a primary+verified address.
	if p.EmailsURL != "" {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := p.getJSON(ctx, p.EmailsURL, token, &emails); err != nil {
			return nil, err
		}
		for _, e := range emails {
			if e.Primary && e.Verified {
				info.Email = e.Email
				info.EmailVerified = true
				break
			}
		}
	}
	return info, nil
}

// OAuthRegistry holds the configured providers. Providers absent from the map
// are treated as disabled (their endpoints return 404 and they are reported
// disabled by /providers).
type OAuthRegistry struct {
	providers map[string]*OAuthProvider
}

// ProviderStatus is one entry in the GET /auth/oauth/providers response.
type ProviderStatus struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// knownProviders is the fixed set of providers this build understands. The
// /providers endpoint reports all of them with an enabled flag so the frontend
// can render (and grey out) each button deterministically.
var knownProviders = []string{ProviderGoogle, ProviderGitHub}

// OAuthConfig configures the registry from environment-derived values.
type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	// RedirectBaseURL is the API's public base; the per-provider redirect URL is
	// {RedirectBaseURL}/auth/oauth/{provider}/callback.
	RedirectBaseURL string
}

// NewOAuthRegistry builds the registry, including only providers whose client
// id AND secret are both set.
func NewOAuthRegistry(cfg OAuthConfig) *OAuthRegistry {
	base := strings.TrimRight(cfg.RedirectBaseURL, "/")
	reg := &OAuthRegistry{providers: map[string]*OAuthProvider{}}

	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		reg.providers[ProviderGoogle] = &OAuthProvider{
			Name:         ProviderGoogle,
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
			RedirectURL:  base + "/auth/oauth/google/callback",
			UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
		}
	}
	if cfg.GitHubClientID != "" && cfg.GitHubClientSecret != "" {
		reg.providers[ProviderGitHub] = &OAuthProvider{
			Name:         ProviderGitHub,
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
			RedirectURL:  base + "/auth/oauth/github/callback",
			UserInfoURL:  "https://api.github.com/user",
			EmailsURL:    "https://api.github.com/user/emails",
		}
	}
	return reg
}

// NewOAuthRegistryWithProviders builds a registry from explicit providers. Used
// by tests to inject providers pointed at an httptest server.
func NewOAuthRegistryWithProviders(providers ...*OAuthProvider) *OAuthRegistry {
	reg := &OAuthRegistry{providers: map[string]*OAuthProvider{}}
	for _, p := range providers {
		reg.providers[p.Name] = p
	}
	return reg
}

// Get returns the provider by name and whether it is enabled (configured).
func (r *OAuthRegistry) Get(name string) (*OAuthProvider, bool) {
	p, ok := r.providers[strings.ToLower(name)]
	return p, ok
}

// Statuses reports every known provider with its enabled flag.
func (r *OAuthRegistry) Statuses() []ProviderStatus {
	out := make([]ProviderStatus, 0, len(knownProviders))
	for _, name := range knownProviders {
		_, ok := r.providers[name]
		out = append(out, ProviderStatus{Name: name, Enabled: ok})
	}
	return out
}

// --- find-or-create (on AuthService, reusing the Phase 1 session path) ---

// ConfigureOAuth wires the OAuth identity repository. Mirrors ConfigureMFA so
// the base AuthService constructor and every existing caller/test stay
// unchanged; LoginWithOAuth guards against a nil repo.
func (s *AuthService) ConfigureOAuth(identities port.OAuthIdentityRepository) {
	s.oauthIdentities = identities
}

// randomPasswordHash returns a bcrypt hash of 32 random bytes. OAuth-created
// users have no password they know; they must use the password-reset flow to
// set one. This keeps users.password_hash NOT NULL without a schema change and
// leaves no guessable credential.
func randomPasswordHash() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random password: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(base64.RawURLEncoding.EncodeToString(b)), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash random password: %w", err)
	}
	return string(hash), nil
}

// LoginWithOAuth implements find-or-create for a verified OAuth identity and
// opens a normal session (the exact Phase 1 path). Rules:
//
//  1. If an identity (provider, providerUserID) exists → log in that user.
//  2. Else if a user with the (verified) email exists → link a new identity to
//     them and log in.
//  3. Else → create a new tenant + owner user (self-serve signup) and link.
//
// For steps 2 and 3 the email MUST be verified; the callback enforces this for
// Google, and the GitHub path only ever yields verified primary emails.
func (s *AuthService) LoginWithOAuth(ctx context.Context, provider string, info *OAuthUserInfo, userAgent string) (*domain.User, string, error) {
	if s.oauthIdentities == nil {
		return nil, "", fmt.Errorf("oauth is not configured")
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	email := strings.ToLower(strings.TrimSpace(info.Email))

	// 1. Existing identity → that user.
	if identity, err := s.oauthIdentities.GetByProviderUserID(ctx, provider, info.ProviderUserID); err == nil {
		user, err := s.users.GetByIDGlobal(ctx, identity.UserID)
		if err != nil {
			return nil, "", fmt.Errorf("oauth identity references missing user: %w", err)
		}
		token, err := s.openSession(ctx, user, userAgent)
		if err != nil {
			return nil, "", err
		}
		return user, token, nil
	}

	// Steps 2 and 3 require a verified email we can trust.
	if email == "" || !info.EmailVerified {
		return nil, "", domain.ErrOAuthEmailUnverified
	}

	// 2. Existing user with this email → link + log in.
	if user, err := s.users.GetByEmail(ctx, email); err == nil {
		if err := s.linkIdentity(ctx, user.ID, provider, info.ProviderUserID, email); err != nil {
			return nil, "", err
		}
		token, err := s.openSession(ctx, user, userAgent)
		if err != nil {
			return nil, "", err
		}
		return user, token, nil
	}

	// 3. Brand-new email → create tenant + owner (self-serve signup).
	user, err := s.createOAuthUser(ctx, provider, info, email)
	if err != nil {
		return nil, "", err
	}
	token, err := s.openSession(ctx, user, userAgent)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *AuthService) linkIdentity(ctx context.Context, userID uuid.UUID, provider, providerUserID, email string) error {
	return s.oauthIdentities.Create(ctx, &domain.OAuthIdentity{
		ID:             uuid.New(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
		CreatedAt:      time.Now().UTC(),
	})
}

func (s *AuthService) createOAuthUser(ctx context.Context, provider string, info *OAuthUserInfo, email string) (*domain.User, error) {
	companyName := companyNameFromEmail(email)
	tenant, _, err := s.tenants.Register(ctx, companyName, email)
	if err != nil {
		return nil, err
	}
	pwHash, err := randomPasswordHash()
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(info.Name)
	if name == "" {
		name = localPart(email)
	}
	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		Email:        email,
		PasswordHash: pwHash,
		Name:         name,
		Role:         domain.RoleOwner,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	if err := s.linkIdentity(ctx, user.ID, provider, info.ProviderUserID, email); err != nil {
		return nil, err
	}
	return user, nil
}

// companyNameFromEmail derives a friendly workspace name from an email's domain
// ("alice@acme.com" → "Acme"), falling back to "My Workspace".
func companyNameFromEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 || at+1 >= len(email) {
		return "My Workspace"
	}
	domainPart := email[at+1:]
	label := domainPart
	if dot := strings.Index(domainPart, "."); dot > 0 {
		label = domainPart[:dot]
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return "My Workspace"
	}
	return strings.ToUpper(label[:1]) + label[1:]
}

func localPart(email string) string {
	if at := strings.Index(email, "@"); at > 0 {
		return email[:at]
	}
	return email
}
