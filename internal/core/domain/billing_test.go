package domain

import (
	"testing"
	"time"
)

func TestCalculateDueDate(t *testing.T) {
	baseTime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		start    time.Time
		terms    string
		expected time.Time
	}{
		{
			name:     "empty terms returns start date",
			start:    baseTime,
			terms:    "",
			expected: baseTime,
		},
		{
			name:     "due_on_receipt returns start date",
			start:    baseTime,
			terms:    "due_on_receipt",
			expected: baseTime,
		},
		{
			name:     "net0 returns start date",
			start:    baseTime,
			terms:    "net0",
			expected: baseTime,
		},
		{
			name:     "net15 adds 15 days",
			start:    baseTime,
			terms:    "net15",
			expected: baseTime.AddDate(0, 0, 15),
		},
		{
			name:     "net30 adds 30 days",
			start:    baseTime,
			terms:    "net30",
			expected: baseTime.AddDate(0, 0, 30),
		},
		{
			name:     "net45 adds 45 days",
			start:    baseTime,
			terms:    "net45",
			expected: baseTime.AddDate(0, 0, 45),
		},
		{
			name:     "net60 adds 60 days",
			start:    baseTime,
			terms:    "net60",
			expected: baseTime.AddDate(0, 0, 60),
		},
		{
			name:     "invalid netX returns start date",
			start:    baseTime,
			terms:    "netABC",
			expected: baseTime,
		},
		{
			name:     "unknown term returns start date",
			start:    baseTime,
			terms:    "custom_term",
			expected: baseTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateDueDate(tc.start, tc.terms)
			if !got.Equal(tc.expected) {
				t.Errorf("CalculateDueDate(%v, %q) = %v, want %v", tc.start, tc.terms, got, tc.expected)
			}
		})
	}
}

func TestCalculateNextBillingDate_Acquisition(t *testing.T) {
	start := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	sub := &Subscription{
		CurrentPeriodEnd:  start,
		BillingAnchorType: "acquisition",
	}

	// Monthly
	next := sub.CalculateNextBillingDate("month", 1)
	expected := start.AddDate(0, 1, 0)
	if !next.Equal(expected) {
		t.Errorf("monthly acquisition: got %v, want %v", next, expected)
	}

	// Yearly
	next = sub.CalculateNextBillingDate("year", 1)
	expected = start.AddDate(1, 0, 0)
	if !next.Equal(expected) {
		t.Errorf("yearly acquisition: got %v, want %v", next, expected)
	}

	// Empty anchor type defaults to acquisition
	sub.BillingAnchorType = ""
	next = sub.CalculateNextBillingDate("month", 1)
	expected = start.AddDate(0, 1, 0)
	if !next.Equal(expected) {
		t.Errorf("empty anchor type: got %v, want %v", next, expected)
	}
}

func TestCalculateNextBillingDate_CalendarBilling(t *testing.T) {
	// Subscription started mid-month — first renewal should be 1st of next month
	sub := &Subscription{
		CurrentPeriodEnd:  time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		BillingAnchorType: "first_of_month",
	}

	next := sub.CalculateNextBillingDate("month", 1)
	expected := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("calendar billing mid-month: got %v, want %v", next, expected)
	}

	// Already aligned to 1st — should add full interval
	sub.CurrentPeriodEnd = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	next = sub.CalculateNextBillingDate("month", 1)
	expected = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("calendar billing already aligned: got %v, want %v", next, expected)
	}

	// Year end — should wrap to next year
	sub.CurrentPeriodEnd = time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)
	next = sub.CalculateNextBillingDate("month", 1)
	expected = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("calendar billing year wrap: got %v, want %v", next, expected)
	}
}

func TestCalculateNextBillingDate_ZeroDate(t *testing.T) {
	sub := &Subscription{
		BillingAnchorType: "acquisition",
	}

	// When CurrentPeriodEnd is zero, should use time.Now()
	next := sub.CalculateNextBillingDate("month", 1)
	// Should be roughly now + 1 month
	if next.Before(time.Now()) {
		t.Error("zero date: next billing date should be in the future")
	}
}

func TestAddInterval(t *testing.T) {
	base := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		unit     string
		count    int
		expected time.Time
	}{
		{"month", 1, base.AddDate(0, 1, 0)},
		{"month", 3, base.AddDate(0, 3, 0)},
		{"year", 1, base.AddDate(1, 0, 0)},
		{"year", 2, base.AddDate(2, 0, 0)},
		{"unknown", 1, base.AddDate(0, 1, 0)}, // defaults to 1 month
	}

	for _, tc := range tests {
		t.Run(tc.unit, func(t *testing.T) {
			got := addInterval(base, tc.unit, tc.count)
			if !got.Equal(tc.expected) {
				t.Errorf("addInterval(%v, %q, %d) = %v, want %v", base, tc.unit, tc.count, got, tc.expected)
			}
		})
	}
}
