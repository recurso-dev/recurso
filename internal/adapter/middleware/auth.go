package middleware

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/adapter/httperr"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// verifiedKeyCache remembers API keys that already passed bcrypt
// verification, keyed by their SHA-256 (never the plaintext). Without it
// every request pays a full bcrypt compare (~60-100ms of CPU), capping the
// whole API at ~100 rps. Entries expire so key revocation takes effect
// within the TTL.
type verifiedKeyCache struct {
	mu      sync.RWMutex
	entries map[[32]byte]verifiedKeyEntry
}

type verifiedKeyEntry struct {
	tenantID  uuid.UUID
	livemode  bool
	expiresAt time.Time
}

const (
	verifiedKeyTTL      = 5 * time.Minute
	verifiedKeyCacheMax = 10000
)

func (vc *verifiedKeyCache) get(token string) (uuid.UUID, bool, bool) {
	k := sha256.Sum256([]byte(token))
	vc.mu.RLock()
	e, ok := vc.entries[k]
	vc.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return uuid.Nil, false, false
	}
	return e.tenantID, e.livemode, true
}

func (vc *verifiedKeyCache) put(token string, tenantID uuid.UUID, livemode bool) {
	k := sha256.Sum256([]byte(token))
	vc.mu.Lock()
	defer vc.mu.Unlock()
	if len(vc.entries) >= verifiedKeyCacheMax {
		// Simple pressure valve: drop everything rather than evict finely.
		vc.entries = make(map[[32]byte]verifiedKeyEntry, verifiedKeyCacheMax/4)
	}
	vc.entries[k] = verifiedKeyEntry{tenantID: tenantID, livemode: livemode, expiresAt: time.Now().Add(verifiedKeyTTL)}
}

// extractBearerToken pulls the credential out of the Authorization header,
// accepting both "Bearer <token>" and a bare token. The bool is false when no
// Authorization header is present.
func extractBearerToken(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", false
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1], true
	}
	return authHeader, true
}

// resolveAPIKey applies the dev bypass, the verified-key cache, and finally the
// DB bcrypt compare. It returns the tenant ID on success without writing to the
// context or aborting. This is the single source of truth shared by both the
// API-key-only middleware and the dual (session-or-key) middleware, so the
// cache/perf characteristics never diverge.
// resolveAPIKey returns (tenantID, livemode, ok). serverLive is the server's own
// mode; the dev-bypass path reports the server's mode so it always passes the
// gate the caller applies.
func resolveAPIKey(c *gin.Context, token string, cache *verifiedKeyCache, repo *db.TenantRepository, logger *slog.Logger, serverLive bool) (uuid.UUID, bool, bool) {
	// Dev bypass — ONLY in development mode AND when explicitly enabled.
	if token == "recurso_secret" {
		if os.Getenv("APP_ENV") == "development" && os.Getenv("ALLOW_DEV_BYPASS") == "true" {
			devTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
			logger.Debug("dev bypass auth used", "tenant_id", devTenantID)
			return devTenantID, serverLive, true
		}
		logger.Warn("dev bypass attempted but not enabled",
			"app_env", os.Getenv("APP_ENV"),
			"dev_bypass", os.Getenv("ALLOW_DEV_BYPASS"),
			"ip", c.ClientIP(),
		)
	}

	// Cache of already-verified keys (avoids per-request bcrypt).
	if tenantID, livemode, ok := cache.get(token); ok {
		return tenantID, livemode, true
	}

	// Validate against DB (bcrypt compare).
	tenant, livemode, err := repo.GetTenantByKey(c.Request.Context(), token)
	if err != nil {
		logger.Warn("invalid API key attempt",
			"ip", c.ClientIP(),
			"user_agent", c.GetHeader("User-Agent"),
		)
		return uuid.Nil, false, false
	}
	cache.put(token, tenant.ID, livemode)
	return tenant.ID, livemode, true
}

// keyModeMismatchMessage explains why a key was rejected by the mode gate.
func keyModeMismatchMessage(serverLive bool) string {
	if serverLive {
		return "This is a test-mode API key (rsk_test_…) but the server is configured for live payments. Use a live-mode key."
	}
	return "This is a live-mode API key (rsk_live_…) but the server is not configured with live payment gateways. Use a test-mode key."
}

