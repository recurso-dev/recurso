package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// AuditLogRepository is the Postgres implementation of
// port.AuditLogRepository. Insert-only by design; the table's trigger
// rejects UPDATE/DELETE.
type AuditLogRepository struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) port.AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Insert(ctx context.Context, a *domain.AuditLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, tenant_id, actor, action, entity_type, entity_id, status, request_body, ip, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		a.ID, a.TenantID, a.Actor, a.Action, a.EntityType, a.EntityID, a.Status, a.RequestBody, a.IP, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}
	return nil
}

func (r *AuditLogRepository) List(ctx context.Context, tenantID uuid.UUID, filter domain.AuditLogFilter) ([]domain.AuditLog, error) {
	query := `
		SELECT id, tenant_id, actor, action, entity_type, entity_id, status, request_body, ip, created_at
		FROM audit_logs
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	if filter.EntityType != "" {
		args = append(args, filter.EntityType)
		query += fmt.Sprintf(" AND entity_type = $%d", len(args))
	}
	if filter.EntityID != "" {
		args = append(args, filter.EntityID)
		query += fmt.Sprintf(" AND entity_id = $%d", len(args))
	}
	if filter.Actor != "" {
		args = append(args, filter.Actor)
		query += fmt.Sprintf(" AND actor = $%d", len(args))
	}
	if !filter.From.IsZero() {
		args = append(args, filter.From)
		query += fmt.Sprintf(" AND created_at >= $%d", len(args))
	}
	if !filter.To.IsZero() {
		args = append(args, filter.To)
		query += fmt.Sprintf(" AND created_at < $%d", len(args))
	}
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))
	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	logs := []domain.AuditLog{}
	for rows.Next() {
		var a domain.AuditLog
		if err := rows.Scan(&a.ID, &a.TenantID, &a.Actor, &a.Action, &a.EntityType, &a.EntityID,
			&a.Status, &a.RequestBody, &a.IP, &a.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, a)
	}
	return logs, rows.Err()
}
