package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// The dedup key must prefer Razorpay's event id and, when the (unsigned)
// header is absent, derive a stable key from the signed body so a replayed
// delivery still dedups.
func TestRazorpayEventID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"event":"payment.captured","payload":{}}`)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/webhooks/razorpay", nil)
	c.Request.Header.Set("X-Razorpay-Event-Id", "evt_123")
	if got := razorpayEventID(c, body); got != "evt_123" {
		t.Fatalf("with header: %q", got)
	}

	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest("POST", "/webhooks/razorpay", nil)
	first := razorpayEventID(c2, body)
	second := razorpayEventID(c2, body)
	if first == "" || first != second {
		t.Fatalf("body fallback must be stable and non-empty: %q vs %q", first, second)
	}
	if other := razorpayEventID(c2, []byte(`{"event":"payment.captured","payload":{"x":1}}`)); other == first {
		t.Fatalf("different bodies must not collide")
	}
}
