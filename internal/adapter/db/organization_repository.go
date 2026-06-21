package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

type OrganizationRepository struct {
	db *sql.DB
}

func NewOrganizationRepository(db *sql.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	query := `INSERT INTO organizations (id, name, owner_email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, org.ID, org.Name, org.OwnerEmail, org.CreatedAt, org.UpdatedAt)
	return err
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	query := `SELECT id, name, owner_email, created_at, updated_at FROM organizations WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	var org domain.Organization
	err := row.Scan(&org.ID, &org.Name, &org.OwnerEmail, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrganizationRepository) AddTenant(ctx context.Context, orgID, tenantID uuid.UUID) error {
	query := `UPDATE tenants SET organization_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, orgID, tenantID)
	return err
}

func (r *OrganizationRepository) ListTenants(ctx context.Context, orgID uuid.UUID) ([]*domain.Tenant, error) {
	query := `SELECT id, name, email, created_at, updated_at FROM tenants WHERE organization_id = $1`
	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []*domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Email, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, &t)
	}
	return tenants, nil
}
