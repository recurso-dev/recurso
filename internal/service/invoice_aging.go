package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// InvoiceAgingStore reads AR aging aggregates for a tenant.
type InvoiceAgingStore interface {
	GetInvoiceAgingRows(ctx context.Context, tenantID uuid.UUID) ([]domain.InvoiceAgingRow, error)
}

// SetInvoiceAgingStore wires the invoice-aging source.
func (s *AnalyticsService) SetInvoiceAgingStore(store InvoiceAgingStore) {
	s.agingStore = store
}

// GetInvoiceAging returns outstanding receivables bucketed by how far past due
// they are, normalized to the tenant's reporting currency. All buckets are
// present (zero when empty) and ordered current → 90+.
func (s *AnalyticsService) GetInvoiceAging(ctx context.Context, tenantID uuid.UUID) (*domain.InvoiceAgingReport, error) {
	rows, err := s.agingStore.GetInvoiceAgingRows(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	reporting := s.resolveReportingCurrency(ctx, tenantID)
	normalizer := newFXNormalizer(s.fxProvider, s.fxFallback)

	idx := make(map[string]int, len(domain.InvoiceAgingBuckets))
	buckets := make([]domain.InvoiceAgingBucket, len(domain.InvoiceAgingBuckets))
	for i, label := range domain.InvoiceAgingBuckets {
		buckets[i] = domain.InvoiceAgingBucket{Label: label}
		idx[label] = i
	}

	report := &domain.InvoiceAgingReport{ReportingCurrency: reporting}
	for _, row := range rows {
		i, ok := idx[row.Bucket]
		if !ok {
			continue
		}
		cur := row.Currency
		if cur == "" {
			cur = reporting
		}
		amt, _, err := normalizer.convert(ctx, row.Amount, cur, reporting)
		if err != nil {
			continue // unconvertible currency excluded from the normalized total
		}
		buckets[i].Amount += amt
		buckets[i].Count += row.Count
		report.TotalOutstanding += amt
		report.TotalCount += row.Count
	}
	report.Buckets = buckets
	return report, nil
}
