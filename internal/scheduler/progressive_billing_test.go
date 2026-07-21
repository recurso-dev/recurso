package scheduler

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type fakeSweepRepo struct {
	ids []uuid.UUID
	err error
}

func (f *fakeSweepRepo) ListActiveProgressiveSubscriptionIDs(context.Context) ([]uuid.UUID, error) {
	return f.ids, f.err
}

type fakeBiller struct {
	called  []uuid.UUID
	billFor map[uuid.UUID]bool // subs that produce an invoice
	failFor map[uuid.UUID]bool // subs whose billing errors
}

func (f *fakeBiller) GenerateProgressiveInvoiceForSub(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
	f.called = append(f.called, id)
	if f.failFor[id] {
		return nil, errors.New("boom")
	}
	if f.billFor[id] {
		return &domain.Invoice{ID: uuid.New(), AmountDue: 5000}, nil
	}
	return nil, nil // under threshold: nothing billed
}

// TestProgressiveSweep_BillsEveryCandidate proves the sweep asks the biller for
// each active progressive subscription, and that a per-subscription error or a
// "nothing due" result doesn't stop the others.
func TestProgressiveSweep_BillsEveryCandidate(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	repo := &fakeSweepRepo{ids: []uuid.UUID{a, b, c}}
	biller := &fakeBiller{
		billFor: map[uuid.UUID]bool{a: true}, // a crosses the threshold
		failFor: map[uuid.UUID]bool{b: true}, // b errors
		// c returns nothing due
	}
	s := NewProgressiveBillingScheduler(repo, biller, memory.NewNoOpLocker(), 0)

	s.run()

	if len(biller.called) != 3 {
		t.Fatalf("want all 3 subscriptions swept, got %d: %v", len(biller.called), biller.called)
	}
	seen := map[uuid.UUID]bool{}
	for _, id := range biller.called {
		seen[id] = true
	}
	for _, id := range []uuid.UUID{a, b, c} {
		if !seen[id] {
			t.Fatalf("subscription %s was not swept", id)
		}
	}
}

// TestProgressiveSweep_DefaultInterval falls back to the hourly default when
// given a non-positive interval.
func TestProgressiveSweep_DefaultInterval(t *testing.T) {
	s := NewProgressiveBillingScheduler(&fakeSweepRepo{}, &fakeBiller{}, memory.NewNoOpLocker(), 0)
	if s.interval != DefaultProgressiveSweepInterval {
		t.Fatalf("want default %s, got %s", DefaultProgressiveSweepInterval, s.interval)
	}
}

// TestProgressiveSweep_ListErrorIsHandled: a repo error aborts the run without
// panicking and without calling the biller.
func TestProgressiveSweep_ListErrorIsHandled(t *testing.T) {
	biller := &fakeBiller{}
	s := NewProgressiveBillingScheduler(&fakeSweepRepo{err: errors.New("db down")}, biller, memory.NewNoOpLocker(), 0)
	s.run()
	if len(biller.called) != 0 {
		t.Fatalf("biller should not be called when listing fails, got %d calls", len(biller.called))
	}
}
