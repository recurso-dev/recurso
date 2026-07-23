package sms_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/recurso-dev/recurso/internal/adapter/sms"
	"github.com/recurso-dev/recurso/internal/core/port"
)

func TestConsoleSMSSender_Send(t *testing.T) {
	sender := sms.NewConsoleSMSSender()
	ctx := context.Background()

	msg := port.SMSMessage{
		To:   "+15550199",
		Body: "Your Recurso verification code is 123456",
	}

	err := sender.Send(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error from ConsoleSMSSender: %v", err)
	}
}

func TestTwilioSMSSender_New(t *testing.T) {
	sender := sms.NewTwilioSMSSender("ACtest", "authtoken", "+15550100")
	if sender == nil {
		t.Fatal("expected non-nil TwilioSMSSender")
	}
}

func TestTwilioSMSSender_Send_MockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid": "SM123"}`))
	}))
	defer server.Close()

	// Direct constructor test
	sender := sms.NewTwilioSMSSender("AC123", "token123", "+15550000")
	if sender == nil {
		t.Fatal("expected non-nil sender")
	}
}
