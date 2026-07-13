package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type OfflinePaymentRepository interface {
	CreateVirtualAccount(ctx context.Context, va *domain.VirtualAccount) error
	GetVirtualAccountByID(ctx context.Context, id uuid.UUID) (*domain.VirtualAccount, error)
	GetVirtualAccountByRazorpayID(ctx context.Context, razorpayVAID string) (*domain.VirtualAccount, error)
	ListVirtualAccounts(ctx context.Context, tenantID uuid.UUID) ([]*domain.VirtualAccount, error)
	UpdateVirtualAccount(ctx context.Context, va *domain.VirtualAccount) error
	// IncrementAmountReceived atomically adds `amount` to the VA's
	// amount_received (closing it when amount_expected is reached) and returns
	// the updated row. Replaces a read-modify-write that dropped concurrent
	// credits under multi-instance webhook delivery.
	IncrementAmountReceived(ctx context.Context, razorpayVAID string, amount int64) (*domain.VirtualAccount, error)
	CreateOfflinePayment(ctx context.Context, payment *domain.OfflinePayment) error
	ListOfflinePayments(ctx context.Context, tenantID uuid.UUID) ([]*domain.OfflinePayment, error)
}
