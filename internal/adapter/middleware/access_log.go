package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessLog writes one structured line per request through slog, so access
// logs land in the same JSON stream as everything else (gin's default logger
// printed plain text into it). Health probes are skipped: they are the
// noisiest and least interesting lines.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"route", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"bytes", c.Writer.Size(),
			"ip", c.ClientIP(),
		}
		if rid := c.Writer.Header().Get("X-Request-ID"); rid != "" {
			attrs = append(attrs, "request_id", rid)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}
		switch {
		case c.Writer.Status() >= 500:
			slog.ErrorContext(c.Request.Context(), "request", attrs...)
		case c.Writer.Status() >= 400:
			slog.WarnContext(c.Request.Context(), "request", attrs...)
		default:
			slog.InfoContext(c.Request.Context(), "request", attrs...)
		}
	}
}
