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
		credit_applied, mandate_cycle_key, billing_reason, tax_type, entity_id
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37)
`

// allocateInvoiceNumber draws the next number in the issuing entity's gapless
// series (Multi-Entity Books Inc 3a): {invoice_prefix}-{seq:06d}. A nil entity
// resolves to the tenant's primary entity, so single-entity tenants get one
// continuous "INV-000001…" series. The UPDATE…RETURNING is atomic per entity
// row, so concurrent finalizations serialize into gapless, unique numbers; run
// on the caller's execer it shares the invoice-insert transaction, so a rolled
// back insert (e.g. a rejected mandate-cycle claim) returns the number too.
func allocateInvoiceNumber(ctx context.Context, ex execer, tenantID uuid.UUID, entityID *uuid.UUID) (string, error) {
	var entID uuid.UUID
	var prefix string
	var row *sql.Row
	if entityID != nil {
		row = ex.QueryRowContext(ctx,
			`SELECT id, invoice_prefix FROM entities WHERE id = $1 AND tenant_id = $2`, *entityID, tenantID)
	} else {
		row = ex.QueryRowContext(ctx,
			`SELECT id, invoice_prefix FROM entities WHERE tenant_id = $1 AND is_primary`, tenantID)
	}
	if err := row.Scan(&entID, &prefix); err != nil {
		return "", fmt.Errorf("resolve invoice-series entity: %w", err)
	}
	var seq int64
	if err := ex.QueryRowContext(ctx,
		`UPDATE entity_invoice_sequences SET next_number = next_number + 1 WHERE entity_id = $1 RETURNING next_number - 1`,
		entID).Scan(&seq); err != nil {
		return "", fmt.Errorf("allocate invoice number for entity %s: %w", entID, err)
	}
	return fmt.Sprintf("%s-%06d", prefix, seq), nil
}

// insertInvoiceRow writes the invoice row against any execer (*sql.DB or *sql.Tx).
func insertInvoiceRow(ctx context.Context, ex execer, inv *domain.Invoice) error {
	// Allocate the gapless per-entity invoice number when the caller left it
	// blank (all billing paths do — see Inc 3a). Same execer as the insert, so
	// the sequence draw commits or rolls back with the row.
	if inv.InvoiceNumber == "" {
		num, err := allocateInvoiceNumber(ctx, ex, inv.TenantID, inv.EntityID)
		if err != nil {
			return err
		}
		inv.InvoiceNumber = num
	}

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
		inv.CreditApplied, nilIfEmpty(inv.MandateCycleKey), nilIfEmpty(inv.BillingReason), inv.TaxType, inv.EntityID,
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
			COALESCE(gateway_payment_id, ''),
			COALESCE(billing_reason, ''), entity_id
		FROM invoices WHERE id = $1
	`
	args := []interface{}{id}
	if tenantID != nil {
		query += " AND tenant_id = $2"
		args = append(args, *tenantID)
	}

	var hsnCode, irn, signedQRCode, eInvoiceStatus, ackNo sql.NullString
	var entityID uuid.NullUUID

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
		&inv.BillingReason, &entityID,
	)

	inv.HSNCode = hsnCode.String
	inv.IRN = irn.String
	inv.AckNo = ackNo.String
	inv.SignedQRCode = signedQRCode.String
	if entityID.Valid {
		inv.EntityID = &entityID.UUID
	}
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
func (r *InvoiceRepository) MarkPaid(ctx context.Context, tenantID, invoiceID uuid.UUID, paidAt time.Time) (bool, error) {
	// tenant_id is scoped in the WHERE (defense-in-depth). Callers pass the
	// loaded invoice's own TenantID, so the real settler still matches exactly
	// one row; the guard only ever excludes a cross-tenant id — settlement can
	// never flip an invoice belonging to another tenant.
	res, err := r.db.ExecContext(ctx, `
		UPDATE invoices
		SET status = 'paid', amount_paid = total - credit_applied, paid_at = $2, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $3 AND status <> 'paid'
	`, invoiceID, paidAt, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to mark invoice paid: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}

// ReverseToUnpaid is the inverse of MarkPaid: it reopens an invoice whose
// payment the bank later clawed back (an ACH return, Inc 3c). The
// `AND status = 'paid'` guard makes it idempotent and safe against a
// redelivered return webhook — only a currently-paid row transitions, and it
// lands in 'past_due' so dunning picks it back up (the in-flight guard has
// already cleared because the attempt is now 'returned'). amount_paid resets to
// 0 and paid_at to NULL so the invoice reads as fully outstanding again.
// Returns true iff this call performed the transition.
func (r *InvoiceRepository) ReverseToUnpaid(ctx context.Context, tenantID, invoiceID uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE invoices
		SET status = 'past_due', amount_paid = 0, paid_at = NULL, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND status = 'paid'
	`, invoiceID, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to reverse invoice to unpaid: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}

// retryInvoiceColumns is the projection shared by GetDueForRetry and
// ClaimDueForRetry (as a RETURNING list) — the two must stay column-aligned
// with scanRetryInvoices.
const retryInvoiceColumns = `
	id, tenant_id, subscription_id, customer_id, invoice_number, status,
	currency, subtotal, tax_amount, total, amount_paid, COALESCE(credit_applied, 0),
	igst_amount, cgst_amount, sgst_amount, hsn_code, irn, ack_no,
	signed_qr_code, e_invoice_status, tds_amount,
	created_at, due_date, paid_at, next_retry_at, retry_count,
	COALESCE(dunning_action_id, ''), COALESCE(dunning_context_key, ''),
	COALESCE(last_payment_error, ''), COALESCE(dunning_managed_by, 'scheduler')`

func (r *InvoiceRepository) GetDueForRetry(ctx context.Context) ([]*domain.Invoice, error) {
	query := `
		SELECT` + retryInvoiceColumns + `
		FROM invoices
		WHERE status IN ('open', 'past_due')
		  AND next_retry_at IS NOT NULL
		  AND next_retry_at <= $1
		  AND dunning_managed_by = 'worker'
		  AND NOT dunning_paused
		LIMIT 10
	`
	rows, err := r.db.QueryContext(ctx, query, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query retry invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanRetryInvoices(rows)
}

// ClaimDueForRetry atomically leases up to `limit` due retry invoices for THIS
// worker instance and returns them, pushing each claimed row's next_retry_at
// forward by `lease` so a second instance (Cloud Run scales to many, and the
// Locker is a no-op without Redis) cannot claim the same rows in the same cycle
// (ADR-003). FOR UPDATE SKIP LOCKED makes concurrent claimers take disjoint
// sets instead of blocking. The caller overwrites next_retry_at with the real
// next-retry time (or clears it on success); if the worker dies mid-process the
// lease lapses and the row is retried on a later tick.
func (r *InvoiceRepository) ClaimDueForRetry(ctx context.Context, lease time.Duration, limit int) ([]*domain.Invoice, error) {
	query := `
		UPDATE invoices
		SET next_retry_at = NOW() + $1 * INTERVAL '1 second', updated_at = NOW()
		WHERE id IN (
			SELECT id FROM invoices
			WHERE status IN ('open', 'past_due')
			  AND next_retry_at IS NOT NULL
			  AND next_retry_at <= NOW()
			  AND dunning_managed_by = 'worker'
			  AND NOT dunning_paused
			ORDER BY next_retry_at
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		RETURNING` + retryInvoiceColumns
	rows, err := r.db.QueryContext(ctx, query, int64(lease.Seconds()), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to claim retry invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanRetryInvoices(rows)
}

// scanRetryInvoices reads the retryInvoiceColumns projection into invoices,
// tolerating the NULL e-invoice/due-date columns that failed (non-e-invoiced)
// rows carry — a NULL scanned into a plain string/time would abort the sweep.
func scanRetryInvoices(rows *sql.Rows) ([]*domain.Invoice, error) {
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
			created_at, updated_at, due_date, paid_at, next_retry_at, retry_count,
			COALESCE(billing_reason, '')
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
			&inv.BillingReason,
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
			created_at, updated_at, due_date, paid_at, next_retry_at, retry_count,
			COALESCE(billing_reason, '')
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
			&inv.BillingReason,
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
			AND NOT i.dunning_paused
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
func (r *InvoiceRepository) GetInvoiceAgingRows(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID) ([]domain.InvoiceAgingRow, error) {
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
		  AND ($2::uuid IS NULL OR entity_id = $2)
		GROUP BY currency, bucket`
	rows, err := r.db.QueryContext(ctx, query, tenantID, entityID)
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

// collectionsQueueWhere builds the shared WHERE clause + args for the
// collections worklist (list + count must stay identical, or the pagination
// total lies). The population is every invoice still owing money in a recovery
// state: past_due (dunning/retry in progress) or uncollectible (given up on but
// not yet written off). Optional status/managed_by narrowing. $1 is always the
// tenant id; filter args start at $2.
func collectionsQueueWhere(tenantID uuid.UUID, f domain.CollectionsQueueFilter) (string, []interface{}) {
	where := `WHERE i.tenant_id = $1
		AND i.status IN ('past_due', 'uncollectible')
		AND i.amount_remaining > 0`
	args := []interface{}{tenantID}
	if f.Status != "" {
		args = append(args, f.Status)
		where += fmt.Sprintf(" AND i.status = $%d", len(args))
	}
	if f.ManagedBy != "" {
		args = append(args, f.ManagedBy)
		where += fmt.Sprintf(" AND COALESCE(i.dunning_managed_by, 'scheduler') = $%d", len(args))
	}
	return where, args
}

// ListCollectionsQueue returns the operator-facing collections worklist for a
// tenant — currently-failing invoices with their recovery state, customer, and
// latest payment-attempt status (Collections Intelligence Inc 1). Ordered oldest
// due-date first (most urgent). Read-only; no money-path.
func (r *InvoiceRepository) ListCollectionsQueue(ctx context.Context, tenantID uuid.UUID, f domain.CollectionsQueueFilter) ([]domain.CollectionsQueueItem, error) {
	where, args := collectionsQueueWhere(tenantID, f)
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	args = append(args, limit, f.Offset)
	query := `
		SELECT
			i.id, i.customer_id, COALESCE(c.name, ''), COALESCE(c.email, ''),
			i.invoice_number, i.status, i.currency, i.amount_remaining, i.due_date,
			GREATEST(0, DATE_PART('day', NOW() - i.due_date))::int AS days_overdue,
			i.retry_count, COALESCE(i.last_payment_error, ''), i.next_retry_at,
			COALESCE(i.dunning_managed_by, 'scheduler'),
			COALESCE(att.status, ''), i.dunning_paused
		FROM invoices i
		JOIN customers c ON c.id = i.customer_id
		LEFT JOIN LATERAL (
			SELECT status FROM payment_attempts pa
			WHERE pa.invoice_id = i.id
			ORDER BY pa.created_at DESC
			LIMIT 1
		) att ON true
		` + where + `
		ORDER BY i.due_date ASC
		LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query collections queue: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.CollectionsQueueItem
	for rows.Next() {
		var it domain.CollectionsQueueItem
		if err := rows.Scan(
			&it.ID, &it.CustomerID, &it.CustomerName, &it.CustomerEmail,
			&it.InvoiceNumber, &it.Status, &it.Currency, &it.AmountRemaining, &it.DueDate,
			&it.DaysOverdue, &it.RetryCount, &it.LastPaymentError, &it.NextRetryAt,
			&it.ManagedBy, &it.AttemptStatus, &it.DunningPaused,
		); err != nil {
			return nil, fmt.Errorf("scan collections queue row: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// CountCollectionsQueue returns the total number of invoices matching the same
// filter (for pagination). Must use the identical predicate as
// ListCollectionsQueue.
func (r *InvoiceRepository) CountCollectionsQueue(ctx context.Context, tenantID uuid.UUID, f domain.CollectionsQueueFilter) (int, error) {
	where, args := collectionsQueueWhere(tenantID, f)
	query := `SELECT COUNT(*) FROM invoices i ` + where
	var n int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count collections queue: %w", err)
	}
	return n, nil
}

// GetRetryEligibility reads the minimal tenant-scoped state a manual "retry now"
// needs (Collections Intelligence Inc 3) — status, paused flag, and whether it's
// a mandate — so the service can return a precise reason without hydrating the
// full invoice. Found=false when no such invoice exists for the tenant.
func (r *InvoiceRepository) GetRetryEligibility(ctx context.Context, tenantID, invoiceID uuid.UUID) (domain.InvoiceRetryEligibility, error) {
	var e domain.InvoiceRetryEligibility
	err := r.db.QueryRowContext(ctx, `
		SELECT status, dunning_paused, (mandate_cycle_key IS NOT NULL)
		FROM invoices WHERE id = $1 AND tenant_id = $2`,
		invoiceID, tenantID).Scan(&e.Status, &e.Paused, &e.IsMandate)
	if err == sql.ErrNoRows {
		return domain.InvoiceRetryEligibility{Found: false}, nil
	}
	if err != nil {
		return e, fmt.Errorf("query retry eligibility: %w", err)
	}
	e.Found = true
	return e, nil
}

// RequeueForRetry hands a failing invoice to the smart-retry worker for an
// immediate attempt (Collections Intelligence Inc 3, "retry now"): it sets
// next_retry_at to now and dunning_managed_by='worker' so the next worker tick
// (≤10s) claims it. The WHERE clause is the safety envelope — it fires only for a
// tenant-owned, currently-past_due, un-paused, NON-mandate invoice with NO
// in-flight payment attempt, so a manual retry can never double-charge a UPI
// mandate (ENG-168), fight a paused row, or stack a second charge on an ACH
// debit that is still settling. The in-flight check is INSIDE the atomic UPDATE
// (not just the caller's pre-read) so a payment_intent.processing webhook
// landing between the eligibility read and this statement still blocks the
// requeue. Returns true iff a row was requeued.
func (r *InvoiceRepository) RequeueForRetry(ctx context.Context, tenantID, invoiceID uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE invoices
		SET next_retry_at = NOW(), dunning_managed_by = 'worker', updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
		  AND status = 'past_due'
		  AND NOT dunning_paused
		  AND mandate_cycle_key IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM payment_attempts pa
			WHERE pa.invoice_id = invoices.id
			  AND pa.status IN ('initiated', 'processing')
		  )
	`, invoiceID, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to requeue invoice for retry: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}

// SetDunningPaused pauses or resumes automated dunning on a single invoice
// (Collections Intelligence Inc 3). Tenant-scoped; only touches an invoice still
// in a dunnable/recovery state. Returns true iff a row changed ownership state.
func (r *InvoiceRepository) SetDunningPaused(ctx context.Context, tenantID, invoiceID uuid.UUID, paused bool) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE invoices
		SET dunning_paused = $3, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
		  AND status IN ('open', 'past_due', 'uncollectible')
	`, invoiceID, tenantID, paused)
	if err != nil {
		return false, fmt.Errorf("failed to set dunning paused: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}

// MarkUncollectibleScoped is the tenant-scoped, operator-initiated write-off
// (Collections Intelligence Inc 3) — the manual counterpart of the automated
// MarkAsUncollectible. It only flips a still-collectible invoice, so it can't
// resurrect a paid one, and (matching the automated path) posts no ledger leg:
// uncollectible is a status, and AR is excluded from owing reports by status.
// Returns true iff this call performed the transition.
func (r *InvoiceRepository) MarkUncollectibleScoped(ctx context.Context, tenantID, invoiceID uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE invoices
		SET status = 'uncollectible', next_retry_at = NULL, marked_uncollectible_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND status IN ('open', 'past_due')
	`, invoiceID, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to mark invoice uncollectible: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}

// GetOutstandingByEntity sums open AR (amount_remaining on open/past_due
// invoices) per legal entity + currency, for the multi-entity overview. A NULL
// entity_id (rare — pre-backfill rows) is returned as-is and attributed to the
// primary by the service. Read-only.
func (r *InvoiceRepository) GetOutstandingByEntity(ctx context.Context, tenantID uuid.UUID) ([]domain.EntityOutstandingRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT entity_id, currency, COALESCE(SUM(amount_remaining), 0)
		FROM invoices
		WHERE tenant_id = $1 AND status IN ('open', 'past_due') AND amount_remaining > 0
		GROUP BY entity_id, currency`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query outstanding by entity: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.EntityOutstandingRow
	for rows.Next() {
		var row domain.EntityOutstandingRow
		var eid uuid.NullUUID
		if err := rows.Scan(&eid, &row.Currency, &row.Amount); err != nil {
			return nil, fmt.Errorf("scan outstanding-by-entity row: %w", err)
		}
		if eid.Valid {
			id := eid.UUID
			row.EntityID = &id
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CountUncollectibleSince counts invoices written off at-or-after `since` — the
// written-off side of the windowed recovery-rate cohort (QA finding D). Rows
// written off before migration 000147 carry a best-effort backfilled timestamp.
func (r *InvoiceRepository) CountUncollectibleSince(ctx context.Context, tenantID uuid.UUID, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM invoices
		WHERE tenant_id = $1 AND status = 'uncollectible'
		  AND marked_uncollectible_at IS NOT NULL AND marked_uncollectible_at >= $2`,
		tenantID, since).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count uncollectible since: %w", err)
	}
	return n, nil
}

