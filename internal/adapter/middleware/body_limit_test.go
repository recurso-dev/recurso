package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func bodyLimitRouter(max int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimitMiddleware(max, "/v1/import/"))
	echo := func(c *gin.Context) {
		b, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, "read error")
			return
		}
		c.String(http.StatusOK, "%d", len(b))
	}
	r.POST("/webhooks/x", echo)
	r.POST("/v1/import/stripe/preview", echo)
	return r
}

// A declared Content-Length over the cap never reaches the handler.
func TestBodyLimit_RejectsDeclaredOversize(t *testing.T) {
	r := bodyLimitRouter(16)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/x", strings.NewReader(strings.Repeat("a", 32)))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
}

// A chunked body (no Content-Length) that runs past the cap fails at read time
// instead of being buffered whole.
func TestBodyLimit_ChunkedOversizeFailsOnRead(t *testing.T) {
	r := bodyLimitRouter(16)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/x", io.NopCloser(strings.NewReader(strings.Repeat("a", 32))))
	req.ContentLength = -1
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || w.Body.String() != "read error" {
		t.Fatalf("got %d %q, want 400 read error", w.Code, w.Body.String())
	}
}

// Under the cap passes untouched; exempt prefixes are not capped at all.
func TestBodyLimit_UnderCapAndExempt(t *testing.T) {
	r := bodyLimitRouter(16)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/webhooks/x", strings.NewReader("small")))
	if w.Code != http.StatusOK || w.Body.String() != "5" {
		t.Fatalf("under cap: got %d %q", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/import/stripe/preview", strings.NewReader(strings.Repeat("a", 64))))
	if w.Code != http.StatusOK || w.Body.String() != "64" {
		t.Fatalf("exempt: got %d %q", w.Code, w.Body.String())
	}
}
