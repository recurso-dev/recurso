package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// RecoveredPaymentRepository persists and aggregates dunning recovery records.
type RecoveredPaymentRepository interface {
	// Insert writes a recovery record. Implementations must be idempotent on
	// invoice_id (ON CONFLICT DO NOTHING) so the multiple payment-success
	// paths can all call it safely.
	Insert(ctx context.Context, rec *domain.RecoveredPayment) error
	GetRecoveryTotals(ctx context.Context, tenantID uuid.UUID) (*domain.RecoveryTotals, error)
	GetMonthlyRecoveries(ctx context.Context, tenantID uuid.UUID, months int) ([]domain.RecoveryMonthBucket, error)
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
		repo:     repo,
		strategy: strategy,
		logger:   slog.Default().With("service", "dunning_recovery"),
		now:      time.Now,
	}
}

// SetCampaignLookup injects the campaign execution lookup after construction.
func (s *DunningRecoveryService) SetCampaignLookup(lookup CampaignExecutionLookup) {
	s.campaignLookup = lookup
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

	return &DunningRecoverySummary{
		RecoveredAmountTotal: amounts,
		RecoveredCount:       totals.RecoveredCount,
		AvgAttempts:          totals.AvgAttempts,
		AvgDaysToRecover:     totals.AvgDaysToRecover,
		Monthly:              monthly,
	}, nil
}
