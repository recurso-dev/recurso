package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// IntegrationConnectionRepository persists per-tenant integration credentials.
// The config blob is already sealed by the service before it reaches here.
type IntegrationConnectionRepository interface {
	// Upsert replaces the tenant's active connection for a (category, provider).
	Upsert(ctx context.Context, conn *domain.IntegrationConnection) error
	// GetActive returns the tenant's active connection for a (category,
	// provider), or domain.ErrIntegrationConnectionNotFound.
	GetActive(ctx context.Context, tenantID uuid.UUID, category domain.IntegrationCategory, provider string) (*domain.IntegrationConnection, error)
	// ListByTenant returns the tenant's active connections across categories.
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.IntegrationConnection, error)
	// Deactivate soft-disconnects the tenant's active connection for a
	// (category, provider). Returns domain.ErrIntegrationConnectionNotFound when
	// none is active.
	Deactivate(ctx context.Context, tenantID uuid.UUID, category domain.IntegrationCategory, provider string) error
}
