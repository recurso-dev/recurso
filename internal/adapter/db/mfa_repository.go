package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// MFABackupCodeRepository is the Postgres-backed store for hashed one-time MFA
// recovery codes.
type MFABackupCodeRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewMFABackupCodeRepository(db *sql.DB) *MFABackupCodeRepository {
	return &MFABackupCodeRepository{db: db, logger: slog.Default().With("repo", "mfa_backup_code")}
}

func (r *MFABackupCodeRepository) CreateMany(ctx context.Context, codes []*domain.MFABackupCode) error {
	if len(codes) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO mfa_backup_codes (id, user_id, code_hash, used_at, created_at)
		 VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, c := range codes {
		if _, err := stmt.ExecContext(ctx, c.ID, c.UserID, c.CodeHash, c.UsedAt, c.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *MFABackupCodeRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.MFABackupCode, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, code_hash, used_at, created_at
		 FROM mfa_backup_codes WHERE user_id = $1 ORDER BY created_at ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*domain.MFABackupCode
	for rows.Next() {
		var c domain.MFABackupCode
		var usedAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.UserID, &c.CodeHash, &usedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		if usedAt.Valid {
			c.UsedAt = &usedAt.Time
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *MFABackupCodeRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mfa_backup_codes SET used_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *MFABackupCodeRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mfa_backup_codes WHERE user_id = $1`, userID)
	return err
}

// MFALoginTokenRepository is the Postgres-backed store for short-lived MFA
// challenge tokens (only the SHA-256 hash is persisted).
type MFALoginTokenRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewMFALoginTokenRepository(db *sql.DB) *MFALoginTokenRepository {
	return &MFALoginTokenRepository{db: db, logger: slog.Default().With("repo", "mfa_login_token")}
}

func (r *MFALoginTokenRepository) Create(ctx context.Context, t *domain.MFALoginToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO mfa_login_tokens (id, token_hash, user_id, expires_at, used_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.TokenHash, t.UserID, t.ExpiresAt, t.UsedAt, t.CreatedAt,
	)
	return err
}

func (r *MFALoginTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.MFALoginToken, error) {
	var t domain.MFALoginToken
	var usedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, token_hash, user_id, expires_at, used_at, created_at
		 FROM mfa_login_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&t.ID, &t.TokenHash, &t.UserID, &t.ExpiresAt, &usedAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrInvalidMFAToken
	}
	if err != nil {
		return nil, err
	}
	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}
	return &t, nil
}

func (r *MFALoginTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE mfa_login_tokens SET used_at = NOW() WHERE id = $1`, id)
	return err
}
