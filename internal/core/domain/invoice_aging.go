package domain

// InvoiceAgingBuckets is the fixed, ordered set of AR aging buckets by how far
// past due an open invoice is.
var InvoiceAgingBuckets = []string{"current", "1-30", "31-60", "61-90", "90+"}

// InvoiceAgingRow is a per-currency, per-bucket aggregate straight from the DB
// (outstanding = amount_remaining, in the invoice's own currency).
type InvoiceAgingRow struct {
	Currency string
	Bucket   string
	Count    int
	Amount   int64
}

// InvoiceAgingBucket is one aging band after currency normalization.
type InvoiceAgingBucket struct {
	Label  string `json:"label"`
	Count  int    `json:"count"`
	Amount int64  `json:"amount"` // outstanding, reporting currency minor units
}

// InvoiceAgingReport is the outstanding-receivables aging: how much unpaid
// invoice value sits in each band, in the tenant's reporting currency.
type InvoiceAgingReport struct {
	ReportingCurrency string               `json:"reporting_currency"`
	Buckets           []InvoiceAgingBucket `json:"buckets"`
	TotalOutstanding  int64                `json:"total_outstanding"`
	TotalCount        int                  `json:"total_count"`
}
