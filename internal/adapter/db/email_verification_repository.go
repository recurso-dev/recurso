package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// EmailVerificationRepository is the Postgres-backed store for single-use email
// verification tokens (only the SHA-256 hash of the token is persisted). It is a
// deliberate mirror of PasswordResetRepository.
type EmailVerificationRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewEmailVerificationRepository(db *sql.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db, logger: slog.Default().With("repo", "email_verification")}
}

func (r *EmailVerificationRepository) Create(ctx context.Context, t *domain.EmailVerificationToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO email_verification_tokens (id, token_hash, user_id, expires_at, used_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.TokenHash, t.UserID, t.ExpiresAt, t.UsedAt, t.CreatedAt,
	)
	return err
}

func (r *EmailVerificationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.EmailVerificationToken, error) {
	var t domain.EmailVerificationToken
	var usedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, token_hash, user_id, expires_at, used_at, created_at
		 FROM email_verification_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&t.ID, &t.TokenHash, &t.UserID, &t.ExpiresAt, &usedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrInvalidVerificationToken
	}
	if err != nil {
		return nil, err
	}
	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}
	return &t, nil
}

func (r *EmailVerificationRepository) MarkUsed(ctx context.Context, id uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE email_verification_tokens SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}
