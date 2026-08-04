package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// magicLinkExpiredRepo makes VerifyMagicLink fail with the "expired" state.
type magicLinkExpiredRepo struct {
	port.MagicLinkRepository
}

func (m *magicLinkExpiredRepo) GetByToken(_ context.Context, _ string) (*domain.MagicLink, error) {
	// A link that exists but is past its expiry — the service returns
	// ErrMagicLinkExpired, which the handler must NOT leak to the client.
	past := time.Unix(0, 0)
	return &domain.MagicLink{ID: uuid.New(), CustomerID: uuid.New(), ExpiresAt: past}, nil
}

func newVerifyHandler() *PortalAPIHandler {
	svc := service.NewPortalService(nil, nil, &magicLinkExpiredRepo{}, nil, nil, nil, nil, "")
	return NewPortalAPIHandler(svc)
}

// TestVerifyMagicLink_GenericError proves #18: the verify endpoint must return
// ONE generic message for every failure state (invalid / expired / used) so the
// response can't be used to probe a token's state. Here the token is expired;
// the body must not say "expired".
func TestVerifyMagicLink_GenericError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/portal/auth/verify", newVerifyHandler().VerifyMagicLink)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"token": "some-token"})
	req := httptest.NewRequest(http.MethodPost, "/portal/auth/verify", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	msg := w.Body.String()
	// The specific service error strings must NOT reach the client — only the
	// single generic message. ("expired" appears in the generic copy itself, so
	// match the distinctive service phrasings instead.)
	for _, leak := range []string{"magic link has expired", "already been used", "invalid magic link"} {
		if strings.Contains(strings.ToLower(msg), leak) {
			t.Errorf("response leaks the token state %q (must be generic): %s", leak, msg)
		}
	}
	if !strings.Contains(msg, "invalid or has expired. Please request a new one") {
		t.Errorf("expected the generic message, got: %s", msg)
	}
}

// TestVerifyMagicLink_AcceptsPostBody confirms the token is read from the JSON
// body (the #18 fix — off the query string, which leaks via Referer/logs).
func TestVerifyMagicLink_AcceptsPostBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/portal/auth/verify", newVerifyHandler().VerifyMagicLink)

	// No query string at all — the token must be picked up from the body. The
	// repo is wired to fail, so we assert we got PAST the "token required" 400
	// (i.e. the body token was read) into the 401 verify path.
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"token": "body-token"})
	req := httptest.NewRequest(http.MethodPost, "/portal/auth/verify", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest {
		t.Fatalf("got 400 (token not read from body); want the 401 verify path")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
