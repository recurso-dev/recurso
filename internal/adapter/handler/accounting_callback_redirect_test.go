package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestAccountingOAuthCallback_RedirectsToDashboard proves the callback answers
// top-level browser navigations with a 302 back to the dashboard's
// Integrations page instead of raw JSON — for every failure class reachable
// without a live provider token exchange.
func TestAccountingOAuthCallback_RedirectsToDashboard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Trailing slash on DASHBOARD_URL must not produce a double slash.
	h := NewAccountingHandler(nil, nil, []byte("test-secret"), "https://app.example.com/")

	router := gin.New()
	router.GET("/v1/accounting/callback/:provider", h.OAuthCallback)

	validState := h.generateOAuthState(uuid.New(), "quickbooks")

	cases := []struct {
		name         string
		url          string
		wantLocation string
	}{
		{
			name:         "missing authorization code",
			url:          "/v1/accounting/callback/quickbooks?state=" + validState,
			wantLocation: "https://app.example.com/integrations?error=missing_code",
		},
		{
			name:         "tampered state",
			url:          "/v1/accounting/callback/quickbooks?code=abc&state=forged",
			wantLocation: "https://app.example.com/integrations?error=invalid_state",
		},
		{
			name:         "unsupported provider",
			url:          "/v1/accounting/callback/sage?code=abc&state=" + validState,
			wantLocation: "https://app.example.com/integrations?error=unsupported_provider",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusFound, rec.Body.String())
			}
			if got := rec.Header().Get("Location"); got != tc.wantLocation {
				t.Fatalf("Location = %q, want %q", got, tc.wantLocation)
			}
		})
	}
}
