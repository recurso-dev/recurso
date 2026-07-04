package service

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

type ChurnService struct {
	customerRepo port.CustomerRepository
	invoiceRepo  port.InvoiceRepository
	subRepo      port.SubscriptionRepository
	planRepo     port.PlanRepository
	db           *sql.DB
}

func NewChurnService(
	customerRepo port.CustomerRepository,
	invoiceRepo port.InvoiceRepository,
) *ChurnService {
	return &ChurnService{
		customerRepo: customerRepo,
		invoiceRepo:  invoiceRepo,
	}
}

func (s *ChurnService) SetSubscriptionRepo(repo port.SubscriptionRepository) {
	s.subRepo = repo
}

func (s *ChurnService) SetPlanRepo(repo port.PlanRepository) {
	s.planRepo = repo
}

func (s *ChurnService) SetDB(db *sql.DB) {
	s.db = db
}

// AnalyzeCustomer calculates the risk score using the ML model.
func (s *ChurnService) AnalyzeCustomer(ctx context.Context, customerID uuid.UUID) error {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return err
	}

	invoices, err := s.invoiceRepo.GetByCustomerID(ctx, customerID)
	if err != nil {
		return err
	}

	var subscriptions []*domain.Subscription
	if s.subRepo != nil {
		subs, err := s.subRepo.List(ctx, customer.TenantID, domain.SubscriptionFilter{
			CustomerID: customerID,
			Limit:      100,
		})
		if err == nil {
			subscriptions = subs
		}
	}

	features := ExtractFeatures(ctx, customer, invoices, subscriptions, s.planRepo)
	score := ScoreChurn(features)

	// Get previous score for alert comparison
	prevScore := customer.RiskScore

	// Update risk in DB
	factors := map[string]interface{}{
		"payment_failure_rate": features.PaymentFailureRate,
		"avg_days_to_pay":      features.AvgDaysToPay,
		"months_active":        features.MonthsActive,
		"failed_invoices_90d":  features.FailedInvoices90d,
		"model_version":        "v1",
	}

	if err := s.customerRepo.UpdateRisk(ctx, customerID, score, factors); err != nil {
		return err
	}

	// Save feature snapshot
	s.saveSnapshot(ctx, customer.TenantID, customerID, features, score)

	// Check for threshold crossings
	s.checkAlertThresholds(ctx, customer.TenantID, customerID, prevScore, score)

	return nil
}

func (s *ChurnService) saveSnapshot(ctx context.Context, tenantID, customerID uuid.UUID, features *ChurnFeatures, score int) {
	if s.db == nil {
		return
	}

	query := `INSERT INTO churn_feature_snapshots (id, tenant_id, customer_id, days_since_signup,
		total_invoices, failed_invoices_90d, payment_failure_rate, avg_days_to_pay,
		plan_downgrades, months_active, current_mrr, usage_trend, risk_score, model_version, computed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err := s.db.ExecContext(ctx, query,
		uuid.New(), tenantID, customerID, features.DaysSinceSignup,
		features.TotalInvoices, features.FailedInvoices90d, features.PaymentFailureRate,
		features.AvgDaysToPay, features.PlanDowngrades, features.MonthsActive,
		features.CurrentMRR, features.UsageTrend, score, "v1", time.Now(),
	)
	if err != nil {
		slog.Error("failed to save churn snapshot", "error", err)
	}
}

func (s *ChurnService) checkAlertThresholds(ctx context.Context, tenantID, customerID uuid.UUID, prevScore, newScore int) {
	if s.db == nil {
		return
	}

	// Alert on crossing 70 (high risk) or 90 (critical)
	thresholds := []struct {
		threshold int
		alertType string
	}{
		{70, "high_risk"},
		{90, "critical"},
	}

	for _, t := range thresholds {
		if prevScore < t.threshold && newScore >= t.threshold {
			query := `INSERT INTO churn_alerts (id, tenant_id, customer_id, previous_score, new_score, threshold, alert_type, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
			_, err := s.db.ExecContext(ctx, query,
				uuid.New(), tenantID, customerID, prevScore, newScore, t.threshold, t.alertType, time.Now(),
			)
			if err != nil {
				slog.Error("failed to create churn alert", "error", err)
			}
		}
	}
}

// GetCustomerScore returns the current churn score with features.
func (s *ChurnService) GetCustomerScore(ctx context.Context, customerID uuid.UUID) (*ChurnScoreResult, error) {
	customer, err := s.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	invoices, err := s.invoiceRepo.GetByCustomerID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	var subscriptions []*domain.Subscription
	if s.subRepo != nil {
		subs, err := s.subRepo.List(ctx, customer.TenantID, domain.SubscriptionFilter{
			CustomerID: customerID,
			Limit:      100,
		})
		if err == nil {
			subscriptions = subs
		}
	}

	features := ExtractFeatures(ctx, customer, invoices, subscriptions, s.planRepo)
	score := ScoreChurn(features)

	return &ChurnScoreResult{
		CustomerID:   customerID,
		Score:        score,
		RiskLevel:    RiskLevel(score),
		Features:     features,
		ModelVersion: "v1",
	}, nil
}

// GetHighRiskCustomers returns customers above the threshold.
func (s *ChurnService) GetHighRiskCustomers(ctx context.Context, tenantID uuid.UUID, threshold int) ([]*ChurnScoreResult, error) {
	if s.db == nil {
		return nil, nil
	}

	query := `SELECT DISTINCT ON (customer_id) customer_id, risk_score, days_since_signup,
		total_invoices, failed_invoices_90d, payment_failure_rate, avg_days_to_pay,
		plan_downgrades, months_active, current_mrr, usage_trend
		FROM churn_feature_snapshots
		WHERE tenant_id = $1 AND risk_score >= $2
		ORDER BY customer_id, computed_at DESC`

	rows, err := s.db.QueryContext(ctx, query, tenantID, threshold)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []*ChurnScoreResult
	for rows.Next() {
		var r ChurnScoreResult
		var f ChurnFeatures
		err := rows.Scan(&r.CustomerID, &r.Score, &f.DaysSinceSignup,
			&f.TotalInvoices, &f.FailedInvoices90d, &f.PaymentFailureRate,
			&f.AvgDaysToPay, &f.PlanDowngrades, &f.MonthsActive,
			&f.CurrentMRR, &f.UsageTrend)
		if err != nil {
			return nil, err
		}
		r.Features = &f
		r.RiskLevel = RiskLevel(r.Score)
		r.ModelVersion = "v1"
		results = append(results, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// AnalyzeAllCustomers iterates and analyzes all customers for a tenant.
func (s *ChurnService) AnalyzeAllCustomers(ctx context.Context, tenantID uuid.UUID) error {
	offset := 0
	limit := 100

	for {
		customers, err := s.customerRepo.List(ctx, tenantID, domain.CustomerFilter{Limit: limit, Offset: offset})
		if err != nil {
			return err
		}
		if len(customers) == 0 {
			break
		}

		for _, customer := range customers {
			if err := s.AnalyzeCustomer(ctx, customer.ID); err != nil {
				slog.Error("failed to analyze customer", "customer_id", customer.ID, "error", err)
			}
		}

		offset += limit
	}

	return nil
}
