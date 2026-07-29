package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSentryMiddleware_PassesThroughNormalRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SentryMiddleware())
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Fatalf("pass-through broken: code=%d body=%q", w.Code, w.Body.String())
	}
}

func TestSentryMiddleware_PanicStillBecomes500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Recovery outermost (as gin.Default provides), Sentry inner — a panic is
	// captured then re-raised, and Recovery turns it into a 500 (no crash).
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(SentryMiddleware())
	r.GET("/boom", func(c *gin.Context) { panic("kaboom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic should yield 500, got %d", w.Code)
	}
}
