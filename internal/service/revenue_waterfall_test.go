package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestSummarizeWaterfall totals recognized and scheduled across buckets and
// carries the buckets and tenant through unchanged.
func TestSummarizeWaterfall(t *testing.T) {
	tenantID := uuid.New()
	buckets := []domain.RevenueWaterfallBucket{
		{Year: 2026, Month: 1, Recognized: 8000, Scheduled: 0},
		{Year: 2026, Month: 2, Recognized: 0, Scheduled: 10000},
		{Year: 2026, Month: 3, Recognized: 0, Scheduled: 12000},
	}

	w := summarizeWaterfall(tenantID, buckets)

	if w.TotalRecognized != 8000 {
		t.Errorf("TotalRecognized = %d, want 8000", w.TotalRecognized)
	}
	if w.TotalScheduled != 22000 {
		t.Errorf("TotalScheduled = %d, want 22000", w.TotalScheduled)
	}
	if len(w.Buckets) != 3 || w.TenantID != tenantID {
		t.Errorf("buckets/tenant not carried through: got %d buckets, tenant=%v", len(w.Buckets), w.TenantID)
	}
}

// TestSummarizeWaterfall_Empty: no events -> zero totals, empty (non-nil-safe) series.
func TestSummarizeWaterfall_Empty(t *testing.T) {
	w := summarizeWaterfall(uuid.New(), nil)
	if w.TotalRecognized != 0 || w.TotalScheduled != 0 {
		t.Errorf("empty waterfall totals = R%d/S%d, want 0/0", w.TotalRecognized, w.TotalScheduled)
	}
}
