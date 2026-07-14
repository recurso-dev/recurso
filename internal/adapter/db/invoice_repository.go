package db

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type InvoiceRepository struct {
	db    *sql.DB
	items *InvoiceItemRepository
}

func NewInvoiceRepository(db *sql.DB) port.InvoiceRepository {
	return &InvoiceRepository{db: db, items: NewInvoiceItemRepository(db)}
}

const invoiceInsertQuery = `
	INSERT INTO invoices (
		id, tenant_id, subscription_id, customer_id, invoice_number, status,
		currency, subtotal, tax_amount, total, amount_paid,
		igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
		signed_qr_code, e_invoice_status, tds_amount,
		created_at, due_date, next_retry_at, retry_count,
		ack_date, e_invoice_retry_count, e_invoice_next_retry_at, e_invoice_error_message,
		dunning_action_id, dunning_context_key, last_payment_error, dunning_managed_by,
		credit_applied, mandate_cycle_key
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34)
`

// insertInvoiceRow writes the invoice row against any execer (*sql.DB or *sql.Tx).
func insertInvoiceRow(ctx context.Context, ex execer, inv *domain.Invoice) error {
	// amount_paid default 0 if not set
	amountPaid := int64(0)
	if inv.PaidAt != nil {
		amountPaid = inv.Total
	}

	var eInvoiceStatus interface{} = inv.EInvoiceStatus
	if inv.EInvoiceStatus == "" {
		eInvoiceStatus = nil
	}

	managedBy := inv.DunningManagedBy
	if managedBy == "" {
		managedBy = "scheduler"
	}

	_, err := ex.ExecContext(ctx, invoiceInsertQuery,
		inv.ID, inv.TenantID, inv.SubscriptionID, inv.CustomerID, inv.InvoiceNumber, inv.Status,
		inv.Currency, inv.Subtotal, inv.TaxAmount, inv.Total, amountPaid,
		inv.IGSTAmount, inv.CGSTAmount, inv.SGSTAmount, inv.HSNCode, inv.IRN, inv.AckNo,
		inv.SignedQRCode, eInvoiceStatus, inv.TDSAmount,
		inv.CreatedAt, inv.DueDate, inv.NextRetryAt, inv.RetryCount,
		inv.AckDate, inv.EInvoiceRetryCount, inv.EInvoiceNextRetryAt, inv.EInvoiceErrorMessage,
		nilIfEmpty(inv.DunningActionID), nilIfEmpty(inv.DunningContextKey), nilIfEmpty(inv.LastPaymentError), managedBy,
		inv.CreditApplied, nilIfEmpty(inv.MandateCycleKey),
	)
	if err != nil {
		return fmt.Errorf("failed to insert invoice: %w", err)
	}
	return nil
}

// lineItemPtrs returns the invoice's line items as pointers with InvoiceID set,
// ready for the item repository's bulk insert.
func lineItemPtrs(inv *domain.Invoice) []*domain.InvoiceItem {
	if len(inv.LineItems) == 0 {
		return nil
	}
	items := make([]*domain.InvoiceItem, 0, len(inv.LineItems))
	for i := range inv.LineItems {
		it := &inv.LineItems[i]
		if it.InvoiceID == uuid.Nil {
			it.InvoiceID = inv.ID
		}
		if it.CreatedAt.IsZero() {
			it.CreatedAt = inv.CreatedAt
		}
		items = append(items, it)
	}
	return items
}

func (r *InvoiceRepository) Create(ctx context.Context, inv *domain.Invoice) error {
	items := lineItemPtrs(inv)
	// No line items: preserve the historical single-statement, non-tx insert.
	if len(items) == 0 {
		return insertInvoiceRow(ctx, r.db, inv)
	}
	// With line items: insert the invoice and its items atomically so a partial
	// write can never leave an invoice without its lines (money-path invariant).
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin invoice tx: %w", err)
	}
	if err := insertInvoiceRow(ctx, tx, inv); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := insertInvoiceItems(ctx, tx, items); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit invoice tx: %w", err)
	}
	return nil
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// setInvoiceAmounts populates the amount fields on read: AmountPaid (scanned
// into a local) and AmountDue, which is derived (Total − AmountPaid) and has
// no stored column.
func setInvoiceAmounts(inv *domain.Invoice, amountPaid, creditApplied int64) {
	inv.AmountPaid = amountPaid
	inv.CreditApplied = creditApplied
	// Account credit (ENG-153) settles the invoice alongside cash: amount due is
	// the gross total less both what was paid and what credit was applied.
	inv.AmountDue = inv.Total - amountPaid - creditApplied
}

