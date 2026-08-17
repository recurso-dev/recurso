package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// CloudBillingRepository persists the mapping between a signup tenant and the
// Customer that represents it inside the platform (founder) tenant's account.
type CloudBillingRepository struct {
	db *sql.DB
}

func NewCloudBillingRepository(database *sql.DB) *CloudBillingRepository {
	return &CloudBillingRepository{db: database}
}

// GetByTenant returns the mapping for a signup tenant within a platform tenant,
// or (nil, nil) when none exists. The (nil, nil) case is how callers detect
// "not yet provisioned" and stays distinct from a real query error.
func (r *CloudBillingRepository) GetByTenant(ctx context.Context, platformTenantID, tenantID uuid.UUID) (*domain.CloudTenantCustomer, error) {
	m := &domain.CloudTenantCustomer{}
	var subID uuid.NullUUID
	err := r.db.QueryRowContext(ctx, `
		SELECT id, platform_tenant_id, tenant_id, customer_id, subscription_id, created_at
		FROM cloud_tenant_customer
		WHERE platform_tenant_id = $1 AND tenant_id = $2`,
		platformTenantID, tenantID,
	).Scan(&m.ID, &m.PlatformTenantID, &m.TenantID, &m.CustomerID, &subID, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if subID.Valid {
		m.SubscriptionID = &subID.UUID
	}
	return m, nil
}

// Create inserts a new mapping. The UNIQUE (platform_tenant_id, tenant_id)
// constraint makes a concurrent double-provision fail loudly rather than
// duplicate; callers treat that as already-provisioned.
func (r *CloudBillingRepository) Create(ctx context.Context, m *domain.CloudTenantCustomer) error {
	var subID uuid.NullUUID
	if m.SubscriptionID != nil {
		subID = uuid.NullUUID{UUID: *m.SubscriptionID, Valid: true}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cloud_tenant_customer
			(id, platform_tenant_id, tenant_id, customer_id, subscription_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		m.ID, m.PlatformTenantID, m.TenantID, m.CustomerID, subID, m.CreatedAt,
	)
	return err
}
