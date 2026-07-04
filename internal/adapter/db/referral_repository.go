package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type ReferralRepository struct {
	db *sqlx.DB
}

func NewReferralRepository(db *sqlx.DB) *ReferralRepository {
	return &ReferralRepository{db: db}
}

func (r *ReferralRepository) Create(ctx context.Context, referral *domain.Referral) error {
	query := `
		INSERT INTO referrals (
			id, tenant_id, referrer_id, referred_id, code, status, 
			reward_amount, currency, created_at, updated_at, qualified_at
		) VALUES (
			:id, :tenant_id, :referrer_id, :referred_id, :code, :status,
			:reward_amount, :currency, :created_at, :updated_at, :qualified_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, referral)
	return err
}

func (r *ReferralRepository) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.Referral, error) {
	var referral domain.Referral
	query := `SELECT * FROM referrals WHERE tenant_id = $1 AND id = $2 LIMIT 1`
	err := r.db.GetContext(ctx, &referral, query, tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &referral, nil
}

func (r *ReferralRepository) Update(ctx context.Context, referral *domain.Referral) error {
	query := `
		UPDATE referrals
		SET status = :status, qualified_at = :qualified_at, updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id
	`
	_, err := r.db.NamedExecContext(ctx, query, referral)
	return err
}

func (r *ReferralRepository) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Referral, error) {
	var referral domain.Referral
	query := `SELECT * FROM referrals WHERE tenant_id = $1 AND code = $2 LIMIT 1`
	err := r.db.GetContext(ctx, &referral, query, tenantID, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &referral, nil
}

func (r *ReferralRepository) GetByReferrerID(ctx context.Context, tenantID uuid.UUID, referrerID uuid.UUID) ([]*domain.Referral, error) {
	var referrals []*domain.Referral
	query := `SELECT * FROM referrals WHERE tenant_id = $1 AND referrer_id = $2 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &referrals, query, tenantID, referrerID)
	return referrals, err
}

func (r *ReferralRepository) GetByReferredID(ctx context.Context, tenantID uuid.UUID, referredID uuid.UUID) (*domain.Referral, error) {
	var referral domain.Referral
	query := `SELECT * FROM referrals WHERE tenant_id = $1 AND referred_id = $2 LIMIT 1`
	err := r.db.GetContext(ctx, &referral, query, tenantID, referredID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &referral, nil
}

func (r *ReferralRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Referral, error) {
	var referrals []*domain.Referral
	query := `SELECT * FROM referrals WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &referrals, query, tenantID, limit, offset)
	return referrals, err
}
