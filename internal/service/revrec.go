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

type RevRecRepository interface {
	CreateSchedule(ctx context.Context, schedule *domain.RevenueSchedule) error
	CreateEvents(ctx context.Context, events []*domain.RecognitionEvent) error
	GetDueEvents(ctx context.Context, date time.Time) ([]*domain.RecognitionEvent, error)
	MarkEventRecognized(ctx context.Context, eventID uuid.UUID, ledgerTxID uuid.UUID) error
	MarkEventFailed(ctx context.Context, eventID uuid.UUID, reason string) error
	GetReport(ctx context.Context, tenantID uuid.UUID, month, year int) (*domain.DeferredRevenueReport, error)
	GetWaterfall(ctx context.Context, tenantID uuid.UUID) ([]domain.RevenueWaterfallBucket, error)

	// Unwind support (ENG-147): reverse the still-deferred portion of a schedule
	// when a subscription is canceled or refunded mid-period.
	GetActiveSchedulesBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]*domain.RevenueSchedule, error)
	GetActiveScheduleByInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) (*domain.RevenueSchedule, error)
	// GetPendingEventsBySchedule returns the schedule's not-yet-recognized events,
	// latest recognition_date first (so an unwind reduces from the tail).
	GetPendingEventsBySchedule(ctx context.Context, scheduleID uuid.UUID) ([]*domain.RecognitionEvent, error)
	CancelEvent(ctx context.Context, eventID uuid.UUID) error
	SetEventAmount(ctx context.Context, eventID uuid.UUID, amount int64) error
	MarkScheduleCanceled(ctx context.Context, scheduleID uuid.UUID) error
}

type RevRecService struct {
	repo    RevRecRepository
	subRepo port.SubscriptionRepository
	ledger  *LedgerService
}

func NewRevRecService(repo RevRecRepository, ledger *LedgerService, subRepo port.SubscriptionRepository) *RevRecService {
	return &RevRecService{
		repo:    repo,
		subRepo: subRepo,
		ledger:  ledger,
	}
}

// ProcessDueEvents executes recognition for events whose date has passed
func (s *RevRecService) ProcessDueEvents(ctx context.Context) error {
	events, err := s.repo.GetDueEvents(ctx, time.Now())
	if err != nil {
		return err
	}

	for _, e := range events {
		// 1. Execute Ledger Transfer (reference the event so the posting is
		// attributable and idempotent per event).
		txID, err := s.ledger.RecordRecognition(ctx, e.TenantID, e.Amount, e.ID)
		if err != nil {
			slog.Error("revenue recognition ledger transfer failed", "event_id", e.ID, "error", err)
			if markErr := s.repo.MarkEventFailed(ctx, e.ID, err.Error()); markErr != nil {
				slog.Error("failed to mark recognition event as failed", "event_id", e.ID, "error", markErr)
			}
			continue
		}

		// 2. Mark as Recognized in PG
		if err := s.repo.MarkEventRecognized(ctx, e.ID, txID); err != nil {
			slog.Error("failed to mark recognition event as recognized", "event_id", e.ID, "error", err)
		}
	}
	return nil
}

// UnwindOnCancel forfeits the still-deferred revenue of a subscription's active
// schedules when it is canceled immediately with no refund (ENG-147). Every
// not-yet-recognized event is voided and its total is recognized as breakage
// revenue (DR Deferred / CR Recognized), draining Deferred to 0 so future
// recognition can't keep firing and the deferred balance can't sit forever.
//
// Schedule state is the source of truth: events are voided first, then the
// forfeit is posted to the ledger best-effort (a post failure is logged for
// reconciliation, never fails the cancel). Idempotent: once the schedule is
// canceled a repeat call finds nothing to unwind.
func (s *RevRecService) UnwindOnCancel(ctx context.Context, tenantID, subscriptionID uuid.UUID) (int64, error) {
	schedules, err := s.repo.GetActiveSchedulesBySubscription(ctx, tenantID, subscriptionID)
	if err != nil {
		return 0, fmt.Errorf("load active schedules: %w", err)
	}

	var totalForfeited int64
	for _, sched := range schedules {
		events, err := s.repo.GetPendingEventsBySchedule(ctx, sched.ID)
		if err != nil {
			return totalForfeited, fmt.Errorf("load pending events for schedule %s: %w", sched.ID, err)
		}
		var forfeited int64
		for _, e := range events {
			if err := s.repo.CancelEvent(ctx, e.ID); err != nil {
				return totalForfeited, fmt.Errorf("cancel event %s: %w", e.ID, err)
			}
			forfeited += e.Amount
		}
		if err := s.repo.MarkScheduleCanceled(ctx, sched.ID); err != nil {
			return totalForfeited, fmt.Errorf("cancel schedule %s: %w", sched.ID, err)
		}

		// Forfeited (collected-but-unearned) revenue is recognized as breakage.
		// referenceID is the schedule id, keeping this posting off the per-event
		// recognition keys and idempotent under (reference_id, code=2).
		if forfeited > 0 && s.ledger != nil {
			if _, err := s.ledger.RecordRecognition(ctx, tenantID, forfeited, sched.ID); err != nil {
				slog.Error("forfeit recognition ledger post failed — reconciliation needed",
					"schedule_id", sched.ID, "amount", forfeited, "error", err)
			}
		}
		totalForfeited += forfeited
	}
	return totalForfeited, nil
}

