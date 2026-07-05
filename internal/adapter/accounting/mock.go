package accounting

import (
	"context"
	"log"

	"github.com/swapnull-in/recur-so/internal/core/domain"
)

type MockAccountingAdapter struct{}

func NewMockAccountingAdapter() *MockAccountingAdapter {
	return &MockAccountingAdapter{}
}

func (m *MockAccountingAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer) (string, error) {
	log.Printf("[Accounting Mock] Syncing Customer: %s (%s)", customer.ID, domain.PtrToString(customer.Name))
	return "mock-customer-" + customer.ID.String(), nil
}

func (m *MockAccountingAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice, customerExternalID string) (string, error) {
	log.Printf("[Accounting Mock] Syncing Invoice: %s (Total: %d, CustomerRef: %s)", invoice.ID, invoice.Total, customerExternalID)
	return "mock-invoice-" + invoice.ID.String(), nil
}

func (m *MockAccountingAdapter) SyncProduct(ctx context.Context, plan *domain.Plan) (string, error) {
	log.Printf("[Accounting Mock] Syncing Product: %s (%s)", plan.ID, plan.Name)
	return "mock-product-" + plan.ID.String(), nil
}
