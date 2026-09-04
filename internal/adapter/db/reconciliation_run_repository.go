package db

import (
	"context"
	"database/sql"
	"errors"
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

// Create records one run summary plus its discrepancy rows in a single
// transaction (defaults id + created_at when unset). The discrepancy rows are
// the already-computed output of the run — persisting them makes the recorded
// run explainable, not just a count. An empty discrepancies slice records a
// clean (or detail-less) run without rows.
func (r *ReconciliationRunRepository) Create(ctx context.Context, run *domain.ReconciliationRun, discrepancies []domain.ReconciliationRunDiscrepancy) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin reconciliation run tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO reconciliation_runs
		 (id, tenant_id, run_by, run_at, invoices_checked, paid_invoices_checked, total_discrepancies, tb_compared, tb_accounts_checked, tb_transfers_checked)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		run.ID, run.TenantID, run.RunBy, run.RunAt, run.InvoicesChecked, run.PaidInvoicesChecked,
		run.TotalDiscrepancies, run.TBCompared, run.TBAccountsChecked, run.TBTransfersChecked); err != nil {
		return fmt.Errorf("failed to record reconciliation run: %w", err)
	}

	for i, d := range discrepancies {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO reconciliation_run_discrepancies
			 (run_id, tenant_id, type, invoice_id, transaction_id, reference_id, account_code, expected_amount, found_amount, seq)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			run.ID, run.TenantID, d.Type, d.InvoiceID, d.TransactionID, d.ReferenceID,
			d.AccountCode, d.ExpectedAmount, d.FoundAmount, i); err != nil {
			return fmt.Errorf("failed to record reconciliation discrepancy: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit reconciliation run: %w", err)
	}
	return nil
}

// GetByID returns one recorded run with its stored discrepancy rows, tenant-
// scoped. Returns (nil, nil) when no such run exists for the tenant (handler →
// 404). DiscrepanciesTruncated is set when fewer rows were stored than the run
// counted (the live-run listing cap, or a run recorded before per-run
// persistence) so the caller never treats the stored rows as the complete set.
func (r *ReconciliationRunRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ReconciliationRunDetail, error) {
	var d domain.ReconciliationRunDetail
	err := r.db.QueryRowContext(ctx,
		`SELECT `+reconciliationRunColumns+` FROM reconciliation_runs
		 WHERE id = $1 AND tenant_id = $2`, id, tenantID).
		Scan(&d.ID, &d.TenantID, &d.RunBy, &d.RunAt,
			&d.InvoicesChecked, &d.PaidInvoicesChecked, &d.TotalDiscrepancies,
			&d.TBCompared, &d.TBAccountsChecked, &d.TBTransfersChecked, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reconciliation run: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT type, invoice_id, transaction_id, reference_id, account_code, expected_amount, found_amount
		 FROM reconciliation_run_discrepancies
		 WHERE run_id = $1 AND tenant_id = $2 ORDER BY seq, id`, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load reconciliation discrepancies: %w", err)
	}
	defer func() { _ = rows.Close() }()

	d.Discrepancies = []domain.ReconciliationRunDiscrepancy{}
	for rows.Next() {
		var row domain.ReconciliationRunDiscrepancy
		if err := rows.Scan(&row.Type, &row.InvoiceID, &row.TransactionID, &row.ReferenceID,
			&row.AccountCode, &row.ExpectedAmount, &row.FoundAmount); err != nil {
			return nil, fmt.Errorf("failed to scan reconciliation discrepancy: %w", err)
		}
		d.Discrepancies = append(d.Discrepancies, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	d.DiscrepanciesTruncated = len(d.Discrepancies) < d.TotalDiscrepancies
	return &d, nil
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
