package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func demoGuardRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DemoGuard())
	ok := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }
	// Representative routes on both sides of the guard.
	r.POST("/v1/team/invite", ok)
	r.PUT("/v1/settings/sso", ok)
	r.DELETE("/v1/developer/keys/abc", ok)
	r.POST("/auth/register", ok)
	r.POST("/auth/forgot-password", ok)
	r.PUT("/v1/account", ok)
	r.GET("/v1/team", ok)
	r.POST("/v1/plans", ok)
	r.POST("/v1/customers", ok)
	r.POST("/v1/usage/events", ok)
	r.POST("/v1/wallets", ok)
	r.POST("/auth/login", ok)
	return r
}

func TestDemoGuardBlocksDestructiveEdges(t *testing.T) {
	r := demoGuardRouter()
	blocked := []struct{ method, path string }{
		{"POST", "/v1/team/invite"},
		{"PUT", "/v1/settings/sso"},
		{"DELETE", "/v1/developer/keys/abc"},
		{"POST", "/auth/register"},
		{"POST", "/auth/forgot-password"},
		{"PUT", "/v1/account"},
	}
	for _, tc := range blocked {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}")))
		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s = %d, want 403", tc.method, tc.path, w.Code)
			continue
		}
		var body struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body.Error.Code != "demo_mode" {
			t.Errorf("%s %s error code = %q, want demo_mode", tc.method, tc.path, body.Error.Code)
		}
	}
}

func TestDemoGuardAllowsThePointOfTheDemo(t *testing.T) {
	r := demoGuardRouter()
	allowed := []struct{ method, path string }{
		{"POST", "/v1/plans"}, // creating things IS the demo (D4)
		{"POST", "/v1/customers"},
		{"POST", "/v1/usage/events"},
		{"POST", "/v1/wallets"},
		{"POST", "/auth/login"}, // the demo session logs in normally
		{"GET", "/v1/team"},     // reads are always fine
	}
	for _, tc := range allowed {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}")))
		if w.Code != http.StatusOK {
			t.Errorf("%s %s = %d, want 200 (must not be guarded)", tc.method, tc.path, w.Code)
		}
	}
}
