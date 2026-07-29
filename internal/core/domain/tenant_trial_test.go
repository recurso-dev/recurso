package domain

import (
	"testing"
	"time"
)

func ptr(t time.Time) *time.Time { return &t }

func TestTenant_TrialDaysLeft(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		end  *time.Time
		want int
	}{
		{"no trial", nil, 0},
		{"expired", ptr(now.Add(-time.Hour)), 0},
		{"exactly now", ptr(now), 0},
		{"18 hours left rounds up to 1", ptr(now.Add(18 * time.Hour)), 1},
		{"exactly 1 day", ptr(now.Add(24 * time.Hour)), 1},
		{"13 days + 1h rounds up to 14", ptr(now.Add(13*24*time.Hour + time.Hour)), 14},
	}
	for _, c := range cases {
		tn := &Tenant{TrialEndsAt: c.end}
		if got := tn.TrialDaysLeft(now); got != c.want {
			t.Errorf("%s: TrialDaysLeft = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestTenant_IsTrialingAndExpired(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)

	active := &Tenant{BillingStatus: BillingStatusActive, TrialEndsAt: ptr(now.Add(-time.Hour))}
	if active.IsTrialing() {
		t.Error("active tenant should not be trialing")
	}
	if active.IsTrialExpired(now) {
		t.Error("non-trialing tenant is never 'trial expired'")
	}

	future := &Tenant{BillingStatus: BillingStatusTrialing, TrialEndsAt: ptr(now.Add(48 * time.Hour))}
	if !future.IsTrialing() {
		t.Error("should be trialing")
	}
	if future.IsTrialExpired(now) {
		t.Error("trial in the future is not expired")
	}

	past := &Tenant{BillingStatus: BillingStatusTrialing, TrialEndsAt: ptr(now.Add(-time.Minute))}
	if !past.IsTrialExpired(now) {
		t.Error("trialing tenant past its end is expired")
	}
}
