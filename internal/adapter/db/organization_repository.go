package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type OrganizationRepository struct {
	db *sql.DB
}

func NewOrganizationRepository(db *sql.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	query := `INSERT INTO organizations (id, name, owner_tenant_id, owner_email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, org.ID, org.Name, org.OwnerTenantID, org.OwnerEmail, org.CreatedAt, org.UpdatedAt)
	return err
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	query := `SELECT id, name, owner_tenant_id, owner_email, created_at, updated_at FROM organizations WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanOrganization(row)
}

// scanOrganization reads a row into an Organization, tolerating a NULL
// owner_tenant_id on legacy rows (which then fail every ownership check).
func scanOrganization(row interface{ Scan(...any) error }) (*domain.Organization, error) {
	var org domain.Organization
	var ownerTenant uuid.NullUUID
	if err := row.Scan(&org.ID, &org.Name, &ownerTenant, &org.OwnerEmail, &org.CreatedAt, &org.UpdatedAt); err != nil {
		return nil, err
	}
	if ownerTenant.Valid {
		org.OwnerTenantID = ownerTenant.UUID
	}
	return &org, nil
}

func (r *OrganizationRepository) Update(ctx context.Context, org *domain.Organization) error {
	query := `UPDATE organizations SET name = $1, owner_email = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, org.Name, org.OwnerEmail, org.UpdatedAt, org.ID)
	return err
}

func (r *OrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Detach tenants first
	_, err := r.db.ExecContext(ctx, `UPDATE tenants SET organization_id = NULL, updated_at = NOW() WHERE organization_id = $1`, id)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	return err
}

func (r *OrganizationRepository) ListByOwner(ctx context.Context, ownerTenantID uuid.UUID) ([]*domain.Organization, error) {
	query := `SELECT id, name, owner_tenant_id, owner_email, created_at, updated_at
		FROM organizations WHERE owner_tenant_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, ownerTenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var orgs []*domain.Organization
	for rows.Next() {
		org, err := scanOrganization(rows)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}
	return orgs, rows.Err()
}

func (r *OrganizationRepository) AddTenant(ctx context.Context, orgID, tenantID uuid.UUID) error {
	query := `UPDATE tenants SET organization_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, orgID, tenantID)
	return err
}

func (r *OrganizationRepository) RemoveTenant(ctx context.Context, orgID, tenantID uuid.UUID) error {
	query := `UPDATE tenants SET organization_id = NULL, updated_at = NOW() WHERE id = $1 AND organization_id = $2`
	_, err := r.db.ExecContext(ctx, query, tenantID, orgID)
	return err
}

func (r *OrganizationRepository) ListTenants(ctx context.Context, orgID uuid.UUID) ([]*domain.Tenant, error) {
	query := `SELECT id, name, email, created_at, updated_at FROM tenants WHERE organization_id = $1`
	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var tenants []*domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Email, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tenants, nil
}
