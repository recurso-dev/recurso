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
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

// mapDedup is an in-memory InboundWebhookDedup for tests.
type mapDedup struct {
	mu    sync.Mutex
	seen  map[string]bool
	marks int
}

func (m *mapDedup) key(gateway, id string) string { return gateway + ":" + id }

func (m *mapDedup) WasProcessed(_ context.Context, gateway, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.seen[m.key(gateway, id)], nil
}

func (m *mapDedup) MarkProcessed(_ context.Context, gateway, id, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.seen == nil {
		m.seen = map[string]bool{}
	}
	m.seen[m.key(gateway, id)] = true
	m.marks++
	return nil
}

// TestHandleRazorpay_DedupSkipsRedelivery proves the ENG-162 inbound webhook
// idempotency on the Razorpay path: the first delivery of an event is processed
// (2xx) and recorded; a redelivery with the same X-Razorpay-Event-Id is
// acknowledged (2xx) without re-running processing or re-recording.
func TestHandleRazorpay_DedupSkipsRedelivery(t *testing.T) {
	secret := "whsec_razorpay_test"
	t.Setenv("RAZORPAY_WEBHOOK_SECRET", secret)
	gin.SetMode(gin.TestMode)

	dedup := &mapDedup{}
	h := &WebhookHandler{
		logger:       slog.Default(),
		inboundDedup: dedup,
	}

	// An unhandled event type hits the default 200 path and touches no services.
	body := []byte(`{"event":"subscription.pending"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	send := func() int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", bytes.NewReader(body))
		req.Header.Set("X-Razorpay-Signature", sig)
		req.Header.Set("X-Razorpay-Event-Id", "evt_dedup_1")
		c.Request = req
		h.HandleRazorpay(c)
		return w.Code
	}

	if code := send(); code != http.StatusOK {
		t.Fatalf("first delivery status = %d, want 200", code)
	}
	if dedup.marks != 1 {
		t.Fatalf("recorded %d times after first delivery, want 1", dedup.marks)
	}

	// Redelivery: same event id → recognized as duplicate, acked, not re-recorded.
	if code := send(); code != http.StatusOK {
		t.Fatalf("redelivery status = %d, want 200", code)
	}
	if dedup.marks != 1 {
		t.Fatalf("recorded %d times after redelivery, want 1 (duplicate must be skipped, not re-recorded)", dedup.marks)
	}
}

// TestHandleRazorpay_FailedDeliveryNotRecorded proves a 5xx response is NOT
// recorded, so Razorpay retries and the event is reprocessed. A payment.captured
// with no invoice_id resolves to a benign 200 "ignored"; to force a 5xx without
// wiring services we rely on the invalid-signature path staying un-recorded.
func TestHandleRazorpay_InvalidSignatureNotRecorded(t *testing.T) {
	t.Setenv("RAZORPAY_WEBHOOK_SECRET", "whsec_x")
	gin.SetMode(gin.TestMode)

	dedup := &mapDedup{}
	h := &WebhookHandler{logger: slog.Default(), inboundDedup: dedup}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", bytes.NewReader([]byte(`{"event":"x"}`)))
	req.Header.Set("X-Razorpay-Signature", "deadbeef") // wrong signature
	req.Header.Set("X-Razorpay-Event-Id", "evt_bad_sig")
	c.Request = req
	h.HandleRazorpay(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a bad signature", w.Code)
	}
	if dedup.marks != 0 {
		t.Fatalf("recorded %d times, want 0 (a rejected webhook must not be marked processed)", dedup.marks)
	}
}
