package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
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

// UpdateReason updates an open dispute's reason. It is scoped by customer_id
// (defense-in-depth): the portal is customer-facing, so the customer that owns
// the dispute — not a tenant admin — is the authorization boundary. The caller
// loads the dispute for the authenticated portal customer, so customerID is the
// verified owner and this can't touch another customer's dispute.
func (r *DisputeRepository) UpdateReason(ctx context.Context, id, customerID uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE invoice_disputes SET reason = $1 WHERE id = $2 AND customer_id = $3 AND status = 'open'`,
		reason, id, customerID,
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

// ListByCustomerIDPaged bounds the portal-facing dispute list.
func (r *DisputeRepository) ListByCustomerIDPaged(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*domain.InvoiceDispute, error) {
	query := `SELECT ` + disputeColumns + ` FROM invoice_disputes WHERE customer_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, customerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanDisputeRows(rows)
}

func (r *DisputeRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status string, limit, offset int) ([]*domain.InvoiceDispute, error) {
	query := `SELECT ` + disputeColumns + ` FROM invoice_disputes WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argIdx := 2
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanDisputeRows(rows)
}

func (r *DisputeRepository) Resolve(ctx context.Context, tenantID, id uuid.UUID, note string) error {
	return r.Close(ctx, tenantID, id, domain.DisputeStatusResolved, note)
}

// Close transitions an open dispute to a terminal status (resolved or rejected)
// with an optional note. Only 'open' rows are affected, so re-closing a
// terminal dispute is a no-op that returns ErrDisputeNotFound.
func (r *DisputeRepository) Close(ctx context.Context, tenantID, id uuid.UUID, status domain.DisputeStatus, note string) error {
	if status != domain.DisputeStatusResolved && status != domain.DisputeStatusRejected {
		return fmt.Errorf("invalid terminal dispute status %q", status)
	}
	var noteArg interface{}
	if note != "" {
		noteArg = note
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE invoice_disputes
			SET status = $1, note = $2, resolved_at = NOW()
			WHERE id = $3 AND tenant_id = $4 AND status = 'open'`,
		string(status), noteArg, id, tenantID,
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
