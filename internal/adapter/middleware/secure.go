package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SecureMiddleware adds security headers to responses.
func SecureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// HSTS only makes sense on an HTTPS response: emitted over plain HTTP
		// (local dev, an internal health probe) it is at best ignored and at
		// worst pins a browser to a scheme the host does not serve. Behind the
		// production proxies TLS terminates upstream, so trust the forwarded
		// scheme as well as a direct TLS connection.
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
