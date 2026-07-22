package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

// GetByTenantEntity resolves the EU seller config for a specific issuing entity
// (Multi-Entity Books Inc 3b). A nil entityID matches the tenant/primary default
// (entity_id IS NULL); a non-primary entity matches only its own row, so an
// unconfigured non-primary entity returns (nil, nil) and its EU e-invoice is
// skipped rather than filed under the default's VAT id.
func (r *TenantEUConfigRepository) GetByTenantEntity(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID) (*domain.TenantEUConfig, error) {
	c := &domain.TenantEUConfig{}
	err := r.db.QueryRowContext(ctx,
		`SELECT tenant_id, enabled, legal_name, vat_number, country_code, street, city, postal_zone
		   FROM tenant_eu_config WHERE tenant_id = $1 AND entity_id IS NOT DISTINCT FROM $2`, tenantID, entityID,
	).Scan(&c.TenantID, &c.Enabled, &c.LegalName, &c.VATNumber, &c.CountryCode, &c.Street, &c.City, &c.PostalZone)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get EU config: %w", err)
	}
	return c, nil
}

// GetByTenantID returns the tenant/primary default EU config (entity_id NULL),
// or nil when none is set (EU e-invoicing simply stays off for that tenant).
func (r *TenantEUConfigRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantEUConfig, error) {
	return r.GetByTenantEntity(ctx, tenantID, nil)
}

// Upsert creates or replaces the tenant's EU config.
func (r *TenantEUConfigRepository) Upsert(ctx context.Context, c *domain.TenantEUConfig) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tenant_eu_config (tenant_id, enabled, legal_name, vat_number, country_code, street, city, postal_zone, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		 ON CONFLICT (tenant_id) WHERE entity_id IS NULL DO UPDATE SET
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
// on invoice_id, so re-generating overwrites rather than duplicating. Regenerating
// resets the delivery retry state (recipient/count/next_retry_at) from the record.
func (r *EUInvoiceRepository) Upsert(ctx context.Context, e *domain.EUInvoice) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO eu_einvoices (id, tenant_id, invoice_id, syntax, status, document, recipient_vat_id, message_id, error_message, retry_count, next_retry_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
		 ON CONFLICT (invoice_id) DO UPDATE SET
		   syntax = EXCLUDED.syntax, status = EXCLUDED.status, document = EXCLUDED.document,
		   recipient_vat_id = EXCLUDED.recipient_vat_id, message_id = EXCLUDED.message_id,
		   error_message = EXCLUDED.error_message, retry_count = EXCLUDED.retry_count,
		   next_retry_at = EXCLUDED.next_retry_at, updated_at = NOW()`,
		e.ID, e.TenantID, e.InvoiceID, string(e.Syntax), string(e.Status), e.Document,
		e.RecipientVATID, e.MessageID, e.ErrorMessage, e.RetryCount, e.NextRetryAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert EU e-invoice: %w", err)
	}
	return nil
}

const euInvoiceColumns = `id, tenant_id, invoice_id, syntax, status, document, recipient_vat_id, message_id, error_message, retry_count, next_retry_at, created_at, updated_at`

// scanEUInvoice reads one row in euInvoiceColumns order.
func scanEUInvoice(s interface{ Scan(...any) error }) (*domain.EUInvoice, error) {
	e := &domain.EUInvoice{}
	var syntax, status string
	if err := s.Scan(&e.ID, &e.TenantID, &e.InvoiceID, &syntax, &status, &e.Document,
		&e.RecipientVATID, &e.MessageID, &e.ErrorMessage, &e.RetryCount, &e.NextRetryAt,
		&e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	e.Syntax = domain.EUInvoiceSyntax(syntax)
	e.Status = domain.EUInvoiceStatus(status)
	return e, nil
}

// GetByInvoiceID returns the stored e-invoice for an invoice, or nil.
func (r *EUInvoiceRepository) GetByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*domain.EUInvoice, error) {
	e, err := scanEUInvoice(r.db.QueryRowContext(ctx,
		`SELECT `+euInvoiceColumns+` FROM eu_einvoices WHERE invoice_id = $1`, invoiceID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get EU e-invoice: %w", err)
	}
	return e, nil
}

// ClaimFailedEUInvoices atomically claims up to `limit` delivery-failed records
// whose next_retry_at is due, leasing each by pushing next_retry_at out to
// `leaseUntil` so a second instance (or a crashed runner's row) doesn't
// re-transmit the same document. FOR UPDATE SKIP LOCKED is the claim primitive
// (ADR-003) — the scheduler lock is a no-op without Redis, so this is what makes
// the poll safe across instances. `now`/`leaseUntil` must be UTC to match the
// TIMESTAMPTZ column comparison.
//
// Only rows with a non-empty document are eligible: a generation failure has no
// document to re-transmit and is left for manual correction (its next_retry_at is
// NULL, so it never appears here anyway — the document filter is belt-and-braces).
func (r *EUInvoiceRepository) ClaimFailedEUInvoices(ctx context.Context, now, leaseUntil time.Time, limit int) ([]*domain.EUInvoice, error) {
	rows, err := r.db.QueryContext(ctx,
		`UPDATE eu_einvoices SET next_retry_at = $2, updated_at = NOW()
		   WHERE id IN (
		     SELECT id FROM eu_einvoices
		       WHERE status = 'failed' AND document <> '' AND next_retry_at IS NOT NULL AND next_retry_at <= $1
		       ORDER BY next_retry_at ASC
		       LIMIT $3
		       FOR UPDATE SKIP LOCKED
		   )
		 RETURNING `+euInvoiceColumns,
		now, leaseUntil, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to claim failed EU e-invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*domain.EUInvoice
	for rows.Next() {
		e, err := scanEUInvoice(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan claimed EU e-invoice: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpdateDelivery persists the outcome of a delivery attempt (status, message id,
// error, retry count, next retry) by id. The worker holds the leased row and is
// the sole writer, so this is a plain targeted update — no read-modify-write race.
func (r *EUInvoiceRepository) UpdateDelivery(ctx context.Context, e *domain.EUInvoice) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE eu_einvoices
		    SET status = $2, message_id = $3, error_message = $4,
		        retry_count = $5, next_retry_at = $6, updated_at = NOW()
		  WHERE id = $1`,
		e.ID, string(e.Status), e.MessageID, e.ErrorMessage, e.RetryCount, e.NextRetryAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update EU e-invoice delivery: %w", err)
	}
	return nil
}
