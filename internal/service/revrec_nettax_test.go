package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// captureRevRecRepo records the schedule and events CreateScheduleForInvoice
// builds, so we can assert what gets deferred without a database.
type captureRevRecRepo struct {
	RevRecRepository // embed for the methods this test never calls
	schedule         *domain.RevenueSchedule
	events           []*domain.RecognitionEvent
}

func (c *captureRevRecRepo) CreateSchedule(_ context.Context, s *domain.RevenueSchedule) error {
	c.schedule = s
	return nil
}

func (c *captureRevRecRepo) CreateEvents(_ context.Context, e []*domain.RecognitionEvent) error {
	c.events = append(c.events, e...)
	return nil
}

// TestCreateScheduleForInvoice_DefersNetOfTax proves the ENG-191 fix: a GST
// subscription invoice defers only the taxable revenue (Total-Tax), not the
// gross. GST is reclassified out of Deferred into Tax Payable at invoice time,
// so deferring the gross here drove Deferred negative by the tax and recognized
// the tax as revenue.
func TestCreateScheduleForInvoice_DefersNetOfTax(t *testing.T) {
	repo := &captureRevRecRepo{}
	svc := NewRevRecService(repo, nil, nil)

	subID := uuid.New()
	inv := &domain.Invoice{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		SubscriptionID: &subID,
		Total:          118000, // ₹1180.00 gross
		TaxAmount:      18000,  // ₹180.00 GST (18%)
		Currency:       "INR",
		CreatedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := svc.CreateScheduleForInvoice(context.Background(), inv, nil); err != nil {
		t.Fatalf("CreateScheduleForInvoice: %v", err)
	}

	const wantNet = 100000 // 118000 - 18000
	if repo.schedule == nil {
		t.Fatal("no schedule created")
	}
	if repo.schedule.TotalAmount != wantNet {
		t.Errorf("schedule TotalAmount = %d, want %d (net of tax)", repo.schedule.TotalAmount, wantNet)
	}

	var eventSum int64
	for _, e := range repo.events {
		eventSum += e.Amount
	}
	if eventSum != wantNet {
		t.Errorf("recognition events sum = %d, want %d (must never recognize the tax)", eventSum, wantNet)
	}
}

// TestCreateScheduleForInvoice_ZeroTaxUnchanged guards the non-GST path: with no
// tax, net == gross, so nothing about the pre-ENG-191 behavior changes.
func TestCreateScheduleForInvoice_ZeroTaxUnchanged(t *testing.T) {
	repo := &captureRevRecRepo{}
	svc := NewRevRecService(repo, nil, nil)

	subID := uuid.New()
	inv := &domain.Invoice{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		SubscriptionID: &subID,
		Total:          120000,
		TaxAmount:      0,
		Currency:       "USD",
		CreatedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if err := svc.CreateScheduleForInvoice(context.Background(), inv, nil); err != nil {
		t.Fatalf("CreateScheduleForInvoice: %v", err)
	}
	if repo.schedule.TotalAmount != 120000 {
		t.Errorf("schedule TotalAmount = %d, want 120000 (gross==net when no tax)", repo.schedule.TotalAmount)
	}
}
