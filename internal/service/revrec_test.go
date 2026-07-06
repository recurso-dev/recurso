package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

func TestCalculateMonthlyAllocation_Monthly(t *testing.T) {
	svc := &RevRecService{}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	schedule := &domain.RevenueSchedule{
		ID:          uuid.New(),
		TotalAmount: 1000, // $10.00
		StartDate:   start,
		EndDate:     end,
	}

	events := svc.CalculateMonthlyAllocation(schedule)

	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	if events[0].Amount != 1000 {
		t.Errorf("expected amount 1000, got %d", events[0].Amount)
	}
}

func TestCalculateMonthlyAllocation_Annual(t *testing.T) {
	svc := &RevRecService{}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0) // 1 year

	schedule := &domain.RevenueSchedule{
		ID:          uuid.New(),
		TotalAmount: 12000, // $120.00
		StartDate:   start,
		EndDate:     end,
	}

	events := svc.CalculateMonthlyAllocation(schedule)

	if len(events) != 12 {
		t.Errorf("expected 12 events, got %d", len(events))
	}

	// Each month should be 1000
	for i, e := range events {
		if e.Amount != 1000 {
			t.Errorf("event %d: expected amount 1000, got %d", i, e.Amount)
		}
		expectedDate := start.AddDate(0, i, 0)
		if !e.RecognitionDate.Equal(expectedDate) {
			t.Errorf("event %d: expected date %v, got %v", i, expectedDate, e.RecognitionDate)
		}
	}
}

func TestCalculateMonthlyAllocation_Remainder(t *testing.T) {
	svc := &RevRecService{}

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	schedule := &domain.RevenueSchedule{
		ID:          uuid.New(),
		TotalAmount: 12005, // $120.05
		StartDate:   start,
		EndDate:     end,
	}

	events := svc.CalculateMonthlyAllocation(schedule)

	var total int64
	for _, e := range events {
		total += e.Amount
	}

	if total != schedule.TotalAmount {
		t.Errorf("expected total %d, got %d", schedule.TotalAmount, total)
	}

	// Last event should have the remainder
	if events[11].Amount != 1005 {
		t.Errorf("expected last event amount 1005, got %d", events[11].Amount)
	}
}