// CreateWithTx creates an invoice within an existing transaction for atomic
// operations. Line items (if any) are written on the same transaction so they
// commit atomically with the invoice.
func (r *InvoiceRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, inv *domain.Invoice) error {
	if err := insertInvoiceRow(ctx, tx, inv); err != nil {
		return fmt.Errorf("failed to insert invoice in tx: %w", err)
	}
	if items := lineItemPtrs(inv); len(items) > 0 {
		if err := insertInvoiceItems(ctx, tx, items); err != nil {
			return fmt.Errorf("failed to insert invoice items in tx: %w", err)
		}
	}
	return nil
}

func (r *InvoiceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	tenantID, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
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
	var amountPaid, creditApplied int64

	query := `
		SELECT
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid, COALESCE(credit_applied, 0),
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, updated_at, due_date, paid_at, next_retry_at, retry_count,
			COALESCE(ack_date, ''), e_invoice_retry_count,
			e_invoice_next_retry_at, COALESCE(e_invoice_error_message, ''),
			COALESCE(dunning_action_id, ''), COALESCE(dunning_context_key, ''),
			COALESCE(last_payment_error, ''), COALESCE(dunning_managed_by, 'scheduler'),
			COALESCE(payment_wall_active, FALSE),
			COALESCE(gateway_payment_id, '')
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
		&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid, &creditApplied,
		&inv.IGSTAmount, &inv.CGSTAmount, &inv.SGSTAmount, &hsnCode, &irn, &ackNo,
		&signedQRCode, &eInvoiceStatus, &inv.TDSAmount,
		&inv.CreatedAt, &inv.UpdatedAt, &inv.DueDate, &inv.PaidAt, &inv.NextRetryAt, &inv.RetryCount,
		&inv.AckDate, &inv.EInvoiceRetryCount,
		&inv.EInvoiceNextRetryAt, &inv.EInvoiceErrorMessage,
		&inv.DunningActionID, &inv.DunningContextKey,
		&inv.LastPaymentError, &inv.DunningManagedBy,
		&inv.PaymentWallActive,
		&inv.GatewayPaymentID,
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

	setInvoiceAmounts(inv, amountPaid, creditApplied)

	if items, itErr := r.items.ListByInvoiceID(ctx, inv.ID); itErr != nil {
		return nil, itErr
	} else {
		inv.LineItems = items
	}

	return inv, nil
}

// hydrateLineItems batch-loads and attaches line items for a slice of invoices,
// avoiding an N+1 query on list endpoints.
func (r *InvoiceRepository) hydrateLineItems(ctx context.Context, invoices []*domain.Invoice) error {
	if len(invoices) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(invoices))
	for _, inv := range invoices {
		ids = append(ids, inv.ID)
	}
	byInvoice, err := r.items.ListByInvoiceIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, inv := range invoices {
		inv.LineItems = byInvoice[inv.ID]
	}
	return nil
}

func (r *InvoiceRepository) Update(ctx context.Context, inv *domain.Invoice) error {
	query := `
		UPDATE invoices
		SET status = $1, amount_paid = $2, paid_at = $3, next_retry_at = $4, retry_count = $5,
		    tds_amount = $6, signed_qr_code = $7, e_invoice_status = $8, irn = $9,
		    ack_no = $10, ack_date = $11, e_invoice_retry_count = $12,
		    e_invoice_next_retry_at = $13, e_invoice_error_message = $14,
		    dunning_action_id = $15, dunning_context_key = $16,
		    last_payment_error = $17, dunning_managed_by = $18,
		    payment_wall_active = $19,
		    updated_at = NOW()
		WHERE id = $20 AND tenant_id = $21
	`
	// Persist the invoice's actual amount_paid — NOT the total. Update is used
	// for non-payment mutations (retry reschedule, e-invoice status, dunning) on
	// invoices that are usually UNPAID; hardcoding amount_paid = total corrupted
	// AR every time one of those ran (ENG-144). The paid transition goes through
	// MarkPaid, not here.
	_, err := r.db.ExecContext(ctx, query,
		inv.Status, inv.AmountPaid, inv.PaidAt, inv.NextRetryAt, inv.RetryCount,
		inv.TDSAmount, inv.SignedQRCode, inv.EInvoiceStatus, inv.IRN,
		inv.AckNo, inv.AckDate, inv.EInvoiceRetryCount,
		inv.EInvoiceNextRetryAt, inv.EInvoiceErrorMessage,
		nilIfEmpty(inv.DunningActionID), nilIfEmpty(inv.DunningContextKey),
		nilIfEmpty(inv.LastPaymentError), nilIfEmpty(inv.DunningManagedBy),
		inv.PaymentWallActive,
		inv.ID, inv.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}
	return nil
}

// MarkPaid atomically settles an invoice via a single conditional UPDATE. The
// `AND status <> 'paid'` guard means only the first of several concurrent
// settlers (inline checkout, gateway webhook, retry worker, offline payment)
// transitions the row; the rest affect zero rows. amount_paid is the cash
// portion — total less any account credit already applied (ENG-153) — so
// amount_paid + credit_applied = total and no read-then-write is needed.
// Returns true iff this call performed the transition.
func (r *InvoiceRepository) MarkPaid(ctx context.Context, invoiceID uuid.UUID, paidAt time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE invoices
		SET status = 'paid', amount_paid = total - credit_applied, paid_at = $2, updated_at = NOW()
		WHERE id = $1 AND status <> 'paid'
	`, invoiceID, paidAt)
	if err != nil {
		return false, fmt.Errorf("failed to mark invoice paid: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}

func (r *InvoiceRepository) GetDueForRetry(ctx context.Context) ([]*domain.Invoice, error) {
	query := `
		SELECT
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid, COALESCE(credit_applied, 0),
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, due_date, paid_at, next_retry_at, retry_count,
			COALESCE(dunning_action_id, ''), COALESCE(dunning_context_key, ''),
			COALESCE(last_payment_error, ''), COALESCE(dunning_managed_by, 'scheduler')
		FROM invoices
		WHERE status IN ('open', 'past_due')
		  AND next_retry_at IS NOT NULL
		  AND next_retry_at <= $1
		  AND dunning_managed_by = 'worker'
		LIMIT 10
	`
	rows, err := r.db.QueryContext(ctx, query, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query retry invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var invoices []*domain.Invoice
	for rows.Next() {
		inv := &domain.Invoice{}
		var amountPaid, creditApplied int64
		// e-invoice columns are nullable and NULL on non-e-invoiced rows (the
		// failed invoices this query targets); scanning NULL into a plain
		// string would abort the whole retry sweep.
		var hsn, irn, ackNo, qr, einvStatus, dunAction, dunCtx, lastErr, dunMgr sql.NullString
		// due_date is a nullable column scanned into a non-pointer time.Time;
		// guard it the same way so a NULL can't abort the sweep.
		var dueDate sql.NullTime
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
			&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid, &creditApplied,
			&inv.IGSTAmount, &inv.CGSTAmount, &inv.SGSTAmount, &hsn, &irn, &ackNo,
			&qr, &einvStatus, &inv.TDSAmount,
			&inv.CreatedAt, &dueDate, &inv.PaidAt, &inv.NextRetryAt, &inv.RetryCount,
			&dunAction, &dunCtx,
			&lastErr, &dunMgr,
		); err != nil {
			return nil, err
		}
		inv.DueDate = dueDate.Time
		inv.HSNCode = hsn.String
		inv.IRN = irn.String
		inv.AckNo = ackNo.String
		inv.SignedQRCode = qr.String
		inv.EInvoiceStatus = einvStatus.String
		inv.DunningActionID = dunAction.String
		inv.DunningContextKey = dunCtx.String
		inv.LastPaymentError = lastErr.String
		inv.DunningManagedBy = dunMgr.String
		setInvoiceAmounts(inv, amountPaid, creditApplied)
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

func (r *InvoiceRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error) {
	query := `
		SELECT 
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid, COALESCE(credit_applied, 0),
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, updated_at, due_date, paid_at, next_retry_at, retry_count
		FROM invoices
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch customer invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var invoices []*domain.Invoice
	for rows.Next() {
		inv := &domain.Invoice{}
		var amountPaid, creditApplied int64
		var hsnCode, irn, signedQRCode, eInvoiceStatus, ackNo sql.NullString
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
			&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid, &creditApplied,
			&inv.IGSTAmount, &inv.CGSTAmount, &inv.SGSTAmount, &hsnCode, &irn, &ackNo,
			&signedQRCode, &eInvoiceStatus, &inv.TDSAmount,
			&inv.CreatedAt, &inv.UpdatedAt, &inv.DueDate, &inv.PaidAt, &inv.NextRetryAt, &inv.RetryCount,
		); err != nil {
			return nil, err
		}
		inv.HSNCode = hsnCode.String
		inv.IRN = irn.String
		inv.AckNo = ackNo.String
		inv.SignedQRCode = signedQRCode.String
		inv.EInvoiceStatus = eInvoiceStatus.String
		setInvoiceAmounts(inv, amountPaid, creditApplied)
		invoices = append(invoices, inv)
	}
	if err := r.hydrateLineItems(ctx, invoices); err != nil {
		return nil, err
	}
	return invoices, nil
}

func (r *InvoiceRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invoice, error) {
	query := `
		SELECT 
			id, tenant_id, subscription_id, customer_id, invoice_number, status,
			currency, subtotal, tax_amount, total, amount_paid, COALESCE(credit_applied, 0),
			igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
			signed_qr_code, e_invoice_status, tds_amount,
			created_at, updated_at, due_date, paid_at, next_retry_at, retry_count
		FROM invoices
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var invoices []*domain.Invoice
	for rows.Next() {
		inv := &domain.Invoice{}
		var amountPaid, creditApplied int64
		var hsnCode, irn, signedQRCode, eInvoiceStatus, ackNo sql.NullString
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
			&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid, &creditApplied,
			&inv.IGSTAmount, &inv.CGSTAmount, &inv.SGSTAmount, &hsnCode, &irn, &ackNo,
			&signedQRCode, &eInvoiceStatus, &inv.TDSAmount,
			&inv.CreatedAt, &inv.UpdatedAt, &inv.DueDate, &inv.PaidAt, &inv.NextRetryAt, &inv.RetryCount,
		); err != nil {
			return nil, err
		}
		inv.HSNCode = hsnCode.String
		inv.IRN = irn.String
		inv.AckNo = ackNo.String
		inv.SignedQRCode = signedQRCode.String
		inv.EInvoiceStatus = eInvoiceStatus.String
		setInvoiceAmounts(inv, amountPaid, creditApplied)
		invoices = append(invoices, inv)
	}
	if err := r.hydrateLineItems(ctx, invoices); err != nil {
		return nil, err
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
			i.due_date, i.retry_count, i.next_retry_at, (i.mandate_cycle_key IS NOT NULL)
		FROM invoices i
		JOIN customers c ON i.customer_id = c.id
		WHERE i.status IN ('open', 'past_due')
			AND i.due_date < CURRENT_TIMESTAMP
			AND (i.next_retry_at IS NULL OR i.next_retry_at <= CURRENT_TIMESTAMP)
			AND (i.dunning_managed_by = 'scheduler' OR i.dunning_managed_by IS NULL)
		ORDER BY i.due_date ASC
		LIMIT 50
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query overdue invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var invoices []domain.OverdueInvoice
	for rows.Next() {
		var inv domain.OverdueInvoice
		// customers.name is nullable — scanning it into a plain string would
		// abort the whole dunning sweep on the first nameless customer.
		var name sql.NullString
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.CustomerID,
			&name, &inv.CustomerEmail,
			&inv.InvoiceNumber, &inv.Amount, &inv.Currency,
			&inv.DueDate, &inv.RetryCount, &inv.NextRetryAt, &inv.IsMandate,
		); err != nil {
			return nil, err
		}
		inv.CustomerName = name.String
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

// GetInvoiceAgingRows aggregates open/past-due invoices for a tenant into AR
// aging buckets by how far past due they are, per currency. Outstanding is the
// generated amount_remaining (total - amount_paid); fully-paid rows are excluded.
func (r *InvoiceRepository) GetInvoiceAgingRows(ctx context.Context, tenantID uuid.UUID) ([]domain.InvoiceAgingRow, error) {
	query := `
		SELECT currency,
		       CASE
		         WHEN due_date IS NULL OR due_date >= NOW()        THEN 'current'
		         WHEN due_date >  NOW() - INTERVAL '30 days'       THEN '1-30'
		         WHEN due_date >  NOW() - INTERVAL '60 days'       THEN '31-60'
		         WHEN due_date >  NOW() - INTERVAL '90 days'       THEN '61-90'
		         ELSE '90+'
		       END AS bucket,
		       COUNT(*)                         AS cnt,
		       COALESCE(SUM(amount_remaining),0) AS amt
		FROM invoices
		WHERE tenant_id = $1 AND status IN ('open', 'past_due') AND amount_remaining > 0
		GROUP BY currency, bucket`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query invoice aging: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.InvoiceAgingRow
	for rows.Next() {
		var row domain.InvoiceAgingRow
		if err := rows.Scan(&row.Currency, &row.Bucket, &row.Count, &row.Amount); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
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

// UpdateRetryInfoWithDunning updates retry info and sets dunning_managed_by for handoff
func (r *InvoiceRepository) UpdateRetryInfoWithDunning(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int, managedBy string) error {
	query := `
		UPDATE invoices
		SET next_retry_at = $1, retry_count = $2, status = 'past_due', dunning_managed_by = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, nextRetry, retryCount, managedBy, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to update retry info with dunning: %w", err)
	}
	return nil
}

// GetFailedEInvoices fetches FAILED e-invoices that are due for retry
// failedEInvoiceColumns is the shared projection for the failed-e-invoice
// read/claim paths; the two must return the same columns in the same order so
// scanFailedEInvoiceRows works for both.
const failedEInvoiceColumns = `id, tenant_id, subscription_id, customer_id, invoice_number, status,
	currency, subtotal, tax_amount, total, amount_paid, COALESCE(credit_applied, 0),
	igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
	signed_qr_code, e_invoice_status, tds_amount,
	created_at, due_date, paid_at, next_retry_at, retry_count,
	COALESCE(ack_date, ''), e_invoice_retry_count,
	e_invoice_next_retry_at, COALESCE(e_invoice_error_message, '')`

func scanFailedEInvoiceRows(rows *sql.Rows) ([]*domain.Invoice, error) {
	var invoices []*domain.Invoice
	for rows.Next() {
		inv := &domain.Invoice{}
		var amountPaid, creditApplied int64
		var hsnCode, irn, signedQRCode, eInvoiceStatus, ackNo sql.NullString
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.SubscriptionID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
			&inv.Currency, &inv.Subtotal, &inv.TaxAmount, &inv.Total, &amountPaid, &creditApplied,
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
		setInvoiceAmounts(inv, amountPaid, creditApplied)
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

func (r *InvoiceRepository) GetFailedEInvoices(ctx context.Context) ([]*domain.Invoice, error) {
	query := `SELECT ` + failedEInvoiceColumns + `
		FROM invoices
		WHERE e_invoice_status = 'FAILED'
		  AND e_invoice_next_retry_at IS NOT NULL
		  AND e_invoice_next_retry_at <= $1
		ORDER BY e_invoice_next_retry_at ASC
		LIMIT 20`
	rows, err := r.db.QueryContext(ctx, query, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query failed e-invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanFailedEInvoiceRows(rows)
}

// ClaimFailedEInvoices atomically leases up to `limit` failed e-invoices due for
// retry, pushing e_invoice_next_retry_at forward to `leaseUntil` so a concurrent
// runner can't see them — preventing duplicate government IRN submissions when
// the retry worker runs on more than one instance (the distributed lock is a
// no-op without Redis). FOR UPDATE SKIP LOCKED serializes the claim; the retry
// path then overwrites e_invoice_next_retry_at with the real backoff (on
// failure) or moves the row to a non-FAILED status (on success), so the lease is
// only a placeholder and a crashed runner's row re-surfaces after leaseUntil.
func (r *InvoiceRepository) ClaimFailedEInvoices(ctx context.Context, now, leaseUntil time.Time, limit int) ([]*domain.Invoice, error) {
	query := `UPDATE invoices
		SET e_invoice_next_retry_at = $2
		WHERE id IN (
			SELECT id FROM invoices
			WHERE e_invoice_status = 'FAILED'
			  AND e_invoice_next_retry_at IS NOT NULL
			  AND e_invoice_next_retry_at <= $1
			ORDER BY e_invoice_next_retry_at ASC
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		RETURNING ` + failedEInvoiceColumns
	rows, err := r.db.QueryContext(ctx, query, now, leaseUntil, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to claim failed e-invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanFailedEInvoiceRows(rows)
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

// GetGSTR1Invoices returns the tenant's finalized (non-draft, non-void)
// invoices issued in [start, end), flattened with the buyer's GST identity —
// the input for the GSTR-1 export. TaxableValue is the invoice subtotal (the
// GST base); the tax split is what was billed.
func (r *InvoiceRepository) GetGSTR1Invoices(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]domain.GSTR1Invoice, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT i.invoice_number, i.created_at,
		        COALESCE(c.gstin, ''), COALESCE(c.place_of_supply, ''),
		        i.subtotal::bigint, i.igst_amount, i.cgst_amount, i.sgst_amount, COALESCE(i.hsn_code, '')
		 FROM invoices i
		 JOIN customers c ON c.id = i.customer_id
		 WHERE i.tenant_id = $1
		   AND i.status NOT IN ('draft', 'void')
		   AND i.created_at >= $2 AND i.created_at < $3
		 ORDER BY i.created_at, i.invoice_number`, tenantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query gstr-1 invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.GSTR1Invoice
	for rows.Next() {
		var g domain.GSTR1Invoice
		if err := rows.Scan(&g.InvoiceNumber, &g.Date, &g.BuyerGSTIN, &g.PlaceOfSupply,
			&g.TaxableValue, &g.IGST, &g.CGST, &g.SGST, &g.HSNCode); err != nil {
			return nil, fmt.Errorf("failed to scan gstr-1 invoice: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GetGSTR1CreditNotes returns refund credit notes issued in [start, end) against
// an invoice, for the CDNR section. A credit note stores only its gross amount,
// so the tax it reversed is derived proportionally from the originating
// invoice's tax split — matching how RecordRefundTaxReversal reverses the ledger.
func (r *InvoiceRepository) GetGSTR1CreditNotes(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]domain.GSTR1CreditNote, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT COALESCE(NULLIF(cn.reference, ''), cn.id::text), cn.created_at,
		        COALESCE(c.gstin, ''), COALESCE(c.place_of_supply, ''),
		        i.invoice_number, cn.amount, i.total, i.igst_amount, i.cgst_amount, i.sgst_amount
		 FROM credit_notes cn
		 JOIN invoices i ON i.id = cn.invoice_id
		 JOIN customers c ON c.id = cn.customer_id
		 WHERE cn.tenant_id = $1 AND cn.type = 'refund' AND cn.invoice_id IS NOT NULL
		   AND cn.created_at >= $2 AND cn.created_at < $3
		 ORDER BY cn.created_at`, tenantID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query gstr-1 credit notes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.GSTR1CreditNote
	for rows.Next() {
		var cn domain.GSTR1CreditNote
		var amount, invTotal, invIGST, invCGST, invSGST int64
		if err := rows.Scan(&cn.NoteNumber, &cn.Date, &cn.BuyerGSTIN, &cn.PlaceOfSupply,
			&cn.OriginalInvoiceNumber, &amount, &invTotal, &invIGST, &invCGST, &invSGST); err != nil {
			return nil, fmt.Errorf("failed to scan gstr-1 credit note: %w", err)
		}
		cn.IGST = proportionalTax(amount, invIGST, invTotal)
		cn.CGST = proportionalTax(amount, invCGST, invTotal)
		cn.SGST = proportionalTax(amount, invSGST, invTotal)
		cn.TaxableValue = amount - (cn.IGST + cn.CGST + cn.SGST)
		out = append(out, cn)
	}
	return out, rows.Err()
}

// proportionalTax slices a component of the invoice's tax in proportion to how
// much of the invoice a credit note refunds.
func proportionalTax(creditAmount, invoiceComponent, invoiceTotal int64) int64 {
	if invoiceTotal <= 0 || invoiceComponent <= 0 || creditAmount <= 0 {
		return 0
	}
	return int64(math.Round(float64(creditAmount) * float64(invoiceComponent) / float64(invoiceTotal)))
}

// SetGatewayPaymentID records the gateway-side payment identifier (Stripe
// pi_*/ch_*, Razorpay pay_*) that settled the invoice. Called from the
// payment-success webhook paths; the id is what refunds are issued against.
func (r *InvoiceRepository) SetGatewayPaymentID(ctx context.Context, invoiceID uuid.UUID, gatewayPaymentID string) error {
	query := `
		UPDATE invoices
		SET gateway_payment_id = $1
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, gatewayPaymentID, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to set gateway payment id: %w", err)
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
