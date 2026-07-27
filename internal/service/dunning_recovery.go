package service

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// RecoveredPaymentRepository persists and aggregates dunning recovery records.
type RecoveredPaymentRepository interface {
	// Insert writes a recovery record. Implementations must be idempotent on
	// invoice_id (ON CONFLICT DO NOTHING) so the multiple payment-success
	// paths can all call it safely.
	Insert(ctx context.Context, rec *domain.RecoveredPayment) error
	GetRecoveryTotals(ctx context.Context, tenantID uuid.UUID) (*domain.RecoveryTotals, error)
	GetMonthlyRecoveries(ctx context.Context, tenantID uuid.UUID, months int) ([]domain.RecoveryMonthBucket, error)
	// CountRecoveredSince counts recoveries in a trailing window — the
	// recovered side of the windowed recovery-rate cohort.
	CountRecoveredSince(ctx context.Context, tenantID uuid.UUID, since time.Time) (int, error)
}

// CampaignExecutionLookup resolves the dunning campaign (if any) driving an
// invoice. Satisfied by port.DunningCampaignRepository.
type CampaignExecutionLookup interface {
	GetExecutionByInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.DunningCampaignExecution, error)
}

// PaymentRecoveryRecorder is the hook the payment-success paths call after an
// invoice is marked paid. Implemented by DunningRecoveryService.
type PaymentRecoveryRecorder interface {
	RecordIfRecovered(ctx context.Context, inv *domain.Invoice) bool
}

// DunningRecoverySummary is the API response shape for recovered-revenue analytics.
type DunningRecoverySummary struct {
	// ReportingCurrency is the tenant's reporting currency; ReportingTotal is
	// the recovered total converted into it (via FX), so the headline never
	// gets hijacked by whichever currency happens to have the largest raw
	// minor-unit amount. RecoveredAmountTotal keeps the raw per-currency
	// breakdown for transparency.
	ReportingCurrency    string                       `json:"reporting_currency"`
	ReportingTotal       int64                        `json:"reporting_total"`
	RecoveredAmountTotal map[string]int64             `json:"recovered_amount_total"`
	RecoveredCount       int                          `json:"recovered_count"`
	AvgAttempts          float64                      `json:"avg_attempts"`
	AvgDaysToRecover     float64                      `json:"avg_days_to_recover"`
	Monthly              []domain.RecoveryMonthBucket `json:"monthly"`
}

// DunningRecoveryService measures what the retry/dunning engine actually
// recovers: it records an invoice as RECOVERED when it transitions to paid
// after at least one failed payment attempt.
type DunningRecoveryService struct {
	repo           RecoveredPaymentRepository
	campaignLookup CampaignExecutionLookup
	strategy       string
	logger         *slog.Logger
	now            func() time.Time

	fxProvider        port.ExchangeRateProvider
	fxFallback        port.ExchangeRateProvider
	tenantLookup      TenantLookup
	reportingCurrency string // env-level default when the tenant has no base currency

	collectionsAgg collectionsAggregator // nil-safe; powers the funnel + failure breakdown (Inc 2)
}

// collectionsAggregator is the invoice-side aggregation the recovery funnel and
// failure breakdown need (Collections Intelligence Inc 2). *db.InvoiceRepository
// satisfies it.
type collectionsAggregator interface {
	GetCollectionsAtRisk(ctx context.Context, tenantID uuid.UUID) ([]domain.CollectionsAtRiskRow, error)
	GetCollectionsFailureBreakdown(ctx context.Context, tenantID uuid.UUID) ([]domain.CollectionsFailureRow, error)
	// CountUncollectibleSince counts write-offs in a trailing window — the
	// written-off side of the windowed recovery-rate cohort.
	CountUncollectibleSince(ctx context.Context, tenantID uuid.UUID, since time.Time) (int, error)
}

// NewDunningRecoveryService builds the service. strategy is the tenant-wide
// dunning strategy in effect (env DUNNING_STRATEGY); it defaults to the smart
// retry engine's default when empty. A campaign execution on the invoice
// overrides it with "campaign".
func NewDunningRecoveryService(repo RecoveredPaymentRepository, strategy string) *DunningRecoveryService {
	if strategy == "" {
		strategy = string(StrategyEpsilonGreedy)
	}
	return &DunningRecoveryService{
		repo:              repo,
		strategy:          strategy,
		logger:            slog.Default().With("service", "dunning_recovery"),
		now:               time.Now,
		reportingCurrency: "USD",
	}
}

// SetCampaignLookup injects the campaign execution lookup after construction.
func (s *DunningRecoveryService) SetCampaignLookup(lookup CampaignExecutionLookup) {
	s.campaignLookup = lookup
}

