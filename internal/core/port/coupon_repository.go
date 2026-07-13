package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type CouponRepository interface {
	Create(ctx context.Context, coupon *domain.Coupon) error
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Coupon, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Coupon, error)
}
