package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CompareReportRepository persists migration Compare runs so each proof of a
// migration survives as a citable, printable receipt.
type CompareReportRepository struct {
	db *sql.DB
}

func NewCompareReportRepository(db *sql.DB) *CompareReportRepository {
	return &CompareReportRepository{db: db}
}

// StoredCompareReport is a persisted run: the raw report plus its envelope.
type StoredCompareReport struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Source      string          `json:"source"`
	Ready       bool            `json:"ready"`
	Report      json.RawMessage `json:"report"`
	GeneratedAt time.Time       `json:"generated_at"`
}

// Create stores a run and returns its id.
func (r *CompareReportRepository) Create(ctx context.Context, tenantID uuid.UUID, source string, ready bool, report any) (uuid.UUID, error) {
	raw, err := json.Marshal(report)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal compare report: %w", err)
	}
	id := uuid.New()
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO import_compare_reports (id, tenant_id, source, ready, report) VALUES ($1,$2,$3,$4,$5)`,
		id, tenantID, source, ready, raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("store compare report: %w", err)
	}
	return id, nil
}

// GetByID returns a tenant's stored run, or (nil, nil) when absent — the
// tenant scope in the query is the authorization boundary.
func (r *CompareReportRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*StoredCompareReport, error) {
	rec := &StoredCompareReport{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, source, ready, report, generated_at
		 FROM import_compare_reports WHERE tenant_id=$1 AND id=$2`, tenantID, id,
	).Scan(&rec.ID, &rec.TenantID, &rec.Source, &rec.Ready, &rec.Report, &rec.GeneratedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get compare report: %w", err)
	}
	return rec, nil
}

// List returns a tenant's runs, newest first.
func (r *CompareReportRepository) List(ctx context.Context, tenantID uuid.UUID, limit int) ([]*StoredCompareReport, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, source, ready, report, generated_at
		 FROM import_compare_reports WHERE tenant_id=$1 ORDER BY generated_at DESC LIMIT $2`,
		tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list compare reports: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []*StoredCompareReport
	for rows.Next() {
		rec := &StoredCompareReport{}
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.Source, &rec.Ready, &rec.Report, &rec.GeneratedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