// SetFX wires the FX provider (with optional static fallback) and default
// reporting currency used to normalize recovered revenue into one currency.
func (s *DunningRecoveryService) SetFX(provider, fallback port.ExchangeRateProvider, reportingCurrency string) {
	s.fxProvider = provider
	s.fxFallback = fallback
	if reportingCurrency != "" {
		s.reportingCurrency = reportingCurrency
	}
}

// SetTenantLookup enables per-tenant reporting currency (tenant.BaseCurrency).
func (s *DunningRecoveryService) SetTenantLookup(l TenantLookup) {
	s.tenantLookup = l
}

// SetCollectionsAggregator wires the invoice-side aggregation used by the
// collections funnel + failure breakdown (Inc 2). Nil-safe: without it those
// endpoints return empty results rather than failing.
func (s *DunningRecoveryService) SetCollectionsAggregator(a collectionsAggregator) {
	s.collectionsAgg = a
}

func (s *DunningRecoveryService) resolveReportingCurrency(ctx context.Context, tenantID uuid.UUID) string {
	if s.tenantLookup != nil && tenantID != uuid.Nil {
		if tenant, err := s.tenantLookup.GetByID(ctx, tenantID); err == nil && tenant != nil && tenant.BaseCurrency != "" {
			return tenant.BaseCurrency
		}
	}
	return s.reportingCurrency
}

// RecordIfRecovered inspects the invoice state at payment time and records a
// recovery if the invoice needed at least one retry or an active dunning
// action/campaign to get paid. It is idempotent (unique invoice_id, conflicts
// ignored) and non-fatal: failures are logged, never propagated.
// Callers must pass the invoice snapshot from before dunning fields are
// cleared. Returns true when the invoice qualified and the insert succeeded.
func (s *DunningRecoveryService) RecordIfRecovered(ctx context.Context, inv *domain.Invoice) bool {
	if s == nil || s.repo == nil || inv == nil {
		return false
	}

	strategy := s.strategy
	var campaignID *uuid.UUID
	if s.campaignLookup != nil {
		exec, err := s.campaignLookup.GetExecutionByInvoice(ctx, inv.ID)
		if err != nil {
			s.logger.Error("failed to look up dunning campaign execution", "invoice_id", inv.ID, "error", err)
		} else if exec != nil {
			campaignID = &exec.CampaignID
			strategy = "campaign"
		}
	}

	// Qualification: at least one failed attempt or an active dunning action.
	if inv.RetryCount < 1 && inv.DunningActionID == "" && campaignID == nil {
		return false // paid on the first try — nothing was recovered
	}

	recoveredAt := s.now().UTC()
	if inv.PaidAt != nil {
		recoveredAt = inv.PaidAt.UTC()
	}

	days := 0
	if !inv.CreatedAt.IsZero() {
		if d := int(recoveredAt.Sub(inv.CreatedAt).Hours() / 24); d > 0 {
			days = d
		}
	}

	rec := &domain.RecoveredPayment{
		ID:            uuid.New(),
		TenantID:      inv.TenantID,
		InvoiceID:     inv.ID,
		Amount:        inv.Total,
		Currency:      inv.Currency,
		Attempts:      inv.RetryCount,
		Strategy:      strategy,
		CampaignID:    campaignID,
		DaysToRecover: days,
		RecoveredAt:   recoveredAt,
	}

	if err := s.repo.Insert(ctx, rec); err != nil {
		s.logger.Error("failed to record dunning recovery", "invoice_id", inv.ID, "error", err)
		return false
	}

	s.logger.Info("dunning recovery recorded",
		"invoice_id", inv.ID,
		"amount", rec.Amount,
		"currency", rec.Currency,
		"attempts", rec.Attempts,
		"strategy", rec.Strategy,
	)
	return true
}

