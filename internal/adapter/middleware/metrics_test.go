package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/metrics"
)

func TestMetricsMiddleware_UsesRouteTemplateNotRawPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := metrics.NewHTTPMetrics()
	r := gin.New()
	r.Use(MetricsMiddleware(m))
	r.GET("/v1/customers/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/v1/customers/abc123", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	var b strings.Builder
	m.WriteProm(&b)
	out := b.String()

	// The label must be the bounded route TEMPLATE, so cardinality can't explode.
	if !strings.Contains(out, `route="/v1/customers/:id"`) {
		t.Errorf("expected route template label, got:\n%s", out)
	}
	// The raw id must never appear as a label value.
	if strings.Contains(out, "abc123") {
		t.Errorf("raw path segment leaked into labels (cardinality risk):\n%s", out)
	}
}

func TestMetricsMiddleware_SkipsMetricsScrape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := metrics.NewHTTPMetrics()
	r := gin.New()
	r.Use(MetricsMiddleware(m))
	r.GET("/metrics", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	var b strings.Builder
	m.WriteProm(&b)
	// The scrape itself must not be counted.
	if strings.Contains(b.String(), `route="/metrics"`) {
		t.Error("/metrics scrape should not record its own request metric")
	}
}
