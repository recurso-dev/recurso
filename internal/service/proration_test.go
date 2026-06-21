package service

import (
	"testing"
	"time"
)

func TestCalculateProration(t *testing.T) {
	svc := &SubscriptionService{}

	tests := []struct {
		name           string
		currentPrice   int64
		newPrice       int64
		periodStart    time.Time
		periodEnd      time.Time
		prorationDate  time.Time
		expectedCredit int64
		expectedCharge int64
	}{
		{
			name:           "upgrade mid-period (50%)",
			currentPrice:   1000, // $10
			newPrice:       2000, // $20
			periodStart:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			prorationDate:  time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC), // ~50% through
			expectedCredit: 500,                                           // ~$5 unused on old plan
			expectedCharge: 1000,                                          // ~$10 remaining on new plan
		},
		{
			name:           "downgrade mid-period (50%)",
			currentPrice:   2000,
			newPrice:       1000,
			periodStart:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			prorationDate:  time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC),
			expectedCredit: 1000,
			expectedCharge: 500,
		},
		{
			name:           "change at period start (100% remaining)",
			currentPrice:   1000,
			newPrice:       3000,
			periodStart:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			prorationDate:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			expectedCredit: 1000, // Full credit
			expectedCharge: 3000, // Full charge
		},
		{
			name:           "change at period end (0% remaining)",
			currentPrice:   1000,
			newPrice:       2000,
			periodStart:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			prorationDate:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			expectedCredit: 0,
			expectedCharge: 0,
		},
		{
			name:           "same price (no net change)",
			currentPrice:   1500,
			newPrice:       1500,
			periodStart:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:      time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			prorationDate:  time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC),
			expectedCredit: 0, // Will be close to same
			expectedCharge: 0,
		},
		{
			name:           "zero duration period",
			currentPrice:   1000,
			newPrice:       2000,
			periodStart:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			prorationDate:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			expectedCredit: 0,
			expectedCharge: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := svc.CalculateProration(
				tc.currentPrice, tc.newPrice,
				tc.periodStart, tc.periodEnd,
				tc.prorationDate,
			)

			if tc.name == "zero duration period" {
				if result.CreditAmount != 0 || result.ChargeAmount != 0 {
					t.Errorf("zero duration: expected 0/0, got %d/%d", result.CreditAmount, result.ChargeAmount)
				}
				return
			}

			if tc.name == "same price (no net change)" {
				if result.NetAmount != 0 {
					t.Errorf("same price: expected net 0, got %d", result.NetAmount)
				}
				return
			}

			// Allow 5% tolerance for rounding
			creditTolerance := int64(float64(tc.expectedCredit) * 0.05)
			if creditTolerance < 1 {
				creditTolerance = 1
			}

			chargeTolerance := int64(float64(tc.expectedCharge) * 0.05)
			if chargeTolerance < 1 {
				chargeTolerance = 1
			}

			creditDiff := abs(result.CreditAmount - tc.expectedCredit)
			chargeDiff := abs(result.ChargeAmount - tc.expectedCharge)

			if creditDiff > creditTolerance {
				t.Errorf("credit: got %d, want ~%d (diff %d > tolerance %d)",
					result.CreditAmount, tc.expectedCredit, creditDiff, creditTolerance)
			}

			if chargeDiff > chargeTolerance {
				t.Errorf("charge: got %d, want ~%d (diff %d > tolerance %d)",
					result.ChargeAmount, tc.expectedCharge, chargeDiff, chargeTolerance)
			}

			// Net = Charge - Credit
			expectedNet := result.ChargeAmount - result.CreditAmount
			if result.NetAmount != expectedNet {
				t.Errorf("net amount mismatch: got %d, want %d", result.NetAmount, expectedNet)
			}
		})
	}
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
