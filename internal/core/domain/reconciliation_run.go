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

// ReconciliationRunDiscrepancy is one persisted disagreement from a recorded
// run — the per-run detail that makes a historical run explainable ("what
// disagreed, by how much, and why") rather than just a count. Mirrors the live
// service discrepancy shape; the amounts are minor units of the run's reporting
// currency.
type ReconciliationRunDiscrepancy struct {
	Type           string     `json:"type"`
	InvoiceID      *uuid.UUID `json:"invoice_id,omitempty"`
	TransactionID  *uuid.UUID `json:"transaction_id,omitempty"`
	ReferenceID    *uuid.UUID `json:"reference_id,omitempty"`
	AccountCode    int        `json:"account_code,omitempty"`
	ExpectedAmount int64      `json:"expected_amount"`
	FoundAmount    int64      `json:"found_amount"`
}

// ReconciliationRunDetail is a recorded run plus its stored discrepancy rows —
// the addressable run object. Discrepancies holds the rows captured at record
// time (capped at the live run's listed maximum). Runs recorded before per-run
// persistence have an empty Discrepancies list while TotalDiscrepancies still
// reflects the true count found at run time; DiscrepanciesTruncated flags when
// fewer rows were stored than counted (cap or a pre-persistence run) so the UI
// never implies the stored rows are the complete set.
type ReconciliationRunDetail struct {
	ReconciliationRun
	Discrepancies          []ReconciliationRunDiscrepancy `json:"discrepancies"`
	DiscrepanciesTruncated bool                           `json:"discrepancies_truncated"`
}
