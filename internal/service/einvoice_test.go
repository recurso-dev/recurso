package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/gsp"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// --- Mock repos for e-invoice tests ---

type mockEInvInvoiceRepo struct {
	port.InvoiceRepository
	invoices map[uuid.UUID]*domain.Invoice
}

func newMockEInvInvoiceRepo() *mockEInvInvoiceRepo {
	return &mockEInvInvoiceRepo{invoices: make(map[uuid.UUID]*domain.Invoice)}
}

func (m *mockEInvInvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error {
	m.invoices[inv.ID] = inv
	return nil
}

func (m *mockEInvInvoiceRepo) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	inv, ok := m.invoices[id]
	if !ok {
		return nil, nil
	}
	return inv, nil
}

// GetByID mirrors the real repo's tenant scoping: the tenant must be present in
// the context and match the invoice, otherwise the row is invisible. This is
// what makes the e-invoice service's cross-tenant guard (ENG-165 C3) testable.
func (m *mockEInvInvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	tenantID, ok := ctx.Value(domain.TenantIDKey).(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id missing from context")
	}
	inv, ok := m.invoices[id]
	if !ok || inv.TenantID != tenantID {
		return nil, sql.ErrNoRows
	}
	return inv, nil
}

func (m *mockEInvInvoiceRepo) Update(ctx context.Context, inv *domain.Invoice) error {
	m.invoices[inv.ID] = inv
	return nil
}

func (m *mockEInvInvoiceRepo) ClaimFailedEInvoices(ctx context.Context, _, _ time.Time, _ int) ([]*domain.Invoice, error) {
	return m.GetFailedEInvoices(ctx)
}

func (m *mockEInvInvoiceRepo) GetFailedEInvoices(ctx context.Context) ([]*domain.Invoice, error) {
	var result []*domain.Invoice
	for _, inv := range m.invoices {
		if inv.EInvoiceStatus == "FAILED" && inv.EInvoiceNextRetryAt != nil && inv.EInvoiceNextRetryAt.Before(time.Now()) {
			result = append(result, inv)
		}
	}
	return result, nil
}

func (m *mockEInvInvoiceRepo) UpdateEInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status, irn, ackNo, signedQR, ackDate, errorMsg string) error {
	if inv, ok := m.invoices[invoiceID]; ok {
		inv.EInvoiceStatus = status
		inv.IRN = irn
		inv.AckNo = ackNo
		inv.SignedQRCode = signedQR
		inv.AckDate = ackDate
		inv.EInvoiceErrorMessage = errorMsg
	}
	return nil
}

type mockEInvCustomerRepo struct {
	port.CustomerRepository
	customer *domain.Customer
}

func (m *mockEInvCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return m.customer, nil
}

// --- Tests ---

func TestGenerateEInvoice_Eligible(t *testing.T) {
	invRepo := newMockEInvInvoiceRepo()
	custRepo := &mockEInvCustomerRepo{
		customer: &domain.Customer{
			ID: uuid.New(),
			BillingAddress: domain.BillingAddress{
				Country: "India",
				State:   "TN",
				Line1:   "123 Test Street",
				City:    "Chennai",
				Zip:     "600001",
			},
			GSTIN:         domain.StringPtr("33ABCDE1234F1Z5"),
			TaxType:       "business",
			Name:          domain.StringPtr("Test Corp"),
			PlaceOfSupply: domain.StringPtr("33"),
		},
	}
	mockGSP := gsp.NewMockGSPAdapter()

	svc := NewEInvoiceService(mockGSP, invRepo, custRepo, nil, nil)

	inv := &domain.Invoice{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CustomerID: custRepo.customer.ID,
		Subtotal:   100000,
		TaxAmount:  18000,
		Total:      118000,
		IGSTAmount: 18000,
		CreatedAt:  time.Now(),
	}

	resp, err := svc.GenerateEInvoice(context.Background(), inv)
	if err != nil {
		t.Fatalf("GenerateEInvoice failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response for eligible B2B invoice")
	}

	if inv.EInvoiceStatus != "GENERATED" {
		t.Errorf("Expected status GENERATED, got %s", inv.EInvoiceStatus)
	}
	if inv.IRN == "" {
		t.Error("Expected IRN to be set")
	}
	if inv.SignedQRCode == "" {
		t.Error("Expected SignedQRCode to be set")
	}
	if inv.AckNo == "" {
		t.Error("Expected AckNo to be set")
	}
}

