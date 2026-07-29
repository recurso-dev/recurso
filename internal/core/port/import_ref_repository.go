package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ImportRefRepository persists the source-id → Recurso-id mappings that make
// data imports idempotent.
type ImportRefRepository interface {
	// Create records one mapping. It must be a no-op-safe insert: a duplicate
	// (tenant, source, external_id) returns ErrDuplicateImportRef so the caller
	// can treat a concurrent/re-run collision as "already imported".
	Create(ctx context.Context, ref *domain.ImportExternalRef) error
	// ListExternalIDs returns the set of external ids already imported for the
	// tenant + source (used to build the idempotency skip-set before a run).
	ListExternalIDs(ctx context.Context, tenantID uuid.UUID, source string) (map[string]bool, error)
}
