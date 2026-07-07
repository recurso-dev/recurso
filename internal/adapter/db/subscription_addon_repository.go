package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type SubscriptionAddonRepository struct {
	db *sql.DB
}

func NewSubscriptionAddonRepository(db *sql.DB) port.SubscriptionAddonRepository {
	return &SubscriptionAddonRepository{db: db}
}

func (r *SubscriptionAddonRepository) Create(ctx context.Context, addon *domain.SubscriptionAddon) error {
	query := `
		INSERT INTO subscription_addons (id, tenant_id, subscription_id, plan_id, quantity, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		addon.ID, addon.TenantID, addon.SubscriptionID, addon.PlanID, addon.Quantity, addon.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert subscription add-on: %w", err)
	}
	return nil
}

func (r *SubscriptionAddonRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SubscriptionAddon, error) {
	query := `
		SELECT id, tenant_id, subscription_id, plan_id, quantity, created_at
		FROM subscription_addons
		WHERE id = $1 AND tenant_id = $2
	`
	addon := &domain.SubscriptionAddon{}
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&addon.ID, &addon.TenantID, &addon.SubscriptionID, &addon.PlanID, &addon.Quantity, &addon.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription add-on: %w", err)
	}
	return addon, nil
}

func (r *SubscriptionAddonRepository) ListBySubscriptionID(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]*domain.SubscriptionAddon, error) {
	query := `
		SELECT id, tenant_id, subscription_id, plan_id, quantity, created_at
		FROM subscription_addons
		WHERE tenant_id = $1 AND subscription_id = $2
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription add-ons: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var addons []*domain.SubscriptionAddon
	for rows.Next() {
		addon := &domain.SubscriptionAddon{}
		if err := rows.Scan(
			&addon.ID, &addon.TenantID, &addon.SubscriptionID, &addon.PlanID, &addon.Quantity, &addon.CreatedAt,
		); err != nil {
			return nil, err
		}
		addons = append(addons, addon)
	}
	return addons, rows.Err()
}

func (r *SubscriptionAddonRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM subscription_addons WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete subscription add-on: %w", err)
	}
	return nil
}
