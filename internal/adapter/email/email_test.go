package email_test

import (
	"context"
	"errors"
	"testing"

	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type mockInnerSender struct {
	shouldErr bool
	sentMsg   port.EmailMessage
}

func (m *mockInnerSender) Send(_ context.Context, msg port.EmailMessage) error {
	if m.shouldErr {
		return errors.New("send failed")
	}
	m.sentMsg = msg
	return nil
}

func TestConsoleSender_Send(t *testing.T) {
	sender := email.NewConsoleSender()
	ctx := context.Background()

	msg := port.EmailMessage{
		To:       "user@example.com",
		Subject:  "Test Subject",
		HTMLBody: "<h1>Hello</h1>",
	}

	err := sender.Send(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error from ConsoleSender: %v", err)
	}
}

func TestLoggingSender_Success(t *testing.T) {
	inner := &mockInnerSender{}
	loggingSender := email.NewLoggingSender(inner)
	ctx := context.Background()

	msg := port.EmailMessage{
		To:       "test@recurso.dev",
		Subject:  "Invoice Created",
		HTMLBody: "<p>Your invoice #1001 is ready</p>",
	}

	err := loggingSender.Send(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error from LoggingSender: %v", err)
	}

	if inner.sentMsg.To != msg.To {
		t.Errorf("expected inner to receive To %s, got %s", msg.To, inner.sentMsg.To)
	}
}

func TestLoggingSender_Error(t *testing.T) {
	inner := &mockInnerSender{shouldErr: true}
	loggingSender := email.NewLoggingSender(inner)
	ctx := context.Background()

	msg := port.EmailMessage{
		To:       "fail@recurso.dev",
		Subject:  "Dunning Notice",
		HTMLBody: "<p>Payment failed</p>",
	}

	err := loggingSender.Send(ctx, msg)
	if err == nil {
		t.Fatal("expected error from LoggingSender when inner fails")
	}
}
