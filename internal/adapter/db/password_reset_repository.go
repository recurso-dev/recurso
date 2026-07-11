package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// PasswordResetRepository is the Postgres-backed store for single-use password
// reset tokens (only the SHA-256 hash of the token is persisted).
type PasswordResetRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewPasswordResetRepository(db *sql.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db, logger: slog.Default().With("repo", "password_reset")}
}

func (r *PasswordResetRepository) Create(ctx context.Context, t *domain.PasswordResetToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (id, token_hash, user_id, expires_at, used_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.TokenHash, t.UserID, t.ExpiresAt, t.UsedAt, t.CreatedAt,
	)
	return err
}

func (r *PasswordResetRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	var t domain.PasswordResetToken
	var usedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, token_hash, user_id, expires_at, used_at, created_at
		 FROM password_reset_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&t.ID, &t.TokenHash, &t.UserID, &t.ExpiresAt, &usedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrInvalidResetToken
	}
	if err != nil {
		return nil, err
	}
	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}
	return &t, nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`, id)
	return err
}
