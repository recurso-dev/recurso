package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/recur-so/recurso/internal/core/domain"
)

type CustomerRepository struct {
	db *sqlx.DB
}

func NewCustomerRepository(db *sqlx.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) Create(ctx context.Context, customer *domain.Customer) error {
	addressJSON, err := json.Marshal(customer.BillingAddress)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO customers (
			id, tenant_id, email, name, phone, tax_id, 
			line1, city, state, zip, country, 
			billing_address, ledger_account_id, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err = r.db.ExecContext(ctx, query,
		customer.ID, customer.TenantID, customer.Email, customer.Name,
		customer.Phone, customer.TaxID,
		customer.BillingAddress.Line1, customer.BillingAddress.City, customer.BillingAddress.State, customer.BillingAddress.Zip, customer.BillingAddress.Country,
		addressJSON, customer.LedgerAccountID, customer.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert customer: %w", err)
	}

	return nil
}

func (r *CustomerRepository) Update(ctx context.Context, customer *domain.Customer) error {
	addressJSON, err := json.Marshal(customer.BillingAddress)
	if err != nil {
		return err
	}

	query := `
		UPDATE customers SET
			email = :email,
			name = :name,
			phone = :phone,
			tax_id = :tax_id,
			line1 = :billing_address.line1,
			city = :billing_address.city,
			state = :billing_address.state,
			zip = :billing_address.zip,
			country = :billing_address.country,
			billing_address = :billing_address_json,
			ledger_account_id = :ledger_account_id,
			referral_code = :referral_code
		WHERE id = :id AND tenant_id = :tenant_id
	`
	// NamedExec requires map or struct with matching tags.
	// Our struct tags use `db:"..."`.
	// However, `billing_address` is a struct inside a struct. SQLX NamedExec supports dot notation?
	// The Create method used `:billing_address.line1`. So yes.
	// But `billing_address` column stores json. We need to pass the JSON bytes.
	// We can use a map or an anonymous struct wrapper?
	// Or just extend the Customer struct with a temporary field? No.
	// Better: Use a map for parameters.

	params := map[string]interface{}{
		"id":                      customer.ID,
		"tenant_id":               customer.TenantID,
		"email":                   customer.Email,
		"name":                    customer.Name,
		"phone":                   customer.Phone,
		"tax_id":                  customer.TaxID,
		"billing_address.line1":   customer.BillingAddress.Line1,
		"billing_address.city":    customer.BillingAddress.City,
		"billing_address.state":   customer.BillingAddress.State,
		"billing_address.zip":     customer.BillingAddress.Zip,
		"billing_address.country": customer.BillingAddress.Country,
		"billing_address_json":    addressJSON,
		"ledger_account_id":       customer.LedgerAccountID,
		"referral_code":           customer.ReferralCode,
	}

	_, err = r.db.NamedExecContext(ctx, query, params)
	return err
}

func (r *CustomerRepository) UpdateRisk(ctx context.Context, customerID uuid.UUID, score int, factors map[string]interface{}) error {
	factorsJSON, err := json.Marshal(factors)
	if err != nil {
		return err
	}

	query := `UPDATE customers SET risk_score = $1, risk_factors = $2 WHERE id = $3`
	_, err = r.db.ExecContext(ctx, query, score, factorsJSON, customerID)
	return err
}

func (r *CustomerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id missing from context")
	}
	return r.getByIDInternal(ctx, id, &tenantID)
}

func (r *CustomerRepository) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return r.getByIDInternal(ctx, id, nil)
}

func (r *CustomerRepository) GetByReferralCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Customer, error) {
	var customer domain.Customer
	var addressJSON []byte
	query := `
		SELECT id, tenant_id, email, name, phone, tax_id, line1, city, state, zip, country, billing_address, ledger_account_id, referral_code, created_at
		FROM customers 
		WHERE tenant_id = $1 AND referral_code = $2 LIMIT 1
	`
	err := r.db.QueryRowContext(ctx, query, tenantID, code).Scan(
		&customer.ID, &customer.TenantID, &customer.Email, &customer.Name,
		&customer.Phone, &customer.TaxID,
		&customer.BillingAddress.Line1, &customer.BillingAddress.City, &customer.BillingAddress.State, &customer.BillingAddress.Zip, &customer.BillingAddress.Country,
		&addressJSON, &customer.LedgerAccountID, &customer.ReferralCode, &customer.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}

	if customer.BillingAddress.Line1 == "" && len(addressJSON) > 0 {
		_ = json.Unmarshal(addressJSON, &customer.BillingAddress)
	}

	return &customer, nil
}

func (r *CustomerRepository) getByIDInternal(ctx context.Context, id uuid.UUID, tenantID *uuid.UUID) (*domain.Customer, error) {
	customer := &domain.Customer{}
	var addressJSON []byte // We scan this but prefer individual columns if populated

	query := `
		SELECT id, tenant_id, email, name, phone, tax_id, line1, city, state, zip, country, billing_address, ledger_account_id, referral_code, created_at
		FROM customers WHERE id = $1
	`
	args := []interface{}{id}
	if tenantID != nil {
		query += " AND tenant_id = $2"
		args = append(args, *tenantID)
	}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&customer.ID, &customer.TenantID, &customer.Email, &customer.Name,
		&customer.Phone, &customer.TaxID,
		&customer.BillingAddress.Line1, &customer.BillingAddress.City, &customer.BillingAddress.State, &customer.BillingAddress.Zip, &customer.BillingAddress.Country,
		&addressJSON, &customer.LedgerAccountID, &customer.ReferralCode, &customer.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	// Fallback/Validation: If individual columns are empty but JSON exists, potentially use JSON?
	// For now, trusting individual columns as source of truth for new records.
	// Legacy records might have empty columns but populated JSON.
	// simple migration logic:
	if customer.BillingAddress.Line1 == "" && len(addressJSON) > 0 {
		_ = json.Unmarshal(addressJSON, &customer.BillingAddress)
	}

	return customer, nil
}

func (r *CustomerRepository) List(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error) {
	query := `
		SELECT id, tenant_id, email, name, phone, tax_id, line1, city, state, zip, country, billing_address, ledger_account_id, referral_code, created_at
		FROM customers WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.Status == "active" {
		query += " AND EXISTS (SELECT 1 FROM subscriptions s WHERE s.customer_id = customers.id AND s.status = 'active')"
	} else if filter.Status == "inactive" {
		query += " AND NOT EXISTS (SELECT 1 FROM subscriptions s WHERE s.customer_id = customers.id AND s.status = 'active')"
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.Country != "" {
		query += fmt.Sprintf(" AND country = $%d", argIdx)
		args = append(args, filter.Country)
		argIdx++
	}

	if filter.Email != "" {
		query += fmt.Sprintf(" AND email = $%d", argIdx)
		args = append(args, filter.Email)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
		argIdx++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []*domain.Customer
	for rows.Next() {
		var c domain.Customer
		var addressJSON []byte
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.Email, &c.Name,
			&c.Phone, &c.TaxID,
			&c.BillingAddress.Line1, &c.BillingAddress.City, &c.BillingAddress.State, &c.BillingAddress.Zip, &c.BillingAddress.Country,
			&addressJSON, &c.LedgerAccountID, &c.ReferralCode, &c.CreatedAt,
		); err != nil {
			return nil, err
		}

		if c.BillingAddress.Line1 == "" && len(addressJSON) > 0 {
			_ = json.Unmarshal(addressJSON, &c.BillingAddress)
		}

		customers = append(customers, &c)
	}

	return customers, nil
}
