package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrDuplicateImportRef is returned when a (tenant, source, external_id) mapping
// already exists — the idempotency signal that the object was already imported.
var ErrDuplicateImportRef = errors.New("import reference already exists")

// Import source identifiers (the `source` column of import_external_refs).
const (
	ImportSourceStripe = "stripe"
)

// Import object kinds (the `kind` column).
const (
	ImportKindCustomer     = "customer"
	ImportKindPlan         = "plan"
	ImportKindSubscription = "subscription"
)

// ImportExternalRef maps a source system's object id (e.g. a Stripe customer or
// price id) to the Recurso record created from it, so re-running an import is
// idempotent — an already-mapped external id is skipped, not duplicated.
type ImportExternalRef struct {
	ID         uuid.UUID `db:"id"`
	TenantID   uuid.UUID `db:"tenant_id"`
	Source     string    `db:"source"`
	Kind       string    `db:"kind"`
	ExternalID string    `db:"external_id"`
	RecursoID  uuid.UUID `db:"recurso_id"`
	CreatedAt  time.Time `db:"created_at"`
}
