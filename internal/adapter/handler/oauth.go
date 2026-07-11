package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
	"golang.org/x/oauth2"
)

// oauthStateCookieName carries the signed, short-lived CSRF-state + PKCE
// verifier binding between /start and /callback. It is httpOnly and scoped to
// the OAuth path so it never leaks to the rest of the app.
const oauthStateCookieName = "recurso_oauth_state"

// oauthStateTTL bounds how long a login attempt may sit at the provider before
// the callback is rejected.
const oauthStateTTL = 10 * time.Minute

// OAuthHandler serves the public social-login endpoints. It reuses the Phase 1
// session path (via AuthService) to log users in and only ever redirects to the
// configured DASHBOARD_URL (no open redirects).
type OAuthHandler struct {
	auth         *service.AuthService
	registry     *service.OAuthRegistry
	dashboardURL string
	stateSecret  []byte
	secure       bool
	logger       *slog.Logger
}

// NewOAuthHandler builds the handler. dashboardURL is the SPA base to redirect
// to after login (success → {dashboardURL}/, failure → {dashboardURL}/login?
// error=oauth). stateSecret signs the state cookie (HMAC-SHA256).
func NewOAuthHandler(auth *service.AuthService, registry *service.OAuthRegistry, dashboardURL string, stateSecret []byte, secureCookie bool) *OAuthHandler {
	return &OAuthHandler{
		auth:         auth,
		registry:     registry,
		dashboardURL: strings.TrimRight(dashboardURL, "/"),
		stateSecret:  stateSecret,
		secure:       secureCookie,
		logger:       slog.Default().With("handler", "oauth"),
	}
}

// oauthStatePayload is the JSON bound (and HMAC-signed) into the state cookie.
type oauthStatePayload struct {
	Provider string `json:"p"`
	State    string `json:"s"`
	Verifier string `json:"v"`
	Issued   int64  `json:"t"` // unix seconds
}

// --- GET /auth/oauth/providers ---

// Providers reports every known provider and whether it is enabled (its client
// id + secret are configured). The frontend uses this to show/hide buttons.
func (h *OAuthHandler) Providers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"providers": h.registry.Statuses()})
}

// --- GET /auth/oauth/:provider/start ---

// Start begins a login: it generates a CSRF state and a PKCE verifier, binds
// them into a signed httpOnly cookie, and 302-redirects to the provider's
// authorize URL. Unknown/disabled providers get 404.
func (h *OAuthHandler) Start(c *gin.Context) {
	name := c.Param("provider")
	provider, ok := h.registry.Get(name)
	if !ok {
		respondError(c, http.StatusNotFound, codeNotFound, "unknown or disabled oauth provider")
		return
	}

	state, err := randomToken()
	if err != nil {
		respondError(c, http.StatusInternalServerError, codeInternalError, "failed to start oauth")
		return
	}
	verifier := oauth2.GenerateVerifier()

	payload := oauthStatePayload{
		Provider: provider.Name,
		State:    state,
		Verifier: verifier,
		Issued:   time.Now().Unix(),
	}
	h.setStateCookie(c, payload)

	authURL := provider.AuthCodeURL(state, verifier)
	c.Redirect(http.StatusFound, authURL)
}

// --- GET /auth/oauth/:provider/callback ---

