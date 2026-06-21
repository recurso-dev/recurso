package accounting

import (
	"context"
	"log"

	"github.com/recur-so/recurso/internal/core/domain"
)

type MockAccountingAdapter struct{}

func NewMockAccountingAdapter() *MockAccountingAdapter {
	return &MockAccountingAdapter{}
}

func (m *MockAccountingAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer) error {
	log.Printf("[Accounting Mock] Syncing Customer: %s (%s)", customer.ID, domain.PtrToString(customer.Name))
	return nil
}

func (m *MockAccountingAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice) error {
	log.Printf("[Accounting Mock] Syncing Invoice: %s (Total: %d)", invoice.ID, invoice.Total)
	return nil
}

func (m *MockAccountingAdapter) SyncProduct(ctx context.Context, plan *domain.Plan) error {
	log.Printf("[Accounting Mock] Syncing Product: %s (%s)", plan.ID, plan.Name)
	return nil
}
