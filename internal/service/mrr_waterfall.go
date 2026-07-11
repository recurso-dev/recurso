package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// MRRSnapshotStore persists and reads per-subscription MRR history.
type MRRSnapshotStore interface {
	UpsertSnapshots(ctx context.Context, snaps []domain.MRRSnapshot) error
	ResolveSnapshotDate(ctx context.Context, tenantID uuid.UUID, onOrBefore time.Time) (time.Time, bool, error)
	GetSnapshotsOn(ctx context.Context, tenantID uuid.UUID, date time.Time) ([]domain.MRRSnapshot, error)
	SubscriptionIDsSeenBefore(ctx context.Context, tenantID uuid.UUID, date time.Time) (map[uuid.UUID]bool, error)
}

// dayUTC reduces a timestamp to its calendar day in UTC, matching the DATE
// column so snapshot dates compare exactly.
func dayUTC(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// CaptureMRRSnapshot records every active subscription's monthly-normalized MRR
// for the tenant on the given date (idempotent per day). This is the history the
// waterfall diffs; run it daily. Returns the number of subscriptions captured.
func (s *AnalyticsService) CaptureMRRSnapshot(ctx context.Context, tenantID uuid.UUID, date time.Time) (int, error) {
	// The plan repo reads the tenant from the context, so scope it to THIS
	// tenant (a background sweep otherwise reads the wrong/empty tenant — the
	// tenant-context bug class).
	tctx := context.WithValue(ctx, domain.TenantIDKey, tenantID)

	subs, err := s.subRepo.GetActiveSubscriptions(tctx, tenantID)
	if err != nil {
		return 0, err
	}
	reporting := s.resolveReportingCurrency(ctx, tenantID)
	day := dayUTC(date)

	planCache := make(map[uuid.UUID]*domain.Plan)
	snaps := make([]domain.MRRSnapshot, 0, len(subs))
	for _, sub := range subs {
		plan, ok := planCache[sub.PlanID]
		if !ok {
			p, err := s.planRepo.GetByID(tctx, sub.PlanID)
			if err != nil || p == nil {
				continue
			}
			plan = p
			planCache[sub.PlanID] = plan
		}
		if len(plan.Prices) == 0 {
			continue
		}
		currency := plan.Prices[0].Currency
		if currency == "" {
			currency = reporting
		}
		cust := sub.CustomerID
		planID := sub.PlanID
		snaps = append(snaps, domain.MRRSnapshot{
			TenantID:       tenantID,
			SubscriptionID: sub.ID,
			SnapshotDate:   day,
			MRRAmount:      monthlyMinorUnits(plan.Prices[0].Amount, plan.IntervalUnit, plan.IntervalCount),
			Currency:       currency,
			CustomerID:     &cust,
			PlanID:         &planID,
		})
	}
	if err := s.snapshots.UpsertSnapshots(ctx, snaps); err != nil {
		return 0, err
	}
	return len(snaps), nil
}

// GetMRRWaterfall breaks the change in MRR between two dates into movement
// components (new/expansion/contraction/churned/reactivation), in the tenant's
// reporting currency. Period boundaries resolve to the nearest captured snapshot
// on or before each date. When no snapshot exists on or before the start,
// HasStartHistory is false and everything present at the end counts as New.
func (s *AnalyticsService) GetMRRWaterfall(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*domain.MRRWaterfall, error) {
	reporting := s.resolveReportingCurrency(ctx, tenantID)
	wf := &domain.MRRWaterfall{StartDate: start, EndDate: end, ReportingCurrency: reporting}

	endDate, hasEnd, err := s.snapshots.ResolveSnapshotDate(ctx, tenantID, end)
	if err != nil {
		return nil, err
	}
	if !hasEnd {
		return wf, nil // no history captured yet
	}
	startDate, hasStart, err := s.snapshots.ResolveSnapshotDate(ctx, tenantID, start)
	if err != nil {
		return nil, err
	}
	wf.HasStartHistory = hasStart

	endSnaps, err := s.snapshots.GetSnapshotsOn(ctx, tenantID, endDate)
	if err != nil {
		return nil, err
	}
	startMap := make(map[uuid.UUID]domain.MRRSnapshot)
	seenBefore := make(map[uuid.UUID]bool)
	if hasStart {
		startSnaps, err := s.snapshots.GetSnapshotsOn(ctx, tenantID, startDate)
		if err != nil {
			return nil, err
		}
		for _, sn := range startSnaps {
			startMap[sn.SubscriptionID] = sn
		}
		if seenBefore, err = s.snapshots.SubscriptionIDsSeenBefore(ctx, tenantID, startDate); err != nil {
			return nil, err
		}
	}

	normalizer := newFXNormalizer(s.fxProvider, s.fxFallback)
	conv := func(amount int64, currency string) (int64, bool) {
		if currency == "" {
			currency = reporting
		}
		c, _, err := normalizer.convert(ctx, amount, currency, reporting)
		if err != nil {
			return 0, false
		}
		return c, true
	}

	endMap := make(map[uuid.UUID]domain.MRRSnapshot, len(endSnaps))
	for _, sn := range endSnaps {
		endMap[sn.SubscriptionID] = sn
	}

	for _, sn := range startMap {
		if v, ok := conv(sn.MRRAmount, sn.Currency); ok {
			wf.StartingMRR += v
		}
	}
	for _, sn := range endMap {
		if v, ok := conv(sn.MRRAmount, sn.Currency); ok {
			wf.EndingMRR += v
		}
	}

	// Movement: subscriptions present at the end.
	for subID, es := range endMap {
		en, okE := conv(es.MRRAmount, es.Currency)
		if !okE {
			continue
		}
		if ss, inStart := startMap[subID]; inStart {
			st, okS := conv(ss.MRRAmount, ss.Currency)
			if !okS {
				continue
			}
			if d := en - st; d > 0 {
				wf.Expansion += d
			} else if d < 0 {
				wf.Contraction += -d
			}
		} else if seenBefore[subID] {
			wf.Reactivation += en
		} else {
			wf.New += en
		}
	}
	// Churned: present at the start, gone at the end.
	for subID, ss := range startMap {
		if _, inEnd := endMap[subID]; inEnd {
			continue
		}
		if st, ok := conv(ss.MRRAmount, ss.Currency); ok {
			wf.Churned += st
		}
	}

	return wf, nil
}
