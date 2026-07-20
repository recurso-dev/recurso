package handler

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// fakeConnResolver is an in-memory gatewayConnResolver for the per-connection
// webhook tests.
type fakeConnResolver struct {
	conns     map[uuid.UUID]*domain.GatewayConnection
	secret    string
	secretErr error
}

func (f *fakeConnResolver) GetByID(_ context.Context, id uuid.UUID) (*domain.GatewayConnection, error) {
	if c, ok := f.conns[id]; ok {
		return c, nil
	}
	return nil, domain.ErrGatewayConnectionNotFound
}
func (f *fakeConnResolver) OpenWebhookSecret(*domain.GatewayConnection) (string, error) {
	return f.secret, f.secretErr
}

// postWithConn drives HandleStripe against the per-connection route with the
// given :connID path param.
func postWithConn(h *WebhookHandler, connID string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/webhooks/stripe/"+connID, bytes.NewReader([]byte(`{"type":"x"}`)))
	c.Request.Header.Set("Stripe-Signature", "t=1,v1=bogus")
	c.Params = gin.Params{{Key: "connID", Value: connID}}
	h.HandleStripe(c)
	return w
}

func TestPerConnectionWebhook_NoResolver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &WebhookHandler{logger: slog.Default()} // gatewayConns nil
	if got := postWithConn(h, uuid.New().String()).Code; got != http.StatusServiceUnavailable {
		t.Fatalf("no resolver: got %d want 503", got)
	}
}

func TestPerConnectionWebhook_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &WebhookHandler{logger: slog.Default(), gatewayConns: &fakeConnResolver{}}
	if got := postWithConn(h, "not-a-uuid").Code; got != http.StatusBadRequest {
		t.Fatalf("invalid id: got %d want 400", got)
	}
}

func TestPerConnectionWebhook_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &WebhookHandler{logger: slog.Default(), gatewayConns: &fakeConnResolver{conns: map[uuid.UUID]*domain.GatewayConnection{}}}
	if got := postWithConn(h, uuid.New().String()).Code; got != http.StatusNotFound {
		t.Fatalf("unknown connection: got %d want 404", got)
	}
}

func TestPerConnectionWebhook_WrongProviderIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New()
	// A Razorpay connection must not resolve a secret for the Stripe route.
	h := &WebhookHandler{logger: slog.Default(), gatewayConns: &fakeConnResolver{
		conns:  map[uuid.UUID]*domain.GatewayConnection{id: {ID: id, Provider: domain.GatewayRazorpay, Active: true}},
		secret: "whsec_x",
	}}
	if got := postWithConn(h, id.String()).Code; got != http.StatusNotFound {
		t.Fatalf("wrong provider: got %d want 404", got)
	}
}

func TestPerConnectionWebhook_NoSecretFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New()
	h := &WebhookHandler{logger: slog.Default(), gatewayConns: &fakeConnResolver{
		conns:  map[uuid.UUID]*domain.GatewayConnection{id: {ID: id, Provider: domain.GatewayStripe, Active: true}},
		secret: "", // no webhook secret configured on the connection
	}}
	if got := postWithConn(h, id.String()).Code; got != http.StatusServiceUnavailable {
		t.Fatalf("no secret: got %d want 503 (fail closed)", got)
	}
}

func TestPerConnectionWebhook_ResolvedSecretReachesVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New()
	h := &WebhookHandler{logger: slog.Default(), gatewayConns: &fakeConnResolver{
		conns:  map[uuid.UUID]*domain.GatewayConnection{id: {ID: id, Provider: domain.GatewayStripe, Active: true}},
		secret: "whsec_realish",
	}}
	// A valid connection + secret means verification runs; the bogus signature
	// is then rejected 401 (proves the per-connection secret was used, not a
	// 503/404 short-circuit).
	if got := postWithConn(h, id.String()).Code; got != http.StatusUnauthorized {
		t.Fatalf("resolved secret: got %d want 401 (reached signature verification)", got)
	}
}
