package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
)

type InvoiceRepository struct {
	db *sql.DB
}

func NewInvoiceRepository(db *sql.DB) port.InvoiceRepository {
	return &InvoiceRepository{db: db}
}

func (r *InvoiceRepository) Create(ctx context.Context, inv *domain.Invoice) error {
	query := `
		INSERT INTO invoices (
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid,
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, due_date, next_retry_at, retry_count,
			ack_date, e_invoice_retry_count, e_invoice_next_retry_at, e_invoice_error_message
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
	`
	// amount_paid default 0 if not set
	amountPaid := int64(0)
	if inv.PaidAt != nil {
		amountPaid = inv.Total
	}

	var eInvoiceStatus interface{} = inv.EInvoiceStatus
	if inv.EInvoiceStatus == "" {
		eInvoiceStatus = nil
	}

	_, err := r.db.ExecContext(ctx, query,
		inv.ID, inv.TenantID, inv.SubscriptionID, inv.CustomerID, inv.InvoiceNumber, inv.Status,
		inv.Currency, inv.Subtotal, inv.TaxAmount, inv.Total, amountPaid,
		inv.IGSTAmount, inv.CGSTAmount, inv.SGSTAmount, inv.HSNCode, inv.IRN, inv.AckNo,
		inv.SignedQRCode, eInvoiceStatus, inv.TDSAmount,
		inv.CreatedAt, inv.DueDate, inv.NextRetryAt, inv.RetryCount,
		inv.AckDate, inv.EInvoiceRetryCount, inv.EInvoiceNextRetryAt, inv.EInvoiceErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to insert invoice: %w", err)
	}
	return nil
}

// CreateWithTx creates an invoice within an existing transaction for atomic operations.
func (r *InvoiceRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, inv *domain.Invoice) error {
	query := `
		INSERT INTO invoices (
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid,
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, due_date, next_retry_at, retry_count,
			ack_date, e_invoice_retry_count, e_invoice_next_retry_at, e_invoice_error_message
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
	`
	amountPaid := int64(0)
	if inv.PaidAt != nil {
		amountPaid = inv.Total
	}

	var eInvoiceStatus interface{} = inv.EInvoiceStatus
	if inv.EInvoiceStatus == "" {
		eInvoiceStatus = nil
	}

	_, err := tx.ExecContext(ctx, query,
		inv.ID, inv.TenantID, inv.SubscriptionID, inv.CustomerID, inv.InvoiceNumber, inv.Status,
		inv.Currency, inv.Subtotal, inv.TaxAmount, inv.Total, amountPaid,
		inv.IGSTAmount, inv.CGSTAmount, inv.SGSTAmount, inv.HSNCode, inv.IRN, inv.AckNo,
		inv.SignedQRCode, eInvoiceStatus, inv.TDSAmount,
		inv.CreatedAt, inv.DueDate, inv.NextRetryAt, inv.RetryCount,
		inv.AckDate, inv.EInvoiceRetryCount, inv.EInvoiceNextRetryAt, inv.EInvoiceErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to insert invoice in tx: %w", err)
	}
	return nil
}

func (r *InvoiceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id missing from context")
	}

	return r.getByIDInternal(ctx, id, &tenantID)
}

// GetByIDPublic fetches invoice without tenant context check (for public pages)
func (r *InvoiceRepository) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return r.getByIDInternal(ctx, id, nil)
}

func (r *InvoiceRepository) getByIDInternal(ctx context.Context, id uuid.UUID, tenantID *uuid.UUID) (*domain.Invoice, error) {
	inv := &domain.Invoice{}
	var amountPaid int64

	query := `
		SELECT
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid,
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, due_date, paid_at, next_retry_at, retry_count,
			COALESCE(ack_date, ''), e_invoice_retry_count,
			e_invoice_next_retry_at, COALESCE(e_invoice_error_message, '')
		FROM invoices WHERE id = $1
	`
	args := []interface{}{id}
	if tenantID != nil {
		query += " AND tenant_id = $2"
		args = append(args, *tenantID)
	}

	var hsnCode, irn, signedQRCode, eInvoiceStatus, ackNo sql.NullString

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
		&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid,
		&inv.IGSTAmount, &inv.CGSTAmount, &inv.SGSTAmount, &hsnCode, &irn, &ackNo,
		&signedQRCode, &eInvoiceStatus, &inv.TDSAmount,
		&inv.CreatedAt, &inv.DueDate, &inv.PaidAt, &inv.NextRetryAt, &inv.RetryCount,
		&inv.AckDate, &inv.EInvoiceRetryCount,
		&inv.EInvoiceNextRetryAt, &inv.EInvoiceErrorMessage,
	)

	inv.HSNCode = hsnCode.String
	inv.IRN = irn.String
	inv.AckNo = ackNo.String
	inv.SignedQRCode = signedQRCode.String
	inv.EInvoiceStatus = eInvoiceStatus.String
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	return inv, nil
}

