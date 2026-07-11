package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TaxNexusRepository stores the US states a tenant has declared sales-tax
// nexus in. All methods are tenant-scoped.
type TaxNexusRepository interface {
	// ListByTenant returns the tenant's declared nexus states.
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.TaxNexus, error)
	// SetStates replaces the tenant's entire nexus set (used by the PUT config).
	SetStates(ctx context.Context, tenantID uuid.UUID, states []domain.TaxNexus) error
	// Delete removes one state.
	Delete(ctx context.Context, tenantID uuid.UUID, stateCode string) error
	// NexusFor tells the tax resolver, in one query, whether the tenant has
	// declared ANY nexus (declaredAny) and whether it has nexus in stateCode
	// (inState). declaredAny gates the opt-in behaviour: a tenant with zero
	// declared nexus is never gated.
	NexusFor(ctx context.Context, tenantID uuid.UUID, stateCode string) (declaredAny, inState bool, err error)
}
