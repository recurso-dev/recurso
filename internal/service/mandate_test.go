package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

// --- Mocks ---

type mandateMockRepo struct {
	port.MandateRepository
	mandate     *domain.Mandate
	updated     *domain.Mandate
	updateCalls int
	updateErr   error
}

func (m *mandateMockRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Mandate, error) {
	if m.mandate == nil || m.mandate.ID != id {
		return nil, fmt.Errorf("mandate %s not found", id)
	}
	return m.mandate, nil
}

func (m *mandateMockRepo) GetByRazorpayTokenID(ctx context.Context, tokenID string) (*domain.Mandate, error) {
	if m.mandate == nil || m.mandate.RazorpayTokenID != tokenID {
		return nil, fmt.Errorf("mandate for token %s not found", tokenID)
	}
	return m.mandate, nil
}

func (m *mandateMockRepo) Update(ctx context.Context, mandate *domain.Mandate) error {
	m.updateCalls++
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = mandate
	return nil
}

type mandateRevokeCall struct {
	customerID string
	tokenID    string
}

type mandateMockGateway struct {
	port.PaymentGateway
	revokeCalls []mandateRevokeCall
	revokeErr   error
}

func (m *mandateMockGateway) RevokeMandate(ctx context.Context, customerID, tokenID string) error {
	m.revokeCalls = append(m.revokeCalls, mandateRevokeCall{customerID: customerID, tokenID: tokenID})
	return m.revokeErr
}

func newTestMandate() *domain.Mandate {
	return &domain.Mandate{
		ID:                 uuid.New(),
		TenantID:           uuid.New(),
		CustomerID:         uuid.New(),
		RazorpayTokenID:    "token_abc123",
		RazorpayCustomerID: "cust_xyz789",
		Status:             domain.MandateStatusActive,
	}
}

// --- Revoke tests ---

func TestMandateRevoke_Success(t *testing.T) {
	mandate := newTestMandate()
	repo := &mandateMockRepo{mandate: mandate}
	gw := &mandateMockGateway{}
	svc := NewMandateService(repo, gw, nil, nil)

	if err := svc.Revoke(context.Background(), mandate.ID); err != nil {
		t.Fatalf("Revoke returned error: %v", err)
	}

	if len(gw.revokeCalls) != 1 {
		t.Fatalf("expected 1 gateway revoke call, got %d", len(gw.revokeCalls))
	}
	call := gw.revokeCalls[0]
	if call.customerID != "cust_xyz789" || call.tokenID != "token_abc123" {
		t.Errorf("gateway called with (%q, %q), want (%q, %q)",
			call.customerID, call.tokenID, "cust_xyz789", "token_abc123")
	}

	if repo.updateCalls != 1 {
		t.Fatalf("expected 1 repo update, got %d", repo.updateCalls)
	}
	if repo.updated.Status != domain.MandateStatusRevoked {
		t.Errorf("status = %q, want %q", repo.updated.Status, domain.MandateStatusRevoked)
	}
	if repo.updated.RevokedAt == nil {
		t.Error("RevokedAt not set on revoked mandate")
	}
}

func TestMandateRevoke_GatewayFailureLeavesStatusUntouched(t *testing.T) {
	mandate := newTestMandate()
	repo := &mandateMockRepo{mandate: mandate}
	gw := &mandateMockGateway{revokeErr: fmt.Errorf("razorpay unavailable")}
	svc := NewMandateService(repo, gw, nil, nil)

	err := svc.Revoke(context.Background(), mandate.ID)
	if err == nil {
		t.Fatal("expected error when gateway revoke fails, got nil")
	}

	if repo.updateCalls != 0 {
		t.Errorf("repo update called %d times after gateway failure, want 0", repo.updateCalls)
	}
	if mandate.Status != domain.MandateStatusActive {
		t.Errorf("status = %q after gateway failure, want %q", mandate.Status, domain.MandateStatusActive)
	}
	if mandate.RevokedAt != nil {
		t.Error("RevokedAt set even though gateway revoke failed")
	}
}

func TestMandateRevoke_NoTokenSkipsGateway(t *testing.T) {
	mandate := newTestMandate()
	mandate.RazorpayTokenID = ""
	repo := &mandateMockRepo{mandate: mandate}
	gw := &mandateMockGateway{}
	svc := NewMandateService(repo, gw, nil, nil)

	if err := svc.Revoke(context.Background(), mandate.ID); err != nil {
		t.Fatalf("Revoke returned error: %v", err)
	}

	if len(gw.revokeCalls) != 0 {
		t.Errorf("gateway called %d times for tokenless mandate, want 0", len(gw.revokeCalls))
	}
	if repo.updated == nil || repo.updated.Status != domain.MandateStatusRevoked {
		t.Error("tokenless mandate was not marked revoked")
	}
}

func TestMandateRevoke_NotFound(t *testing.T) {
	repo := &mandateMockRepo{}
	gw := &mandateMockGateway{}
	svc := NewMandateService(repo, gw, nil, nil)

	if err := svc.Revoke(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected error for unknown mandate id, got nil")
	}
	if len(gw.revokeCalls) != 0 {
		t.Errorf("gateway called %d times for unknown mandate, want 0", len(gw.revokeCalls))
	}
}

// --- HandleAuthorization tests ---

func TestMandateHandleAuthorization_PersistsCustomerID(t *testing.T) {
	mandate := newTestMandate()
	mandate.Status = domain.MandateStatusCreated
	mandate.RazorpayCustomerID = ""
	repo := &mandateMockRepo{mandate: mandate}
	svc := NewMandateService(repo, &mandateMockGateway{}, nil, nil)

	if err := svc.HandleAuthorization(context.Background(), mandate.RazorpayTokenID, "cust_from_webhook"); err != nil {
		t.Fatalf("HandleAuthorization returned error: %v", err)
	}

	if repo.updated == nil {
		t.Fatal("repo update not called")
	}
	if repo.updated.RazorpayCustomerID != "cust_from_webhook" {
		t.Errorf("RazorpayCustomerID = %q, want %q", repo.updated.RazorpayCustomerID, "cust_from_webhook")
	}
	if repo.updated.Status != domain.MandateStatusActive {
		t.Errorf("status = %q, want %q", repo.updated.Status, domain.MandateStatusActive)
	}
	if repo.updated.AuthorizedAt == nil || repo.updated.ActivatedAt == nil {
		t.Error("AuthorizedAt/ActivatedAt not set")
	}
}

func TestMandateHandleAuthorization_EmptyCustomerIDKeepsExisting(t *testing.T) {
	mandate := newTestMandate()
	mandate.Status = domain.MandateStatusCreated
	repo := &mandateMockRepo{mandate: mandate}
	svc := NewMandateService(repo, &mandateMockGateway{}, nil, nil)

	if err := svc.HandleAuthorization(context.Background(), mandate.RazorpayTokenID, ""); err != nil {
		t.Fatalf("HandleAuthorization returned error: %v", err)
	}

	if repo.updated.RazorpayCustomerID != "cust_xyz789" {
		t.Errorf("RazorpayCustomerID = %q, want existing %q kept", repo.updated.RazorpayCustomerID, "cust_xyz789")
	}
}