// UnwindOnRefund reverses the still-deferred portion of an invoice's schedule
// when a refund is issued against it (ENG-147). Up to the unrecognized amount is
// pulled out of Deferred to offset the Refunds expense (DR Deferred / CR
// Refunds), and the matching future events are voided/reduced from the tail so
// they stop recognizing the refunded portion. Amounts already recognized are a
// genuine refund expense and are left untouched. Returns the amount reversed.
//
// Called exactly once per credit note: createRefund creates each credit note
// once (the over-refund guard blocks a duplicate), so this does not attempt to
// be idempotent against replays — a replay would reduce the schedule twice while
// the code-5 ledger posting deduped, diverging the two. The ledger post is
// best-effort (a failure is logged for reconciliation, never fails the refund).
func (s *RevRecService) UnwindOnRefund(ctx context.Context, tenantID, invoiceID, creditNoteID uuid.UUID, refundAmount int64) (int64, error) {
	if refundAmount <= 0 {
		return 0, nil
	}
	sched, err := s.repo.GetActiveScheduleByInvoice(ctx, tenantID, invoiceID)
	if err != nil {
		return 0, fmt.Errorf("load schedule for invoice %s: %w", invoiceID, err)
	}
	if sched == nil {
		return 0, nil // one-off invoice, or already fully recognized/canceled
	}
	events, err := s.repo.GetPendingEventsBySchedule(ctx, sched.ID)
	if err != nil {
		return 0, fmt.Errorf("load pending events for schedule %s: %w", sched.ID, err)
	}

	var deferred int64
	for _, e := range events {
		deferred += e.Amount
	}
	reverse := refundAmount
	if reverse > deferred {
		reverse = deferred // only the unearned portion comes out of Deferred
	}
	if reverse <= 0 {
		return 0, nil
	}

	// Remove exactly `reverse` from the tail (latest events first): void whole
	// events, splitting the boundary event by reducing its amount.
	remaining := reverse
	for _, e := range events {
		if remaining <= 0 {
			break
		}
		if remaining >= e.Amount {
			if err := s.repo.CancelEvent(ctx, e.ID); err != nil {
				return 0, fmt.Errorf("cancel event %s: %w", e.ID, err)
			}
			remaining -= e.Amount
		} else {
			if err := s.repo.SetEventAmount(ctx, e.ID, e.Amount-remaining); err != nil {
				return 0, fmt.Errorf("reduce event %s: %w", e.ID, err)
			}
			remaining = 0
		}
	}
	if reverse >= deferred {
		if err := s.repo.MarkScheduleCanceled(ctx, sched.ID); err != nil {
			return 0, fmt.Errorf("cancel schedule %s: %w", sched.ID, err)
		}
	}

	// Post the deferred reversal best-effort (schedule state already corrected).
	if s.ledger != nil {
		if _, err := s.ledger.RecordDeferredRefundReversal(ctx, tenantID, creditNoteID, reverse,
			"Deferred reversal for refund"); err != nil {
			slog.Error("deferred reversal ledger post failed — reconciliation needed",
				"credit_note_id", creditNoteID, "amount", reverse, "error", err)
		}
	}
	return reverse, nil
}

// ReduceScheduleForDowngrade shrinks a subscription's active recognition
// schedule(s) by `amount` when it is downgraded mid-period (ENG-154): the
// over-deferred portion is removed from the tail (latest events first, splitting
// the boundary event) so the remaining schedule recognizes only the new, lower
// plan's amount for the rest of the period. No ledger posting here — the caller
// pairs this with RecordDowngradeCredit so Deferred and the schedule move
// together. Returns the amount actually removed (capped at what was pending).
func (s *RevRecService) ReduceScheduleForDowngrade(ctx context.Context, tenantID, subscriptionID uuid.UUID, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, nil
	}
	schedules, err := s.repo.GetActiveSchedulesBySubscription(ctx, tenantID, subscriptionID)
	if err != nil {
		return 0, fmt.Errorf("load active schedules: %w", err)
	}

	remaining := amount
	var reduced int64
	for _, sched := range schedules {
		if remaining <= 0 {
			break
		}
		events, err := s.repo.GetPendingEventsBySchedule(ctx, sched.ID)
		if err != nil {
			return reduced, fmt.Errorf("load pending events for schedule %s: %w", sched.ID, err)
		}
		for _, e := range events {
			if remaining <= 0 {
				break
			}
			if remaining >= e.Amount {
				if err := s.repo.CancelEvent(ctx, e.ID); err != nil {
					return reduced, fmt.Errorf("cancel event %s: %w", e.ID, err)
				}
				reduced += e.Amount
				remaining -= e.Amount
			} else {
				if err := s.repo.SetEventAmount(ctx, e.ID, e.Amount-remaining); err != nil {
					return reduced, fmt.Errorf("reduce event %s: %w", e.ID, err)
				}
				reduced += remaining
				remaining = 0
			}
		}
	}
	return reduced, nil
}

