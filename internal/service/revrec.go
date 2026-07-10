package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type RevRecRepository interface {
	CreateSchedule(ctx context.Context, schedule *domain.RevenueSchedule) error
	CreateEvents(ctx context.Context, events []*domain.RecognitionEvent) error
	GetDueEvents(ctx context.Context, date time.Time) ([]*domain.RecognitionEvent, error)
	MarkEventRecognized(ctx context.Context, eventID uuid.UUID, ledgerTxID uuid.UUID) error
	MarkEventFailed(ctx context.Context, eventID uuid.UUID, reason string) error
	GetReport(ctx context.Context, tenantID uuid.UUID, month, year int) (map[string]interface{}, error)
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

	schedule := &domain.RevenueSchedule{
		ID:             uuid.New(),
		TenantID:       invoice.TenantID,
		InvoiceID:      invoice.ID,
		SubscriptionID: invoice.SubscriptionID,
		TotalAmount:    invoice.Total,
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
func (s *RevRecService) GetReport(ctx context.Context, tenantID uuid.UUID, month, year int) (map[string]interface{}, error) {
	return s.repo.GetReport(ctx, tenantID, month, year)
}
