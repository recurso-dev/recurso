package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// IntegrationCategory groups the operator-style integrations that a tenant can
// bring their own credentials for (the non-gateway BYO surface).
type IntegrationCategory string

const (
	IntegrationTax     IntegrationCategory = "tax"
	IntegrationCRM     IntegrationCategory = "crm"
	IntegrationStorage IntegrationCategory = "storage"
)

// ValidIntegration reports whether (category, provider) is a supported pair.
func ValidIntegration(category IntegrationCategory, provider string) bool {
	switch category {
	case IntegrationTax:
		return provider == "taxjar" || provider == "avalara"
	case IntegrationCRM:
		return provider == "hubspot"
	case IntegrationStorage:
		return provider == "s3"
	}
	return false
}

// IntegrationConnection is a tenant's own credentials for one integration. The
// provider's config (API keys and any endpoint/region fields) is sealed as one
// AES-256-GCM JSON blob in ConfigEnc — never plaintext, never serialized.
type IntegrationConnection struct {
	ID        uuid.UUID           `db:"id" json:"id"`
	TenantID  uuid.UUID           `db:"tenant_id" json:"-"`
	Category  IntegrationCategory `db:"category" json:"category"`
	Provider  string              `db:"provider" json:"provider"`
	ConfigEnc string              `db:"config_enc" json:"-"`
	Active    bool                `db:"active" json:"active"`
	CreatedAt time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt time.Time           `db:"updated_at" json:"updated_at"`
}

// Integration connection domain errors.
var (
	ErrIntegrationConnectionNotFound = errors.New("integration connection not found")
)
