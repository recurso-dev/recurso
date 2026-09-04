package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// cloudUsageReader reads the stored usage readings a preview is computed from.
type cloudUsageReader interface {
	ListByPeriod(ctx context.Context, periodStart time.Time) ([]*domain.CloudTenantUsage, error)
}

// cloudChargeRepo persists the dry-run previews.
type cloudChargeRepo interface {
	UpsertPreview(ctx context.Context, rows []*domain.CloudChargePreview) error
}

// CloudChargeService computes the Recurso Cloud DRY-RUN charge for each tenant:
// it normalizes their per-currency usage to the reporting currency (USD) via
// FX, applies the published pricing, and stores what they WOULD be charged.
//
// It is money-free — a preview only. No plan, subscription, invoice, or ledger
// leg. It exists so the pricing + quota can be reviewed before real charging is
// enabled in a later increment.
type CloudChargeService struct {
	usage  cloudUsageReader
	repo   cloudChargeRepo
	logger *slog.Logger

	// FX (optional): converts each tenant's per-currency totals into the
	// reporting currency before the pricing thresholds apply. Wired via SetFX,
	// mirroring the analytics/org services.
	fxProvider        port.ExchangeRateProvider
	fxFallback        port.ExchangeRateProvider
	reportingCurrency string
}

func NewCloudChargeService(usage cloudUsageReader, repo cloudChargeRepo, logger *slog.Logger) *CloudChargeService {
	if logger == nil {
		logger = slog.Default()
	}
	return &CloudChargeService{usage: usage, repo: repo, logger: logger, reportingCurrency: "USD"}
}

// SetFX wires the FX provider (+ fallback) used to normalize each tenant's
// per-currency usage to the reporting currency the pricing is denominated in.
func (s *CloudChargeService) SetFX(provider, fallback port.ExchangeRateProvider, reportingCurrency string) {
	s.fxProvider = provider
	s.fxFallback = fallback
	if reportingCurrency != "" {
		s.reportingCurrency = reportingCurrency
	}
}

// PreviewPeriod computes and stores one dry-run charge per tenant for the period
// whose readings start at periodStart. Idempotent (upsert). Returns the previews
// it wrote. NOTHING is billed.
func (s *CloudChargeService) PreviewPeriod(ctx context.Context, periodStart, periodEnd time.Time) ([]*domain.CloudChargePreview, error) {
	readings, err := s.usage.ListByPeriod(ctx, periodStart)
	if err != nil {
		return nil, fmt.Errorf("read cloud usage: %w", err)
	}

	// Sum each tenant's usage into the reporting currency.
	type totals struct {
		tracked, collected int64
	}
	perTenant := map[uuid.UUID]*totals{}
	order := []uuid.UUID{}
	var normalizer *fxNormalizer
	if s.fxProvider != nil {
		normalizer = newFXNormalizer(s.fxProvider, s.fxFallback)
	}
	for _, u := range readings {
		t, ok := perTenant[u.TenantID]
		if !ok {
			t = &totals{}
			perTenant[u.TenantID] = t
			order = append(order, u.TenantID)
		}
		tracked, collected := u.TrackedRevenueMinor, u.CollectedVolumeMinor
		if u.Currency != s.reportingCurrency {
			if normalizer == nil {
				// No FX available and a foreign-currency reading — skip it rather
				// than mis-price by treating, say, INR minor units as USD cents.
				s.logger.WarnContext(ctx, "cloud charge preview: no FX to normalize a reading; skipping",
					"tenant_id", u.TenantID, "currency", u.Currency)
				continue
			}
			if tracked, _, err = normalizer.convert(ctx, tracked, u.Currency, s.reportingCurrency); err != nil {
				s.logger.WarnContext(ctx, "cloud charge preview: FX convert (tracked) failed; skipping reading",
					"tenant_id", u.TenantID, "currency", u.Currency, "error", err)
				continue
			}
			if collected, _, err = normalizer.convert(ctx, collected, u.Currency, s.reportingCurrency); err != nil {
				s.logger.WarnContext(ctx, "cloud charge preview: FX convert (collected) failed; skipping reading",
					"tenant_id", u.TenantID, "currency", u.Currency, "error", err)
				continue
			}
		}
		t.tracked += tracked
		t.collected += collected
	}

	now := time.Now().UTC()
	previews := make([]*domain.CloudChargePreview, 0, len(order))
	for _, tid := range order {
		t := perTenant[tid]
		charge, reason := domain.ComputeCloudCharge(t.tracked, t.collected)
		previews = append(previews, &domain.CloudChargePreview{
			ID:                   uuid.New(),
			PeriodStart:          periodStart,
			PeriodEnd:            periodEnd,
			TenantID:             tid,
			Currency:             s.reportingCurrency,
			TrackedRevenueMinor:  t.tracked,
			CollectedVolumeMinor: t.collected,
			WouldChargeMinor:     charge,
			Reason:               reason,
			ComputedAt:           now,
		})
	}

	if err := s.repo.UpsertPreview(ctx, previews); err != nil {
		return nil, fmt.Errorf("store cloud charge preview: %w", err)
	}

	var totalCharge int64
	for _, p := range previews {
		totalCharge += p.WouldChargeMinor
	}
	s.logger.InfoContext(ctx, "Recurso Cloud dry-run charge preview computed (no money moved)",
		"tenants", len(previews), "would_charge_total_minor", totalCharge, "currency", s.reportingCurrency)
	return previews, nil
}

// PreviewCurrentPeriod previews the current calendar month (UTC).
func (s *CloudChargeService) PreviewCurrentPeriod(ctx context.Context) error {
	start, end := MonthBounds(time.Now().UTC())
	_, err := s.PreviewPeriod(ctx, start, end)
	return err
}