func (r *InvoiceRepository) Update(ctx context.Context, inv *domain.Invoice) error {
	query := `
		UPDATE invoices
		SET status = $1, amount_paid = $2, paid_at = $3, next_retry_at = $4, retry_count = $5,
		    tds_amount = $6, signed_qr_code = $7, e_invoice_status = $8, irn = $9,
		    ack_no = $10, ack_date = $11, e_invoice_retry_count = $12,
		    e_invoice_next_retry_at = $13, e_invoice_error_message = $14
		WHERE id = $15
	`
	amountPaid := inv.Total // Simplification as we update usually for generic payment

	_, err := r.db.ExecContext(ctx, query,
		inv.Status, amountPaid, inv.PaidAt, inv.NextRetryAt, inv.RetryCount,
		inv.TDSAmount, inv.SignedQRCode, inv.EInvoiceStatus, inv.IRN,
		inv.AckNo, inv.AckDate, inv.EInvoiceRetryCount,
		inv.EInvoiceNextRetryAt, inv.EInvoiceErrorMessage,
		inv.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}
	return nil
}

func (r *InvoiceRepository) GetDueForRetry(ctx context.Context) ([]*domain.Invoice, error) {
	query := `
		SELECT 
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid,
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, due_date, paid_at, next_retry_at, retry_count
		FROM invoices 
		WHERE status IN ('open', 'past_due') 
		  AND next_retry_at IS NOT NULL 
		  AND next_retry_at <= $1
		LIMIT 10
	`
	rows, err := r.db.QueryContext(ctx, query, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query retry invoices: %w", err)
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		inv := &domain.Invoice{}
		var amountPaid int64
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
			&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid,
			&inv.IGSTAmount, &inv.CGSTAmount, &inv.SGSTAmount, &inv.HSNCode, &inv.IRN, &inv.AckNo,
			&inv.SignedQRCode, &inv.EInvoiceStatus, &inv.TDSAmount,
			&inv.CreatedAt, &inv.DueDate, &inv.PaidAt, &inv.NextRetryAt, &inv.RetryCount,
		); err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

func (r *InvoiceRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error) {
	query := `
		SELECT 
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid,
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, due_date, paid_at, next_retry_at, retry_count
		FROM invoices 
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch customer invoices: %w", err)
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		inv := &domain.Invoice{}
		var amountPaid int64
		var hsnCode, irn, signedQRCode, eInvoiceStatus, ackNo sql.NullString
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
			&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid,
			&inv.IGSTAmount, &inv.CGSTAmount, &inv.SGSTAmount, &hsnCode, &irn, &ackNo,
			&signedQRCode, &eInvoiceStatus, &inv.TDSAmount,
			&inv.CreatedAt, &inv.DueDate, &inv.PaidAt, &inv.NextRetryAt, &inv.RetryCount,
		); err != nil {
			return nil, err
		}
		inv.HSNCode = hsnCode.String
		inv.IRN = irn.String
		inv.AckNo = ackNo.String
		inv.SignedQRCode = signedQRCode.String
		inv.EInvoiceStatus = eInvoiceStatus.String
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

func (r *InvoiceRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invoice, error) {
	query := `
		SELECT 
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid,
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, due_date, paid_at, next_retry_at, retry_count
		FROM invoices 
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invoices: %w", err)
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		inv := &domain.Invoice{}
		var amountPaid int64
		var hsnCode, irn, signedQRCode, eInvoiceStatus, ackNo sql.NullString
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
			&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid,
			&inv.IGSTAmount, &inv.CGSTAmount, &inv.SGSTAmount, &hsnCode, &irn, &ackNo,
			&signedQRCode, &eInvoiceStatus, &inv.TDSAmount,
			&inv.CreatedAt, &inv.DueDate, &inv.PaidAt, &inv.NextRetryAt, &inv.RetryCount,
		); err != nil {
			return nil, err
		}
		inv.HSNCode = hsnCode.String
		inv.IRN = irn.String
		inv.AckNo = ackNo.String
		inv.SignedQRCode = signedQRCode.String
		inv.EInvoiceStatus = eInvoiceStatus.String
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

// GetOverdueInvoices returns unpaid invoices that are past due
func (r *InvoiceRepository) GetOverdueInvoices(ctx context.Context) ([]domain.OverdueInvoice, error) {
	query := `
		SELECT 
			i.id, i.tenant_id, i.customer_id,
			c.name as customer_name, c.email as customer_email,
			i.invoice_number, i.total as amount, i.currency,
			i.due_date, i.retry_count, i.next_retry_at
		FROM invoices i
		JOIN customers c ON i.customer_id = c.id
		WHERE i.status IN ('open', 'past_due')
			AND i.due_date < CURRENT_TIMESTAMP
			AND (i.next_retry_at IS NULL OR i.next_retry_at <= CURRENT_TIMESTAMP)
		ORDER BY i.due_date ASC
		LIMIT 50
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query overdue invoices: %w", err)
	}
	defer rows.Close()

	var invoices []domain.OverdueInvoice
	for rows.Next() {
		var inv domain.OverdueInvoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.CustomerID,
			&inv.CustomerName, &inv.CustomerEmail,
			&inv.InvoiceNumber, &inv.Amount, &inv.Currency,
			&inv.DueDate, &inv.RetryCount, &inv.NextRetryAt,
		); err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

// UpdateRetryInfo updates the retry count and next retry date
func (r *InvoiceRepository) UpdateRetryInfo(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int) error {
	query := `
		UPDATE invoices 
		SET next_retry_at = $1, retry_count = $2, status = 'past_due'
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, nextRetry, retryCount, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to update retry info: %w", err)
	}
	return nil
}

