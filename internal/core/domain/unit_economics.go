package domain

// UnitEconomics holds the per-account revenue metrics, all in the tenant's
// reporting currency. LTV depends on a churn rate, which needs MRR history — so
// HasLTV is false (and LTV 0) until there is enough snapshot history and a
// non-zero churn to divide by.
type UnitEconomics struct {
	ReportingCurrency   string  `json:"reporting_currency"`
	MRR                 int64   `json:"mrr"`              // normalized, minor units
	ActiveCustomers     int     `json:"active_customers"` // customers with ≥1 active subscription
	ActiveSubscriptions int     `json:"active_subscriptions"`
	ARPA                int64   `json:"arpa"`               // MRR / active customers, minor units
	ARPU                int64   `json:"arpu"`               // MRR / active subscriptions, minor units
	MonthlyChurnRate    float64 `json:"monthly_churn_rate"` // percent; revenue churn over the trailing period
	LTV                 int64   `json:"ltv"`                // ARPA / monthly churn rate, minor units
	HasLTV              bool    `json:"has_ltv"`
}
