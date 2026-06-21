package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type PlanRepository interface {
	Create(ctx context.Context, plan *domain.Plan) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error)
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Plan, error)
	List(ctx context.Context, tenantID uuid.UUID, filter domain.PlanFilter) ([]*domain.Plan, error)
}

type InvoiceRepository interface {
	Create(ctx context.Context, invoice *domain.Invoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invoice, error)
	Update(ctx context.Context, invoice *domain.Invoice) error
	GetDueForRetry(ctx context.Context) ([]*domain.Invoice, error)
	UpdateRetryInfo(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int) error
	MarkAsUncollectible(ctx context.Context, invoiceID uuid.UUID) error
	GetOverdueInvoices(ctx context.Context) ([]domain.OverdueInvoice, error)
	GetFailedEInvoices(ctx context.Context) ([]*domain.Invoice, error)
	UpdateEInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status, irn, ackNo, signedQR, ackDate, errorMsg string) error
}

type ReferralRepository interface {
	Create(ctx context.Context, referral *domain.Referral) error
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Referral, error)
	GetByReferrerID(ctx context.Context, tenantID uuid.UUID, referrerID uuid.UUID) ([]*domain.Referral, error)
	GetByReferredID(ctx context.Context, tenantID uuid.UUID, referredID uuid.UUID) (*domain.Referral, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Referral, error)
}

type GiftRepository interface {
	Create(ctx context.Context, gift *domain.Gift) error
	GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Gift, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Gift, error)
	Update(ctx context.Context, gift *domain.Gift) error
}
