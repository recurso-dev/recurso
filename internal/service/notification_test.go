package service_test

import (
	"context"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

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