// Callback validates the CSRF state against the cookie (constant-time),
// exchanges the code (with PKCE), fetches userinfo, enforces email
// verification, then find-or-creates a user and opens a session. On success it
// 302s to {dashboardURL}/; on ANY failure it 302s to
// {dashboardURL}/login?error=oauth.
func (h *OAuthHandler) Callback(c *gin.Context) {
	name := c.Param("provider")
	provider, ok := h.registry.Get(name)
	if !ok {
		respondError(c, http.StatusNotFound, codeNotFound, "unknown or disabled oauth provider")
		return
	}

	// Always clear the state cookie: the attempt is single-use either way.
	defer h.clearStateCookie(c)

	payload, err := h.readStateCookie(c)
	if err != nil || payload.Provider != provider.Name {
		h.logger.Warn("oauth callback: invalid state cookie", "provider", provider.Name, "ip", c.ClientIP())
		h.failRedirect(c)
		return
	}
	if time.Since(time.Unix(payload.Issued, 0)) > oauthStateTTL {
		h.logger.Warn("oauth callback: state expired", "provider", provider.Name)
		h.failRedirect(c)
		return
	}

	// CSRF: constant-time compare of the returned state to the bound one.
	returnedState := c.Query("state")
	if subtle.ConstantTimeCompare([]byte(returnedState), []byte(payload.State)) != 1 {
		h.logger.Warn("oauth callback: state mismatch", "provider", provider.Name, "ip", c.ClientIP())
		// A state mismatch is a distinct, security-relevant condition. Answer
		// 403 directly (tests assert this) rather than a soft redirect.
		respondError(c, http.StatusForbidden, codeForbidden, "oauth state mismatch")
		return
	}

	code := c.Query("code")
	if code == "" {
		h.failRedirect(c)
		return
	}

	ctx := c.Request.Context()
	token, err := provider.Exchange(ctx, code, payload.Verifier)
	if err != nil {
		h.logger.Warn("oauth callback: token exchange failed", "provider", provider.Name, "error", err)
		h.failRedirect(c)
		return
	}

	info, err := provider.FetchUserInfo(ctx, token)
	if err != nil {
		h.logger.Warn("oauth callback: userinfo failed", "provider", provider.Name, "error", err)
		h.failRedirect(c)
		return
	}

	// Google (and our GitHub path) MUST yield a verified email before we link or
	// create an account on it.
	if info.Email == "" || !info.EmailVerified {
		h.logger.Warn("oauth callback: unverified or missing email", "provider", provider.Name)
		h.failRedirect(c)
		return
	}

	user, sessionToken, err := h.auth.LoginWithOAuth(ctx, provider.Name, info, c.GetHeader("User-Agent"))
	if err != nil {
		h.logger.Warn("oauth callback: login failed", "provider", provider.Name, "error", err)
		h.failRedirect(c)
		return
	}

	h.setSessionCookie(c, sessionToken)
	h.logger.Info("oauth login", "provider", provider.Name, "user_id", user.ID, "tenant_id", user.TenantID)
	c.Redirect(http.StatusFound, h.dashboardURL+"/")
}

// setSessionCookie mirrors AuthHandler.setSessionCookie exactly so OAuth logins
// issue an identical recurso_session cookie.
func (h *OAuthHandler) setSessionCookie(c *gin.Context, token string) {
	maxAge := int(h.auth.SessionTTL().Seconds())
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(domain.SessionCookieName, token, maxAge, "/", "", h.secure, true)
}

func (h *OAuthHandler) failRedirect(c *gin.Context) {
	c.Redirect(http.StatusFound, h.dashboardURL+"/login?error=oauth")
}

// --- signed state cookie ---

func (h *OAuthHandler) setStateCookie(c *gin.Context, payload oauthStatePayload) {
	raw, _ := json.Marshal(payload)
	value := h.sign(raw)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookieName, value, int(oauthStateTTL.Seconds()), "/auth/oauth", "", h.secure, true)
}

func (h *OAuthHandler) clearStateCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oauthStateCookieName, "", -1, "/auth/oauth", "", h.secure, true)
}

func (h *OAuthHandler) readStateCookie(c *gin.Context) (oauthStatePayload, error) {
	var payload oauthStatePayload
	value, err := c.Cookie(oauthStateCookieName)
	if err != nil || value == "" {
		return payload, errors.New("missing state cookie")
	}
	raw, err := h.verify(value)
	if err != nil {
		return payload, err
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

// sign returns base64(payload) + "." + base64(HMAC-SHA256(payload)).
func (h *OAuthHandler) sign(raw []byte) string {
	mac := hmac.New(sha256.New, h.stateSecret)
	mac.Write(raw)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// verify checks the HMAC (constant-time) and returns the payload bytes.
func (h *OAuthHandler) verify(value string) ([]byte, error) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("malformed state cookie")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, h.stateSecret)
	mac.Write(raw)
	wantSig := mac.Sum(nil)
	if subtle.ConstantTimeCompare(gotSig, wantSig) != 1 {
		return nil, errors.New("bad state signature")
	}
	return raw, nil
}

// randomToken returns 32 bytes of CSPRNG entropy as a URL-safe string. Used for
// the CSRF state.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
