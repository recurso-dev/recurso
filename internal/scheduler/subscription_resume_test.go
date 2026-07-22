package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/memory"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type fakeResumeClaimer struct {
	batches [][]*domain.Subscription // one slice returned per call, in order
	calls   int
	lease   time.Duration
	limit   int
}

func (f *fakeResumeClaimer) ClaimDueForResume(ctx context.Context, lease time.Duration, limit int) ([]*domain.Subscription, error) {
	f.lease, f.limit = lease, limit
	if f.calls >= len(f.batches) {
		f.calls++
		return nil, nil
	}
	b := f.batches[f.calls]
	f.calls++
	return b, nil
}

type fakeResumer struct {
	resumed []uuid.UUID
	tenants []uuid.UUID
	failIDs map[uuid.UUID]bool
}

func (f *fakeResumer) ResumeSubscription(ctx context.Context, tenantID, subID uuid.UUID) (*domain.Subscription, error) {
	if f.failIDs[subID] {
		return nil, errors.New("boom")
	}
	f.resumed = append(f.resumed, subID)
	f.tenants = append(f.tenants, tenantID)
	return &domain.Subscription{ID: subID, TenantID: tenantID}, nil
}

func TestSubscriptionResume_ResumesClaimedSubs(t *testing.T) {
	t1, t2 := uuid.New(), uuid.New()
	s1, s2 := uuid.New(), uuid.New()
	claimer := &fakeResumeClaimer{batches: [][]*domain.Subscription{{
		{ID: s1, TenantID: t1}, {ID: s2, TenantID: t2},
	}}}
	resumer := &fakeResumer{}
	sched := NewSubscriptionResumeScheduler(claimer, resumer, memory.NewNoOpLocker())

	sched.runResumes()

	if len(resumer.resumed) != 2 {
		t.Fatalf("resumed %d subs, want 2", len(resumer.resumed))
	}
	// Each resume must carry the subscription's own tenant (scheduler ctx has none).
	if resumer.tenants[0] != t1 || resumer.tenants[1] != t2 {
		t.Errorf("resume tenants = %v, want [%v %v]", resumer.tenants, t1, t2)
	}
	// The claim must use the configured lease/limit.
	if claimer.lease != resumeClaimWindow || claimer.limit != resumeBatchLimit {
		t.Errorf("claim args = (%v, %d), want (%v, %d)", claimer.lease, claimer.limit, resumeClaimWindow, resumeBatchLimit)
	}
}

func TestSubscriptionResume_EmptyClaimIsNoOp(t *testing.T) {
	claimer := &fakeResumeClaimer{} // returns nil
	resumer := &fakeResumer{}
	sched := NewSubscriptionResumeScheduler(claimer, resumer, memory.NewNoOpLocker())

	sched.runResumes()

	if len(resumer.resumed) != 0 {
		t.Fatalf("no subs claimed, but %d resumed", len(resumer.resumed))
	}
}

func TestSubscriptionResume_OneFailureDoesNotStopBatch(t *testing.T) {
	good1, bad, good2 := uuid.New(), uuid.New(), uuid.New()
	tenant := uuid.New()
	claimer := &fakeResumeClaimer{batches: [][]*domain.Subscription{{
		{ID: good1, TenantID: tenant}, {ID: bad, TenantID: tenant}, {ID: good2, TenantID: tenant},
	}}}
	resumer := &fakeResumer{failIDs: map[uuid.UUID]bool{bad: true}}
	sched := NewSubscriptionResumeScheduler(claimer, resumer, memory.NewNoOpLocker())

	sched.runResumes()

	// The failing one is skipped; the other two still resume.
	if len(resumer.resumed) != 2 {
		t.Fatalf("resumed %d, want 2 (the good ones)", len(resumer.resumed))
	}
	for _, id := range resumer.resumed {
		if id == bad {
			t.Error("the failing subscription must not be recorded as resumed")
		}
	}
}
