package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type GiftRepository struct {
	db *sqlx.DB
}

func NewGiftRepository(db *sqlx.DB) *GiftRepository {
	return &GiftRepository{db: db}
}

func (r *GiftRepository) Create(ctx context.Context, gift *domain.Gift) error {
	query := `
		INSERT INTO gifts (
			id, tenant_id, code, plan_id, buyer_customer_id, recipient_email, 
			status, redeemed_by_customer_id, redeemed_at, duration_months, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :code, :plan_id, :buyer_customer_id, :recipient_email,
			:status, :redeemed_by_customer_id, :redeemed_at, :duration_months, :created_at, :updated_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, gift)
	return err
}

func (r *GiftRepository) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Gift, error) {
	var gift domain.Gift
	query := `SELECT * FROM gifts WHERE tenant_id = $1 AND code = $2 LIMIT 1`
	err := r.db.GetContext(ctx, &gift, query, tenantID, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &gift, nil
}

func (r *GiftRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Gift, error) {
	var gifts []*domain.Gift
	query := `SELECT * FROM gifts WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &gifts, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	return gifts, nil
}

// MarkRedeemed atomically flips purchased -> redeemed. Returns true only when
// this call made the transition (WHERE status = 'purchased'), so concurrent
// redemptions of the same gift can't both win.
func (r *GiftRepository) MarkRedeemed(ctx context.Context, giftID, tenantID, redeemedBy uuid.UUID, at time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE gifts SET status = $1, redeemed_by_customer_id = $2, redeemed_at = $3, updated_at = $3
		 WHERE id = $4 AND tenant_id = $5 AND status = $6`,
		domain.GiftStatusRedeemed, redeemedBy, at, giftID, tenantID, domain.GiftStatusPurchased)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (r *GiftRepository) RevertRedemption(ctx context.Context, giftID, tenantID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE gifts SET status = $1, redeemed_by_customer_id = NULL, redeemed_at = NULL, updated_at = NOW()
		 WHERE id = $2 AND tenant_id = $3`,
		domain.GiftStatusPurchased, giftID, tenantID)
	return err
}

func (r *GiftRepository) Update(ctx context.Context, gift *domain.Gift) error {
	query := `
		UPDATE gifts SET
			status = :status,
			redeemed_by_customer_id = :redeemed_by_customer_id,
			redeemed_at = :redeemed_at,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id
	`
	_, err := r.db.NamedExecContext(ctx, query, gift)
	return err
}

// GetByID loads one gift, tenant-scoped. Nil when absent.
func (r *GiftRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Gift, error) {
	var gift domain.Gift
	err := r.db.GetContext(ctx, &gift, `SELECT * FROM gifts WHERE tenant_id = $1 AND id = $2 LIMIT 1`, tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &gift, nil
}

// SetInvoiceID links the buyer's purchase invoice to the gift (best-effort at
// purchase time; cancellation acts on the link).
func (r *GiftRepository) SetInvoiceID(ctx context.Context, giftID, tenantID, invoiceID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE gifts SET invoice_id = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		invoiceID, giftID, tenantID)
	return err
}

// Cancel atomically flips purchased -> canceled. Returns true only when this
// call made the transition (WHERE status = 'purchased'), so a concurrent
// cancel — or a cancel racing a redemption — can't both win, and the winner is
// the only caller that issues the buyer's credit.
func (r *GiftRepository) Cancel(ctx context.Context, giftID, tenantID uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE gifts SET status = $1, updated_at = NOW()
		 WHERE id = $2 AND tenant_id = $3 AND status = $4`,
		domain.GiftStatusCanceled, giftID, tenantID, domain.GiftStatusPurchased)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}
