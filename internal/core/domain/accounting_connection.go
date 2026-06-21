package domain

import (
	"time"

	"github.com/google/uuid"
)

type AccountingConnection struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Provider       string     `json:"provider" db:"provider"`
	AccessToken    string     `json:"-" db:"access_token"`
	RefreshToken   string     `json:"-" db:"refresh_token"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty" db:"token_expires_at"`
	RealmID        string     `json:"realm_id,omitempty" db:"realm_id"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty" db:"last_sync_at"`
	SyncStatus     string     `json:"sync_status" db:"sync_status"`
	LastError      string     `json:"last_error,omitempty" db:"last_error"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

type AccountingSyncLog struct {
	ID           uuid.UUID `json:"id" db:"id"`
	TenantID     uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ConnectionID uuid.UUID `json:"connection_id" db:"connection_id"`
	EntityType   string    `json:"entity_type" db:"entity_type"`
	EntityID     uuid.UUID `json:"entity_id" db:"entity_id"`
	ExternalID   string    `json:"external_id,omitempty" db:"external_id"`
	Action       string    `json:"action" db:"action"`
	Status       string    `json:"status" db:"status"`
	ErrorMessage string    `json:"error_message,omitempty" db:"error_message"`
	SyncedAt     time.Time `json:"synced_at" db:"synced_at"`
}
