package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recur-so/recurso/internal/core/domain"
	"github.com/recur-so/recurso/internal/core/port"
	"github.com/recur-so/recurso/internal/service"
)

// --- Mock InvoiceRepository ---

type mockInvoiceRepo struct {
	invoices []*domain.Invoice
	updated  []*domain.Invoice
}

func (m *mockInvoiceRepo) GetDueForRetry(ctx context.Context) ([]*domain.Invoice, error) {
	return m.invoices, nil
}

func (m *mockInvoiceRepo) Update(ctx context.Context, inv *domain.Invoice) error {
	m.updated = append(m.updated, inv)
	return nil
}

// Stubs for interface compliance
func (m *mockInvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error { return nil }
func (m *mockInvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) UpdateRetryInfo(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int) error {
	return nil
}
func (m *mockInvoiceRepo) UpdateRetryInfoWithDunning(ctx context.Context, invoiceID uuid.UUID, nextRetry time.Time, retryCount int, managedBy string) error {
	return nil
}
func (m *mockInvoiceRepo) MarkAsUncollectible(ctx context.Context, invoiceID uuid.UUID) error {
	return nil
}
func (m *mockInvoiceRepo) GetOverdueInvoices(ctx context.Context) ([]domain.OverdueInvoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) GetFailedEInvoices(ctx context.Context) ([]*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) UpdateEInvoiceStatus(ctx context.Context, invoiceID uuid.UUID, status, irn, ackNo, signedQR, ackDate, errorMsg string) error {
	return nil
}

// --- Mock PaymentGateway ---

type mockGateway struct {
	result *port.PaymentResult
	err    error
}

func (m *mockGateway) RetryPayment(ctx context.Context, invoiceID string, amount int64, currency string) (*port.PaymentResult, error) {
	return m.result, m.err
}

func (m *mockGateway) CreateOrder(ctx context.Context, amount int64, currency string, receipt string) (*port.PaymentOrder, error) {
	return nil, nil
}
func (m *mockGateway) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) error {
	return nil
}
func (m *mockGateway) CreateSubscription(ctx context.Context, planID string, totalCount int, customerEmail string, startAt *int64, currency string) (string, error) {
	return "", nil
}

// --- Mock Notifier ---

type mockNotifier struct{}

func (m *mockNotifier) SendEmail(ctx context.Context, to string, subject string, body string) error {
	return nil
}

// --- Mock DunningRepository ---

type mockDunningRepo struct {
	weights map[string]domain.DunningWeight
	history []domain.DunningHistory
}

func (m *mockDunningRepo) GetWeights(ctx context.Context, contextKey string) ([]domain.DunningWeight, error) {
	var results []domain.DunningWeight
	for _, v := range m.weights {
		if v.ContextKey == contextKey {
			results = append(results, v)
		}
	}
	return results, nil
}

func (m *mockDunningRepo) UpdateWeight(ctx context.Context, weight domain.DunningWeight) error {
	m.weights[weight.ContextKey+":"+weight.ActionID] = weight
	return nil
}

func (m *mockDunningRepo) RecordHistory(ctx context.Context, history domain.DunningHistory) error {
	m.history = append(m.history, history)
	return nil
}

// --- Tests ---

func TestRetryWorker_PaymentSuccess(t *testing.T) {
	now := time.Now()
	inv := &domain.Invoice{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		InvoiceNumber:     "INV-001",
		Total:             10000,
		Currency:          "USD",
		Status:            domain.InvoiceStatusPastDue,
		RetryCount:        2,
		DunningActionID:   "24h",
		DunningContextKey: "USD:insufficient_funds",
		DunningManagedBy:  "worker",
		NextRetryAt:       &now,
	}

	repo := &mockInvoiceRepo{invoices: []*domain.Invoice{inv}}
	gw := &mockGateway{result: &port.PaymentResult{
		Success:   true,
		PaymentID: "pay_123",
	}}
	dunningRepo := &mockDunningRepo{weights: make(map[string]domain.DunningWeight)}
	retrySvc := service.NewSmartRetryService(dunningRepo)

	w := NewRetryWorker(repo, retrySvc, gw, &mockNotifier{})
	w.processRetries(context.Background())

	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 update, got %d", len(repo.updated))
	}

	updated := repo.updated[0]
	if updated.Status != domain.InvoiceStatusPaid {
		t.Errorf("expected status=paid, got %s", updated.Status)
	}
	if updated.PaidAt == nil {
		t.Error("expected PaidAt to be set")
	}
	if updated.NextRetryAt != nil {
		t.Error("expected NextRetryAt to be nil")
	}

	// Check outcome was recorded
	if len(dunningRepo.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(dunningRepo.history))
	}
	if dunningRepo.history[0].Outcome != "success" {
		t.Errorf("expected outcome=success, got %s", dunningRepo.history[0].Outcome)
	}
	if dunningRepo.history[0].Reward != 1.0 {
		t.Errorf("expected reward=1.0, got %f", dunningRepo.history[0].Reward)
	}
}

