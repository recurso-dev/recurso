package middleware

import (
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
	expiresAt time.Time
}

const (
	verifiedKeyTTL      = 5 * time.Minute
	verifiedKeyCacheMax = 10000
)

func (vc *verifiedKeyCache) get(token string) (uuid.UUID, bool) {
	k := sha256.Sum256([]byte(token))
	vc.mu.RLock()
	e, ok := vc.entries[k]
	vc.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return uuid.Nil, false
	}
	return e.tenantID, true
}

func (vc *verifiedKeyCache) put(token string, tenantID uuid.UUID) {
	k := sha256.Sum256([]byte(token))
	vc.mu.Lock()
	defer vc.mu.Unlock()
	if len(vc.entries) >= verifiedKeyCacheMax {
		// Simple pressure valve: drop everything rather than evict finely.
		vc.entries = make(map[[32]byte]verifiedKeyEntry, verifiedKeyCacheMax/4)
	}
	vc.entries[k] = verifiedKeyEntry{tenantID: tenantID, expiresAt: time.Now().Add(verifiedKeyTTL)}
}

// AuthMiddleware checks for a valid API Key using the DB.
// API keys are validated against the tenants/api_keys table.
func AuthMiddleware(repo *db.TenantRepository) gin.HandlerFunc {
	logger := slog.Default().With("middleware", "auth")
	cache := &verifiedKeyCache{entries: make(map[[32]byte]verifiedKeyEntry)}

	return func(c *gin.Context) {
		// 1. Extract Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httperr.Abort(c, http.StatusUnauthorized, httperr.CodeUnauthorized, "Authorization header required")
			return
		}

		parts := strings.Split(authHeader, " ")
		var token string
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token = parts[1]
		} else {
			token = authHeader
		}

		// 2. Dev bypass — ONLY in development mode AND when explicitly enabled
		if token == "recurso_secret" {
			appEnv := os.Getenv("APP_ENV")
			devBypass := os.Getenv("ALLOW_DEV_BYPASS")

			if appEnv == "development" && devBypass == "true" {
				devTenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
				c.Set("tenant_id", devTenantID)
				logger.Debug("dev bypass auth used", "tenant_id", devTenantID)
				c.Next()
				return
			}

			// If not explicitly enabled, treat as invalid key
			logger.Warn("dev bypass attempted but not enabled",
				"app_env", appEnv,
				"dev_bypass", devBypass,
				"ip", c.ClientIP(),
			)
		}

		// 3. Cache of already-verified keys (avoids per-request bcrypt)
		if tenantID, ok := cache.get(token); ok {
			c.Set("tenant_id", tenantID)
			c.Next()
			return
		}

		// 4. Validate against DB (bcrypt compare)
		tenant, err := repo.GetTenantByKey(c.Request.Context(), token)
		if err != nil {
			logger.Warn("invalid API key attempt",
				"ip", c.ClientIP(),
				"user_agent", c.GetHeader("User-Agent"),
			)
			httperr.Abort(c, http.StatusUnauthorized, httperr.CodeInvalidAPIKey, "Invalid API Key")
			return
		}

		// 5. Cache and set tenant context
		cache.put(token, tenant.ID)
		c.Set("tenant_id", tenant.ID)
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
