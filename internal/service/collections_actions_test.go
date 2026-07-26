package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// stubActionRepo records calls and returns scripted eligibility/results.
type stubActionRepo struct {
	elig          domain.InvoiceRetryEligibility
	eligErr       error
	requeueOK     bool
	requeueCalled bool
	pausedOK      bool
	pausedSet     *bool
	uncollOK      bool
	uncollCalled  bool
}

func (s *stubActionRepo) GetRetryEligibility(_ context.Context, _, _ uuid.UUID) (domain.InvoiceRetryEligibility, error) {
	return s.elig, s.eligErr
}
func (s *stubActionRepo) RequeueForRetry(_ context.Context, _, _ uuid.UUID) (bool, error) {
	s.requeueCalled = true
	return s.requeueOK, nil
}
func (s *stubActionRepo) SetDunningPaused(_ context.Context, _, _ uuid.UUID, paused bool) (bool, error) {
	s.pausedSet = &paused
	return s.pausedOK, nil
}
func (s *stubActionRepo) MarkUncollectibleScoped(_ context.Context, _, _ uuid.UUID) (bool, error) {
	s.uncollCalled = true
	return s.uncollOK, nil
}

type stubInFlight struct{ inFlight bool }

func (s *stubInFlight) HasInFlightForInvoice(_ context.Context, _ uuid.UUID) (bool, error) {
	return s.inFlight, nil
}

func pastDueEligible() domain.InvoiceRetryEligibility {
	return domain.InvoiceRetryEligibility{Found: true, Status: "past_due"}
}

func TestRetryNow_HappyPath(t *testing.T) {
	repo := &stubActionRepo{elig: pastDueEligible(), requeueOK: true}
	svc := NewCollectionsActionService(repo)
	svc.SetInFlightChecker(&stubInFlight{inFlight: false})

	if err := svc.RetryNow(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("RetryNow: %v", err)
	}
	if !repo.requeueCalled {
		t.Error("expected RequeueForRetry to be called")
	}
}

func TestRetryNow_Guards(t *testing.T) {
	cases := []struct {
		name     string
		elig     domain.InvoiceRetryEligibility
		inFlight bool
		want     error
	}{
		{"not found", domain.InvoiceRetryEligibility{Found: false}, false, ErrCollectionInvoiceNotFound},
		{"not past due", domain.InvoiceRetryEligibility{Found: true, Status: "paid"}, false, ErrRetryNotPastDue},
		{"paused", domain.InvoiceRetryEligibility{Found: true, Status: "past_due", Paused: true}, false, ErrRetryPaused},
		{"mandate", domain.InvoiceRetryEligibility{Found: true, Status: "past_due", IsMandate: true}, false, ErrRetryMandate},
		{"in flight", pastDueEligible(), true, ErrRetryInFlight},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubActionRepo{elig: tc.elig, requeueOK: true}
			svc := NewCollectionsActionService(repo)
			svc.SetInFlightChecker(&stubInFlight{inFlight: tc.inFlight})

			err := svc.RetryNow(context.Background(), uuid.New(), uuid.New())
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if repo.requeueCalled {
				t.Error("a guard must reject before requeuing (no charge scheduled)")
			}
		})
	}
}

// Eligibility passes but the atomic requeue affects no row (state changed under
// us) → reported as not-retryable, not a false success.
func TestRetryNow_LostRace(t *testing.T) {
	repo := &stubActionRepo{elig: pastDueEligible(), requeueOK: false}
	svc := NewCollectionsActionService(repo)
	if err := svc.RetryNow(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, ErrRetryNotPastDue) {
		t.Fatalf("got %v, want ErrRetryNotPastDue", err)
	}
}

// With no in-flight checker wired, retry-now still works (nil-safe) — it just
// skips the ACH guard.
func TestRetryNow_NilInFlightChecker(t *testing.T) {
	repo := &stubActionRepo{elig: pastDueEligible(), requeueOK: true}
	svc := NewCollectionsActionService(repo)
	if err := svc.RetryNow(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("RetryNow (nil checker): %v", err)
	}
}

func TestSetPaused(t *testing.T) {
	repo := &stubActionRepo{pausedOK: true}
	svc := NewCollectionsActionService(repo)
	if err := svc.SetPaused(context.Background(), uuid.New(), uuid.New(), true); err != nil {
		t.Fatalf("SetPaused: %v", err)
	}
	if repo.pausedSet == nil || *repo.pausedSet != true {
		t.Error("expected paused=true forwarded to repo")
	}

	repo2 := &stubActionRepo{pausedOK: false}
	svc2 := NewCollectionsActionService(repo2)
	if err := svc2.SetPaused(context.Background(), uuid.New(), uuid.New(), true); !errors.Is(err, ErrCollectionInvoiceNotFound) {
		t.Fatalf("got %v, want ErrCollectionInvoiceNotFound", err)
	}
}

func TestMarkUncollectible(t *testing.T) {
	repo := &stubActionRepo{uncollOK: true}
	svc := NewCollectionsActionService(repo)
	if err := svc.MarkUncollectible(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("MarkUncollectible: %v", err)
	}
	if !repo.uncollCalled {
		t.Error("expected MarkUncollectibleScoped called")
	}

	repo2 := &stubActionRepo{uncollOK: false}
	svc2 := NewCollectionsActionService(repo2)
	if err := svc2.MarkUncollectible(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, ErrCollectionInvoiceNotFound) {
		t.Fatalf("got %v, want ErrCollectionInvoiceNotFound", err)
	}
}
