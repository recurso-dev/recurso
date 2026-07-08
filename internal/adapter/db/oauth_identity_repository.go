package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// OAuthIdentityRepository is the Postgres-backed store for links between
// dashboard users and external identity-provider accounts.
type OAuthIdentityRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewOAuthIdentityRepository(db *sql.DB) *OAuthIdentityRepository {
	return &OAuthIdentityRepository{db: db, logger: slog.Default().With("repo", "oauth_identity")}
}

func (r *OAuthIdentityRepository) Create(ctx context.Context, i *domain.OAuthIdentity) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_oauth_identities (id, user_id, provider, provider_user_id, email, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		i.ID, i.UserID, i.Provider, i.ProviderUserID, normalizeEmail(i.Email), i.CreatedAt,
	)
	if err != nil && strings.Contains(err.Error(), "user_oauth_identities_provider_uid_unique") {
		return domain.ErrDuplicateEmail
	}
	return err
}

func (r *OAuthIdentityRepository) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*domain.OAuthIdentity, error) {
	var i domain.OAuthIdentity
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, provider, provider_user_id, email, created_at
		 FROM user_oauth_identities WHERE provider = $1 AND provider_user_id = $2`,
		provider, providerUserID,
	).Scan(&i.ID, &i.UserID, &i.Provider, &i.ProviderUserID, &i.Email, &i.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *OAuthIdentityRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.OAuthIdentity, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, provider, provider_user_id, email, created_at
		 FROM user_oauth_identities WHERE user_id = $1 ORDER BY created_at ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*domain.OAuthIdentity
	for rows.Next() {
		var i domain.OAuthIdentity
		if err := rows.Scan(&i.ID, &i.UserID, &i.Provider, &i.ProviderUserID, &i.Email, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &i)
	}
	return out, rows.Err()
}
