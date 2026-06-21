package middleware

import "github.com/gin-gonic/gin"

// SecureMiddleware adds security headers to responses
func SecureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Only set HSTS if using HTTPS (detect via headers or environment)
		// For consistency in this project, we'll set it but browsers might ignore on localhost http
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		c.Next()
	}
}