func TestGenerateEInvoice_ConsumerNotEligible(t *testing.T) {
	invRepo := newMockEInvInvoiceRepo()
	custRepo := &mockEInvCustomerRepo{
		customer: &domain.Customer{
			ID: uuid.New(),
			BillingAddress: domain.BillingAddress{
				Country: "India",
				State:   "TN",
			},
			GSTIN:   nil, // Consumer — no GSTIN
			TaxType: "consumer",
		},
	}
	mockGSP := gsp.NewMockGSPAdapter()

	svc := NewEInvoiceService(mockGSP, invRepo, custRepo, nil, nil)

	inv := &domain.Invoice{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CustomerID: custRepo.customer.ID,
		Subtotal:   100000,
		Total:      100000,
		CreatedAt:  time.Now(),
	}

	resp, err := svc.GenerateEInvoice(context.Background(), inv)
	if err != nil {
		t.Fatalf("GenerateEInvoice should not fail for consumer: %v", err)
	}

	if resp != nil {
		t.Error("Expected nil response for non-eligible consumer")
	}

	if inv.EInvoiceStatus != "NA" {
		t.Errorf("Expected status NA, got %s", inv.EInvoiceStatus)
	}
}

func TestGenerateEInvoice_ForeignCustomerNotEligible(t *testing.T) {
	invRepo := newMockEInvInvoiceRepo()
	custRepo := &mockEInvCustomerRepo{
		customer: &domain.Customer{
			ID: uuid.New(),
			BillingAddress: domain.BillingAddress{
				Country: "United States",
			},
			TaxType: "business",
		},
	}
	mockGSP := gsp.NewMockGSPAdapter()

	svc := NewEInvoiceService(mockGSP, invRepo, custRepo, nil, nil)

	inv := &domain.Invoice{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		CustomerID: custRepo.customer.ID,
		Subtotal:   100000,
		Total:      100000,
		CreatedAt:  time.Now(),
	}

	resp, err := svc.GenerateEInvoice(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("Expected nil response for non-India customer")
	}
	if inv.EInvoiceStatus != "NA" {
		t.Errorf("Expected status NA, got %s", inv.EInvoiceStatus)
	}
}

func TestCancelEInvoice_WithinWindow(t *testing.T) {
	invRepo := newMockEInvInvoiceRepo()
	custRepo := &mockEInvCustomerRepo{customer: &domain.Customer{ID: uuid.New()}}
	mockGSP := gsp.NewMockGSPAdapter()

	svc := NewEInvoiceService(mockGSP, invRepo, custRepo, nil, nil)

	invID := uuid.New()
	tenantID := uuid.New()
	inv := &domain.Invoice{
		ID:             invID,
		TenantID:       tenantID,
		CustomerID:     custRepo.customer.ID,
		EInvoiceStatus: "GENERATED",
		IRN:            "test-irn-12345",
		AckDate:        time.Now().Format("02/01/2006 15:04:05"),
		CreatedAt:      time.Now(), // Within 24hr window
	}
	invRepo.invoices[invID] = inv

	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenantID)
	err := svc.CancelEInvoice(ctx, invID, 1, "Duplicate invoice")
	if err != nil {
		t.Fatalf("CancelEInvoice failed: %v", err)
	}

	if inv.EInvoiceStatus != "CANCELLED" {
		t.Errorf("Expected status CANCELLED, got %s", inv.EInvoiceStatus)
	}
}

func TestCancelEInvoice_AfterWindow(t *testing.T) {
	invRepo := newMockEInvInvoiceRepo()
	custRepo := &mockEInvCustomerRepo{customer: &domain.Customer{ID: uuid.New()}}
	mockGSP := gsp.NewMockGSPAdapter()

	svc := NewEInvoiceService(mockGSP, invRepo, custRepo, nil, nil)

	invID := uuid.New()
	tenantID := uuid.New()
	inv := &domain.Invoice{
		ID:             invID,
		TenantID:       tenantID,
		CustomerID:     custRepo.customer.ID,
		EInvoiceStatus: "GENERATED",
		IRN:            "test-irn-12345",
		AckDate:        time.Now().Add(-25 * time.Hour).Format("02/01/2006 15:04:05"),
		CreatedAt:      time.Now().Add(-25 * time.Hour), // Past 24hr window
	}
	invRepo.invoices[invID] = inv

	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenantID)
	err := svc.CancelEInvoice(ctx, invID, 1, "Duplicate invoice")
	if err == nil {
		t.Fatal("Expected error when cancelling after 24hr window")
	}
}

