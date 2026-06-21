package middleware

import (
	"bytes"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

// responseWriter is a wrapper to capture the response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// IdempotencyMiddleware ensures that requests with the same Idempotency-Key
// return the same response without re-processing.
func IdempotencyMiddleware(store port.IdempotencyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Check for Header
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		// 2. Identify Tenant (Scope key by tenant for security)
		// Assuming AuthMiddleware has already run and set tenant_id
		// If not, we might process it anyway or wait?
		// Better to run AFTER auth middleware.
		var tenantIDPrefix string
		if tID, ok := c.Get("tenant_id"); ok {
			tenantIDPrefix = fmt.Sprintf("%v:", tID)
		} else {
			// If unauthenticated (e.g. login tokens), maybe scope by IP?
			// For this system, critical actions are authenticated.
			tenantIDPrefix = "global:"
		}

		storageKey := fmt.Sprintf("%s%s", tenantIDPrefix, key)

		// 3. Check Storage (Hit)
		cached, err := store.Get(c.Request.Context(), storageKey)
		if err != nil {
			// On error, log and proceed primarily? Or fail closed?
			// Proceeding effectively disables idempotency which is safer than blocking.
			c.Next()
			return
		}

		if cached != nil {
			// HIT: Return cached response
			c.Header("X-Idempotency-Hit", "true")

			// Replay headers
			for k, v := range cached.Headers {
				c.Header(k, v)
			}

			c.Data(cached.Status, c.Writer.Header().Get("Content-Type"), cached.Body)
			c.Abort()
			return
		}

		// 4. Wrap Writer (Miss)
		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = w

		// 5. Process Request
		c.Next()

		// 6. Store Response (After successful processing)
		// Only store success/failure codes that are deterministic?
		// Usually we store 2xx, 4xx, 5xx (except transient).
		status := c.Writer.Status()

		// Capture headers
		headers := make(map[string]string)
		for k, v := range c.Writer.Header() {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		res := &domain.StoredResponse{
			Status:  status,
			Body:    w.body.Bytes(),
			Headers: headers,
		}

		// Asynchronously save to avoid blocking response?
		// Better to wait to ensure consistency for client immediately checking?
		// Let's blocking save for consistency.
		_ = store.Set(c.Request.Context(), storageKey, res)
	}
}
