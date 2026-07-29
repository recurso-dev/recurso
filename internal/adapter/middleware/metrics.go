package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/metrics"
)

// MetricsMiddleware records request count + latency for every request, labelled
// by method, the gin route TEMPLATE (bounded cardinality — never the raw path),
// and status. The /metrics scrape itself is not measured.
func MetricsMiddleware(m *metrics.HTTPMetrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		m.Observe(c.Request.Method, c.FullPath(), c.Writer.Status(), time.Since(start).Seconds())
	}
}
