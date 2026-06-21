package service

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

// ChurnFeatures holds extracted features for a customer.
type ChurnFeatures struct {
	DaysSinceSignup    int     `json:"days_since_signup"`
	TotalInvoices      int     `json:"total_invoices"`
	FailedInvoices90d  int     `json:"failed_invoices_90d"`
	PaymentFailureRate float64 `json:"payment_failure_rate"`
	AvgDaysToPay       float64 `json:"avg_days_to_pay"`
	PlanDowngrades     int     `json:"plan_downgrades"`
	MonthsActive       int     `json:"months_active"`
	CurrentMRR         int64   `json:"current_mrr"`
	UsageTrend         float64 `json:"usage_trend"`
}

// ChurnModelWeights are hand-tuned logistic regression weights.
var ChurnModelWeights = struct {
	Intercept          float64
	PaymentFailureRate float64
	AvgDaysToPay       float64
	MonthsActive       float64
	PlanDowngrades     float64
	UsageTrend         float64
	FailedInvoices90d  float64
}{
	Intercept:          -2.0,
	PaymentFailureRate: 5.0,
	AvgDaysToPay:       0.05,
	MonthsActive:       -0.1,
	PlanDowngrades:     1.5,
	UsageTrend:         -3.0,
	FailedInvoices90d:  0.8,
}

// sigmoid computes the logistic function.
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// ScoreChurn computes a risk score (0-100) using logistic regression.
func ScoreChurn(features *ChurnFeatures) int {
	w := ChurnModelWeights
	logit := w.Intercept +
		w.PaymentFailureRate*features.PaymentFailureRate +
		w.AvgDaysToPay*features.AvgDaysToPay +
		w.MonthsActive*float64(features.MonthsActive) +
		w.PlanDowngrades*float64(features.PlanDowngrades) +
		w.UsageTrend*features.UsageTrend +
		w.FailedInvoices90d*float64(features.FailedInvoices90d)

	score := int(sigmoid(logit) * 100)
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// ExtractFeatures computes churn features from invoices and customer data.
func ExtractFeatures(ctx context.Context, customer *domain.Customer, invoices []*domain.Invoice, subscriptions []*domain.Subscription, planRepo port.PlanRepository) *ChurnFeatures {
	features := &ChurnFeatures{}

	// Days since signup
	features.DaysSinceSignup = int(time.Since(customer.CreatedAt).Hours() / 24)
	features.MonthsActive = features.DaysSinceSignup / 30

	// Invoice analysis
	features.TotalInvoices = len(invoices)
	failedCount := 0
	var totalDaysToPay float64
	paidCount := 0

	for _, inv := range invoices {
		if time.Since(inv.CreatedAt) < 90*24*time.Hour {
			if inv.Status == domain.InvoiceStatusVoid ||
				inv.Status == domain.InvoiceStatusPastDue ||
				inv.Status == domain.InvoiceStatusUncollectible {
				failedCount++
			}
		}

		if inv.PaidAt != nil {
			daysToPay := inv.PaidAt.Sub(inv.CreatedAt).Hours() / 24
			totalDaysToPay += daysToPay
			paidCount++
		}
	}

	features.FailedInvoices90d = failedCount
	if features.TotalInvoices > 0 {
		features.PaymentFailureRate = float64(failedCount) / float64(features.TotalInvoices)
	}
	if paidCount > 0 {
		features.AvgDaysToPay = totalDaysToPay / float64(paidCount)
	}

	// Current MRR from active subscriptions
	for _, sub := range subscriptions {
		if sub.Status == domain.SubscriptionStatusActive {
			plan, err := planRepo.GetByID(ctx, sub.PlanID)
			if err == nil && len(plan.Prices) > 0 {
				features.CurrentMRR += plan.Prices[0].Amount
			}
		}
	}

	// Usage trend: placeholder (would come from usage metrics)
	features.UsageTrend = 0.0

	return features
}

// ChurnScoreResult is returned by the churn scoring endpoint.
type ChurnScoreResult struct {
	CustomerID uuid.UUID      `json:"customer_id"`
	Score      int            `json:"score"`
	RiskLevel  string         `json:"risk_level"`
	Features   *ChurnFeatures `json:"features"`
	ModelVersion string       `json:"model_version"`
}

func RiskLevel(score int) string {
	switch {
	case score >= 90:
		return "critical"
	case score >= 70:
		return "high"
	case score >= 40:
		return "medium"
	default:
		return "low"
	}
}
