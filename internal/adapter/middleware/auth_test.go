package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// fakeResolver stands in for *service.AuthService in the dual-auth middleware.
type fakeResolver struct {
	user *domain.User
	err  error
}

func (f *fakeResolver) ResolveSession(_ context.Context, _ string) (*domain.User, error) {
	return f.user, f.err
}

func runMiddleware(mw gin.HandlerFunc, req *http.Request) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	mw(c)
	return c, w
}

func TestDualAuth_SessionCookieAuthenticates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	userID := uuid.New()
	resolver := &fakeResolver{user: &domain.User{ID: userID, TenantID: tenantID, Role: domain.RoleAdmin}}
	mw := SessionOrAPIKeyMiddleware(db.NewTenantRepository(nil), resolver, false)

	req := httptest.NewRequest(http.MethodGet, "/v1/plans", nil)
	req.AddCookie(&http.Cookie{Name: domain.SessionCookieName, Value: "opaque-token"})

	c, w := runMiddleware(mw, req)
	if c.IsAborted() {
		t.Fatalf("request aborted, status=%d body=%s", w.Code, w.Body.String())
	}
	if got := GetTenantID(c); got != tenantID {
		t.Fatalf("tenant_id = %v, want %v", got, tenantID)
	}
	if got := GetUserID(c); got != userID {
		t.Fatalf("user_id = %v, want %v", got, userID)
	}
	if role, ok := GetUserRole(c); !ok || role != "admin" {
		t.Fatalf("user_role = %q ok=%v, want admin", role, ok)
	}
}

func TestDualAuth_DevBypassAPIKeyAuthenticates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// The dev-bypass token exercises the API-key branch (extract header ->
	// resolveAPIKey) without a database, proving backward compatibility of the
	// key path and that it sets the same tenant_id context.
	t.Setenv("APP_ENV", "development")
	t.Setenv("ALLOW_DEV_BYPASS", "true")

	mw := SessionOrAPIKeyMiddleware(db.NewTenantRepository(nil), &fakeResolver{err: domain.ErrSessionNotFound}, false)
	req := httptest.NewRequest(http.MethodGet, "/v1/plans", nil)
	req.Header.Set("Authorization", "Bearer recurso_secret")

	c, w := runMiddleware(mw, req)
	if c.IsAborted() {
		t.Fatalf("request aborted, status=%d body=%s", w.Code, w.Body.String())
	}
	if GetTenantID(c) == uuid.Nil {
		t.Fatal("expected a tenant_id from dev-bypass API key")
	}
	// API-key requests carry no user context.
	if _, ok := GetUserRole(c); ok {
		t.Fatal("API-key request must not set user_role")
	}
}

func TestDualAuth_NoCredentialsRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := SessionOrAPIKeyMiddleware(db.NewTenantRepository(nil), &fakeResolver{err: domain.ErrSessionNotFound}, false)
	req := httptest.NewRequest(http.MethodGet, "/v1/plans", nil)

	c, w := runMiddleware(mw, req)
	if !c.IsAborted() || w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d aborted=%v, want 401 aborted", w.Code, c.IsAborted())
	}
}

func TestDualAuth_InvalidSessionNoKeyRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Expired/invalid session cookie and no API key → unauthorized.
	mw := SessionOrAPIKeyMiddleware(db.NewTenantRepository(nil), &fakeResolver{err: domain.ErrSessionNotFound}, false)
	req := httptest.NewRequest(http.MethodGet, "/v1/plans", nil)
	req.AddCookie(&http.Cookie{Name: domain.SessionCookieName, Value: "expired-token"})

	c, w := runMiddleware(mw, req)
	if !c.IsAborted() || w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d aborted=%v, want 401 aborted", w.Code, c.IsAborted())
	}
}

func TestDualAuth_StaleCookieFallsBackToDevBypassKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("APP_ENV", "development")
	t.Setenv("ALLOW_DEV_BYPASS", "true")

	// Stale cookie AND a valid (dev-bypass) key: the key path must still authenticate.
	mw := SessionOrAPIKeyMiddleware(db.NewTenantRepository(nil), &fakeResolver{err: domain.ErrSessionNotFound}, false)
	req := httptest.NewRequest(http.MethodGet, "/v1/plans", nil)
	req.AddCookie(&http.Cookie{Name: domain.SessionCookieName, Value: "stale"})
	req.Header.Set("Authorization", "Bearer recurso_secret")

	c, w := runMiddleware(mw, req)
	if c.IsAborted() {
		t.Fatalf("request aborted, status=%d body=%s", w.Code, w.Body.String())
	}
	if GetTenantID(c) == uuid.Nil {
		t.Fatal("expected tenant_id from fallback API key")
	}
}
