package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type OfflinePaymentRepository struct {
	db *sql.DB
}

func NewOfflinePaymentRepository(db *sql.DB) *OfflinePaymentRepository {
	return &OfflinePaymentRepository{db: db}
}

func (r *OfflinePaymentRepository) CreateVirtualAccount(ctx context.Context, va *domain.VirtualAccount) error {
	query := `INSERT INTO virtual_accounts (id, tenant_id, customer_id, invoice_id, account_number, ifsc_code,
		bank_name, beneficiary_name, razorpay_va_id, status, amount_expected, amount_received, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.db.ExecContext(ctx, query,
		va.ID, va.TenantID, va.CustomerID, va.InvoiceID,
		va.AccountNumber, va.IFSCCode, va.BankName, va.BeneficiaryName,
		va.RazorpayVAID, va.Status, va.AmountExpected, va.AmountReceived, va.CreatedAt,
	)
	return err
}

func (r *OfflinePaymentRepository) GetVirtualAccountByID(ctx context.Context, id uuid.UUID) (*domain.VirtualAccount, error) {
	query := `SELECT id, tenant_id, customer_id, invoice_id, account_number, ifsc_code,
		bank_name, beneficiary_name, razorpay_va_id, status, amount_expected, amount_received, closed_at, created_at
		FROM virtual_accounts WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanVA(row)
}

func (r *OfflinePaymentRepository) GetVirtualAccountByRazorpayID(ctx context.Context, razorpayVAID string) (*domain.VirtualAccount, error) {
	query := `SELECT id, tenant_id, customer_id, invoice_id, account_number, ifsc_code,
		bank_name, beneficiary_name, razorpay_va_id, status, amount_expected, amount_received, closed_at, created_at
		FROM virtual_accounts WHERE razorpay_va_id = $1`
	row := r.db.QueryRowContext(ctx, query, razorpayVAID)
	return r.scanVA(row)
}

func (r *OfflinePaymentRepository) ListVirtualAccounts(ctx context.Context, tenantID uuid.UUID) ([]*domain.VirtualAccount, error) {
	query := `SELECT id, tenant_id, customer_id, invoice_id, account_number, ifsc_code,
		bank_name, beneficiary_name, razorpay_va_id, status, amount_expected, amount_received, closed_at, created_at
		FROM virtual_accounts WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var accounts []*domain.VirtualAccount
	for rows.Next() {
		var va domain.VirtualAccount
		err := rows.Scan(
			&va.ID, &va.TenantID, &va.CustomerID, &va.InvoiceID,
			&va.AccountNumber, &va.IFSCCode, &va.BankName, &va.BeneficiaryName,
			&va.RazorpayVAID, &va.Status, &va.AmountExpected, &va.AmountReceived,
			&va.ClosedAt, &va.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, &va)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}

// IncrementAmountReceived atomically applies a credit to the VA in one UPDATE,
// so two concurrent credits can't lose an increment (the old read-modify-write
// did last-write-wins). It also closes the VA in the same statement once the
// expected amount is reached, and returns the updated row.
func (r *OfflinePaymentRepository) IncrementAmountReceived(ctx context.Context, razorpayVAID string, amount int64) (*domain.VirtualAccount, error) {
	query := `
		UPDATE virtual_accounts
		SET amount_received = amount_received + $2,
		    status = CASE WHEN amount_received + $2 >= amount_expected THEN 'closed' ELSE status END,
		    closed_at = CASE WHEN amount_received + $2 >= amount_expected AND closed_at IS NULL THEN NOW() ELSE closed_at END
		WHERE razorpay_va_id = $1
		RETURNING id, tenant_id, customer_id, invoice_id, account_number, ifsc_code,
			bank_name, beneficiary_name, razorpay_va_id, status, amount_expected, amount_received, closed_at, created_at`
	row := r.db.QueryRowContext(ctx, query, razorpayVAID, amount)
	return r.scanVA(row)
}

func (r *OfflinePaymentRepository) UpdateVirtualAccount(ctx context.Context, va *domain.VirtualAccount) error {
	query := `UPDATE virtual_accounts SET status = $1, amount_received = $2, closed_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, va.Status, va.AmountReceived, va.ClosedAt, va.ID)
	return err
}

func (r *OfflinePaymentRepository) CreateOfflinePayment(ctx context.Context, payment *domain.OfflinePayment) error {
	query := `INSERT INTO offline_payments (id, tenant_id, customer_id, invoice_id, payment_type, amount,
		tds_amount, currency, reference_number, notes, recorded_by, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.ExecContext(ctx, query,
		payment.ID, payment.TenantID, payment.CustomerID, payment.InvoiceID,
		payment.PaymentType, payment.Amount, payment.TDSAmount, payment.Currency,
		payment.ReferenceNumber, payment.Notes, payment.RecordedBy, payment.RecordedAt,
	)
	return err
}

func (r *OfflinePaymentRepository) ListOfflinePayments(ctx context.Context, tenantID uuid.UUID) ([]*domain.OfflinePayment, error) {
	query := `SELECT id, tenant_id, customer_id, invoice_id, payment_type, amount,
		tds_amount, currency, reference_number, notes, recorded_by, recorded_at
		FROM offline_payments WHERE tenant_id = $1 ORDER BY recorded_at DESC`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var payments []*domain.OfflinePayment
	for rows.Next() {
		var p domain.OfflinePayment
		err := rows.Scan(
			&p.ID, &p.TenantID, &p.CustomerID, &p.InvoiceID,
			&p.PaymentType, &p.Amount, &p.TDSAmount, &p.Currency,
			&p.ReferenceNumber, &p.Notes, &p.RecordedBy, &p.RecordedAt,
		)
		if err != nil {
			return nil, err
		}
		payments = append(payments, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *OfflinePaymentRepository) scanVA(row *sql.Row) (*domain.VirtualAccount, error) {
	var va domain.VirtualAccount
	err := row.Scan(
		&va.ID, &va.TenantID, &va.CustomerID, &va.InvoiceID,
		&va.AccountNumber, &va.IFSCCode, &va.BankName, &va.BeneficiaryName,
		&va.RazorpayVAID, &va.Status, &va.AmountExpected, &va.AmountReceived,
		&va.ClosedAt, &va.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &va, nil
}
