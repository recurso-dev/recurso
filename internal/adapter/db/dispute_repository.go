package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// DisputeRepository implements port.DisputeRepository.
type DisputeRepository struct {
	db *sql.DB
}

func NewDisputeRepository(db *sql.DB) *DisputeRepository {
	return &DisputeRepository{db: db}
}

const disputeColumns = `id, tenant_id, invoice_id, customer_id, reason, status, note, created_at, resolved_at`

func scanDispute(row interface{ Scan(...interface{}) error }) (*domain.InvoiceDispute, error) {
	var d domain.InvoiceDispute
	var note sql.NullString
	var resolvedAt sql.NullTime
	if err := row.Scan(
		&d.ID, &d.TenantID, &d.InvoiceID, &d.CustomerID, &d.Reason, &d.Status,
		&note, &d.CreatedAt, &resolvedAt,
	); err != nil {
		return nil, err
	}
	if note.Valid {
		d.Note = &note.String
	}
	if resolvedAt.Valid {
		d.ResolvedAt = &resolvedAt.Time
	}
	return &d, nil
}

func (r *DisputeRepository) Create(ctx context.Context, d *domain.InvoiceDispute) error {
	query := `
		INSERT INTO invoice_disputes (id, tenant_id, invoice_id, customer_id, reason, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		d.ID, d.TenantID, d.InvoiceID, d.CustomerID, d.Reason, d.Status,
	)
	return err
}

func (r *DisputeRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.InvoiceDispute, error) {
	query := `SELECT ` + disputeColumns + ` FROM invoice_disputes WHERE id = $1`
	return scanDispute(r.db.QueryRowContext(ctx, query, id))
}

func (r *DisputeRepository) GetOpenByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*domain.InvoiceDispute, error) {
	query := `SELECT ` + disputeColumns + ` FROM invoice_disputes WHERE invoice_id = $1 AND status = 'open'`
	d, err := scanDispute(r.db.QueryRowContext(ctx, query, invoiceID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return d, err
}

func (r *DisputeRepository) UpdateReason(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE invoice_disputes SET reason = $1 WHERE id = $2 AND status = 'open'`,
		reason, id,
	)
	return err
}

func (r *DisputeRepository) ListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.InvoiceDispute, error) {
	query := `SELECT ` + disputeColumns + ` FROM invoice_disputes WHERE customer_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanDisputeRows(rows)
}

func (r *DisputeRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]*domain.InvoiceDispute, error) {
	query := `SELECT ` + disputeColumns + ` FROM invoice_disputes WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanDisputeRows(rows)
}

func (r *DisputeRepository) Resolve(ctx context.Context, tenantID, id uuid.UUID, note string) error {
	var noteArg interface{}
	if note != "" {
		noteArg = note
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE invoice_disputes
			SET status = 'resolved', note = $1, resolved_at = NOW()
			WHERE id = $2 AND tenant_id = $3 AND status = 'open'`,
		noteArg, id, tenantID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrDisputeNotFound
	}
	return nil
}

func scanDisputeRows(rows *sql.Rows) ([]*domain.InvoiceDispute, error) {
	disputes := []*domain.InvoiceDispute{}
	for rows.Next() {
		d, err := scanDispute(rows)
		if err != nil {
			return nil, err
		}
		disputes = append(disputes, d)
	}
	return disputes, rows.Err()
}
