package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TenantEUConfigRepository persists a tenant's isolated EU e-invoicing config.
type TenantEUConfigRepository struct {
	db *sql.DB
}

func NewTenantEUConfigRepository(db *sql.DB) *TenantEUConfigRepository {
	return &TenantEUConfigRepository{db: db}
}

// GetByTenantID returns the tenant's EU config, or nil when none is set (EU
// e-invoicing simply stays off for that tenant).
func (r *TenantEUConfigRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantEUConfig, error) {
	c := &domain.TenantEUConfig{}
	err := r.db.QueryRowContext(ctx,
		`SELECT tenant_id, enabled, legal_name, vat_number, country_code, street, city, postal_zone
		   FROM tenant_eu_config WHERE tenant_id = $1`, tenantID,
	).Scan(&c.TenantID, &c.Enabled, &c.LegalName, &c.VATNumber, &c.CountryCode, &c.Street, &c.City, &c.PostalZone)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get EU config: %w", err)
	}
	return c, nil
}

// Upsert creates or replaces the tenant's EU config.
func (r *TenantEUConfigRepository) Upsert(ctx context.Context, c *domain.TenantEUConfig) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenant_eu_config (tenant_id, enabled, legal_name, vat_number, country_code, street, city, postal_zone, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		 ON CONFLICT (tenant_id) DO UPDATE SET
		   enabled = EXCLUDED.enabled, legal_name = EXCLUDED.legal_name, vat_number = EXCLUDED.vat_number,
		   country_code = EXCLUDED.country_code, street = EXCLUDED.street, city = EXCLUDED.city,
		   postal_zone = EXCLUDED.postal_zone, updated_at = NOW()`,
		c.TenantID, c.Enabled, c.LegalName, c.VATNumber, c.CountryCode, c.Street, c.City, c.PostalZone,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert EU config: %w", err)
	}
	return nil
}

// EUInvoiceRepository persists generated EN 16931 documents + delivery status.
type EUInvoiceRepository struct {
	db *sql.DB
}

func NewEUInvoiceRepository(db *sql.DB) *EUInvoiceRepository {
	return &EUInvoiceRepository{db: db}
}

// Upsert stores (or replaces) the e-invoice record for an invoice — idempotent
// on invoice_id, so re-generating overwrites rather than duplicating.
func (r *EUInvoiceRepository) Upsert(ctx context.Context, e *domain.EUInvoice) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO eu_einvoices (id, tenant_id, invoice_id, syntax, status, document, message_id, error_message, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		 ON CONFLICT (invoice_id) DO UPDATE SET
		   syntax = EXCLUDED.syntax, status = EXCLUDED.status, document = EXCLUDED.document,
		   message_id = EXCLUDED.message_id, error_message = EXCLUDED.error_message, updated_at = NOW()`,
		e.ID, e.TenantID, e.InvoiceID, string(e.Syntax), string(e.Status), e.Document, e.MessageID, e.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert EU e-invoice: %w", err)
	}
	return nil
}

// GetByInvoiceID returns the stored e-invoice for an invoice, or nil.
func (r *EUInvoiceRepository) GetByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*domain.EUInvoice, error) {
	e := &domain.EUInvoice{}
	var syntax, status string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, invoice_id, syntax, status, document, message_id, error_message, created_at, updated_at
		   FROM eu_einvoices WHERE invoice_id = $1`, invoiceID,
	).Scan(&e.ID, &e.TenantID, &e.InvoiceID, &syntax, &status, &e.Document, &e.MessageID, &e.ErrorMessage, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get EU e-invoice: %w", err)
	}
	e.Syntax = domain.EUInvoiceSyntax(syntax)
	e.Status = domain.EUInvoiceStatus(status)
	return e, nil
}
