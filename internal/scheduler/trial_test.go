package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/adapter/db"
	"github.com/swapnull-in/recur-so/internal/adapter/email"
	"github.com/swapnull-in/recur-so/internal/adapter/memory"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// --- mocks ---

type mockTrialRepo struct {
	expired       []*domain.Subscription
	ending        []db.TrialEndingNotice
	reminderMarks []uuid.UUID
}

func (m *mockTrialRepo) GetExpiredTrials(ctx context.Context) ([]*domain.Subscription, error) {
	return m.expired, nil
}

func (m *mockTrialRepo) GetTrialsEndingWithin(ctx context.Context, within time.Duration) ([]db.TrialEndingNotice, error) {
	return m.ending, nil
}

func (m *mockTrialRepo) MarkTrialReminderSent(ctx context.Context, id uuid.UUID) error {
	m.reminderMarks = append(m.reminderMarks, id)
	return nil
}

type mockTrialConverter struct {
	converted []uuid.UUID
}

func (m *mockTrialConverter) ConvertTrialToActive(ctx context.Context, sub *domain.Subscription) (*domain.Invoice, error) {
	m.converted = append(m.converted, sub.ID)
	return &domain.Invoice{ID: uuid.New(), SubscriptionID: &sub.ID}, nil
}

type mockTrialNotifier struct {
	sent []email.TrialEndingEmailData
}

func (m *mockTrialNotifier) SendTrialEndingReminder(ctx context.Context, data email.TrialEndingEmailData) error {
	m.sent = append(m.sent, data)
	return nil
}

// --- tests ---

func TestTrialScheduler_ConvertsExpiredTrials(t *testing.T) {
	sub1 := &domain.Subscription{ID: uuid.New(), Status: domain.SubscriptionStatusTrialing}
	sub2 := &domain.Subscription{ID: uuid.New(), Status: domain.SubscriptionStatusTrialing}

	repo := &mockTrialRepo{expired: []*domain.Subscription{sub1, sub2}}
	conv := &mockTrialConverter{}
	notif := &mockTrialNotifier{}

	s := NewTrialScheduler(repo, conv, notif, memory.NewNoOpLocker(), "https://portal.test")
	s.processTrials()

	if len(conv.converted) != 2 {
		t.Fatalf("expected 2 conversions, got %d", len(conv.converted))
	}
	if conv.converted[0] != sub1.ID || conv.converted[1] != sub2.ID {
		t.Errorf("converted wrong subscriptions: %v", conv.converted)
	}
}

func TestTrialScheduler_SendsRemindersAndMarksSent(t *testing.T) {
	trialEnd := time.Now().Add(48 * time.Hour)
	notice := db.TrialEndingNotice{
		SubscriptionID: uuid.New(),
		CustomerName:   "Ada",
		CustomerEmail:  "ada@example.com",
		PlanName:       "Pro",
		Amount:         100000,
		Currency:       "INR",
		TrialEnd:       trialEnd,
	}

	repo := &mockTrialRepo{ending: []db.TrialEndingNotice{notice}}
	conv := &mockTrialConverter{}
	notif := &mockTrialNotifier{}

	s := NewTrialScheduler(repo, conv, notif, memory.NewNoOpLocker(), "https://portal.test")
	s.processTrials()

	if len(notif.sent) != 1 {
		t.Fatalf("expected 1 reminder sent, got %d", len(notif.sent))
	}
	got := notif.sent[0]
	if got.CustomerEmail != "ada@example.com" {
		t.Errorf("reminder email = %q, want ada@example.com", got.CustomerEmail)
	}
	if got.PlanName != "Pro" {
		t.Errorf("reminder plan = %q, want Pro", got.PlanName)
	}
	if got.Amount != "₹1000.00" {
		t.Errorf("reminder amount = %q, want ₹1000.00", got.Amount)
	}
	if got.PortalURL != "https://portal.test/portal" {
		t.Errorf("reminder portal URL = %q", got.PortalURL)
	}

	// The reminder must be marked sent so it is not delivered twice.
	if len(repo.reminderMarks) != 1 || repo.reminderMarks[0] != notice.SubscriptionID {
		t.Errorf("expected reminder marked sent for %s, got %v", notice.SubscriptionID, repo.reminderMarks)
	}
}
