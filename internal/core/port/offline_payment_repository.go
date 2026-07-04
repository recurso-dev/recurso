package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type OfflinePaymentRepository interface {
	CreateVirtualAccount(ctx context.Context, va *domain.VirtualAccount) error
	GetVirtualAccountByID(ctx context.Context, id uuid.UUID) (*domain.VirtualAccount, error)
	GetVirtualAccountByRazorpayID(ctx context.Context, razorpayVAID string) (*domain.VirtualAccount, error)
	ListVirtualAccounts(ctx context.Context, tenantID uuid.UUID) ([]*domain.VirtualAccount, error)
	UpdateVirtualAccount(ctx context.Context, va *domain.VirtualAccount) error
	CreateOfflinePayment(ctx context.Context, payment *domain.OfflinePayment) error
	ListOfflinePayments(ctx context.Context, tenantID uuid.UUID) ([]*domain.OfflinePayment, error)
}
