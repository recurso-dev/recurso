package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// WaitlistRepository stores Recurso Cloud waitlist signups (ENG-12).
// Platform-level — no tenant scoping.
type WaitlistRepository struct {
	db *sql.DB
}

func NewWaitlistRepository(db *sql.DB) *WaitlistRepository {
	return &WaitlistRepository{db: db}
}

// Add records a signup. Emails are stored lower-cased and deduplicated —
// re-joining is a no-op and reports isNew=false.
func (r *WaitlistRepository) Add(ctx context.Context, email, name, company, source string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO waitlist_signups (id, email, name, company, source)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (email) DO NOTHING`,
		uuid.New(), strings.ToLower(strings.TrimSpace(email)), name, company, source)
	if err != nil {
		return false, fmt.Errorf("failed to add waitlist signup: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