// GetRecoveredSummary returns tenant-scoped totals plus a last-12-months
// monthly series of recovered revenue.
func (s *DunningRecoveryService) GetRecoveredSummary(ctx context.Context, tenantID uuid.UUID) (*DunningRecoverySummary, error) {
	totals, err := s.repo.GetRecoveryTotals(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	monthly, err := s.repo.GetMonthlyRecoveries(ctx, tenantID, 12)
	if err != nil {
		return nil, err
	}

	amounts := totals.RecoveredAmountTotal
	if amounts == nil {
		amounts = map[string]int64{}
	}
	if monthly == nil {
		monthly = []domain.RecoveryMonthBucket{}
	}

	// Normalize to the tenant's reporting currency (like MRR and the other
	// analytics do) so the headline isn't decided by raw minor-unit magnitude.
	reporting := s.resolveReportingCurrency(ctx, tenantID)
	normalizer := newFXNormalizer(s.fxProvider, s.fxFallback)
	haveFX := s.fxProvider != nil || s.fxFallback != nil

	var reportingTotal int64
	for ccy, amt := range amounts {
		if !haveFX {
			if ccy == reporting {
				reportingTotal += amt
			}
			continue
		}
		if conv, _, err := normalizer.convert(ctx, amt, ccy, reporting); err == nil {
			reportingTotal += conv
		}
	}

	// Collapse the monthly series into a single reporting-currency series so the
	// chart shows one consistent currency instead of "(INR only)".
	byMonth := map[string]*domain.RecoveryMonthBucket{}
	var monthOrder []string
	for _, b := range monthly {
		amt := b.Amount
		if haveFX {
			conv, _, err := normalizer.convert(ctx, b.Amount, b.Currency, reporting)
			if err != nil {
				continue
			}
			amt = conv
		} else if b.Currency != reporting {
			continue
		}
		if byMonth[b.Month] == nil {
			byMonth[b.Month] = &domain.RecoveryMonthBucket{Month: b.Month, Currency: reporting}
			monthOrder = append(monthOrder, b.Month)
		}
		byMonth[b.Month].Amount += amt
		byMonth[b.Month].Count += b.Count
	}
	normMonthly := make([]domain.RecoveryMonthBucket, 0, len(monthOrder))
	for _, m := range monthOrder {
		normMonthly = append(normMonthly, *byMonth[m])
	}

	return &DunningRecoverySummary{
		ReportingCurrency:    reporting,
		ReportingTotal:       reportingTotal,
		RecoveredAmountTotal: amounts,
		RecoveredCount:       totals.RecoveredCount,
		AvgAttempts:          totals.AvgAttempts,
		AvgDaysToRecover:     totals.AvgDaysToRecover,
		Monthly:              normMonthly,
	}, nil
}

// CollectionsBucket is one stage of the recovery funnel: how many invoices and
// how much money (in the reporting currency), post FX-normalization.
type CollectionsBucket struct {
	Count  int   `json:"count"`
	Amount int64 `json:"amount"`
}

// recoveryRateWindowDays is the trailing cohort window for RecoveryRate. Long
// enough to smooth week-to-week noise, short enough that the KPI reflects the
// CURRENT engine — the old all-time numerator drifted monotonically upward as
// the business aged (QA finding D).
const recoveryRateWindowDays = 90

// CollectionsFunnel is the failed → resolved journey of billed revenue
// (Collections Intelligence Inc 2). PastDue is money still being chased
// (current snapshot), Uncollectible is written off (current snapshot),
// Recovered is what the engine clawed back (all-time) — each labeled for what
// it is. RecoveryRate is a WINDOWED cohort: of the cases that CONCLUDED in the
// trailing RateWindowDays (recovered or written off, both timestamped), the
// fraction saved — so it neither drifts with business age nor gets dragged
// down by in-flight past_due.
type CollectionsFunnel struct {
	ReportingCurrency string            `json:"reporting_currency"`
	PastDue           CollectionsBucket `json:"past_due"`
	Uncollectible     CollectionsBucket `json:"uncollectible"`
	Recovered         CollectionsBucket `json:"recovered"`
	RecoveryRate      float64           `json:"recovery_rate"`
	RateWindowDays    int               `json:"rate_window_days"`
	// FXExcludedCurrencies lists currencies whose amounts could NOT be
	// converted into the reporting currency and were therefore excluded from
	// the bucket AMOUNTS (their invoices still count in the bucket COUNTS).
	// Non-empty means the money figures are understated — mirrored from the
	// MRR pattern of flagged exclusion rather than silent drop.
	FXExcludedCurrencies []string `json:"fx_excluded_currencies,omitempty"`
}

// CollectionsFailureBucket is one failure reason ranked by money at risk, in the
// reporting currency.
type CollectionsFailureBucket struct {
	ErrorCode    string `json:"error_code"`
	Count        int    `json:"count"`
	AmountAtRisk int64  `json:"amount_at_risk"`
}

// GetCollectionsFunnel composes the recovery funnel: currently-failing (past_due)
// and written-off (uncollectible) invoices from the invoice side, recovered
// totals from recovered_payments, all normalized to the tenant's reporting
// currency. Nil aggregator → an empty (but valid) funnel.
func (s *DunningRecoveryService) GetCollectionsFunnel(ctx context.Context, tenantID uuid.UUID) (*CollectionsFunnel, error) {
	reporting := s.resolveReportingCurrency(ctx, tenantID)
	funnel := &CollectionsFunnel{ReportingCurrency: reporting}
	normalizer := newFXNormalizer(s.fxProvider, s.fxFallback)
	haveFX := s.fxProvider != nil || s.fxFallback != nil

	// norm converts a per-currency minor-unit amount into the reporting currency,
	// skipping amounts it can't convert (rather than mis-summing mixed
	// currencies) — and RECORDS the skip so the exclusion is visible on the
	// response instead of silently understating the money figures.
	excluded := map[string]bool{}
	norm := func(amt int64, ccy string) (int64, bool) {
		if ccy == reporting {
			return amt, true
		}
		if !haveFX {
			excluded[ccy] = true
			return 0, false
		}
		conv, _, err := normalizer.convert(ctx, amt, ccy, reporting)
		if err != nil {
			excluded[ccy] = true
			return 0, false
		}
		return conv, true
	}

	if s.collectionsAgg != nil {
		atRisk, err := s.collectionsAgg.GetCollectionsAtRisk(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		for _, row := range atRisk {
			amt, ok := norm(row.Amount, row.Currency)
			bucket := &funnel.PastDue
			if row.Status == string(domain.InvoiceStatusUncollectible) {
				bucket = &funnel.Uncollectible
			}
			bucket.Count += row.Count
			if ok {
				bucket.Amount += amt
			}
		}
	}

	// Recovered totals (all-time) from the recovered-payments ledger.
	if s.repo != nil {
		totals, err := s.repo.GetRecoveryTotals(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		funnel.Recovered.Count = totals.RecoveredCount
		for ccy, amt := range totals.RecoveredAmountTotal {
			if conv, ok := norm(amt, ccy); ok {
				funnel.Recovered.Amount += conv
			}
		}
	}

	// Recovery rate = windowed concluded cohort: recoveries vs write-offs that
	// happened in the trailing window, both sides timestamped (recovered_at /
	// marked_uncollectible_at). Mixing the all-time recovered count with the
	// current uncollectible snapshot — the previous formula — made the KPI
	// drift upward forever as recoveries accumulated (QA finding D).
	funnel.RateWindowDays = recoveryRateWindowDays
	if s.repo != nil && s.collectionsAgg != nil {
		since := s.now().AddDate(0, 0, -recoveryRateWindowDays)
		recovered, err := s.repo.CountRecoveredSince(ctx, tenantID, since)
		if err != nil {
			return nil, err
		}
		writtenOff, err := s.collectionsAgg.CountUncollectibleSince(ctx, tenantID, since)
		if err != nil {
			return nil, err
		}
		if concluded := recovered + writtenOff; concluded > 0 {
			funnel.RecoveryRate = float64(recovered) / float64(concluded)
		}
	}

	for ccy := range excluded {
		funnel.FXExcludedCurrencies = append(funnel.FXExcludedCurrencies, ccy)
	}
	sort.Strings(funnel.FXExcludedCurrencies)
	return funnel, nil
}

// GetFailureBreakdown ranks the failure reasons holding the most billed revenue
// hostage right now, normalized to the reporting currency and sorted by amount
// at risk (desc). Nil aggregator → empty slice.
func (s *DunningRecoveryService) GetFailureBreakdown(ctx context.Context, tenantID uuid.UUID) ([]CollectionsFailureBucket, error) {
	if s.collectionsAgg == nil {
		return []CollectionsFailureBucket{}, nil
	}
	rows, err := s.collectionsAgg.GetCollectionsFailureBreakdown(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	reporting := s.resolveReportingCurrency(ctx, tenantID)
	normalizer := newFXNormalizer(s.fxProvider, s.fxFallback)
	haveFX := s.fxProvider != nil || s.fxFallback != nil

	byCode := map[string]*CollectionsFailureBucket{}
	var order []string
	for _, row := range rows {
		b := byCode[row.ErrorCode]
		if b == nil {
			b = &CollectionsFailureBucket{ErrorCode: row.ErrorCode}
			byCode[row.ErrorCode] = b
			order = append(order, row.ErrorCode)
		}
		b.Count += row.Count
		if row.Currency == reporting {
			b.AmountAtRisk += row.Amount
		} else if haveFX {
			if conv, _, err := normalizer.convert(ctx, row.Amount, row.Currency, reporting); err == nil {
				b.AmountAtRisk += conv
			}
		}
	}

	out := make([]CollectionsFailureBucket, 0, len(order))
	for _, code := range order {
		out = append(out, *byCode[code])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AmountAtRisk > out[j].AmountAtRisk })
	return out, nil
}
