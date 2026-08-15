package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// readOnlyRevRecRepo embeds RevRecRepository (nil) and overrides ONLY the two
// read methods the forfeit sum uses. Because every other method is the nil
// embedded interface, any accidental mutating call (CancelEvent,
// MarkScheduleCanceled, …) panics — so a green test is itself proof that
// RemainingDeferredForSubscription is read-only.
type readOnlyRevRecRepo struct {
	RevRecRepository
	schedules []*domain.RevenueSchedule
	pending   map[uuid.UUID][]*domain.RecognitionEvent
}

func (r *readOnlyRevRecRepo) GetActiveSchedulesBySubscription(_ context.Context, _, _ uuid.UUID) ([]*domain.RevenueSchedule, error) {
	return r.schedules, nil
}
func (r *readOnlyRevRecRepo) GetPendingEventsBySchedule(_ context.Context, scheduleID uuid.UUID) ([]*domain.RecognitionEvent, error) {
	return r.pending[scheduleID], nil
}

func TestRemainingDeferredForSubscription_SumsPendingReadOnly(t *testing.T) {
	sched1, sched2 := uuid.New(), uuid.New()
	repo := &readOnlyRevRecRepo{
		schedules: []*domain.RevenueSchedule{{ID: sched1}, {ID: sched2}},
		pending: map[uuid.UUID][]*domain.RecognitionEvent{
			sched1: {{Amount: 3000}, {Amount: 2000}},
			sched2: {{Amount: 500}},
		},
	}
	svc := NewRevRecService(repo, nil, nil)

	total, err := svc.RemainingDeferredForSubscription(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("RemainingDeferredForSubscription: %v", err)
	}
	// 3000 + 2000 + 500 = 5500 collected-but-unearned → the breakage an immediate
	// cancel would forfeit. No CancelEvent/MarkScheduleCanceled was invoked (nil
	// embedded interface would have panicked).
	if total != 5500 {
		t.Fatalf("forfeit sum = %d, want 5500", total)
	}
}

func TestRemainingDeferredForSubscription_NoSchedulesIsZero(t *testing.T) {
	repo := &readOnlyRevRecRepo{schedules: nil}
	svc := NewRevRecService(repo, nil, nil)
	total, err := svc.RemainingDeferredForSubscription(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("RemainingDeferredForSubscription: %v", err)
	}
	if total != 0 {
		t.Fatalf("cash-basis (no schedules) forfeit = %d, want 0", total)
	}
}
