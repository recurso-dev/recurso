package domain

import (
	"time"

	"github.com/google/uuid"
)

// ReconciliationRun is a persisted summary of one recorded ledger reconciliation
// — the audit trail of "when was it checked, by whom, and did it tie out?".
// Only the counts are kept; the per-run discrepancy list stays ephemeral. A run
// is balanced when TotalDiscrepancies == 0.
type ReconciliationRun struct {
	ID                  uuid.UUID  `json:"id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	RunBy               *uuid.UUID `json:"run_by,omitempty"`
	RunAt               time.Time  `json:"run_at"`
	InvoicesChecked     int        `json:"invoices_checked"`
	PaidInvoicesChecked int        `json:"paid_invoices_checked"`
	TotalDiscrepancies  int        `json:"total_discrepancies"`
	TBCompared          bool       `json:"tb_compared"`
	TBAccountsChecked   int        `json:"tb_accounts_checked"`
	TBTransfersChecked  int        `json:"tb_transfers_checked"`
	CreatedAt           time.Time  `json:"created_at"`
}

// Balanced reports whether the run found the books tying out.
func (r *ReconciliationRun) Balanced() bool { return r.TotalDiscrepancies == 0 }
