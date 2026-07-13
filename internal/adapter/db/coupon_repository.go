package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type CouponRepository struct {
	db *sql.DB
}

func NewCouponRepository(db *sql.DB) *CouponRepository {
	return &CouponRepository{db: db}
}

func (r *CouponRepository) Create(ctx context.Context, coupon *domain.Coupon) error {
	query := `
		INSERT INTO coupons (id, tenant_id, code, discount_type, discount_value, duration, duration_months, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		coupon.ID,
		coupon.TenantID,
		coupon.Code,
		coupon.DiscountType,
		coupon.DiscountValue,
		coupon.Duration,
		coupon.DurationMonths,
		coupon.CreatedAt,
		coupon.UpdatedAt,
	)
	return err
}

// GetByCode looks up a coupon by code, always scoped to the tenant. Previously
// the tenant filter was applied only when the context happened to carry a
// tenant id, silently degrading to a global lookup (first match across all
// tenants) otherwise — an ENG-160 cross-tenant risk. It now requires the tenant
// explicitly, matching Plan/Referral/Gift GetByCode.
func (r *CouponRepository) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Coupon, error) {
	query := `
		SELECT id, tenant_id, code, discount_type, discount_value, duration, duration_months, created_at, updated_at
		FROM coupons
		WHERE code = $1 AND tenant_id = $2
	`

	row := r.db.QueryRowContext(ctx, query, code, tenantID)

	var c domain.Coupon
	var durationMonths sql.NullInt32

	err := row.Scan(
		&c.ID,
		&c.TenantID,
		&c.Code,
		&c.DiscountType,
		&c.DiscountValue,
		&c.Duration,
		&durationMonths,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if durationMonths.Valid {
		val := int(durationMonths.Int32)
		c.DurationMonths = &val
	}

	return &c, nil
}

func (r *CouponRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Coupon, error) {
	query := `
		SELECT id, tenant_id, code, discount_type, discount_value, duration, duration_months, created_at, updated_at
		FROM coupons
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var coupons []*domain.Coupon
	for rows.Next() {
		var c domain.Coupon
		var durationMonths sql.NullInt32
		if err := rows.Scan(
			&c.ID,
			&c.TenantID,
			&c.Code,
			&c.DiscountType,
			&c.DiscountValue,
			&c.Duration,
			&durationMonths,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if durationMonths.Valid {
			val := int(durationMonths.Int32)
			c.DurationMonths = &val
		}
		coupons = append(coupons, &c)
	}
	return coupons, nil
}
