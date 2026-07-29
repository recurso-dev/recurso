package db

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ImportRefRepository is the Postgres-backed store for import idempotency
// mappings (source object id → Recurso record id).
type ImportRefRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewImportRefRepository(db *sql.DB) *ImportRefRepository {
	return &ImportRefRepository{db: db, logger: slog.Default().With("repo", "import_ref")}
}

func (r *ImportRefRepository) Create(ctx context.Context, ref *domain.ImportExternalRef) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO import_external_refs (id, tenant_id, source, kind, external_id, recurso_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		ref.ID, ref.TenantID, ref.Source, ref.Kind, ref.ExternalID, ref.RecursoID,
	)
	if err != nil && strings.Contains(err.Error(), "import_external_refs_unique") {
		return domain.ErrDuplicateImportRef
	}
	return err
}

func (r *ImportRefRepository) ListExternalIDs(ctx context.Context, tenantID uuid.UUID, source string) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT external_id FROM import_external_refs WHERE tenant_id = $1 AND source = $2`,
		tenantID, source,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
