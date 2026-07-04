package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// MagicLinkRepository implements port.MagicLinkRepository
type MagicLinkRepository struct {
	db *sql.DB
}

func NewMagicLinkRepository(db *sql.DB) *MagicLinkRepository {
	return &MagicLinkRepository{db: db}
}

func (r *MagicLinkRepository) Create(ctx context.Context, link *domain.MagicLink) error {
	query := `
		INSERT INTO magic_links (id, customer_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := r.db.ExecContext(ctx, query, link.ID, link.CustomerID, link.Token, link.ExpiresAt)
	return err
}

func (r *MagicLinkRepository) GetByToken(ctx context.Context, token string) (*domain.MagicLink, error) {
	query := `
		SELECT id, customer_id, token, expires_at, used_at, created_at
		FROM magic_links WHERE token = $1
	`
	var link domain.MagicLink
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&link.ID,
		&link.CustomerID,
		&link.Token,
		&link.ExpiresAt,
		&link.UsedAt,
		&link.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *MagicLinkRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE magic_links SET used_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *MagicLinkRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM magic_links WHERE expires_at < NOW()`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// PortalSessionRepository implements port.PortalSessionRepository
type PortalSessionRepository struct {
	db *sql.DB
}

func NewPortalSessionRepository(db *sql.DB) *PortalSessionRepository {
	return &PortalSessionRepository{db: db}
}

func (r *PortalSessionRepository) Create(ctx context.Context, session *domain.PortalSession) error {
	query := `
		INSERT INTO portal_sessions (id, customer_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := r.db.ExecContext(ctx, query, session.ID, session.CustomerID, session.Token, session.ExpiresAt)
	return err
}

func (r *PortalSessionRepository) GetByToken(ctx context.Context, token string) (*domain.PortalSession, error) {
	query := `
		SELECT id, customer_id, token, expires_at, created_at
		FROM portal_sessions WHERE token = $1
	`
	var session domain.PortalSession
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID,
		&session.CustomerID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *PortalSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM portal_sessions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PortalSessionRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM portal_sessions WHERE expires_at < NOW()`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// Helper to generate secure tokens
func GenerateSecureToken() string {
	id := uuid.New()
	return id.String() + "-" + uuid.New().String()
}

// Default expiry durations
const (
	MagicLinkExpiry     = 15 * time.Minute
	PortalSessionExpiry = 24 * time.Hour * 7 // 7 days
)