// GetFailedEInvoices fetches FAILED e-invoices that are due for retry
func (r *InvoiceRepository) GetFailedEInvoices(ctx context.Context) ([]*domain.Invoice, error) {
	query := `
		SELECT
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid,
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, due_date, paid_at, next_retry_at, retry_count,
			COALESCE(ack_date, ''), e_invoice_retry_count,
			e_invoice_next_retry_at, COALESCE(e_invoice_error_message, '')
		FROM invoices
		WHERE e_invoice_status = 'FAILED'
		  AND e_invoice_next_retry_at IS NOT NULL
		  AND e_invoice_next_retry_at <= $1
		ORDER BY e_invoice_next_retry_at ASC
		LIMIT 20
	`
	rows, err := r.db.QueryContext(ctx, query, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query failed e-invoices: %w", err)
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		inv := &domain.Invoice{}
		var amountPaid int64
		var hsnCode, irn, signedQRCode, eInvoiceStatus, ackNo sql.NullString
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
			&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid,
			&inv.IGSTAmount, &inv.CGSTAmount, &inv.SGSTAmount, &hsnCode, &irn, &ackNo,
			&signedQRCode, &eInvoiceStatus, &inv.TDSAmount,
			&inv.CreatedAt, &inv.DueDate, &inv.PaidAt, &inv.NextRetryAt, &inv.RetryCount,
			&inv.AckDate, &inv.EInvoiceRetryCount,
			&inv.EInvoiceNextRetryAt, &inv.EInvoiceErrorMessage,
		); err != nil {
			return nil, err
		}
		inv.HSNCode = hsnCode.String
		inv.IRN = irn.String
		inv.AckNo = ackNo.String
		inv.SignedQRCode = signedQRCode.String
		inv.EInvoiceStatus = eInvoiceStatus.String
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

// UpdateEInvoiceStatus updates e-invoice specific fields on an invoice
func (r *InvoiceRepository) UpdateEInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status, irn, ackNo, signedQR, ackDate, errorMsg string) error {
	query := `
		UPDATE invoices
		SET e_invoice_status = $1, irn = $2, ack_no = $3, signed_qr_code = $4,
		    ack_date = $5, e_invoice_error_message = $6
		WHERE id = $7
	`
	_, err := r.db.ExecContext(ctx, query, status, irn, ackNo, signedQR, ackDate, errorMsg, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to update e-invoice status: %w", err)
	}
	return nil
}

// MarkAsUncollectible marks an invoice as uncollectible after max retries
func (r *InvoiceRepository) MarkAsUncollectible(ctx context.Context, invoiceID uuid.UUID) error {
	query := `
		UPDATE invoices 
		SET status = 'uncollectible', next_retry_at = NULL
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to mark invoice as uncollectible: %w", err)
	}
	return nil
}
