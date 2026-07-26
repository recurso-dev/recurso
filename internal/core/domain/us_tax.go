package domain

import "github.com/google/uuid"

// TenantUSTaxConfig is a tenant's US tax identity (W-9), shown as the seller
// party on US (sales-tax) invoices. Presentation only — it does not affect tax
// computation, which is driven by the buyer address + nexus + provider.
type TenantUSTaxConfig struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	LegalName string    `json:"legal_name"`
	EIN       string    `json:"ein"` // Employer Identification Number (W-9 tax id)
	Address   string    `json:"address"`
}