// GetCollectionsAtRisk aggregates invoices still owing money in a recovery
// state, grouped by status + currency (Collections Intelligence Inc 2). The
// service FX-normalizes these into the recovery-funnel buckets. Read-only.
func (r *InvoiceRepository) GetCollectionsAtRisk(ctx context.Context, tenantID uuid.UUID) ([]domain.CollectionsAtRiskRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, currency, COUNT(*), COALESCE(SUM(amount_remaining), 0)
		FROM invoices
		WHERE tenant_id = $1 AND status IN ('past_due', 'uncollectible') AND amount_remaining > 0
		GROUP BY status, currency`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query collections at-risk: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.CollectionsAtRiskRow
	for rows.Next() {
		var row domain.CollectionsAtRiskRow
		if err := rows.Scan(&row.Status, &row.Currency, &row.Count, &row.Amount); err != nil {
			return nil, fmt.Errorf("scan collections at-risk row: %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// GetCollectionsFailureBreakdown aggregates currently-failing invoices by their
// last failure code + currency (Collections Intelligence Inc 2). A blank code
// (offline/manual invoices that never hit a gateway) folds into "unknown".
// Read-only.
func (r *InvoiceRepository) GetCollectionsFailureBreakdown(ctx context.Context, tenantID uuid.UUID) ([]domain.CollectionsFailureRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT COALESCE(NULLIF(last_payment_error, ''), 'unknown') AS code, currency,
		       COUNT(*), COALESCE(SUM(amount_remaining), 0)
		FROM invoices
		WHERE tenant_id = $1 AND status IN ('past_due', 'uncollectible') AND amount_remaining > 0
		GROUP BY code, currency`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query collections failure breakdown: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []domain.CollectionsFailureRow
	for rows.Next() {
		var row domain.CollectionsFailureRow
		if err := rows.Scan(&row.ErrorCode, &row.Currency, &row.Count, &row.Amount); err != nil {
			return nil, fmt.Errorf("scan collections failure row: %w", err)
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
func (r *InvoiceRepository) UpdateEInvoiceStatus(ctx context.Context, tenantID, invoiceID uuid.UUID, status, irn, ackNo, signedQR, ackDate, errorMsg string) error {
	query := `
		UPDATE invoices
		SET e_invoice_status = $1, irn = $2, ack_no = $3, signed_qr_code = $4,
		    ack_date = $5, e_invoice_error_message = $6
		WHERE id = $7 AND tenant_id = $8
	`
	_, err := r.db.ExecContext(ctx, query, status, irn, ackNo, signedQR, ackDate, errorMsg, invoiceID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update e-invoice status: %w", err)
	}
	return nil
}

// GetGSTR1Invoices returns the tenant's finalized (non-draft, non-void)
// invoices issued in [start, end), flattened with the buyer's GST identity —
// the input for the GSTR-1 export. TaxableValue is the invoice subtotal (the
// GST base); the tax split is what was billed.
func (r *InvoiceRepository) GetGSTR1Invoices(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID, start, end time.Time) ([]domain.GSTR1Invoice, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT i.invoice_number, i.created_at,
		        COALESCE(c.gstin, ''), COALESCE(c.place_of_supply, ''),
		        i.subtotal::bigint, i.igst_amount, i.cgst_amount, i.sgst_amount, COALESCE(i.hsn_code, '')
		 FROM invoices i
		 JOIN customers c ON c.id = i.customer_id
		 WHERE i.tenant_id = $1
		   AND i.status NOT IN ('draft', 'void')
		   AND i.created_at >= $2 AND i.created_at < $3
		   AND ($4::uuid IS NULL OR i.entity_id = $4)
		 ORDER BY i.created_at, i.invoice_number`, tenantID, start, end, entityID)
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
func (r *InvoiceRepository) GetGSTR1CreditNotes(ctx context.Context, tenantID uuid.UUID, entityID *uuid.UUID, start, end time.Time) ([]domain.GSTR1CreditNote, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT COALESCE(NULLIF(cn.reference, ''), cn.id::text), cn.created_at,
		        COALESCE(c.gstin, ''), COALESCE(c.place_of_supply, ''),
		        i.invoice_number, cn.amount, i.total, i.igst_amount, i.cgst_amount, i.sgst_amount
		 FROM credit_notes cn
		 JOIN invoices i ON i.id = cn.invoice_id
		 JOIN customers c ON c.id = cn.customer_id
		 WHERE cn.tenant_id = $1 AND cn.type = 'refund' AND cn.invoice_id IS NOT NULL
		   AND cn.created_at >= $2 AND cn.created_at < $3
		   AND ($4::uuid IS NULL OR cn.entity_id = $4)
		 ORDER BY cn.created_at`, tenantID, start, end, entityID)
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
func (r *InvoiceRepository) SetGatewayPaymentID(ctx context.Context, tenantID, invoiceID uuid.UUID, gatewayPaymentID string) error {
	query := `
		UPDATE invoices
		SET gateway_payment_id = $1
		WHERE id = $2 AND tenant_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, gatewayPaymentID, invoiceID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to set gateway payment id: %w", err)
	}
	return nil
}

// VoidIfOpen atomically voids a still-open (unpaid) invoice. Returns true only
// when this call performed the transition — a paid, already-void, or missing
// invoice is left untouched. Used by gift cancellation: an unpaid purchase
// invoice is voided rather than credited (no money ever arrived).
func (r *InvoiceRepository) VoidIfOpen(ctx context.Context, tenantID, invoiceID uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE invoices SET status = 'void', updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND status = 'open'
	`, invoiceID, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to void invoice: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}

// MarkAsUncollectible marks an invoice as uncollectible after max retries
func (r *InvoiceRepository) MarkAsUncollectible(ctx context.Context, invoiceID uuid.UUID) error {
	query := `
		UPDATE invoices
		SET status = 'uncollectible', next_retry_at = NULL, marked_uncollectible_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, invoiceID)
	if err != nil {
		return fmt.Errorf("failed to mark invoice as uncollectible: %w", err)
	}
	return nil
}
