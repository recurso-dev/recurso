package service_test

import (
	"context"
	"strings"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

func TestNotificationService_NewSignupAlert(t *testing.T) {
	sender := &mockEmailSender{}
	svc := service.NewNotificationService(sender, "https://api.recurso.dev")

	if err := svc.SendNewSignupAlert(context.Background(), "founder@recurso.dev", "Acme Inc", "owner@acme.com", "US"); err != nil {
		t.Fatalf("SendNewSignupAlert: %v", err)
	}
	if sender.sentMsg.To != "founder@recurso.dev" {
		t.Errorf("alert To = %q, want founder@recurso.dev", sender.sentMsg.To)
	}
	if sender.sentMsg.Subject != "New Recurso signup: Acme Inc" {
		t.Errorf("subject = %q, want 'New Recurso signup: Acme Inc'", sender.sentMsg.Subject)
	}
	if !strings.Contains(sender.sentMsg.TextBody, "owner@acme.com") || !strings.Contains(sender.sentMsg.TextBody, "US") {
		t.Errorf("body missing owner/country: %q", sender.sentMsg.TextBody)
	}
}

type mockEmailSender struct {
	sentMsg port.EmailMessage
}

func (m *mockEmailSender) Send(_ context.Context, msg port.EmailMessage) error {
	m.sentMsg = msg
	return nil
}

func TestNotificationService_AppBaseURL(t *testing.T) {
	sender := &mockEmailSender{}
	svc := service.NewNotificationService(sender, "https://api.recurso.dev")

	svc.SetAppBaseURL("https://app.recurso.dev")

	err := svc.SendInvoiceCreated(context.Background(), service.InvoiceData{
		CustomerName:  "Alice",
		CustomerEmail: "alice@example.com",
		InvoiceNumber: "INV-1001",
		InvoiceID:     "inv_123",
		Amount:        "$100.00",
		DueDate:       "2026-08-01",
	})

	if err != nil {
		t.Fatalf("unexpected error sending invoice created email: %v", err)
	}

	if sender.sentMsg.To != "alice@example.com" {
		t.Errorf("expected email to alice@example.com, got %s", sender.sentMsg.To)
	}
}
