package service

import (
	"context"
	"strings"
	"testing"

	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type lastEmailSender struct{ msg port.EmailMessage }

func (s *lastEmailSender) Send(_ context.Context, m port.EmailMessage) error {
	s.msg = m
	return nil
}

// Every dunning level offers Pay Now, and only claims charge attempts when one
// could have happened.
func TestDunningEmails_OnlyClaimAttemptsThatHappened(t *testing.T) {
	claims := map[int]string{1: "attempted to charge", 2: "tried to process", 3: "Despite multiple attempts"}
	for level, claim := range claims {
		for _, attempted := range []bool{true, false} {
			sender := &lastEmailSender{}
			svc := NewNotificationService(sender, "https://api.test")
			err := svc.SendDunningEmail(context.Background(), level, email.DunningEmailData{
				CustomerEmail: "c@example.com", InvoiceNumber: "INV-1", Amount: "$99.00",
				PayNowURL: "https://app.test/checkout/1", ChargeAttempted: attempted,
			})
			if err != nil {
				t.Fatalf("level %d: %v", level, err)
			}
			if got := strings.Contains(sender.msg.HTMLBody, claim); got != attempted {
				t.Errorf("level %d attempted=%v: contains %q = %v", level, attempted, claim, got)
			}
			if !strings.Contains(sender.msg.HTMLBody, "https://app.test/checkout/1") {
				t.Errorf("level %d attempted=%v: no Pay Now link", level, attempted)
			}
		}
	}
}
