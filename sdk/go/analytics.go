package recurso

import (
	"context"
	"net/http"
	"time"
)

// MRRCurrencyBreakdown is per-currency MRR before and after FX normalization.
type MRRCurrencyBreakdown struct {
	Currency        string  `json:"currency"`
	Amount          int64   `json:"amount"`
	ConvertedAmount int64   `json:"converted_amount"`
	Rate            float64 `json:"rate"`
	Subscriptions   int     `json:"subscriptions"`
	Error           string  `json:"error,omitempty"`
}

// FXSnapshot is the FX rate set used to normalize MRR.
type FXSnapshot struct {
	Rates  map[string]float64 `json:"rates"`
	Source string             `json:"source"`
	AsOf   time.Time          `json:"as_of"`
}

// MRRMetrics is the monthly-recurring-revenue report.
type MRRMetrics struct {
	Currency          string                 `json:"currency"`
	Amount            int64                  `json:"amount"`
	MRR               int64                  `json:"mrr"`
	NormalizedMRR     int64                  `json:"normalized_mrr"`
	ReportingCurrency string                 `json:"reporting_currency"`
	Breakdown         []MRRCurrencyBreakdown `json:"breakdown"`
	FX                FXSnapshot             `json:"fx"`
}

// AnalyticsService groups the revenue-analytics endpoints.
type AnalyticsService struct{ client *Client }

// MRR returns the tenant's current monthly recurring revenue, FX-normalized to
// the reporting currency.
func (s *AnalyticsService) MRR(ctx context.Context) (*MRRMetrics, error) {
	var out MRRMetrics
	if err := s.client.do(ctx, http.MethodGet, "/analytics/mrr", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