func TestRetryWorker_PaymentFailure(t *testing.T) {
	now := time.Now()
	inv := &domain.Invoice{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		InvoiceNumber:     "INV-002",
		Total:             5000,
		Currency:          "USD",
		Status:            domain.InvoiceStatusPastDue,
		RetryCount:        1,
		DunningActionID:   "1h",
		DunningContextKey: "USD:card_declined",
		DunningManagedBy:  "worker",
		NextRetryAt:       &now,
	}

	repo := &mockInvoiceRepo{invoices: []*domain.Invoice{inv}}
	gw := &mockGateway{result: &port.PaymentResult{
		Success:   false,
		ErrorCode: "insufficient_funds",
		ErrorMsg:  "not enough balance",
	}}
	dunningRepo := &mockDunningRepo{weights: make(map[string]domain.DunningWeight)}
	retrySvc := service.NewSmartRetryService(dunningRepo)

	w := NewRetryWorker(repo, retrySvc, gw, &mockNotifier{})
	w.processRetries(context.Background())

	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 update, got %d", len(repo.updated))
	}

	updated := repo.updated[0]
	if updated.Status != domain.InvoiceStatusPastDue {
		t.Errorf("expected status=past_due, got %s", updated.Status)
	}
	if updated.NextRetryAt == nil {
		t.Error("expected NextRetryAt to be set for next retry")
	}
	if updated.DunningActionID == "" {
		t.Error("expected DunningActionID to be set")
	}
	if updated.DunningContextKey == "" {
		t.Error("expected DunningContextKey to be set")
	}
	if updated.LastPaymentError != "insufficient_funds" {
		t.Errorf("expected LastPaymentError=insufficient_funds, got %s", updated.LastPaymentError)
	}
	if updated.RetryCount != 2 {
		t.Errorf("expected RetryCount=2, got %d", updated.RetryCount)
	}

	// Check failure outcome was recorded for previous action
	if len(dunningRepo.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(dunningRepo.history))
	}
	if dunningRepo.history[0].Outcome != "failure" {
		t.Errorf("expected outcome=failure, got %s", dunningRepo.history[0].Outcome)
	}
	if dunningRepo.history[0].Reward != 0.0 {
		t.Errorf("expected reward=0.0, got %f", dunningRepo.history[0].Reward)
	}
}

func TestRetryWorker_MaxRetries(t *testing.T) {
	now := time.Now()
	inv := &domain.Invoice{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		InvoiceNumber:     "INV-003",
		Total:             2000,
		Currency:          "USD",
		Status:            domain.InvoiceStatusPastDue,
		RetryCount:        9, // Will become 10 after increment → max reached
		DunningActionID:   "3d",
		DunningContextKey: "USD:card_expired",
		DunningManagedBy:  "worker",
		NextRetryAt:       &now,
	}

	repo := &mockInvoiceRepo{invoices: []*domain.Invoice{inv}}
	gw := &mockGateway{result: &port.PaymentResult{
		Success:   false,
		ErrorCode: "card_expired",
	}}
	dunningRepo := &mockDunningRepo{weights: make(map[string]domain.DunningWeight)}
	retrySvc := service.NewSmartRetryService(dunningRepo)

	w := NewRetryWorker(repo, retrySvc, gw, &mockNotifier{})
	w.processRetries(context.Background())

	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 update, got %d", len(repo.updated))
	}

	updated := repo.updated[0]
	if updated.Status != domain.InvoiceStatusUncollectible {
		t.Errorf("expected status=uncollectible, got %s", updated.Status)
	}
	if updated.NextRetryAt != nil {
		t.Error("expected NextRetryAt to be nil for uncollectible invoice")
	}
}

func TestRetryWorker_GatewayError(t *testing.T) {
	now := time.Now()
	inv := &domain.Invoice{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		InvoiceNumber:     "INV-004",
		Total:             3000,
		Currency:          "USD",
		Status:            domain.InvoiceStatusPastDue,
		RetryCount:        1,
		DunningActionID:   "24h",
		DunningContextKey: "USD:network_error",
		DunningManagedBy:  "worker",
		NextRetryAt:       &now,
	}

	repo := &mockInvoiceRepo{invoices: []*domain.Invoice{inv}}
	gw := &mockGateway{
		result: nil,
		err:    context.DeadlineExceeded, // Infra error
	}
	dunningRepo := &mockDunningRepo{weights: make(map[string]domain.DunningWeight)}
	retrySvc := service.NewSmartRetryService(dunningRepo)

	w := NewRetryWorker(repo, retrySvc, gw, &mockNotifier{})
	w.processRetries(context.Background())

	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 update, got %d", len(repo.updated))
	}

	updated := repo.updated[0]
	// Should schedule a 5-minute retry, NOT record outcome
	if updated.NextRetryAt == nil {
		t.Error("expected NextRetryAt to be set for 5min retry")
	}
	if updated.NextRetryAt != nil {
		diff := updated.NextRetryAt.Sub(time.Now())
		if diff < 4*time.Minute || diff > 6*time.Minute {
			t.Errorf("expected ~5min retry, got %v", diff)
		}
	}

	// No outcome should be recorded for infra errors
	if len(dunningRepo.history) != 0 {
		t.Errorf("expected 0 history entries for infra error, got %d", len(dunningRepo.history))
	}
}
