package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// GatewayConnectionRepository persists per-tenant BYO gateway credentials.
// Secret fields are already sealed (ciphertext) by the service before they
// reach the repository — the repository never encrypts or decrypts.
type GatewayConnectionRepository interface {
	// Upsert inserts or replaces the tenant's active connection for a provider
	// (keyed by the partial-unique (tenant, provider) WHERE active index).
	Upsert(ctx context.Context, conn *domain.GatewayConnection) error
	// GetByID resolves a single connection, used by per-connection webhook
	// routing. Returns domain.ErrGatewayConnectionNotFound when absent.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.GatewayConnection, error)
	// GetActive returns the tenant's active connection for a provider, or
	// domain.ErrGatewayConnectionNotFound.
	GetActive(ctx context.Context, tenantID uuid.UUID, provider domain.GatewayProvider) (*domain.GatewayConnection, error)
	// ListByTenant returns the tenant's active connections (one per provider).
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.GatewayConnection, error)
	// Deactivate flips the tenant's active connection for a provider to
	// inactive (soft disconnect). Returns domain.ErrGatewayConnectionNotFound
	// if there is no active connection.
	Deactivate(ctx context.Context, tenantID uuid.UUID, provider domain.GatewayProvider) error
}
