package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/swapnull-in/recur-so/internal/core/domain"
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
			referral_code = :referral_code,
			updated_at = NOW()
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
	tenantID, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
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
		SELECT id, tenant_id, email, name, phone, tax_id, line1, city, state, zip, country, billing_address, ledger_account_id, referral_code, card_brand, card_last4, card_exp_month, card_exp_year, created_at, updated_at
		FROM customers
		WHERE tenant_id = $1 AND referral_code = $2 LIMIT 1
	`
	err := r.db.QueryRowContext(ctx, query, tenantID, code).Scan(
		&customer.ID, &customer.TenantID, &customer.Email, &customer.Name,
		&customer.Phone, &customer.TaxID,
		&customer.BillingAddress.Line1, &customer.BillingAddress.City, &customer.BillingAddress.State, &customer.BillingAddress.Zip, &customer.BillingAddress.Country,
		&addressJSON, &customer.LedgerAccountID, &customer.ReferralCode,
		&customer.CardBrand, &customer.CardLast4, &customer.CardExpMonth, &customer.CardExpYear,
		&customer.CreatedAt, &customer.UpdatedAt,
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
		SELECT id, tenant_id, email, name, phone, tax_id, line1, city, state, zip, country, billing_address, ledger_account_id, referral_code, card_brand, card_last4, card_exp_month, card_exp_year, created_at, updated_at
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
		&addressJSON, &customer.LedgerAccountID, &customer.ReferralCode,
		&customer.CardBrand, &customer.CardLast4, &customer.CardExpMonth, &customer.CardExpYear,
		&customer.CreatedAt, &customer.UpdatedAt,
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

// FindByEmailAcrossTenants is cross-tenant by design: portal login
// identifies customers by email alone, before any tenant is known.
// Never call it from tenant-scoped request paths.
func (r *CustomerRepository) FindByEmailAcrossTenants(ctx context.Context, email string) ([]*domain.Customer, error) {
	query := `
		SELECT id, tenant_id, email, name, phone, tax_id, line1, city, state, zip, country, billing_address, ledger_account_id, referral_code, card_brand, card_last4, card_exp_month, card_exp_year, created_at, updated_at
		FROM customers WHERE lower(email) = lower($1)
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, email)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var customers []*domain.Customer
	for rows.Next() {
		var c domain.Customer
		var addressJSON []byte
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.Email, &c.Name,
			&c.Phone, &c.TaxID,
			&c.BillingAddress.Line1, &c.BillingAddress.City, &c.BillingAddress.State, &c.BillingAddress.Zip, &c.BillingAddress.Country,
			&addressJSON, &c.LedgerAccountID, &c.ReferralCode,
			&c.CardBrand, &c.CardLast4, &c.CardExpMonth, &c.CardExpYear,
			&c.CreatedAt, &c.UpdatedAt,
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

func (r *CustomerRepository) List(ctx context.Context, tenantID uuid.UUID, filter domain.CustomerFilter) ([]*domain.Customer, error) {
	query := `
		SELECT id, tenant_id, email, name, phone, tax_id, line1, city, state, zip, country, billing_address, ledger_account_id, referral_code, card_brand, card_last4, card_exp_month, card_exp_year, created_at, updated_at
		FROM customers WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	switch filter.Status {
	case "active":
		query += " AND EXISTS (SELECT 1 FROM subscriptions s WHERE s.customer_id = customers.id AND s.status = 'active')"
	case "inactive":
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
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var customers []*domain.Customer
	for rows.Next() {
		var c domain.Customer
		var addressJSON []byte
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.Email, &c.Name,
			&c.Phone, &c.TaxID,
			&c.BillingAddress.Line1, &c.BillingAddress.City, &c.BillingAddress.State, &c.BillingAddress.Zip, &c.BillingAddress.Country,
			&addressJSON, &c.LedgerAccountID, &c.ReferralCode,
			&c.CardBrand, &c.CardLast4, &c.CardExpMonth, &c.CardExpYear,
			&c.CreatedAt, &c.UpdatedAt,
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

// UpdatePaymentMethod updates the stored card details for a customer
func (r *CustomerRepository) UpdatePaymentMethod(ctx context.Context, customerID uuid.UUID, brand, last4 string, expMonth, expYear int) error {
	query := `UPDATE customers SET card_brand = $1, card_last4 = $2, card_exp_month = $3, card_exp_year = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, brand, last4, expMonth, expYear, customerID)
	return err
}

// CustomerWithExpiringCard holds customer info for card expiry notifications
type CustomerWithExpiringCard struct {
	CustomerID    uuid.UUID
	TenantID      uuid.UUID
	CustomerName  string
	CustomerEmail string
	CardBrand     string
	CardLast4     string
	CardExpMonth  int
	CardExpYear   int
}

// GetCustomersWithExpiringCards finds customers with cards expiring in the given month/year
// who have active subscriptions and have not already been notified.
func (r *CustomerRepository) GetCustomersWithExpiringCards(ctx context.Context, month, year int) ([]CustomerWithExpiringCard, error) {
	query := `
		SELECT c.id, c.tenant_id, COALESCE(c.name, ''), c.email, c.card_brand, c.card_last4, c.card_exp_month, c.card_exp_year
		FROM customers c
		INNER JOIN subscriptions s ON s.customer_id = c.id AND s.status = 'active'
		LEFT JOIN card_expiry_notifications cen ON cen.customer_id = c.id AND cen.card_exp_month = $1 AND cen.card_exp_year = $2
		WHERE c.card_exp_month = $1 AND c.card_exp_year = $2
		  AND cen.id IS NULL
		GROUP BY c.id
	`
	rows, err := r.db.QueryContext(ctx, query, month, year)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []CustomerWithExpiringCard
	for rows.Next() {
		var cust CustomerWithExpiringCard
		if err := rows.Scan(&cust.CustomerID, &cust.TenantID, &cust.CustomerName, &cust.CustomerEmail, &cust.CardBrand, &cust.CardLast4, &cust.CardExpMonth, &cust.CardExpYear); err != nil {
			return nil, err
		}
		results = append(results, cust)
	}
	return results, nil
}

// MarkCardExpiryNotificationSent records that a card expiry notification was sent
func (r *CustomerRepository) MarkCardExpiryNotificationSent(ctx context.Context, customerID, tenantID uuid.UUID, expMonth, expYear int, cardLast4 string) error {
	query := `
		INSERT INTO card_expiry_notifications (tenant_id, customer_id, card_exp_month, card_exp_year, card_last4)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (customer_id, card_exp_month, card_exp_year) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, tenantID, customerID, expMonth, expYear, cardLast4)
	return err
}
