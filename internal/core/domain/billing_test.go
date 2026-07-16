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

// TestAddInterval_MonthEndDoesNotOverflow proves the recurring-date-drift fix:
// adding a month to a day that doesn't exist in the target month must CLAMP to
// that month's last day, never overflow into the following month. Raw
// time.AddDate normalizes Jan 31 + 1 month to Mar 3, skipping February entirely
// and corrupting the billing cycle.
func TestAddInterval_MonthEndDoesNotOverflow(t *testing.T) {
	utc := time.UTC
	cases := []struct {
		name  string
		start time.Time
		unit  string
		count int
		wantY int
		wantM time.Month
		wantD int
	}{
		{"jan31 + 1 month clamps to Feb 28 (non-leap)", time.Date(2025, 1, 31, 10, 30, 0, 0, utc), "month", 1, 2025, time.February, 28},
		{"jan31 + 1 month clamps to Feb 29 (leap)", time.Date(2024, 1, 31, 10, 30, 0, 0, utc), "month", 1, 2024, time.February, 29},
		{"mar31 + 1 month clamps to Apr 30", time.Date(2025, 3, 31, 0, 0, 0, 0, utc), "month", 1, 2025, time.April, 30},
		{"jan31 + 2 months stays Mar 31 (no needless clamp)", time.Date(2025, 1, 31, 0, 0, 0, 0, utc), "month", 2, 2025, time.March, 31},
		{"feb29 + 1 year clamps to Feb 28", time.Date(2024, 2, 29, 0, 0, 0, 0, utc), "year", 1, 2025, time.February, 28},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AddInterval(tc.start, tc.unit, tc.count)
			if got.Year() != tc.wantY || got.Month() != tc.wantM || got.Day() != tc.wantD {
				t.Errorf("AddInterval(%v, %q, %d) = %v, want %04d-%02d-%02d", tc.start, tc.unit, tc.count, got.Format("2006-01-02"), tc.wantY, tc.wantM, tc.wantD)
			}
			// Time-of-day must be preserved.
			if got.Hour() != tc.start.Hour() || got.Minute() != tc.start.Minute() {
				t.Errorf("time-of-day drifted: got %02d:%02d, want %02d:%02d", got.Hour(), got.Minute(), tc.start.Hour(), tc.start.Minute())
			}
		})
	}
}

// TestAddInterval_MonthlySeriesSkipsNoMonth proves a year of monthly renewals
// anchored on the 31st never skips a month (the drift bug billed Jan then Mar).
func TestAddInterval_MonthlySeriesSkipsNoMonth(t *testing.T) {
	d := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 12; i++ {
		next := AddInterval(d, "month", 1)
		wantMonth := d.Month()%12 + 1 // consecutive month, no gaps
		if next.Month() != wantMonth {
			t.Fatalf("month %d: AddInterval(%v) = %v (month %v), want consecutive month %v",
				i, d.Format("2006-01-02"), next.Format("2006-01-02"), next.Month(), wantMonth)
		}
		d = next
	}
}

func TestAddInterval(t *testing.T) {
	base := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		unit     string
		count    int
		expected time.Time
	}{
		{"day", 1, base.AddDate(0, 0, 1)},
		{"day", 7, base.AddDate(0, 0, 7)},
		{"week", 1, base.AddDate(0, 0, 7)},
		{"week", 2, base.AddDate(0, 0, 14)},
		{"month", 1, base.AddDate(0, 1, 0)},
		{"month", 3, base.AddDate(0, 3, 0)},
		{"year", 1, base.AddDate(1, 0, 0)},
		{"year", 2, base.AddDate(2, 0, 0)},
		{"unknown", 1, base.AddDate(0, 1, 0)}, // defaults to 1 month
	}

	for _, tc := range tests {
		t.Run(tc.unit, func(t *testing.T) {
			got := AddInterval(base, tc.unit, tc.count)
			if !got.Equal(tc.expected) {
				t.Errorf("AddInterval(%v, %q, %d) = %v, want %v", base, tc.unit, tc.count, got, tc.expected)
			}
		})
	}
}

// A month-end subscription must not get "sticky" at day 28: with the billing
// anchor on the 31st, the cycle is Jan 31 → Feb 28 → Mar 31 → Apr 30 → May 31,
// restoring the anchor day in months long enough to hold it.
func TestCalculateNextBillingDate_ReanchorsAfterClamp(t *testing.T) {
	anchor := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	sub := &Subscription{
		BillingAnchorType: "acquisition",
		BillingAnchor:     anchor,
		CurrentPeriodEnd:  anchor,
	}

	want := []string{"2026-02-28", "2026-03-31", "2026-04-30", "2026-05-31", "2026-06-30", "2026-07-31"}
	for i, w := range want {
		next := sub.CalculateNextBillingDate("month", 1)
		if got := next.Format("2006-01-02"); got != w {
			t.Fatalf("cycle %d: got %s, want %s", i, got, w)
		}
		sub.CurrentPeriodEnd = next
	}
}

// BillingAnchorDay (calendar billing bookkeeping) wins over the anchor date's
// day when set.
func TestCalculateNextBillingDate_AnchorDayFieldWins(t *testing.T) {
	sub := &Subscription{
		BillingAnchorType: "acquisition",
		BillingAnchor:     time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		BillingAnchorDay:  30,
		CurrentPeriodEnd:  time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}
	next := sub.CalculateNextBillingDate("month", 1)
	if got := next.Format("2006-01-02"); got != "2026-03-30" {
		t.Fatalf("got %s, want 2026-03-30", got)
	}
}

// Re-anchoring must only restore days lost to month-end clamping. A cycle that
// runs mid-month (e.g. a trial that converted on the 10th while the original
// anchor was the 25th) keeps billing on its current day.
func TestCalculateNextBillingDate_NoReanchorMidMonth(t *testing.T) {
	sub := &Subscription{
		BillingAnchorType: "acquisition",
		BillingAnchor:     time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:  time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
	}
	next := sub.CalculateNextBillingDate("month", 1)
	if got := next.Format("2006-01-02"); got != "2026-04-10" {
		t.Fatalf("got %s, want 2026-04-10 (no re-anchor from a mid-month day)", got)
	}
}

// Weekly/daily intervals never re-anchor to a day of month.
func TestCalculateNextBillingDate_NoReanchorForDayWeek(t *testing.T) {
	sub := &Subscription{
		BillingAnchorType: "acquisition",
		BillingAnchor:     time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		CurrentPeriodEnd:  time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}
	next := sub.CalculateNextBillingDate("week", 1)
	if got := next.Format("2006-01-02"); got != "2026-03-07" {
		t.Fatalf("got %s, want 2026-03-07", got)
	}
}
