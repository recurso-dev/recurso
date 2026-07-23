package domain

import (
	"time"

	"github.com/google/uuid"
)

// CreditStatement is a customer's consolidated account-credit statement: the
// spendable balance, every grant (credit note) that makes it up, and every
// invoice application that drew it down. It reads existing GL-backed data — the
// spendable balance equals the credit applier's view (type=adjustment,
// status=issued, balance>0) and reconciles to the Customer Credit ledger account.
type CreditStatement struct {
	CustomerID   uuid.UUID               `json:"customer_id"`
	Balances     []CreditBalanceLine     `json:"balances"`     // spendable, per (currency, entity)
	Grants       []CreditNote            `json:"grants"`       // all credit notes, newest first
	Applications []CreditApplicationLine `json:"applications"` // draw-down history, newest first
	Summary      []CreditSummaryLine     `json:"summary"`      // per-currency rollup
}

// CreditBalanceLine is the spendable credit balance for one (currency, entity).
type CreditBalanceLine struct {
	Currency string     `json:"currency"`
	EntityID *uuid.UUID `json:"entity_id,omitempty"`
	Balance  int64      `json:"balance"` // minor units
}

// CreditApplicationLine is one draw-down: a credit note settling an invoice.
type CreditApplicationLine struct {
	CreditNoteID  uuid.UUID `json:"credit_note_id" db:"credit_note_id"`
	InvoiceID     uuid.UUID `json:"invoice_id" db:"invoice_id"`
	InvoiceNumber string    `json:"invoice_number" db:"invoice_number"`
	Currency      string    `json:"currency" db:"currency"`
	Amount        int64     `json:"amount" db:"amount"` // minor units
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// CreditSummaryLine rolls a customer's credit up by currency: total ever issued
// as spendable adjustment credit, total applied to invoices, and the current
// spendable balance.
type CreditSummaryLine struct {
	Currency       string `json:"currency"`
	TotalIssued    int64  `json:"total_issued"`
	TotalApplied   int64  `json:"total_applied"`
	CurrentBalance int64  `json:"current_balance"`
}
