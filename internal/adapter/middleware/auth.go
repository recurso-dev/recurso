package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
)

// AuthMiddleware checks for a valid API Key using the DB.
// API keys are validated against the tenants/api_keys table.
func AuthMiddleware(repo *db.TenantRepository) gin.HandlerFunc {
	logger := slog.Default().With("middleware", "auth")

	return func(c *gin.Context) {
		// 1. Extract Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header required",
				},
			})
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

		// 3. Validate against DB
		tenant, err := repo.GetTenantByKey(c.Request.Context(), token)
		if err != nil {
			logger.Warn("invalid API key attempt",
				"ip", c.ClientIP(),
				"user_agent", c.GetHeader("User-Agent"),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_API_KEY",
					"message": "Invalid API Key",
				},
			})
			return
		}

		// 4. Set Tenant Context
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
