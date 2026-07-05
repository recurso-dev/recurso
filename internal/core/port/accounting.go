package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// AccountingGateway defines the interface for external accounting platforms
// (QBO, Xero, Tally). Each Sync* method returns the provider-side ID of the
// synced object (QuickBooks Customer.Id, Xero ContactID, ...) so callers can
// persist the internal-to-external mapping and reference it in dependent
// syncs. SyncInvoice takes the customer's provider-side ID because invoices
// must reference the provider's customer/contact record, not our internal
// UUID.
type AccountingGateway interface {
	SyncCustomer(ctx context.Context, customer *domain.Customer) (externalID string, err error)
	SyncInvoice(ctx context.Context, invoice *domain.Invoice, customerExternalID string) (externalID string, err error)
	SyncProduct(ctx context.Context, plan *domain.Plan) (externalID string, err error)
}

// AccountingMappingRepository persists internal-to-external entity ID
// mappings per accounting connection.
type AccountingMappingRepository interface {
	// Upsert inserts the mapping or refreshes external_id/updated_at when a
	// row already exists for (connection_id, entity_type, entity_id).
	Upsert(ctx context.Context, m *domain.AccountingEntityMapping) error
	// Get returns the mapping, or (nil, nil) when none exists.
	Get(ctx context.Context, connectionID uuid.UUID, entityType string, entityID uuid.UUID) (*domain.AccountingEntityMapping, error)
}

// AccountingService orchestrates synchronization logic
type AccountingService interface {
	SyncCustomer(ctx context.Context, customerID uuid.UUID) error
	SyncInvoice(ctx context.Context, invoiceID uuid.UUID) error
	SyncProduct(ctx context.Context, planID string) error
	SyncAllForTenant(ctx context.Context, tenantID uuid.UUID) error
}
