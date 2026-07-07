package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// SessionRepository is the Postgres-backed store for opaque login sessions.
type SessionRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db, logger: slog.Default().With("repo", "session")}
}

func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	var userAgent sql.NullString
	if s.UserAgent != "" {
		userAgent = sql.NullString{String: s.UserAgent, Valid: true}
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, token_hash, user_id, tenant_id, expires_at, created_at, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.TokenHash, s.UserID, s.TenantID, s.ExpiresAt, s.CreatedAt, userAgent,
	)
	return err
}

func (r *SessionRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	var s domain.Session
	var userAgent sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, token_hash, user_id, tenant_id, expires_at, created_at, user_agent
		 FROM sessions WHERE token_hash = $1`, tokenHash,
	).Scan(&s.ID, &s.TokenHash, &s.UserID, &s.TenantID, &s.ExpiresAt, &s.CreatedAt, &userAgent)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if userAgent.Valid {
		s.UserAgent = userAgent.String
	}
	return &s, nil
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *SessionRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}
