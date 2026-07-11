package accounting

import (
	"context"
	"log"
	"sync"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// MockCall records one gateway invocation so tests can assert create vs
// update behavior distinctly.
type MockCall struct {
	Entity     string // "customer" | "invoice" | "product"
	EntityID   string // internal entity ID
	ExternalID string // externalID passed by the caller ("" on create)
	Action     string // "create" when ExternalID is empty, "update" otherwise
}

// MockAccountingAdapter is a no-op AccountingGateway that logs and records
// every call. On create it fabricates a deterministic external ID; on update
// it echoes the externalID it was given, like a real provider would.
type MockAccountingAdapter struct {
	mu    sync.Mutex
	calls []MockCall
}

func NewMockAccountingAdapter() *MockAccountingAdapter {
	return &MockAccountingAdapter{}
}

// Calls returns a snapshot of all recorded invocations.
func (m *MockAccountingAdapter) Calls() []MockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MockCall, len(m.calls))
	copy(out, m.calls)
	return out
}

func (m *MockAccountingAdapter) record(entity, entityID, externalID string) {
	action := "create"
	if externalID != "" {
		action = "update"
	}
	m.mu.Lock()
	m.calls = append(m.calls, MockCall{Entity: entity, EntityID: entityID, ExternalID: externalID, Action: action})
	m.mu.Unlock()
}

func (m *MockAccountingAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer, externalID string) (string, error) {
	m.record("customer", customer.ID.String(), externalID)
	if externalID != "" {
		log.Printf("[Accounting Mock] Updating Customer: %s (%s) as %s", customer.ID, domain.PtrToString(customer.Name), externalID)
		return externalID, nil
	}
	log.Printf("[Accounting Mock] Creating Customer: %s (%s)", customer.ID, domain.PtrToString(customer.Name))
	return "mock-customer-" + customer.ID.String(), nil
}

func (m *MockAccountingAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice, refs port.InvoiceSyncRefs, externalID string) (string, error) {
	m.record("invoice", invoice.ID.String(), externalID)
	if externalID != "" {
		log.Printf("[Accounting Mock] Updating Invoice: %s (Total: %d, CustomerRef: %s, ItemRef: %s) as %s",
			invoice.ID, invoice.Total, refs.CustomerExternalID, refs.ProductExternalID, externalID)
		return externalID, nil
	}
	log.Printf("[Accounting Mock] Creating Invoice: %s (Total: %d, CustomerRef: %s, ItemRef: %s)",
		invoice.ID, invoice.Total, refs.CustomerExternalID, refs.ProductExternalID)
	return "mock-invoice-" + invoice.ID.String(), nil
}

func (m *MockAccountingAdapter) SyncProduct(ctx context.Context, plan *domain.Plan, externalID string) (string, error) {
	m.record("product", plan.ID.String(), externalID)
	if externalID != "" {
		log.Printf("[Accounting Mock] Updating Product: %s (%s) as %s", plan.ID, plan.Name, externalID)
		return externalID, nil
	}
	log.Printf("[Accounting Mock] Creating Product: %s (%s)", plan.ID, plan.Name)
	return "mock-product-" + plan.ID.String(), nil
}
