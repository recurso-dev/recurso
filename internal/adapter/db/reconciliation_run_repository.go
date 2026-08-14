package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// ReconciliationRunRepository persists the summary of each recorded ledger
// reconciliation (the audit trail; the discrepancy list itself stays ephemeral).
type ReconciliationRunRepository struct {
	db *sql.DB
}

func NewReconciliationRunRepository(db *sql.DB) *ReconciliationRunRepository {
	return &ReconciliationRunRepository{db: db}
}

const reconciliationRunColumns = `id, tenant_id, run_by, run_at, invoices_checked, paid_invoices_checked, total_discrepancies, tb_compared, tb_accounts_checked, tb_transfers_checked, created_at`

// Create records one run summary (defaults id + created_at when unset).
func (r *ReconciliationRunRepository) Create(ctx context.Context, run *domain.ReconciliationRun) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO reconciliation_runs
		 (id, tenant_id, run_by, run_at, invoices_checked, paid_invoices_checked, total_discrepancies, tb_compared, tb_accounts_checked, tb_transfers_checked)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		run.ID, run.TenantID, run.RunBy, run.RunAt, run.InvoicesChecked, run.PaidInvoicesChecked,
		run.TotalDiscrepancies, run.TBCompared, run.TBAccountsChecked, run.TBTransfersChecked)
	if err != nil {
		return fmt.Errorf("failed to record reconciliation run: %w", err)
	}
	return nil
}

// ListByTenant returns a tenant's recorded runs, newest first, capped by limit.
func (r *ReconciliationRunRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.ReconciliationRun, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+reconciliationRunColumns+` FROM reconciliation_runs
		 WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list reconciliation runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	runs := []domain.ReconciliationRun{}
	for rows.Next() {
		var run domain.ReconciliationRun
		if err := rows.Scan(&run.ID, &run.TenantID, &run.RunBy, &run.RunAt,
			&run.InvoicesChecked, &run.PaidInvoicesChecked, &run.TotalDiscrepancies,
			&run.TBCompared, &run.TBAccountsChecked, &run.TBTransfersChecked, &run.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan reconciliation run: %w", err)
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}
