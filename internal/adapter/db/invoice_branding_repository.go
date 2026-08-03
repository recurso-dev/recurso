package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// InvoiceBrandingRepository persists a tenant's invoice presentation settings
// (logo, signature, bank details, terms). Mirrors the W-9 config repository.
type InvoiceBrandingRepository struct {
	db *sql.DB
}

func NewInvoiceBrandingRepository(db *sql.DB) *InvoiceBrandingRepository {
	return &InvoiceBrandingRepository{db: db}
}

// GetByTenantID returns the tenant's branding, or (nil, nil) when none is set —
// the invoice then falls back to the env seller identity.
func (r *InvoiceBrandingRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantInvoiceBranding, error) {
	b := &domain.TenantInvoiceBranding{}
	err := r.db.QueryRowContext(ctx,
		`SELECT tenant_id, company_name, logo_data_url, signature_data_url, signatory_name, bank_details, terms, updated_at
		 FROM tenant_invoice_branding WHERE tenant_id = $1`, tenantID,
	).Scan(&b.TenantID, &b.CompanyName, &b.LogoDataURL, &b.SignatureDataURL, &b.SignatoryName, &b.BankDetails, &b.Terms, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice branding: %w", err)
	}
	return b, nil
}

// Upsert writes the tenant's invoice branding.
func (r *InvoiceBrandingRepository) Upsert(ctx context.Context, b *domain.TenantInvoiceBranding) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenant_invoice_branding
		   (tenant_id, company_name, logo_data_url, signature_data_url, signatory_name, bank_details, terms, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   company_name = EXCLUDED.company_name,
		   logo_data_url = EXCLUDED.logo_data_url,
		   signature_data_url = EXCLUDED.signature_data_url,
		   signatory_name = EXCLUDED.signatory_name,
		   bank_details = EXCLUDED.bank_details,
		   terms = EXCLUDED.terms,
		   updated_at = NOW()`,
		b.TenantID, b.CompanyName, b.LogoDataURL, b.SignatureDataURL, b.SignatoryName, b.BankDetails, b.Terms,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert invoice branding: %w", err)
	}
	return nil
}
