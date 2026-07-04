package db

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type UnbilledChargeRepository struct {
	DB *sql.DB
}

func NewUnbilledChargeRepository(db *sql.DB) *UnbilledChargeRepository {
	return &UnbilledChargeRepository{DB: db}
}

func (r *UnbilledChargeRepository) Create(charge *domain.UnbilledCharge) error {
	query := `
		INSERT INTO unbilled_charges (id, subscription_id, amount, currency, description, status, period_start, period_end, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.DB.Exec(query,
		charge.ID,
		charge.SubscriptionID,
		charge.Amount,
		charge.Currency,
		charge.Description,
		charge.Status,
		charge.PeriodStart,
		charge.PeriodEnd,
		charge.CreatedAt,
	)
	return err
}

func (r *UnbilledChargeRepository) ListBySubscriptionID(subscriptionID uuid.UUID) ([]*domain.UnbilledCharge, error) {
	query := `
		SELECT id, subscription_id, amount, currency, description, status, period_start, period_end, created_at
		FROM unbilled_charges
		WHERE subscription_id = $1 AND status = 'pending'
		ORDER BY created_at ASC
	`
	rows, err := r.DB.Query(query, subscriptionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var charges []*domain.UnbilledCharge
	for rows.Next() {
		var c domain.UnbilledCharge
		if err := rows.Scan(
			&c.ID,
			&c.SubscriptionID,
			&c.Amount,
			&c.Currency,
			&c.Description,
			&c.Status,
			&c.PeriodStart,
			&c.PeriodEnd,
			&c.CreatedAt,
		); err != nil {
			return nil, err
		}
		charges = append(charges, &c)
	}
	return charges, nil
}

func (r *UnbilledChargeRepository) MarkAsInvoiced(chargeIDs []uuid.UUID) error {
	if len(chargeIDs) == 0 {
		return nil
	}
	query := `
		UPDATE unbilled_charges
		SET status = 'invoiced'
		WHERE id = ANY($1)
	`
	_, err := r.DB.Exec(query, pq.Array(chargeIDs))
	return err
}
