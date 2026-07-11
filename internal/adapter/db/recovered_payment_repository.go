package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// RecoveredPaymentRepository persists dunning recovery attribution records.
type RecoveredPaymentRepository struct {
	db *sql.DB
}

func NewRecoveredPaymentRepository(db *sql.DB) *RecoveredPaymentRepository {
	return &RecoveredPaymentRepository{db: db}
}

// Insert writes a recovery record. Idempotent: the unique constraint on
// invoice_id absorbs duplicate recordings from concurrent payment-success
// paths (webhook + retry worker), so conflicts are silently ignored.
func (r *RecoveredPaymentRepository) Insert(ctx context.Context, rec *domain.RecoveredPayment) error {
	query := `
		INSERT INTO recovered_payments
			(id, tenant_id, invoice_id, amount, currency, attempts, strategy, campaign_id, days_to_recover, recovered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (invoice_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		rec.ID, rec.TenantID, rec.InvoiceID, rec.Amount, rec.Currency,
		rec.Attempts, rec.Strategy, rec.CampaignID, rec.DaysToRecover, rec.RecoveredAt,
	)
	return err
}

// GetRecoveryTotals returns tenant-scoped aggregate recovery stats.
func (r *RecoveredPaymentRepository) GetRecoveryTotals(ctx context.Context, tenantID uuid.UUID) (*domain.RecoveryTotals, error) {
	totals := &domain.RecoveryTotals{
		RecoveredAmountTotal: map[string]int64{},
	}

	statsQuery := `
		SELECT COUNT(*),
		       COALESCE(AVG(attempts), 0),
		       COALESCE(AVG(days_to_recover), 0)
		FROM recovered_payments
		WHERE tenant_id = $1
	`
	if err := r.db.QueryRowContext(ctx, statsQuery, tenantID).Scan(
		&totals.RecoveredCount, &totals.AvgAttempts, &totals.AvgDaysToRecover,
	); err != nil {
		return nil, err
	}

	amountQuery := `
		SELECT currency, SUM(amount)
		FROM recovered_payments
		WHERE tenant_id = $1
		GROUP BY currency
	`
	rows, err := r.db.QueryContext(ctx, amountQuery, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var currency string
		var amount int64
		if err := rows.Scan(&currency, &amount); err != nil {
			return nil, err
		}
		totals.RecoveredAmountTotal[currency] = amount
	}
	return totals, rows.Err()
}

// GetMonthlyRecoveries returns the recovered-revenue series for the last N
// calendar months (including the current one), grouped by month and currency.
func (r *RecoveredPaymentRepository) GetMonthlyRecoveries(ctx context.Context, tenantID uuid.UUID, months int) ([]domain.RecoveryMonthBucket, error) {
	if months <= 0 {
		months = 12
	}
	query := `
		SELECT to_char(date_trunc('month', recovered_at), 'YYYY-MM') AS month,
		       currency,
		       SUM(amount),
		       COUNT(*)
		FROM recovered_payments
		WHERE tenant_id = $1
		  AND recovered_at >= date_trunc('month', NOW()) - make_interval(months => $2 - 1)
		GROUP BY 1, 2
		ORDER BY 1, 2
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, months)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var series []domain.RecoveryMonthBucket
	for rows.Next() {
		var b domain.RecoveryMonthBucket
		if err := rows.Scan(&b.Month, &b.Currency, &b.Amount, &b.Count); err != nil {
			return nil, err
		}
		series = append(series, b)
	}
	return series, rows.Err()
}
