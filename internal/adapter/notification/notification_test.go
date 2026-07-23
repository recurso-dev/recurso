package notification_test

import (
	"context"
	"testing"

	"github.com/recurso-dev/recurso/internal/adapter/notification"
)

func TestConsoleNotifier_SendEmail(t *testing.T) {
	notifier := notification.NewConsoleNotifier()
	ctx := context.Background()

	err := notifier.SendEmail(ctx, "user@example.com", "Test Subject", "Test Body Content")
	if err != nil {
		t.Fatalf("unexpected error from ConsoleNotifier: %v", err)
	}
}

func TestNewSMTPNotifier(t *testing.T) {
	notifier := notification.NewSMTPNotifier("smtp.example.com", "587", "user", "pass", "no-reply@example.com")
	if notifier == nil {
		t.Fatal("expected non-nil SMTPNotifier")
	}
}
