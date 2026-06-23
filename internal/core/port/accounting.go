package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

// AccountingGateway defines the interface for external accounting platforms (QBO, Xero)
type AccountingGateway interface {
	SyncCustomer(ctx context.Context, customer *domain.Customer) error
	SyncInvoice(ctx context.Context, invoice *domain.Invoice) error
	SyncProduct(ctx context.Context, plan *domain.Plan) error
}

// AccountingService orchestrates synchronization logic
type AccountingService interface {
	SyncCustomer(ctx context.Context, customerID uuid.UUID) error
	SyncInvoice(ctx context.Context, invoiceID uuid.UUID) error
	SyncProduct(ctx context.Context, planID string) error
	SyncAllForTenant(ctx context.Context, tenantID uuid.UUID) error
}
