package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/httperr"
)

// BodyLimitMiddleware caps request bodies at maxBytes.
//
// A declared Content-Length above the cap is rejected up front with 413.
// Everything else (chunked or unknown length) is wrapped in http.MaxBytesReader
// so a read past the cap fails with an error instead of buffering the whole
// stream: the inbound webhook handlers io.ReadAll the body BEFORE verifying its
// signature, and the server deliberately sets no ReadTimeout (large exports and
// usage ingests are legitimate), so without this an anonymous client could
// stream an unbounded body into memory.
//
// Paths under an exempt prefix keep their own handler-side limit (the migration
// importers accept 25 MiB CSV/JSON dumps and cap themselves).
func BodyLimitMiddleware(maxBytes int64, exemptPrefixes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, p := range exemptPrefixes {
			if strings.HasPrefix(c.Request.URL.Path, p) {
				c.Next()
				return
			}
		}
		if c.Request.ContentLength > maxBytes {
			httperr.Abort(c, http.StatusRequestEntityTooLarge, httperr.CodeValidationFailed, "request body too large")
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