// AuthMiddleware checks for a valid API Key using the DB. API keys are validated
// against the tenants/api_keys table, and gated by mode: a key's livemode must
// match the server's (serverLive), so a test key can never run on a live-money
// server and vice-versa.
func AuthMiddleware(repo *db.TenantRepository, serverLive bool) gin.HandlerFunc {
	logger := slog.Default().With("middleware", "auth")
	cache := &verifiedKeyCache{entries: make(map[[32]byte]verifiedKeyEntry)}

	return func(c *gin.Context) {
		token, ok := extractBearerToken(c)
		if !ok {
			httperr.Abort(c, http.StatusUnauthorized, httperr.CodeUnauthorized, "Authorization header required")
			return
		}
		tenantID, livemode, ok := resolveAPIKey(c, token, cache, repo, logger, serverLive)
		if !ok {
			httperr.Abort(c, http.StatusUnauthorized, httperr.CodeInvalidAPIKey, "Invalid API Key")
			return
		}
		if livemode != serverLive {
			logger.Warn("api key mode mismatch", "key_livemode", livemode, "server_live", serverLive, "ip", c.ClientIP())
			httperr.Abort(c, http.StatusUnauthorized, httperr.CodeKeyModeMismatch, keyModeMismatchMessage(serverLive))
			return
		}
		c.Set("tenant_id", tenantID)
		c.Set("livemode", livemode)
		c.Next()
	}
}

// SessionResolver validates an opaque session token and returns the owning
// user. Implemented by *service.AuthService; kept as a local interface so the
// middleware package need not import the service package.
type SessionResolver interface {
	ResolveSession(ctx context.Context, rawToken string) (*domain.User, error)
}

// SessionOrAPIKeyMiddleware authenticates a request via EITHER a valid
// recurso_session cookie (a human logged into the dashboard) OR the existing
// Bearer API-key path (a machine / the demo key). Both branches set the SAME
// tenant context (c.Set("tenant_id", uuid)) that the API-key-only middleware
// set, so every existing tenant-scoped handler keeps working unchanged. When
// authenticated by session, the user id and role are also placed on the
// context for role-gated endpoints.
func SessionOrAPIKeyMiddleware(repo *db.TenantRepository, resolver SessionResolver, serverLive bool) gin.HandlerFunc {
	logger := slog.Default().With("middleware", "auth")
	cache := &verifiedKeyCache{entries: make(map[[32]byte]verifiedKeyEntry)}

	return func(c *gin.Context) {
		// 1. Session cookie (dashboard users).
		if cookie, err := c.Cookie(domain.SessionCookieName); err == nil && cookie != "" {
			user, err := resolver.ResolveSession(c.Request.Context(), cookie)
			if err == nil {
				c.Set("tenant_id", user.TenantID)
				c.Set("user_id", user.ID)
				c.Set("user_role", string(user.Role))
				c.Set("user", user)
				c.Next()
				return
			}
			// Invalid/expired cookie: fall through to the API-key path so a
			// request that carries BOTH a stale cookie and a valid key still
			// works; if there is no key either, it is rejected below.
			logger.Debug("invalid session cookie, falling back to API key", "ip", c.ClientIP())
		}

		// 2. API key (machines, CLI, the demo key).
		token, ok := extractBearerToken(c)
		if !ok {
			httperr.Abort(c, http.StatusUnauthorized, httperr.CodeUnauthorized, "Authentication required (session cookie or API key)")
			return
		}
		tenantID, livemode, ok := resolveAPIKey(c, token, cache, repo, logger, serverLive)
		if !ok {
			httperr.Abort(c, http.StatusUnauthorized, httperr.CodeInvalidAPIKey, "Invalid API Key")
			return
		}
		if livemode != serverLive {
			logger.Warn("api key mode mismatch", "key_livemode", livemode, "server_live", serverLive, "ip", c.ClientIP())
			httperr.Abort(c, http.StatusUnauthorized, httperr.CodeKeyModeMismatch, keyModeMismatchMessage(serverLive))
			return
		}
		c.Set("tenant_id", tenantID)
		c.Set("livemode", livemode)
		c.Next()
	}
}

// GetTenantID retrieves the tenant ID from the context.
func GetTenantID(c *gin.Context) uuid.UUID {
	id, ok := c.Get("tenant_id")
	if !ok {
		return uuid.Nil
	}
	return id.(uuid.UUID)
}

// GetUserID returns the authenticated dashboard user's id, or uuid.Nil when the
// request was authenticated by API key (a machine, which has no user).
func GetUserID(c *gin.Context) uuid.UUID {
	id, ok := c.Get("user_id")
	if !ok {
		return uuid.Nil
	}
	if u, ok := id.(uuid.UUID); ok {
		return u
	}
	return uuid.Nil
}

// GetUserRole returns the authenticated user's role and whether a user (session)
// was present. API-key requests return ("", false).
func GetUserRole(c *gin.Context) (string, bool) {
	r, ok := c.Get("user_role")
	if !ok {
		return "", false
	}
	s, ok := r.(string)
	return s, ok
}
