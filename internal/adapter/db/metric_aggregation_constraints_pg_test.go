package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestCreateMetric_AllAggregations_Postgres guards the billable_metrics DB
// constraints against the class of bug where a shipped aggregation type is
// accepted by the domain/service layer but rejected by an out-of-date CHECK
// constraint at INSERT time. It exercises the REAL repository INSERT (not an
// in-memory struct) against a fully migrated database, so a future aggregation
// added without widening the constraint fails here instead of in production.
//
// Regression: `latest` and `percentile` shipped without the constraint widen
// (migration 000115), so every create 500'd until this was fixed.
func TestCreateMetric_AllAggregations_Postgres(t *testing.T) {
	conn := openProgressiveTestDB(t) // RunMigrations + open
	repo := NewBillableMetricRepository(conn)
	ctx := context.Background()

	run := uuid.NewString()[:8]
	tenantID := uuid.New()
	must(t, conn, `INSERT INTO tenants (id, name, email, created_at, updated_at) VALUES ($1,$2,$3,NOW(),NOW())`,
		tenantID, "Agg-"+run, "agg-"+run+"@t.com")

	cases := []struct {
		agg   domain.AggregationType
		field string
	}{
		{domain.AggregationCount, ""},
		{domain.AggregationSum, ""},
		{domain.AggregationMax, ""},
		{domain.AggregationUnique, "region"},
		{domain.AggregationLatest, ""},
		{domain.AggregationPercentile, "95"},
	}
	for _, c := range cases {
		m := &domain.BillableMetric{
			ID:              uuid.New(),
			TenantID:        tenantID,
			Name:            string(c.agg),
			Code:            string(c.agg) + "_" + run,
			AggregationType: c.agg,
			FieldName:       c.field,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("create %s metric rejected by DB (constraint out of date?): %v", c.agg, err)
		}
	}
}
