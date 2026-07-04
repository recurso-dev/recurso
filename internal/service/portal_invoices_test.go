package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// --- Mocks for portal invoice tests ---

type mockInvoiceRepoForPortal struct {
	port.InvoiceRepository
	gotCustomerID uuid.UUID
	invoices      []*domain.Invoice
	err           error
}

func (m *mockInvoiceRepoForPortal) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error) {
	m.gotCustomerID = customerID
	return m.invoices, m.err
}

func TestPortalGetCustomerInvoices_DelegatesToRepo(t *testing.T) {
	customerID := uuid.New()
	want := []*domain.Invoice{
		{ID: uuid.New(), CustomerID: customerID, Total: 118000},
		{ID: uuid.New(), CustomerID: customerID, Total: 5900},
	}
	repo := &mockInvoiceRepoForPortal{invoices: want}
	svc := NewPortalService(nil, repo, nil, nil, nil, nil, "")

	got, err := svc.GetCustomerInvoices(context.Background(), customerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.gotCustomerID != customerID {
		t.Errorf("repo called with customerID %v, want %v", repo.gotCustomerID, customerID)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d invoices, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("invoice[%d] = %v, want same pointer as repo result", i, got[i])
		}
	}
}

func TestPortalGetCustomerInvoices_PropagatesError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	repo := &mockInvoiceRepoForPortal{err: repoErr}
	svc := NewPortalService(nil, repo, nil, nil, nil, nil, "")

	got, err := svc.GetCustomerInvoices(context.Background(), uuid.New())
	if !errors.Is(err, repoErr) {
		t.Fatalf("error = %v, want repo error %v unchanged", err, repoErr)
	}
	if got != nil {
		t.Errorf("expected nil invoices on error, got %v", got)
	}
}
