package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entity is a legal entity under a tenant (e.g. "ACME Inc (US)"). Each entity has
// its own TigerBeetle ledger (TBLedgerID) and its own gapless invoice series
// (InvoicePrefix + entity_invoice_sequences). Every tenant has exactly one
// IsPrimary entity — the backfill target that keeps the tenant's original
// LedgerID of 1, so single-entity tenants are unchanged. (Multi-Entity Books, Inc 1.)
type Entity struct {
	ID            uuid.UUID `json:"id" db:"id"`
	TenantID      uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name          string    `json:"name" db:"name"`
	LegalName     string    `json:"legal_name" db:"legal_name"`
	IsPrimary     bool      `json:"is_primary" db:"is_primary"`
	TBLedgerID    int       `json:"tb_ledger_id" db:"tb_ledger_id"`
	InvoicePrefix string    `json:"invoice_prefix" db:"invoice_prefix"`
	CountryCode   string    `json:"country_code" db:"country_code"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}
