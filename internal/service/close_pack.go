package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// glExportPath is where the full general ledger streams as CSV. The close pack
// links to it rather than embedding every posting, which would make the JSON
// unbounded for a busy tenant.
const glExportPath = "/v1/ledger/export"

// ClosePackPeriod is the calendar month a close pack covers.
type ClosePackPeriod struct {
	Month int       `json:"month"`
	Year  int       `json:"year"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ClosePackDeferred carries the two independent views of deferred (unearned)
// revenue an auditor ties against each other at close: the ledger-sourced
// rollforward of the Deferred Revenue account, and the schedule-sourced
// recognition report. Ties is true when the ledger closing balance equals the
// schedule's still-deferred balance — divergence is a soft signal, not a close
// blocker, so it is surfaced but does not gate ReadyToClose.
type ClosePackDeferred struct {
	Rollforward *domain.DeferredRollforward   `json:"rollforward"`           // ledger-sourced (Deferred Revenue account movement)
	Recognition *domain.DeferredRevenueReport `json:"recognition,omitempty"` // schedule-sourced; nil when rev-rec is not wired
	// AwaitingPayment is the pre-tax deferral funded by unpaid subscription
	// invoices — Deferred the ledger rightly carries but no schedule covers
	// yet, because schedules are created on payment. The tie-out identity is
	// ledger closing == schedule deferred + awaiting payment; comparing the
	// first two directly made the tie-out structurally amber for any tenant
	// with open invoices (recurso-dev/recurso#466).
	AwaitingPayment  int64 `json:"awaiting_payment"`
	UnexplainedDelta int64 `json:"unexplained_delta"` // ledger − (schedule + awaiting); 0 when the books tie
	Ties             bool  `json:"ties"`
}

// ClosePackGL points at the streaming CSV export instead of embedding the full
// journal in the response.
type ClosePackGL struct {
	Format    string `json:"format"`
	ExportURL string `json:"export_url"`
}

// ClosePack is the month-end close artifact: one call that proves the books
// balance (trial balance), that billing records tie to the ledger
// (reconciliation), and that deferred revenue rolls forward correctly, plus a
// pointer to the GL export. Read-only and never persisted — it is computed on
// demand from the same services that back the individual finance endpoints.
type ClosePack struct {
	TenantID       uuid.UUID             `json:"tenant_id"`
	Period         ClosePackPeriod       `json:"period"`
	GeneratedAt    time.Time             `json:"generated_at"`
	ReadyToClose   bool                  `json:"ready_to_close"`
	Blockers       []string              `json:"blockers"`
	TrialBalance   *domain.TrialBalance  `json:"trial_balance"`
	Reconciliation *ReconciliationReport `json:"reconciliation"`
	Deferred       ClosePackDeferred     `json:"deferred_revenue"`
	GeneralLedger  ClosePackGL           `json:"general_ledger"`
	// ReportingCurrency is the tenant's base currency (from the trial balance),
	// so the UI formats the pack's minor-unit totals with the right exponent.
	ReportingCurrency string `json:"reporting_currency"`
}

// ClosePackService composes the existing read-only finance services into a
// single month-end close artifact. The trial-balance and reconciliation
// services are required; the rev-rec service is optional (nil-safe Set* idiom)
// and, when present, adds the schedule-sourced deferred-revenue view.
type ClosePackService struct {
	ledger *LedgerService
	recon  *ReconciliationService
	revrec *RevRecService
	// unscheduledDeferral sums the pre-tax value of subscription invoices that
	// funded Deferred at issuance but have no recognition schedule yet
	// (schedules are created on payment) — the "awaiting payment" bucket the
	// tie-out must account for. Nil-safe; without it the tie-out falls back to
	// the raw (structurally amber) comparison.
	unscheduledDeferral unscheduledDeferralReader
}

// unscheduledDeferralReader is satisfied by *db.InvoiceRepository.
type unscheduledDeferralReader interface {
	SumUnscheduledDeferral(ctx context.Context, tenantID uuid.UUID) (int64, error)
}

// SetUnscheduledDeferralReader wires the awaiting-payment bucket. Nil-safe.
func (s *ClosePackService) SetUnscheduledDeferralReader(r unscheduledDeferralReader) {
	s.unscheduledDeferral = r
}

// NewClosePackService creates a close-pack service over the ledger and
// reconciliation services. Wire the optional rev-rec view with SetRevRecService.
func NewClosePackService(ledger *LedgerService, recon *ReconciliationService) *ClosePackService {
	return &ClosePackService{ledger: ledger, recon: recon}
}

// SetRevRecService wires the optional schedule-sourced deferred-revenue view.
// Nil-safe: when unset, the close pack reports only the ledger rollforward.
func (s *ClosePackService) SetRevRecService(r *RevRecService) { s.revrec = r }

// Generate builds the close pack for a tenant's calendar month. It reads only;
// closing the period is a human decision. ReadyToClose is true when the trial
// balance is in balance AND reconciliation finds zero discrepancies; Blockers
// spells out why when it is not.
func (s *ClosePackService) Generate(ctx context.Context, tenantID uuid.UUID, month, year int) (*ClosePack, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	tb, err := s.ledger.GetTrialBalance(ctx, tenantID, nil) // month-end close is tenant-wide (all entities)
	if err != nil {
		return nil, fmt.Errorf("close pack trial balance: %w", err)
	}

	recon, err := s.recon.Run(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("close pack reconciliation: %w", err)
	}

	rollforward, err := s.ledger.GetDeferredRollforward(ctx, tenantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("close pack deferred rollforward: %w", err)
	}

	deferred := ClosePackDeferred{Rollforward: rollforward}
	if s.revrec != nil {
		recognition, err := s.revrec.GetReport(ctx, tenantID, month, year)
		if err != nil {
			return nil, fmt.Errorf("close pack revrec report: %w", err)
		}
		deferred.Recognition = recognition
		if s.unscheduledDeferral != nil {
			awaiting, aerr := s.unscheduledDeferral.SumUnscheduledDeferral(ctx, tenantID)
			if aerr != nil {
				return nil, fmt.Errorf("close pack awaiting-payment deferral: %w", aerr)
			}
			deferred.AwaitingPayment = awaiting
		}
		if recognition != nil {
			deferred.UnexplainedDelta = rollforward.Closing - recognition.DeferredBalance - deferred.AwaitingPayment
			deferred.Ties = deferred.UnexplainedDelta == 0
		}
	}

	blockers := closeBlockers(tb, recon)

	return &ClosePack{
		TenantID:          tenantID,
		Period:            ClosePackPeriod{Month: month, Year: year, Start: start, End: end},
		GeneratedAt:       time.Now().UTC(),
		ReadyToClose:      len(blockers) == 0,
		Blockers:          blockers,
		TrialBalance:      tb,
		Reconciliation:    recon,
		Deferred:          deferred,
		GeneralLedger:     ClosePackGL{Format: "csv", ExportURL: glExportPath},
		ReportingCurrency: tb.ReportingCurrency,
	}, nil
}

// closeBlockers lists the reasons a period cannot be closed: an out-of-balance
// trial balance and any reconciliation discrepancy. Pure so it is unit-testable
// without a database. Returns a non-nil empty slice when the period is clean, so
// the JSON always serializes "blockers": [] rather than null.
func closeBlockers(tb *domain.TrialBalance, recon *ReconciliationReport) []string {
	blockers := []string{}
	if tb != nil && !tb.Balanced {
		blockers = append(blockers, fmt.Sprintf(
			"trial balance out of balance: debits %d != credits %d", tb.TotalDebits, tb.TotalCredits))
	}
	if recon != nil && recon.TotalDiscrepancies > 0 {
		blockers = append(blockers, fmt.Sprintf(
			"%d reconciliation discrepanc%s", recon.TotalDiscrepancies, plural(recon.TotalDiscrepancies)))
	}
	return blockers
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
