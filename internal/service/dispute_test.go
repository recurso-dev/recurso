package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// --- Mocks ---

// disputeMockRepo is an in-memory port.DisputeRepository that reproduces the
// one-open-per-invoice and tenant-scoped-resolve semantics of the SQL repo.
type disputeMockRepo struct {
	items []*domain.InvoiceDispute
}

func (m *disputeMockRepo) Create(ctx context.Context, d *domain.InvoiceDispute) error {
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now()
	}
	cp := *d
	m.items = append(m.items, &cp)
	return nil
}

func (m *disputeMockRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.InvoiceDispute, error) {
	for _, d := range m.items {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, domain.ErrDisputeNotFound
}

func (m *disputeMockRepo) GetOpenByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*domain.InvoiceDispute, error) {
	for _, d := range m.items {
		if d.InvoiceID == invoiceID && d.Status == domain.DisputeStatusOpen {
			return d, nil
		}
	}
	return nil, nil
}

func (m *disputeMockRepo) UpdateReason(ctx context.Context, id uuid.UUID, reason string) error {
	for _, d := range m.items {
		if d.ID == id && d.Status == domain.DisputeStatusOpen {
			d.Reason = reason
		}
	}
	return nil
}

func (m *disputeMockRepo) ListByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.InvoiceDispute, error) {
	out := []*domain.InvoiceDispute{}
	for _, d := range m.items {
		if d.CustomerID == customerID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (m *disputeMockRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]*domain.InvoiceDispute, error) {
	out := []*domain.InvoiceDispute{}
	for _, d := range m.items {
		if d.TenantID != tenantID {
			continue
		}
		if status != "" && string(d.Status) != status {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (m *disputeMockRepo) Resolve(ctx context.Context, tenantID, id uuid.UUID, note string) error {
	for _, d := range m.items {
		if d.ID == id && d.TenantID == tenantID && d.Status == domain.DisputeStatusOpen {
			d.Status = domain.DisputeStatusResolved
			if note != "" {
				n := note
				d.Note = &n
			}
			now := time.Now()
			d.ResolvedAt = &now
			return nil
		}
	}
	return domain.ErrDisputeNotFound
}

// disputeMockInvoiceRepo overrides only GetByIDPublic.
type disputeMockInvoiceRepo struct {
	port.InvoiceRepository
	invoices map[uuid.UUID]*domain.Invoice
}

func (m *disputeMockInvoiceRepo) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if inv, ok := m.invoices[id]; ok {
		return inv, nil
	}
	return nil, domain.ErrDisputeNotFound // any non-nil error → treated as not found
}

// recordingCustomerRepo records the customer id passed to UpdatePaymentMethod.
type recordingCustomerRepo struct {
	port.CustomerRepository
	gotCustomerID uuid.UUID
	gotBrand      string
	gotLast4      string
	calls         int
}

func (m *recordingCustomerRepo) UpdatePaymentMethod(ctx context.Context, customerID uuid.UUID, brand, last4 string, expMonth, expYear int) error {
	m.calls++
	m.gotCustomerID = customerID
	m.gotBrand = brand
	m.gotLast4 = last4
	return nil
}

// --- Portal payment-method ---

func TestPortalUpdatePaymentMethod_DelegatesWithGivenCustomer(t *testing.T) {
	custRepo := &recordingCustomerRepo{}
	svc := NewPortalService(custRepo, nil, nil, nil, nil, nil, nil, "")

	sessionCustomer := uuid.New()
	if err := svc.UpdatePaymentMethod(context.Background(), sessionCustomer, "visa", "4242", 12, 2030); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if custRepo.calls != 1 {
		t.Fatalf("UpdatePaymentMethod calls = %d, want 1", custRepo.calls)
	}
	if custRepo.gotCustomerID != sessionCustomer {
		t.Errorf("customer id = %v, want session customer %v", custRepo.gotCustomerID, sessionCustomer)
	}
	if custRepo.gotBrand != "visa" || custRepo.gotLast4 != "4242" {
		t.Errorf("card metadata not forwarded: brand=%q last4=%q", custRepo.gotBrand, custRepo.gotLast4)
	}
}

// --- Portal dispute ---

func newDisputePortalService(inv *disputeMockInvoiceRepo, disp *disputeMockRepo) *PortalService {
	return NewPortalService(nil, inv, nil, nil, disp, nil, nil, "")
}

func TestPortalRaiseDispute_GuardsInvoiceOwnership(t *testing.T) {
	owner := uuid.New()
	attacker := uuid.New()
	invID := uuid.New()

	inv := &disputeMockInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{
		invID: {ID: invID, TenantID: uuid.New(), CustomerID: owner},
	}}
	disp := &disputeMockRepo{}
	svc := newDisputePortalService(inv, disp)

	// Attacker (a different session customer) tries to dispute owner's invoice.
	_, err := svc.RaiseDispute(context.Background(), attacker, invID, "not mine")
	if err != ErrInvoiceNotFound {
		t.Fatalf("err = %v, want ErrInvoiceNotFound", err)
	}
	if len(disp.items) != 0 {
		t.Fatalf("dispute created for non-owned invoice: %d", len(disp.items))
	}
}

func TestPortalRaiseDispute_CreatesOpenDisputeFromInvoiceTenant(t *testing.T) {
	owner := uuid.New()
	tenant := uuid.New()
	invID := uuid.New()

	inv := &disputeMockInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{
		invID: {ID: invID, TenantID: tenant, CustomerID: owner},
	}}
	disp := &disputeMockRepo{}
	svc := newDisputePortalService(inv, disp)

	d, err := svc.RaiseDispute(context.Background(), owner, invID, "overcharged")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Status != domain.DisputeStatusOpen {
		t.Errorf("status = %q, want open", d.Status)
	}
	if d.TenantID != tenant {
		t.Errorf("tenant = %v, want %v (from invoice, not caller)", d.TenantID, tenant)
	}
	if d.CustomerID != owner || d.Reason != "overcharged" {
		t.Errorf("unexpected dispute fields: %+v", d)
	}
	if len(disp.items) != 1 {
		t.Fatalf("dispute count = %d, want 1", len(disp.items))
	}
}

func TestPortalRaiseDispute_OneOpenPerInvoice(t *testing.T) {
	owner := uuid.New()
	invID := uuid.New()

	inv := &disputeMockInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{
		invID: {ID: invID, TenantID: uuid.New(), CustomerID: owner},
	}}
	disp := &disputeMockRepo{}
	svc := newDisputePortalService(inv, disp)

	first, err := svc.RaiseDispute(context.Background(), owner, invID, "reason one")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := svc.RaiseDispute(context.Background(), owner, invID, "reason two")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(disp.items) != 1 {
		t.Fatalf("dispute count = %d, want 1 (re-raise must not create a new one)", len(disp.items))
	}
	if first.ID != second.ID {
		t.Errorf("re-raise returned different dispute id")
	}
	if disp.items[0].Reason != "reason two" {
		t.Errorf("reason = %q, want updated to 'reason two'", disp.items[0].Reason)
	}
}

