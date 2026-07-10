package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	DataRegion     string     `json:"data_region" db:"data_region"`
	BaseCurrency   string     `json:"base_currency" db:"base_currency"` // Default: "USD"
	OrganizationID *uuid.UUID `json:"organization_id,omitempty" db:"organization_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// IRPConfig holds per-tenant IRP (Invoice Registration Portal) credentials
type IRPConfig struct {
	ID           string `json:"id" db:"id"`
	TenantID     string `json:"tenant_id" db:"tenant_id"`
	Environment  string `json:"environment" db:"environment"` // "sandbox" or "production"
	ClientID     string `json:"client_id" db:"client_id"`
	ClientSecret string `json:"client_secret" db:"client_secret"`
	Username     string `json:"username" db:"username"`
	Password     string `json:"password" db:"password"`
	GSTIN        string `json:"gstin" db:"gstin"`
	IsEnabled    bool   `json:"is_enabled" db:"is_enabled"`
}

type APIKey struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	KeyValue  string    `json:"key_value"`            // Original key (only shown once at creation)
	KeyHash   string    `json:"-"`                    // bcrypt hash (stored in DB)
	KeyPrefix string    `json:"key_prefix,omitempty"` // First 8 chars (for lookup + display)
	Type      string    `json:"type"`                 // "secret"
	IsActive  bool      `json:"is_active"`
	Livemode  bool      `json:"livemode"` // true = rsk_live_ (real money), false = rsk_test_
	CreatedAt time.Time `json:"created_at"`
}

// NewAPIKeyValue builds a fresh secret key string for the given mode. Live keys
// are prefixed rsk_live_, test keys rsk_test_ — the prefix is what the auth
// layer gates against, so a test key can never run on a live-money server.
func NewAPIKeyValue(livemode bool, randomPart string) string {
	if livemode {
		return "rsk_live_" + randomPart
	}
	return "rsk_test_" + randomPart
}
