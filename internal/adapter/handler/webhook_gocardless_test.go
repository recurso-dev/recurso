package handler

import (
	"bytes"
	"context"
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
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
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

// fakeInvoiceByPayment provides only the gateway-payment lookup capability.
type fakeInvoiceByPayment struct {
	port.InvoiceRepository // embedded nil: only the methods below may be called
	inv                    *domain.Invoice
}

func (f *fakeInvoiceByPayment) GetByGatewayPaymentIDPublic(_ context.Context, id string) (*domain.Invoice, error) {
	if f.inv != nil && id == "PM123" {
		return f.inv, nil
	}
	return nil, nil
}

func TestGoCardlessWebhook_PaymentEventUnknownPaymentIsBenign(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOCARDLESS_WEBHOOK_SECRET", "whsec")
	dedup := &mapDedup{}
	h := &WebhookHandler{
		logger:         slog.Default(),
		inboundDedup:   dedup,
		mandateService: &service.MandateService{}, // non-nil so events are processed
		invoiceRepo:    &fakeInvoiceByPayment{},   // lookup misses
	}
	body := []byte(`{"events":[{"id":"EVP1","resource_type":"payments","action":"confirmed","links":{"payment":"PM_UNKNOWN"}}]}`)
	w := postGoCardless(h, body, gcSign("whsec", body), "")
	if w.Code != http.StatusOK {
		t.Fatalf("unknown payment: got %d want 200 (body %s)", w.Code, w.Body.String())
	}
	// Benign miss is still marked processed so redeliveries don't loop.
	if dedup.marks != 1 {
		t.Fatalf("marks = %d, want 1", dedup.marks)
	}
}

func TestGoCardlessWebhook_FailedPaymentLeavesInvoiceOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOCARDLESS_WEBHOOK_SECRET", "whsec")
	h := &WebhookHandler{
		logger:         slog.Default(),
		mandateService: &service.MandateService{},
		invoiceRepo:    &fakeInvoiceByPayment{inv: &domain.Invoice{ID: uuid.New(), Status: "open"}},
	}
	body := []byte(`{"events":[{"id":"EVP2","resource_type":"payments","action":"failed","links":{"payment":"PM123"}}]}`)
	if got := postGoCardless(h, body, gcSign("whsec", body), "").Code; got != http.StatusOK {
		t.Fatalf("failed payment: got %d want 200", got)
	}
}

// gcReturnRepo serves the gateway-payment lookup AND the reads
// ReverseSettledPayment makes, so the chargeback chain runs in memory.
type gcReturnRepo struct {
	port.InvoiceRepository
	inv          *domain.Invoice
	reverseCalls int
}

func (r *gcReturnRepo) GetByGatewayPaymentIDPublic(_ context.Context, id string) (*domain.Invoice, error) {
	if id == "PM_CB" {
		return r.inv, nil
	}
	return nil, nil
}
func (r *gcReturnRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Invoice, error) {
	return r.inv, nil
}
func (r *gcReturnRepo) ReverseToUnpaid(_ context.Context, _, _ uuid.UUID) (bool, error) {
	r.reverseCalls++
	r.inv.Status = domain.InvoiceStatusPastDue
	return true, nil
}

func TestGoCardlessWebhook_ChargebackReversesSettlement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("GOCARDLESS_WEBHOOK_SECRET", "whsec")
	repo := &gcReturnRepo{inv: &domain.Invoice{
		ID: uuid.New(), TenantID: uuid.New(), Status: domain.InvoiceStatusPaid, Total: 5000,
	}}
	subSvc := service.NewSubscriptionService(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := &WebhookHandler{
		logger:         slog.Default(),
		mandateService: &service.MandateService{},
		invoiceRepo:    repo,
		subService:     subSvc,
	}

	body := []byte(`{"events":[{"id":"EVCB1","resource_type":"payments","action":"charged_back","links":{"payment":"PM_CB"}}]}`)
	if got := postGoCardless(h, body, gcSign("whsec", body), "").Code; got != http.StatusOK {
		t.Fatalf("chargeback: got %d want 200", got)
	}
	if repo.reverseCalls != 1 {
		t.Fatalf("ReverseToUnpaid calls = %d, want 1", repo.reverseCalls)
	}
	if repo.inv.Status != domain.InvoiceStatusPastDue {
		t.Fatalf("invoice status = %s, want past_due", repo.inv.Status)
	}

	// Redelivery (fresh event id so dedup doesn't shortcut): invoice already
	// past_due — the paid guard stops a second reversal.
	body2 := []byte(`{"events":[{"id":"EVCB2","resource_type":"payments","action":"charged_back","links":{"payment":"PM_CB"}}]}`)
	if got := postGoCardless(h, body2, gcSign("whsec", body2), "").Code; got != http.StatusOK {
		t.Fatalf("chargeback redelivery: got %d want 200", got)
	}
	if repo.reverseCalls != 1 {
		t.Fatalf("redelivery re-reversed: calls = %d, want 1", repo.reverseCalls)
	}
}
