package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID   uuid.UUID `json:"id" db:"id"`
	Name string    `json:"name" db:"name"`
	// OwnerTenantID is the tenant that created and owns the organization. Every
	// /v1/organizations operation is scoped to it — a caller may only see and
	// manage organizations its own tenant owns.
	OwnerTenantID uuid.UUID `json:"owner_tenant_id" db:"owner_tenant_id"`
	OwnerEmail    string    `json:"owner_email" db:"owner_email"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

var (
	// ErrOrganizationNotFound is returned when an organization does not exist or
	// is not owned by the calling tenant. The two cases are deliberately merged
	// so a caller cannot probe for the existence of another tenant's orgs.
	ErrOrganizationNotFound = errors.New("organization not found")
	// ErrCrossTenantAttach is returned when a caller tries to attach a tenant
	// other than its own to an organization. Pulling a foreign tenant in without
	// that tenant's consent would expose its revenue via consolidated reporting.
	ErrCrossTenantAttach = errors.New("cannot attach a tenant other than your own to an organization")
)
