package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// DemoGuard blocks the destructive/identity edges of a public sandbox
// (docs/spec_demo_mode.md D4): visitors get full create/edit freedom
// inside the demo tenant, but cannot invite teammates, rewire SSO,
// rotate keys, change auth credentials, or delete the account. Blocked
// requests answer 403 {"error":{"code":"demo_mode"}}.
//
// This is defense-in-depth — the primary guarantee is that demo mode
// constructs mock adapters, so even an unguarded edge cannot reach the
// outside world.

// demoBlockedPrefixes are (method, path-prefix) pairs matched against the
// raw request path. Prefix matching keeps coverage over sub-routes.
var demoBlockedPrefixes = []struct {
	method string
	prefix string
}{
	// Team & identity
	{"POST", "/v1/team"}, {"PUT", "/v1/team"}, {"DELETE", "/v1/team"},
	{"POST", "/auth/invite"}, {"POST", "/auth/accept-invite"},
	{"POST", "/auth/register"},
	{"PUT", "/auth/password"}, {"POST", "/auth/password"},
	{"POST", "/auth/forgot-password"}, {"POST", "/auth/reset-password"},
	// SSO / security config
	{"POST", "/v1/settings/sso"}, {"PUT", "/v1/settings/sso"}, {"DELETE", "/v1/settings/sso"},
	{"POST", "/v1/settings/mfa"}, {"PUT", "/v1/settings/mfa"}, {"DELETE", "/v1/settings/mfa"},
	// Developer keys (rotation would lock later visitors out)
	{"POST", "/v1/developer/keys"}, {"PUT", "/v1/developer/keys"}, {"DELETE", "/v1/developer/keys"},
	// Tenant-level danger
	{"PUT", "/v1/account"}, {"DELETE", "/v1/account"},
	{"PUT", "/v1/settings/data-region"},
}

// DemoGuard returns the blocking middleware. Wire it only when demo mode
// is enabled.
func DemoGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		for _, b := range demoBlockedPrefixes {
			if method == b.method && strings.HasPrefix(path, b.prefix) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": gin.H{
						"code":    "demo_mode",
						"message": "This action is disabled in the public demo. Self-host Recurso to try it: https://docs.recurso.dev/quickstart",
					},
				})
				return
			}
		}
		c.Next()
	}
}