// TestEInvoice_TenantIsolation proves the ENG-165 C3 fix: the e-invoice
// status/cancel/retry paths are tenant-scoped. A caller in a different tenant
// cannot read the status of, or cancel, another tenant's e-invoice.
func TestEInvoice_TenantIsolation(t *testing.T) {
	invRepo := newMockEInvInvoiceRepo()
	custRepo := &mockEInvCustomerRepo{customer: &domain.Customer{ID: uuid.New()}}
	svc := NewEInvoiceService(gsp.NewMockGSPAdapter(), invRepo, custRepo, nil, nil)

	invID := uuid.New()
	owner := uuid.New()
	attacker := uuid.New()
	invRepo.invoices[invID] = &domain.Invoice{
		ID:             invID,
		TenantID:       owner,
		CustomerID:     custRepo.customer.ID,
		EInvoiceStatus: "GENERATED",
		IRN:            "test-irn-99999",
		AckDate:        time.Now().Format("02/01/2006 15:04:05"),
		CreatedAt:      time.Now(),
	}

	attackerCtx := context.WithValue(context.Background(), domain.TenantIDKey, attacker)

	// Attacker cannot read the status.
	if _, err := svc.GetEInvoiceStatus(attackerCtx, invID); err == nil {
		t.Error("cross-tenant GetEInvoiceStatus: expected error, got nil")
	}
	// Attacker cannot cancel the IRN.
	if err := svc.CancelEInvoice(attackerCtx, invID, 1, "malicious cancel"); err == nil {
		t.Error("cross-tenant CancelEInvoice: expected error, got nil")
	}
	// The invoice is untouched by the attacker.
	if got := invRepo.invoices[invID].EInvoiceStatus; got != "GENERATED" {
		t.Errorf("attacker mutated e-invoice status: got %s, want GENERATED", got)
	}

	// The owner can still read its own status.
	ownerCtx := context.WithValue(context.Background(), domain.TenantIDKey, owner)
	if _, err := svc.GetEInvoiceStatus(ownerCtx, invID); err != nil {
		t.Errorf("owner GetEInvoiceStatus: %v", err)
	}
}

func TestRetryFailedEInvoice(t *testing.T) {
	invRepo := newMockEInvInvoiceRepo()
	custRepo := &mockEInvCustomerRepo{
		customer: &domain.Customer{
			ID: uuid.New(),
			BillingAddress: domain.BillingAddress{
				Country: "India",
				State:   "TN",
				Line1:   "123 Test Street",
				City:    "Chennai",
			},
			GSTIN:         domain.StringPtr("33ABCDE1234F1Z5"),
			TaxType:       "business",
			Name:          domain.StringPtr("Retry Corp"),
			PlaceOfSupply: domain.StringPtr("33"),
		},
	}
	mockGSP := gsp.NewMockGSPAdapter()

	svc := NewEInvoiceService(mockGSP, invRepo, custRepo, nil, nil)

	invID := uuid.New()
	tenantID := uuid.New()
	retryAt := time.Now().Add(-1 * time.Minute) // Due for retry
	inv := &domain.Invoice{
		ID:                   invID,
		TenantID:             tenantID,
		CustomerID:           custRepo.customer.ID,
		InvoiceNumber:        "INV-RETRY-001",
		EInvoiceStatus:       "FAILED",
		EInvoiceRetryCount:   1,
		EInvoiceNextRetryAt:  &retryAt,
		EInvoiceErrorMessage: "previous failure",
		Subtotal:             50000,
		TaxAmount:            9000,
		Total:                59000,
		IGSTAmount:           9000,
		CreatedAt:            time.Now(),
	}
	invRepo.invoices[invID] = inv

	ctx := context.WithValue(context.Background(), domain.TenantIDKey, tenantID)
	resp, err := svc.RetryFailedEInvoice(ctx, invID)
	if err != nil {
		t.Fatalf("RetryFailedEInvoice failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response on successful retry")
	}

	if inv.EInvoiceStatus != "GENERATED" {
		t.Errorf("Expected status GENERATED after retry, got %s", inv.EInvoiceStatus)
	}
	if inv.IRN == "" {
		t.Error("Expected IRN to be set after retry")
	}
	if inv.EInvoiceRetryCount != 2 {
		t.Errorf("Expected retry count 2, got %d", inv.EInvoiceRetryCount)
	}
}

// TestGenerateEInvoice_IdempotentWhenIRNExists proves the ENG-146 fix: an
// invoice that already carries an IRN is not re-submitted to NIC (which would
// return "Duplicate IRN" and get recorded as a fresh FAILURE). The mock GSP
// mints a new IRN if called, so an unchanged IRN proves the short-circuit.
func TestGenerateEInvoice_IdempotentWhenIRNExists(t *testing.T) {
	invRepo := newMockEInvInvoiceRepo()
	custRepo := &mockEInvCustomerRepo{customer: &domain.Customer{ID: uuid.New()}}
	mockGSP := gsp.NewMockGSPAdapter()
	svc := NewEInvoiceService(mockGSP, invRepo, custRepo, nil, nil)

	inv := &domain.Invoice{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		IRN:            "EXISTING-IRN-123",
		AckNo:          "ACK-1",
		SignedQRCode:   "QR-1",
		EInvoiceStatus: "FAILED", // stuck: IRN issued at NIC but a prior persist failed
	}
	resp, err := svc.GenerateEInvoice(context.Background(), inv)
	if err != nil {
		t.Fatalf("GenerateEInvoice: %v", err)
	}
	if inv.IRN != "EXISTING-IRN-123" {
		t.Errorf("IRN = %q — the invoice was re-submitted instead of short-circuiting", inv.IRN)
	}
	if inv.EInvoiceStatus != "GENERATED" {
		t.Errorf("status = %q, want GENERATED (recovered from the stuck FAILED state)", inv.EInvoiceStatus)
	}
	if resp == nil || resp.IRN != "EXISTING-IRN-123" {
		t.Errorf("resp = %+v, want the existing IRN", resp)
	}
}
