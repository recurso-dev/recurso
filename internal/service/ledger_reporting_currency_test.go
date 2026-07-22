package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type fakeTenantLookup struct {
	tenant *domain.Tenant
	err    error
}

func (f *fakeTenantLookup) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return f.tenant, f.err
}

// ReportingCurrency prefers the tenant's base currency, falling back to the
// configured default; so the ledger reports (trial balance, deferred
// rollforward, close pack) can be formatted with the right currency exponent.
func TestLedgerReportingCurrency(t *testing.T) {
	tid := uuid.New()

	t.Run("tenant base currency wins", func(t *testing.T) {
		s := NewLedgerService(nil, nil)
		s.SetReporting(&fakeTenantLookup{tenant: &domain.Tenant{ID: tid, BaseCurrency: "JPY"}}, "USD")
		if got := s.ReportingCurrency(context.Background(), tid); got != "JPY" {
			t.Errorf("got %q, want JPY (tenant base currency)", got)
		}
	})

	t.Run("falls back to the configured default when tenant has no base currency", func(t *testing.T) {
		s := NewLedgerService(nil, nil)
		s.SetReporting(&fakeTenantLookup{tenant: &domain.Tenant{ID: tid}}, "EUR")
		if got := s.ReportingCurrency(context.Background(), tid); got != "EUR" {
			t.Errorf("got %q, want EUR (default)", got)
		}
	})

	t.Run("falls back on lookup error", func(t *testing.T) {
		s := NewLedgerService(nil, nil)
		s.SetReporting(&fakeTenantLookup{err: errors.New("db down")}, "GBP")
		if got := s.ReportingCurrency(context.Background(), tid); got != "GBP" {
			t.Errorf("got %q, want GBP (default on error)", got)
		}
	})

	t.Run("defaults to USD with no wiring", func(t *testing.T) {
		s := NewLedgerService(nil, nil)
		if got := s.ReportingCurrency(context.Background(), tid); got != "USD" {
			t.Errorf("got %q, want USD (nil-safe default)", got)
		}
	})
}
