package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func gcSign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func postGoCardless(h *WebhookHandler, body []byte, signature, connID string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	url := "/webhooks/gocardless"
	if connID != "" {
		url += "/" + connID
		c.Params = gin.Params{{Key: "connID", Value: connID}}
	}
	c.Request = httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if signature != "" {
		c.Request.Header.Set("Webhook-Signature", signature)
	}
	h.HandleGoCardless(c)
	return w
}

func TestGoCardlessWebhook_NoSecretFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOCARDLESS_WEBHOOK_SECRET", "")
	h := &WebhookHandler{logger: slog.Default()}
	if got := postGoCardless(h, []byte(`{"events":[]}`), "sig", "").Code; got != http.StatusServiceUnavailable {
		t.Fatalf("no secret: got %d want 503", got)
	}
}

func TestGoCardlessWebhook_BadSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOCARDLESS_WEBHOOK_SECRET", "whsec")
	h := &WebhookHandler{logger: slog.Default()}
	if got := postGoCardless(h, []byte(`{"events":[]}`), "deadbeef", "").Code; got != http.StatusUnauthorized {
		t.Fatalf("bad signature: got %d want 401", got)
	}
}

func TestGoCardlessWebhook_ValidSignatureEmptyBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOCARDLESS_WEBHOOK_SECRET", "whsec")
	h := &WebhookHandler{logger: slog.Default()} // mandateService nil -> ignored
	body := []byte(`{"events":[{"id":"EV1","resource_type":"mandates","action":"active","links":{"mandate":"MD1"}}]}`)
	w := postGoCardless(h, body, gcSign("whsec", body), "")
	if w.Code != http.StatusOK {
		t.Fatalf("valid signature: got %d want 200 (body %s)", w.Code, w.Body.String())
	}
}

func TestGoCardlessWebhook_PerConnectionWrongProviderIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New()
	// A Stripe connection must not resolve a secret for the GoCardless route.
	h := &WebhookHandler{logger: slog.Default(), gatewayConns: &fakeConnResolver{
		conns:  map[uuid.UUID]*domain.GatewayConnection{id: {ID: id, Provider: domain.GatewayStripe, Active: true}},
		secret: "whsec",
	}}
	body := []byte(`{"events":[]}`)
	if got := postGoCardless(h, body, gcSign("whsec", body), id.String()).Code; got != http.StatusNotFound {
		t.Fatalf("wrong provider: got %d want 404", got)
	}
}

func TestGoCardlessWebhook_PerConnectionSecretVerifies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New()
	h := &WebhookHandler{logger: slog.Default(), gatewayConns: &fakeConnResolver{
		conns:  map[uuid.UUID]*domain.GatewayConnection{id: {ID: id, Provider: domain.GatewayGoCardless, Active: true}},
		secret: "conn_secret",
	}}
	body := []byte(`{"events":[]}`)
	// Signed with the connection's secret: accepted even with no env secret.
	t.Setenv("GOCARDLESS_WEBHOOK_SECRET", "")
	if got := postGoCardless(h, body, gcSign("conn_secret", body), id.String()).Code; got != http.StatusOK {
		t.Fatalf("per-connection: got %d want 200", got)
	}
	// Signed with the wrong secret: rejected.
	if got := postGoCardless(h, body, gcSign("other", body), id.String()).Code; got != http.StatusUnauthorized {
		t.Fatalf("wrong secret: got %d want 401", got)
	}
}
