package domain

// RevenueSegment is one slice of MRR (e.g. a plan), in the reporting currency.
type RevenueSegment struct {
	Key           string  `json:"key"`
	Label         string  `json:"label"`
	MRR           int64   `json:"mrr"` // reporting currency, minor units
	Subscriptions int     `json:"subscriptions"`
	SharePct      float64 `json:"share_pct"`
}

// RevenueByPlanReport breaks current MRR down by plan, largest first.
type RevenueByPlanReport struct {
	ReportingCurrency string           `json:"reporting_currency"`
	TotalMRR          int64            `json:"total_mrr"`
	Segments          []RevenueSegment `json:"segments"`
}

// RevenueByGeographyReport breaks current MRR down by customer country.
type RevenueByGeographyReport struct {
	ReportingCurrency string           `json:"reporting_currency"`
	TotalMRR          int64            `json:"total_mrr"`
	Segments          []RevenueSegment `json:"segments"`
}
