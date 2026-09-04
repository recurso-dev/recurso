package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecureMiddleware_HSTSOnlyOverHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecureMiddleware())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	plain := httptest.NewRecorder()
	r.ServeHTTP(plain, httptest.NewRequest(http.MethodGet, "/", nil))
	if plain.Header().Get("Strict-Transport-Security") != "" {
		t.Fatalf("HSTS emitted over plain HTTP")
	}
	if plain.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("nosniff missing on plain HTTP")
	}

	fwd := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	r.ServeHTTP(fwd, req)
	if fwd.Header().Get("Strict-Transport-Security") == "" {
		t.Fatalf("HSTS missing behind an HTTPS proxy")
	}
}
