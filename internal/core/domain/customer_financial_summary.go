package domain

import "github.com/google/uuid"

// CustomerFinancialSummaryCurrency is a customer's invoice-derived financial
// position in ONE currency. Amounts are minor units (int64). Money is never
// summed across currencies — the customer page shows one block per currency —
// so this is reported per-currency, mirroring the credit statement.
type CustomerFinancialSummaryCurrency struct {
	Currency string `json:"currency"`
	// Outstanding is money still owed: SUM(amount_remaining) over open + past_due
	// invoices. Written-off (uncollectible) invoices are excluded — they are no
	// longer expected to collect.
	Outstanding int64 `json:"outstanding"`
	// PastDue is the past_due slice of Outstanding, with its invoice count — the
	// exceptions-first signal for the customer page.
	PastDue      int64 `json:"past_due"`
	PastDueCount int   `json:"past_due_count"`
	// Billed is lifetime issued value: SUM(total) over invoices that actually
	// issued (not draft, not void). Paid is SUM(amount_paid) over the same set.
	Billed int64 `json:"billed"`
	Paid   int64 `json:"paid"`
}

// CustomerFinancialSummary is the customer's whole invoice-derived position,
// one row per currency. It exists because "currently owed" cannot be computed
// correctly on the client from a paginated invoice list — it must aggregate
// every invoice, server-side, per currency.
type CustomerFinancialSummary struct {
	CustomerID uuid.UUID                          `json:"customer_id"`
	Currencies []CustomerFinancialSummaryCurrency `json:"currencies"`
}
