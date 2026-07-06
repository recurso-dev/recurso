package port

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// ErrExternalGone is returned (wrapped) by AccountingGateway implementations
// when the provider reports that the object a stored external ID points to no
// longer exists — deleted at the provider, purged sandbox company, and so on.
// The service reacts by clearing the stale mapping and re-creating the entity.
var ErrExternalGone = errors.New("accounting: external entity no longer exists at provider")

// InvoiceSyncRefs carries the provider-side IDs an invoice sync depends on.
type InvoiceSyncRefs struct {
	// CustomerExternalID is the provider's customer/contact ID (QuickBooks
	// Customer.Id, Xero ContactID). QBO and Xero reject invoices that
	// reference unknown customers, so this must come from a prior
	// SyncCustomer.
	CustomerExternalID string
	// ProductExternalID is the provider's item ID for the plan backing the
	// invoice (QuickBooks Item.Id), when known. Optional: adapters fall back
	// to bare description-only lines without it.
	ProductExternalID string
	// ProductCode is the internal plan code for the same plan. Xero links
	// invoice lines to items by item Code (not ItemID), so the Xero adapter
	// sets it as the line's ItemCode. Optional; other providers ignore it.
	ProductCode string
}

// AccountingGateway defines the interface for external accounting platforms
// (QBO, Xero, Tally). Each Sync* method upserts: externalID is the known
// provider-side ID of a previously synced object (empty string = create).
// The provider-side ID of the resulting object is returned so callers can
// persist the internal-to-external mapping and reference it in dependent
// syncs. When externalID points at an object the provider no longer has,
// implementations return an error wrapping ErrExternalGone.
type AccountingGateway interface {
	SyncCustomer(ctx context.Context, customer *domain.Customer, externalID string) (string, error)
	SyncInvoice(ctx context.Context, invoice *domain.Invoice, refs InvoiceSyncRefs, externalID string) (string, error)
	SyncProduct(ctx context.Context, plan *domain.Plan, externalID string) (string, error)
}

// AccountingMappingRepository persists internal-to-external entity ID
// mappings per accounting connection.
type AccountingMappingRepository interface {
	// Upsert inserts the mapping or refreshes external_id/updated_at when a
	// row already exists for (connection_id, entity_type, entity_id).
	Upsert(ctx context.Context, m *domain.AccountingEntityMapping) error
	// Get returns the mapping, or (nil, nil) when none exists.
	Get(ctx context.Context, connectionID uuid.UUID, entityType string, entityID uuid.UUID) (*domain.AccountingEntityMapping, error)
	// Delete removes the mapping for (connection_id, entity_type, entity_id).
	// Deleting a mapping that does not exist is not an error.
	Delete(ctx context.Context, connectionID uuid.UUID, entityType string, entityID uuid.UUID) error
}

// AccountingService orchestrates synchronization logic
type AccountingService interface {
	SyncCustomer(ctx context.Context, customerID uuid.UUID) error
	SyncInvoice(ctx context.Context, invoiceID uuid.UUID) error
	SyncProduct(ctx context.Context, planID string) error
	// SyncAllForTenant pushes the tenant's entities to every active
	// connection. With force=false, entities unchanged since their last
	// successful sync are skipped; force=true re-pushes everything (the
	// manual sync endpoint's escape hatch).
	SyncAllForTenant(ctx context.Context, tenantID uuid.UUID, force bool) error
}