// CreateScheduleForInvoice generates a recognition schedule for a paid invoice.
// If sub is provided, its period dates are used; otherwise the subscription is looked up.
func (s *RevRecService) CreateScheduleForInvoice(ctx context.Context, invoice *domain.Invoice, sub *domain.Subscription) error {
	// If invoice has no subscription (one-off), we recognize immediately
	if invoice.SubscriptionID == nil {
		return s.createImmediateRecognition(ctx, invoice)
	}

	// Resolve subscription period dates
	if sub == nil && s.subRepo != nil {
		fetched, err := s.subRepo.GetByID(ctx, *invoice.SubscriptionID)
		if err != nil {
			slog.Error("failed to fetch subscription for revrec schedule", "subscription_id", *invoice.SubscriptionID, "error", err)
		} else {
			sub = fetched
		}
	}

	var startDate, endDate time.Time
	if sub != nil {
		startDate = sub.CurrentPeriodStart
		endDate = sub.CurrentPeriodEnd
	} else {
		// Fallback if subscription unavailable
		startDate = invoice.CreatedAt
		endDate = startDate.AddDate(0, 1, 0)
	}

	// Recognize the NET (taxable) revenue, not the gross total. GST is reclassified
	// out of Deferred into Tax Payable at invoice time (ENG-159), so the Deferred
	// ledger account holds only Total-Tax. Scheduling the gross here made the
	// recognition events drain Deferred by more than was ever deferred — Deferred
	// went negative by the tax and Recognized was overstated by it (tax booked as
	// revenue), on every GST subscription invoice (ENG-191).
	netRevenue := invoice.Total - invoice.TaxAmount
	if netRevenue < 0 {
		netRevenue = 0
	}
	schedule := &domain.RevenueSchedule{
		ID:             uuid.New(),
		TenantID:       invoice.TenantID,
		InvoiceID:      invoice.ID,
		SubscriptionID: invoice.SubscriptionID,
		TotalAmount:    netRevenue,
		Currency:       invoice.Currency,
		StartDate:      startDate,
		EndDate:        endDate,
		Status:         domain.RevRecStatusActive,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateSchedule(ctx, schedule); err != nil {
		return err
	}

	events := s.CalculateMonthlyAllocation(schedule)
	return s.repo.CreateEvents(ctx, events)
}

func (s *RevRecService) createImmediateRecognition(ctx context.Context, invoice *domain.Invoice) error {
	schedule := &domain.RevenueSchedule{
		ID:          uuid.New(),
		TenantID:    invoice.TenantID,
		InvoiceID:   invoice.ID,
		TotalAmount: invoice.Total,
		Currency:    invoice.Currency,
		StartDate:   invoice.CreatedAt,
		EndDate:     invoice.CreatedAt,
		Status:      domain.RevRecStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateSchedule(ctx, schedule); err != nil {
		return err
	}

	event := &domain.RecognitionEvent{
		ID:                uuid.New(),
		RevenueScheduleID: schedule.ID,
		TenantID:          schedule.TenantID,
		Amount:            schedule.TotalAmount,
		RecognitionDate:   schedule.StartDate,
		Status:            domain.RecognitionStatusPending,
		CreatedAt:         time.Now(),
	}

	return s.repo.CreateEvents(ctx, []*domain.RecognitionEvent{event})
}

// CalculateMonthlyAllocation splits the total amount into monthly recognition events
// by iterating month-by-month from start to end date.
func (s *RevRecService) CalculateMonthlyAllocation(schedule *domain.RevenueSchedule) []*domain.RecognitionEvent {
	// Count months by iterating from start to end
	months := 0
	cursor := schedule.StartDate
	for !cursor.After(schedule.EndDate) && cursor.Before(schedule.EndDate) {
		months++
		cursor = schedule.StartDate.AddDate(0, months, 0)
	}
	if months < 1 {
		months = 1
	}

	amountPerMonth := schedule.TotalAmount / int64(months)
	remainder := schedule.TotalAmount % int64(months)

	var events []*domain.RecognitionEvent
	for i := 0; i < months; i++ {
		amount := amountPerMonth
		if i == months-1 {
			amount += remainder // Add remainder to last month
		}

		recognitionDate := schedule.StartDate.AddDate(0, i, 0)

		event := &domain.RecognitionEvent{
			ID:                uuid.New(),
			RevenueScheduleID: schedule.ID,
			TenantID:          schedule.TenantID,
			Amount:            amount,
			RecognitionDate:   recognitionDate,
			Status:            domain.RecognitionStatusPending,
			CreatedAt:         time.Now(),
		}
		events = append(events, event)
	}

	return events
}

// GetReport returns the revenue recognition report for a given tenant/month/year.
func (s *RevRecService) GetReport(ctx context.Context, tenantID uuid.UUID, month, year int) (*domain.DeferredRevenueReport, error) {
	return s.repo.GetReport(ctx, tenantID, month, year)
}
