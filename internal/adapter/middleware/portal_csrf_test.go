package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
)

// portalCSRFRouter mounts the CSRF middleware in front of a GET and a POST, the
// same shape as the real /portal/api group (auth would run before it, but the
// CSRF check is independent of the session identity).
func portalCSRFRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/portal/api")
	g.Use(PortalCSRFMiddleware(false)) // secure=false so cookies set over plain http in the test
	g.GET("/profile", func(c *gin.Context) { c.Status(http.StatusOK) })
	g.POST("/redeem", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

var setCookieCSRF = regexp.MustCompile(`portal_csrf=([^;]+)`)

// TestPortalCSRF_GetMintsCookie proves a safe request with no CSRF cookie is
// served AND handed a freshly minted token, so a pre-existing session (or a
// fresh login) always has a token to echo on its next state-changing call.
func TestPortalCSRF_GetMintsCookie(t *testing.T) {
	r := portalCSRFRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/portal/api/profile", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("GET without cookie: got %d, want 200", w.Code)
	}
	if !setCookieCSRF.MatchString(w.Header().Get("Set-Cookie")) {
		t.Fatalf("GET should mint a portal_csrf cookie, got Set-Cookie=%q", w.Header().Get("Set-Cookie"))
	}
}

// TestPortalCSRF_PostRequiresMatchingToken is the core defense: a state-changing
// request is rejected unless the X-CSRF-Token header matches the portal_csrf
// cookie (double-submit). It exercises every failure mode plus the happy path.
func TestPortalCSRF_PostRequiresMatchingToken(t *testing.T) {
	r := portalCSRFRouter()

	post := func(cookie, header string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/portal/api/redeem", nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "portal_csrf", Value: cookie})
		}
		if header != "" {
			req.Header.Set("X-CSRF-Token", header)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	if got := post("", ""); got != http.StatusForbidden {
		t.Errorf("no cookie, no header: got %d, want 403", got)
	}
	if got := post("tok-abc", ""); got != http.StatusForbidden {
		t.Errorf("cookie but no header: got %d, want 403", got)
	}
	if got := post("", "tok-abc"); got != http.StatusForbidden {
		t.Errorf("header but no cookie: got %d, want 403", got)
	}
	if got := post("tok-abc", "tok-xyz"); got != http.StatusForbidden {
		t.Errorf("mismatched cookie/header: got %d, want 403", got)
	}
	if got := post("tok-abc", "tok-abc"); got != http.StatusOK {
		t.Errorf("matching double-submit: got %d, want 200", got)
	}
}
