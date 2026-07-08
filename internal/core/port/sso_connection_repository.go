package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// SSOConnectionRepository persists per-tenant SAML IdP configuration (one row
// per tenant).
type SSOConnectionRepository interface {
	// GetByTenant returns the tenant's connection, or
	// domain.ErrSSOConnectionNotFound if none exists.
	GetByTenant(ctx context.Context, tenantID uuid.UUID) (*domain.SSOConnection, error)
	// Upsert inserts or updates the tenant's connection (keyed by tenant_id).
	Upsert(ctx context.Context, conn *domain.SSOConnection) error
	// Delete removes the tenant's connection. Deleting a non-existent connection
	// returns domain.ErrSSOConnectionNotFound.
	Delete(ctx context.Context, tenantID uuid.UUID) error
}
