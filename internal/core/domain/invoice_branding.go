package domain

import (
	"time"

	"github.com/google/uuid"
)

// TenantInvoiceBranding is a tenant's invoice presentation settings: the
// display name, logo, signature image, signatory, bank details and terms
// rendered on their invoice documents. It is presentation only — statutory
// seller identity (GST legal entity, US W-9) is configured separately and
// overrides the display name on tax invoices where set.
//
// Images are stored as validated data URLs (data:image/png|jpeg;base64,…,
// decoded size-capped) so no blob storage is needed and the PDF stays a
// single self-contained document.
type TenantInvoiceBranding struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	CompanyName      string    `json:"company_name"`
	LogoDataURL      string    `json:"logo_data_url"`
	SignatureDataURL string    `json:"signature_data_url"`
	SignatoryName    string    `json:"signatory_name"`
	BankDetails      string    `json:"bank_details"`
	Terms            string    `json:"terms"`
	UpdatedAt        time.Time `json:"updated_at"`
}
