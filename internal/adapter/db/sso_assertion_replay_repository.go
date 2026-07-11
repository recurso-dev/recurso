package db

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// SSOAssertionReplayRepository is the Postgres-backed replay cache for consumed
// SAML assertion IDs. A single row per assertion enforces one-time use across
// every instance that serves the ACS endpoint.
type SSOAssertionReplayRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewSSOAssertionReplayRepository(db *sql.DB) *SSOAssertionReplayRepository {
	return &SSOAssertionReplayRepository{db: db, logger: slog.Default().With("repo", "sso_assertion_replay")}
}

// MarkConsumed inserts assertionID; the PRIMARY KEY makes the insert atomic, so
// a concurrent second consume for the same ID affects zero rows and is reported
// as a replay. It also opportunistically prunes rows whose assertions have
// expired, keeping the table bounded without a separate sweeper.
func (r *SSOAssertionReplayRepository) MarkConsumed(ctx context.Context, tenantID uuid.UUID, assertionID string, expiresAt time.Time) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO sso_consumed_assertions (assertion_id, tenant_id, expires_at, consumed_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (assertion_id) DO NOTHING`,
		assertionID, tenantID, expiresAt,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// The assertion ID already existed → this SAMLResponse is a replay.
		return domain.ErrSSOAssertionReplay
	}

	// Best-effort prune of expired rows; failure here must not fail the login.
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM sso_consumed_assertions WHERE expires_at < NOW()`); err != nil {
		r.logger.WarnContext(ctx, "prune expired consumed assertions", "err", err)
	}
	return nil
}
