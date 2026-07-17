package service

import (
	"testing"

	"github.com/recurso-dev/recurso/internal/adapter/accounting"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/residency"
)

// Under RESIDENCY_MODE=self_hosted, an existing QuickBooks/Xero connection row
// must not produce a real (network-egressing) adapter — the service falls back
// to the configured mock gateway. Tally stays real: it is a local file export.
func TestGetAdapterForConnection_ResidencyBlocksSaaS(t *testing.T) {
	t.Setenv(residency.EnvVar, residency.ModeSelfHosted)

	mock := accounting.NewMockAccountingAdapter()
	svc := NewAccountingService(mock, nil, nil, nil)

	for _, provider := range []string{"quickbooks", "xero"} {
		conn := &domain.AccountingConnection{Provider: provider, AccessToken: "tok", RealmID: "realm"}
		if got := svc.getAdapterForConnection(conn); got != any(mock) {
			t.Errorf("%s: expected mock fallback under residency mode, got %T", provider, got)
		}
	}

	tally := svc.getAdapterForConnection(&domain.AccountingConnection{Provider: "tally"})
	if _, ok := tally.(*accounting.TallyAdapter); !ok {
		t.Errorf("tally must remain available under residency mode (local export), got %T", tally)
	}
}

// Without residency mode, the same connections produce real adapters.
func TestGetAdapterForConnection_NormalModeUsesRealAdapters(t *testing.T) {
	t.Setenv(residency.EnvVar, "")

	mock := accounting.NewMockAccountingAdapter()
	svc := NewAccountingService(mock, nil, nil, nil)

	got := svc.getAdapterForConnection(&domain.AccountingConnection{Provider: "quickbooks", AccessToken: "tok", RealmID: "realm"})
	if _, ok := got.(*accounting.QuickBooksAdapter); !ok {
		t.Errorf("expected real QuickBooks adapter outside residency mode, got %T", got)
	}
}
