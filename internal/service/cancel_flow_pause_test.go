package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// A retention "pause N months" offer must pause the subscription AND schedule an
// auto-resume N months out (issue #111) — previously PauseMonths was logged but
// ignored, pausing indefinitely.
func TestCancelFlow_PauseOfferSchedulesResume(t *testing.T) {
	tenant := uuid.New()
	sub := activeSub(tenant)
	svc, repo := newLifecycleSvc(sub)
	cf := &CancelFlowService{subscriptionService: svc, logger: slog.Default()}

	session := &domain.CancelFlowSession{TenantID: tenant, SubscriptionID: sub.ID}
	offer := domain.RetentionOffer{Type: domain.OfferTypePause, PauseMonths: 3}

	if err := cf.applyOffer(context.Background(), session, offer); err != nil {
		t.Fatalf("applyOffer(pause): %v", err)
	}

	got := repo.subs[sub.ID]
	if got.Status != domain.SubscriptionStatusPaused {
		t.Fatalf("status = %s, want paused", got.Status)
	}
	if got.ResumeAt == nil {
		t.Fatal("PauseMonths ignored: ResumeAt is nil (would pause indefinitely)")
	}
	want := time.Now().UTC().AddDate(0, 3, 0)
	if diff := got.ResumeAt.Sub(want); diff > time.Minute || diff < -time.Minute {
		t.Errorf("ResumeAt = %v, want ~%v (now + 3 months)", got.ResumeAt, want)
	}
}

// A pause offer with no PauseMonths stays open-ended (nil ResumeAt) — resume is
// manual, matching the pre-#111 behaviour.
func TestCancelFlow_PauseOfferNoMonthsIsIndefinite(t *testing.T) {
	tenant := uuid.New()
	sub := activeSub(tenant)
	svc, repo := newLifecycleSvc(sub)
	cf := &CancelFlowService{subscriptionService: svc, logger: slog.Default()}

	session := &domain.CancelFlowSession{TenantID: tenant, SubscriptionID: sub.ID}
	if err := cf.applyOffer(context.Background(), session, domain.RetentionOffer{Type: domain.OfferTypePause}); err != nil {
		t.Fatalf("applyOffer(pause, no months): %v", err)
	}

	if got := repo.subs[sub.ID]; got.Status != domain.SubscriptionStatusPaused || got.ResumeAt != nil {
		t.Fatalf("status/resume = %s/%v, want paused/nil (indefinite)", got.Status, got.ResumeAt)
	}
}
