package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestHandleStripe_FailsClosedWithoutSecret proves the ENG-175 fix: when
// STRIPE_WEBHOOK_SECRET is unset the handler REJECTS the webhook (503) instead
// of processing it unverified. A forged charge/invoice event must never settle
// an invoice on a misconfigured deploy. The reject happens before any service
// is touched, so a handler with nil services is sufficient (and proves nothing
// downstream runs).
func TestHandleStripe_FailsClosedWithoutSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &WebhookHandler{logger: slog.Default(), stripeWebhookSecret: ""}

	// A forged "payment succeeded" event with a plausible invoice id.
	forged := []byte(`{"type":"invoice.payment_succeeded","data":{"object":{"id":"in_forged","metadata":{"invoice_id":"11111111-1111-1111-1111-111111111111"}}}}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(forged))

	h.HandleStripe(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured-secret webhook status = %d, want 503 (fail closed)", w.Code)
	}
}

// TestHandleStripe_RejectsBadSignature proves that with a secret configured, an
// event with a missing/invalid Stripe-Signature is rejected (401), never
// processed.
func TestHandleStripe_RejectsBadSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &WebhookHandler{logger: slog.Default(), stripeWebhookSecret: "whsec_test_secret"}

	body := []byte(`{"type":"invoice.payment_succeeded"}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(body))
	c.Request.Header.Set("Stripe-Signature", "t=123,v1=deadbeef") // bogus

	h.HandleStripe(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad-signature webhook status = %d, want 401", w.Code)
	}
}
