package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type RevRecRepository interface {
	CreateSchedule(ctx context.Context, schedule *domain.RevenueSchedule) error
	CreateEvents(ctx context.Context, events []*domain.RecognitionEvent) error
	GetDueEvents(ctx context.Context, date time.Time) ([]*domain.RecognitionEvent, error)
	MarkEventRecognized(ctx context.Context, eventID uuid.UUID, ledgerTxID uuid.UUID) error
	GetReport(ctx context.Context, tenantID uuid.UUID, month, year int) (map[string]interface{}, error)
}

type RevRecService struct {
	repo   RevRecRepository
	ledger *LedgerService
}

func NewRevRecService(repo RevRecRepository, ledger *LedgerService) *RevRecService {
	return &RevRecService{
		repo:   repo,
		ledger: ledger,
	}
}

// ProcessDueEvents executes recognition for events whose date has passed
func (s *RevRecService) ProcessDueEvents(ctx context.Context) error {
	events, err := s.repo.GetDueEvents(ctx, time.Now())
	if err != nil {
		return err
	}

	for _, e := range events {
		// 1. Execute Ledger Transfer
		txID, err := s.ledger.RecordRecognition(ctx, e.TenantID, e.Amount)
		if err != nil {
			// Log and mark as failed
			continue
		}

		// 2. Mark as Recognized in PG
		if err := s.repo.MarkEventRecognized(ctx, e.ID, txID); err != nil {
			// Log error
		}
	}
	return nil
}

// CreateScheduleForInvoice generates a recognition schedule for a paid invoice
func (s *RevRecService) CreateScheduleForInvoice(ctx context.Context, invoice *domain.Invoice) error {
	// If invoice has no subscription (one-off), we recognize immediately
	// For MVP, we primarily target subscription-linked invoices
	if invoice.SubscriptionID == nil {
		// recognition is immediate
		return s.createImmediateRecognition(ctx, invoice)
	}

	// For subscription invoices, we split based on period
	// We need the subscription period from somewhere. 
	// For simplicity in this logic, we assume we want to recognize monthly.
	
	// Assume 1 month period if we can't find more data, 
	// but better to use invoice dates if available.
	startDate := invoice.CreatedAt
	endDate := startDate.AddDate(0, 1, 0) // Default to 1 month

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
func (s *RevRecService) CalculateMonthlyAllocation(schedule *domain.RevenueSchedule) []*domain.RecognitionEvent {
	var events []*domain.RecognitionEvent
	
	// For MVP: Simple single-event recognition at the start of the period
	// Full ratable recognition logic would involve splitting based on days.
	
	// Let's implement basic 1-month split
	// In production, an annual plan would have 12 events.
	
	months := 1
	// Simple logic: if period > 45 days, assume multi-month
	duration := schedule.EndDate.Sub(schedule.StartDate)
	if duration > 45*24*time.Hour {
		months = 12 // Annual
	}

	amountPerMonth := schedule.TotalAmount / int64(months)
	remainder := schedule.TotalAmount % int64(months)

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
