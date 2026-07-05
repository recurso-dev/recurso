package gateway

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSmartRouter_Refund_PrefixRouting(t *testing.T) {
	tests := []struct {
		name        string
		paymentID   string
		currency    string
		wantGateway string // "razorpay" or "stripe"
	}{
		{"stripe payment intent", "pi_3OqXyz", "USD", "stripe"},
		{"stripe charge", "ch_3OqAbc", "USD", "stripe"},
		{"stripe id wins over INR currency", "pi_3OqXyz", "INR", "stripe"},
		{"razorpay payment", "pay_NxYz12", "INR", "razorpay"},
		{"razorpay id wins over USD currency", "pay_NxYz12", "USD", "razorpay"},
		{"unknown prefix falls back to currency INR", "mockpay_1", "INR", "razorpay"},
		{"unknown prefix falls back to currency USD", "mockpay_1", "USD", "stripe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			razorpay := NewMockGateway()
			stripe := NewMockGateway()
			router := NewSmartRouter(razorpay, stripe)

			res, err := router.Refund(context.Background(), tt.paymentID, 500, tt.currency)
			if err != nil {
				t.Fatalf("Refund returned error: %v", err)
			}
			if !strings.HasPrefix(res.RefundID, "rfnd_mock_") {
				t.Errorf("refund id = %s, want rfnd_mock_* from mock gateway", res.RefundID)
			}

			gotRazorpay := len(razorpay.RefundCalls())
			gotStripe := len(stripe.RefundCalls())
			switch tt.wantGateway {
			case "razorpay":
				if gotRazorpay != 1 || gotStripe != 0 {
					t.Errorf("calls razorpay=%d stripe=%d, want razorpay only", gotRazorpay, gotStripe)
				}
			case "stripe":
				if gotStripe != 1 || gotRazorpay != 0 {
					t.Errorf("calls razorpay=%d stripe=%d, want stripe only", gotRazorpay, gotStripe)
				}
			}
		})
	}
}

func TestSmartRouter_Refund_RecordsArguments(t *testing.T) {
	razorpay := NewMockGateway()
	router := NewSmartRouter(razorpay, NewMockGateway())

	if _, err := router.Refund(context.Background(), "pay_123", 750, "INR"); err != nil {
		t.Fatalf("Refund returned error: %v", err)
	}

	calls := razorpay.RefundCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 refund call, got %d", len(calls))
	}
	if calls[0].PaymentID != "pay_123" || calls[0].Amount != 750 || calls[0].Currency != "INR" {
		t.Errorf("recorded call = %+v, want {pay_123 750 INR}", calls[0])
	}
}

func TestSmartRouter_Refund_UnconfiguredGateway(t *testing.T) {
	t.Run("stripe missing", func(t *testing.T) {
		router := NewSmartRouter(NewMockGateway(), nil)
		if _, err := router.Refund(context.Background(), "pi_123", 100, "USD"); err == nil {
			t.Fatal("expected error when stripe gateway is not configured")
		}
	})
	t.Run("razorpay missing", func(t *testing.T) {
		router := NewSmartRouter(nil, NewMockGateway())
		if _, err := router.Refund(context.Background(), "pay_123", 100, "INR"); err == nil {
			t.Fatal("expected error when razorpay gateway is not configured")
		}
	})
}

func TestMockGateway_Refund_ErrorInjection(t *testing.T) {
	mock := NewMockGateway()
	mock.RefundErr = errors.New("simulated gateway outage")

	_, err := mock.Refund(context.Background(), "pay_1", 100, "INR")
	if err == nil {
		t.Fatal("expected injected error")
	}
	if len(mock.RefundCalls()) != 1 {
		t.Errorf("failed calls should still be recorded, got %d", len(mock.RefundCalls()))
	}
}
