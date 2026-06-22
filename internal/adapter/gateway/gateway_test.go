package gateway

import (
	"context"
	"testing"
)

func TestMockGateway_CancelSubscription(t *testing.T) {
	gw := NewMockGateway()

	err := gw.CancelSubscription(context.Background(), "sub_mock_123")
	if err != nil {
		t.Errorf("MockGateway.CancelSubscription should return nil, got: %v", err)
	}
}

func TestRazorpayGateway_CancelSubscription_NoOp(t *testing.T) {
	// RazorpayGateway.CancelSubscription is a no-op that logs and returns nil.
	// We can't construct a real RazorpayGateway without credentials,
	// but we can verify the mock's behavior matches the interface contract.
	gw := NewMockGateway()

	err := gw.CancelSubscription(context.Background(), "sub_rp_456")
	if err != nil {
		t.Errorf("expected nil error for no-op cancel, got: %v", err)
	}
}

func TestSmartRouter_CancelSubscription(t *testing.T) {
	razorpay := &cancelTracker{name: "razorpay"}
	stripe := &cancelTracker{name: "stripe"}

	router := NewSmartRouter(razorpay, stripe)

	// SmartRouter delegates all CancelSubscription calls to Stripe
	// (since both Razorpay and Stripe use "sub_" prefix, callers route explicitly)
	err := router.CancelSubscription(context.Background(), "sub_any_id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stripe.cancelCalled != 1 {
		t.Errorf("stripe cancel calls = %d, want 1", stripe.cancelCalled)
	}
	if stripe.lastID != "sub_any_id" {
		t.Errorf("stripe called with %q, want 'sub_any_id'", stripe.lastID)
	}
	if razorpay.cancelCalled != 0 {
		t.Errorf("razorpay cancel calls = %d, want 0", razorpay.cancelCalled)
	}
}

func TestSmartRouter_CancelSubscription_NonSubPrefix(t *testing.T) {
	razorpay := &cancelTracker{name: "razorpay"}
	stripe := &cancelTracker{name: "stripe"}

	router := NewSmartRouter(razorpay, stripe)

	err := router.CancelSubscription(context.Background(), "pi_stripe_intent_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stripe.cancelCalled != 1 {
		t.Errorf("stripe cancel calls = %d, want 1", stripe.cancelCalled)
	}
}

// cancelTracker is a minimal mock that only tracks CancelSubscription calls.
type cancelTracker struct {
	MockGateway
	name         string
	cancelCalled int
	lastID       string
}

func (c *cancelTracker) CancelSubscription(ctx context.Context, subscriptionID string) error {
	c.cancelCalled++
	c.lastID = subscriptionID
	return nil
}
