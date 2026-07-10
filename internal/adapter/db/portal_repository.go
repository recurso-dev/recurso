package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// hashPortalToken is the at-rest form of a portal magic-link / session token.
// Only the SHA-256 is stored, so a database read never yields a usable token
// (mirrors the dashboard sessions' token_hash). Lookups hash the presented token
// and compare.
func hashPortalToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

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
	_, err := r.db.ExecContext(ctx, query, link.ID, link.CustomerID, hashPortalToken(link.Token), link.ExpiresAt)
	return err
}

func (r *MagicLinkRepository) GetByToken(ctx context.Context, token string) (*domain.MagicLink, error) {
	query := `
		SELECT id, customer_id, token, expires_at, used_at, created_at
		FROM magic_links WHERE token = $1
	`
	var link domain.MagicLink
	err := r.db.QueryRowContext(ctx, query, hashPortalToken(token)).Scan(
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

// MarkUsed atomically stamps used_at only if the link is still unused, and
// reports whether THIS call claimed it. The `AND used_at IS NULL` guard closes
// the single-use race: of two concurrent verifies, only one affects a row.
func (r *MagicLinkRepository) MarkUsed(ctx context.Context, id uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE magic_links SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
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
	_, err := r.db.ExecContext(ctx, query, session.ID, session.CustomerID, hashPortalToken(session.Token), session.ExpiresAt)
	return err
}

func (r *PortalSessionRepository) GetByToken(ctx context.Context, token string) (*domain.PortalSession, error) {
	query := `
		SELECT id, customer_id, token, expires_at, created_at
		FROM portal_sessions WHERE token = $1
	`
	var session domain.PortalSession
	err := r.db.QueryRowContext(ctx, query, hashPortalToken(token)).Scan(
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
