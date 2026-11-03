package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type recordingEmailSender struct {
	sent []port.EmailMessage
	err  error
}

func (s *recordingEmailSender) Send(_ context.Context, msg port.EmailMessage) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, msg)
	return nil
}

func TestProcessDueRenewals_AlertsOnFailure(t *testing.T) {
	svc, subRepo, invoicer, sub := renewalFixture()
	subRepo.claimed = []*domain.Subscription{sub}
	invoicer.invErr = errors.New("tenant_id missing from context")
	sender := &recordingEmailSender{}
	svc.SetFailureAlerter(NewRenewalFailureAlerter(sender, "ops@example.com"))

	if _, err := svc.ProcessDueRenewals(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sender.sent) != 1 {
		t.Fatalf("sent %d alerts, want 1", len(sender.sent))
	}
	msg := sender.sent[0]
	if msg.To != "ops@example.com" || !strings.Contains(msg.Subject, "1 of 1") {
		t.Errorf("to=%q subject=%q", msg.To, msg.Subject)
	}
	if !strings.Contains(msg.TextBody, "tenant_id missing") || !strings.Contains(msg.TextBody, sub.ID.String()) {
		t.Errorf("body should name the error and subscription: %q", msg.TextBody)
	}
}

func TestProcessDueRenewals_AlertsWhenClaimFails(t *testing.T) {
	svc, subRepo, _, _ := renewalFixture()
	subRepo.claimErr = errors.New("connection refused")
	sender := &recordingEmailSender{}
	svc.SetFailureAlerter(NewRenewalFailureAlerter(sender, "ops@example.com"))

	if _, err := svc.ProcessDueRenewals(context.Background()); err == nil {
		t.Fatal("want the claim error returned")
	}
	if len(sender.sent) != 1 || !strings.Contains(sender.sent[0].Subject, "can't claim") {
		t.Fatalf("sent = %+v, want one claim alert", sender.sent)
	}
}

// A decline leaves the invoice open for dunning and the renewal succeeded, so
// it must not page anyone.
func TestProcessDueRenewals_NoAlertWhenRenewalsSucceed(t *testing.T) {
	svc, subRepo, _, sub := renewalFixture()
	subRepo.claimed = []*domain.Subscription{sub}
	svc.SetSavedMethodCharging(&fakeRenewalCharger{result: &port.PaymentResult{Success: false, ErrorCode: "card_declined"}},
		&fakeRenewalLookup{stripeID: "cus_1", methodID: "pm_1"}, &fakeRenewalSettler{})
	sender := &recordingEmailSender{}
	svc.SetFailureAlerter(NewRenewalFailureAlerter(sender, "ops@example.com"))

	if _, err := svc.ProcessDueRenewals(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sender.sent) != 0 {
		t.Fatalf("sent = %+v, want no alert", sender.sent)
	}
}

func TestRenewalFailureAlerter_Cooldown(t *testing.T) {
	sender := &recordingEmailSender{}
	a := NewRenewalFailureAlerter(sender, "ops@example.com")
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return now }
	fail := []error{errors.New("boom")}
	ctx := context.Background()

	a.RenewalsFailed(ctx, 1, 1, fail)
	now = now.Add(59 * time.Minute)
	a.RenewalsFailed(ctx, 1, 1, fail)
	if len(sender.sent) != 1 {
		t.Fatalf("sent %d inside the cooldown, want 1", len(sender.sent))
	}
	now = now.Add(time.Minute)
	a.RenewalsFailed(ctx, 1, 1, fail)
	if len(sender.sent) != 2 {
		t.Fatalf("sent %d after the cooldown, want 2", len(sender.sent))
	}
}

func TestRenewalFailureAlerter_FailedSendRetriesNextSweep(t *testing.T) {
	sender := &recordingEmailSender{err: errors.New("smtp down")}
	a := NewRenewalFailureAlerter(sender, "ops@example.com")
	fail := []error{errors.New("boom")}

	a.RenewalsFailed(context.Background(), 1, 1, fail)
	sender.err = nil
	a.RenewalsFailed(context.Background(), 1, 1, fail)

	if len(sender.sent) != 1 {
		t.Fatalf("sent %d, want the retry to go out", len(sender.sent))
	}
}

func TestRenewalFailureAlerter_QuotesAFewEscapedErrors(t *testing.T) {
	sender := &recordingEmailSender{}
	a := NewRenewalFailureAlerter(sender, "ops@example.com")
	var errs []error
	for range 5 {
		errs = append(errs, errors.New("<bad> "+uuid.NewString()))
	}

	a.RenewalsFailed(context.Background(), 5, 5, errs)

	msg := sender.sent[0]
	if got := strings.Count(msg.TextBody, "<bad>"); got != renewalAlertSampleErrors {
		t.Errorf("quoted %d errors, want %d", got, renewalAlertSampleErrors)
	}
	if strings.Contains(msg.HTMLBody, "<bad>") {
		t.Error("html body must escape error text")
	}
}
