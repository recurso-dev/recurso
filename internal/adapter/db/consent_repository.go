package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
)

// ConsentRepository handles consent database operations
type ConsentRepository struct {
	db *sql.DB
}

// NewConsentRepository creates a new ConsentRepository
func NewConsentRepository(db *sql.DB) *ConsentRepository {
	return &ConsentRepository{db: db}
}

// Create stores a new consent record
func (r *ConsentRepository) Create(ctx context.Context, tenantID uuid.UUID, record domain.ConsentRecord) (*domain.Consent, error) {
	consent := &domain.Consent{
		ID:             uuid.New(),
		TenantID:       tenantID,
		CustomerID:     record.CustomerID,
		SubscriptionID: record.SubscriptionID,
		ConsentType:    record.ConsentType,
		Granted:        record.Granted,
		GrantedAt:      time.Now(),
		IPAddress:      record.IPAddress,
		UserAgent:      record.UserAgent,
		ConsentText:    record.ConsentText,
		Version:        record.Version,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	query := `
		INSERT INTO consents (
			id, tenant_id, customer_id, subscription_id, consent_type,
			granted, granted_at, ip_address, user_agent, consent_text,
			version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.ExecContext(ctx, query,
		consent.ID,
		consent.TenantID,
		consent.CustomerID,
		consent.SubscriptionID,
		consent.ConsentType,
		consent.Granted,
		consent.GrantedAt,
		consent.IPAddress,
		consent.UserAgent,
		consent.ConsentText,
		consent.Version,
		consent.CreatedAt,
		consent.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return consent, nil
}

// GetByCustomer retrieves all consent records for a customer
func (r *ConsentRepository) GetByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Consent, error) {
	query := `
		SELECT id, tenant_id, customer_id, subscription_id, consent_type,
			granted, granted_at, revoked_at, ip_address, user_agent,
			consent_text, version, created_at, updated_at
		FROM consents
		WHERE tenant_id = $1 AND customer_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []domain.Consent
	for rows.Next() {
		var c domain.Consent
		err := rows.Scan(
			&c.ID, &c.TenantID, &c.CustomerID, &c.SubscriptionID, &c.ConsentType,
			&c.Granted, &c.GrantedAt, &c.RevokedAt, &c.IPAddress, &c.UserAgent,
			&c.ConsentText, &c.Version, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}

	return consents, rows.Err()
}

// GetBySubscription retrieves consent for a specific subscription
func (r *ConsentRepository) GetBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) (*domain.Consent, error) {
	query := `
		SELECT id, tenant_id, customer_id, subscription_id, consent_type,
			granted, granted_at, revoked_at, ip_address, user_agent,
			consent_text, version, created_at, updated_at
		FROM consents
		WHERE tenant_id = $1 AND subscription_id = $2 AND consent_type = $3
		ORDER BY created_at DESC
		LIMIT 1
	`

	var c domain.Consent
	err := r.db.QueryRowContext(ctx, query, tenantID, subscriptionID, domain.ConsentTypeRecurringBilling).Scan(
		&c.ID, &c.TenantID, &c.CustomerID, &c.SubscriptionID, &c.ConsentType,
		&c.Granted, &c.GrantedAt, &c.RevokedAt, &c.IPAddress, &c.UserAgent,
		&c.ConsentText, &c.Version, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// Revoke marks a consent as revoked
func (r *ConsentRepository) Revoke(ctx context.Context, tenantID, consentID uuid.UUID) error {
	query := `
		UPDATE consents
		SET granted = false, revoked_at = $1, updated_at = $1
		WHERE tenant_id = $2 AND id = $3
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), tenantID, consentID)
	return err
}

// RevokeBySubscription revokes consent for a subscription
func (r *ConsentRepository) RevokeBySubscription(ctx context.Context, tenantID, subscriptionID uuid.UUID) error {
	query := `
		UPDATE consents
		SET granted = false, revoked_at = $1, updated_at = $1
		WHERE tenant_id = $2 AND subscription_id = $3 AND granted = true
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), tenantID, subscriptionID)
	return err
}

// GetActiveByType gets active consent of a specific type for a customer
func (r *ConsentRepository) GetActiveByType(ctx context.Context, tenantID, customerID uuid.UUID, consentType domain.ConsentType) (*domain.Consent, error) {
	query := `
		SELECT id, tenant_id, customer_id, subscription_id, consent_type,
			granted, granted_at, revoked_at, ip_address, user_agent,
			consent_text, version, created_at, updated_at
		FROM consents
		WHERE tenant_id = $1 AND customer_id = $2 AND consent_type = $3 AND granted = true
		ORDER BY created_at DESC
		LIMIT 1
	`

	var c domain.Consent
	err := r.db.QueryRowContext(ctx, query, tenantID, customerID, consentType).Scan(
		&c.ID, &c.TenantID, &c.CustomerID, &c.SubscriptionID, &c.ConsentType,
		&c.Granted, &c.GrantedAt, &c.RevokedAt, &c.IPAddress, &c.UserAgent,
		&c.ConsentText, &c.Version, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}
