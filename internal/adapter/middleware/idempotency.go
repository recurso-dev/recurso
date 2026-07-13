package middleware

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/httperr"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
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

// mutatingMethods are the HTTP methods covered by idempotency replay.
// Applied group-wide, this covers every money-mutating POST
// (subscriptions, charges, advance invoices, credit notes, usage events,
// e-invoice retries, gift purchases, offline payments, quote conversion, ...)
// as well as PUT/PATCH/DELETE.
var mutatingMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// IdempotencyMiddleware ensures that mutating requests carrying the same
// Idempotency-Key return the stored response (status, headers, body)
// without re-processing.
//
// Behavior:
//   - The Idempotency-Key header is RECOMMENDED but not required. Requests
//     without it are processed normally and never replayed.
//   - Keys are scoped per tenant, HTTP method, and request path, so the
//     same key sent to different endpoints (or by different tenants) is
//     processed independently.
//   - Only mutating methods (POST/PUT/PATCH/DELETE) participate; GETs pass
//     through untouched.
//   - 5xx responses are NOT stored, so transient server failures can be
//     retried with the same key.
//   - Storage errors degrade gracefully: the request is processed as if no
//     key were present (works with the Redis store or the in-memory
//     fallback used when Redis is not configured).
//
// Concurrency: the key is reserved with an atomic Claim before processing, so
// two in-flight requests with the same key cannot both execute — the second is
// rejected with 409 while the first runs, and replays the stored response once
// it completes. A 5xx or panic releases the reservation so the key stays
// retryable.
func IdempotencyMiddleware(store port.IdempotencyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Only mutating methods are subject to idempotency replay.
		if !mutatingMethods[c.Request.Method] {
			c.Next()
			return
		}

		// 2. Check for header (recommended, not enforced).
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		// 3. Scope key by tenant (set by AuthMiddleware), method, and path
		// so a reused key cannot replay a response from another endpoint
		// or another tenant.
		tenantScope := "global"
		if tID, ok := c.Get("tenant_id"); ok {
			tenantScope = fmt.Sprintf("%v", tID)
		}
		storageKey := fmt.Sprintf("idem:%s:%s:%s:%s", tenantScope, c.Request.Method, c.Request.URL.Path, key)

		// 4. Atomically reserve the key. This is the concurrency gate.
		acquired, existing, err := store.Claim(c.Request.Context(), storageKey)
		if err != nil {
			// Storage unavailable: proceed without idempotency rather than
			// blocking the request (graceful degradation).
			c.Next()
			return
		}

		if !acquired {
			if existing != nil {
				// A completed response exists — replay it.
				c.Header("X-Idempotency-Hit", "true")
				for k, v := range existing.Headers {
					c.Header(k, v)
				}
				c.Data(existing.Status, c.Writer.Header().Get("Content-Type"), existing.Body)
				c.Abort()
				return
			}
			// A concurrent request holds the reservation and is still in flight.
			httperr.Respond(c, http.StatusConflict, httperr.CodeConflict,
				"a request with this Idempotency-Key is already in progress")
			c.Abort()
			return
		}

		// 5. Reservation acquired. Wrap writer to capture the response.
		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = w

		// 6. On the way out, either persist the completed response or release the
		// reservation so the key stays retryable. Deferred so a panic (unwinding
		// to an outer recovery) still releases the key rather than wedging it.
		defer func() {
			if rec := recover(); rec != nil {
				_ = store.Delete(c.Request.Context(), storageKey)
				panic(rec) // let the outer recovery middleware handle it
			}
			status := c.Writer.Status()
			// Skip 5xx so transient server failures remain retryable.
			if status >= http.StatusInternalServerError {
				_ = store.Delete(c.Request.Context(), storageKey)
				return
			}
			headers := make(map[string]string)
			for k, v := range c.Writer.Header() {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}
			// Blocking save so a client immediately retrying sees the stored
			// response consistently.
			_ = store.Set(c.Request.Context(), storageKey, &domain.StoredResponse{
				Status:  status,
				Body:    w.body.Bytes(),
				Headers: headers,
			})
		}()

		// 7. Process request.
		c.Next()
	}
}