// --- Admin dispute service ---

func TestDisputeService_ListFiltersByTenantAndStatus(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	disp := &disputeMockRepo{items: []*domain.InvoiceDispute{
		{ID: uuid.New(), TenantID: tenantA, Status: domain.DisputeStatusOpen},
		{ID: uuid.New(), TenantID: tenantA, Status: domain.DisputeStatusResolved},
		{ID: uuid.New(), TenantID: tenantB, Status: domain.DisputeStatusOpen},
	}}
	svc := NewDisputeService(disp)

	all, err := svc.List(context.Background(), tenantA, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("tenant A all = %d, want 2 (tenant isolation)", len(all))
	}

	open, err := svc.List(context.Background(), tenantA, "open")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(open) != 1 || open[0].Status != domain.DisputeStatusOpen {
		t.Errorf("tenant A open = %d, want 1 open", len(open))
	}
}

func TestDisputeService_Resolve_SetsResolved(t *testing.T) {
	tenant := uuid.New()
	id := uuid.New()
	disp := &disputeMockRepo{items: []*domain.InvoiceDispute{
		{ID: id, TenantID: tenant, Status: domain.DisputeStatusOpen},
	}}
	svc := NewDisputeService(disp)

	if err := svc.Resolve(context.Background(), tenant, id, "credited"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := disp.items[0]
	if got.Status != domain.DisputeStatusResolved {
		t.Errorf("status = %q, want resolved", got.Status)
	}
	if got.Note == nil || *got.Note != "credited" {
		t.Errorf("note not set to resolution note")
	}
	if got.ResolvedAt == nil {
		t.Errorf("resolved_at not set")
	}
}

func TestDisputeService_Resolve_TenantIsolation(t *testing.T) {
	tenant := uuid.New()
	otherTenant := uuid.New()
	id := uuid.New()
	disp := &disputeMockRepo{items: []*domain.InvoiceDispute{
		{ID: id, TenantID: tenant, Status: domain.DisputeStatusOpen},
	}}
	svc := NewDisputeService(disp)

	// Resolving another tenant's dispute must fail and leave it untouched.
	if err := svc.Resolve(context.Background(), otherTenant, id, "x"); err != domain.ErrDisputeNotFound {
		t.Fatalf("err = %v, want ErrDisputeNotFound", err)
	}
	if disp.items[0].Status != domain.DisputeStatusOpen {
		t.Errorf("dispute was modified across tenants")
	}
}
