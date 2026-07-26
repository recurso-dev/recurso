package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type achCustStore struct {
	cust     *domain.Customer
	stripeID string
}

func (s *achCustStore) GetByIDPublic(_ context.Context, _ uuid.UUID) (*domain.Customer, error) {
	return s.cust, nil
}
func (s *achCustStore) GetStripeCustomerID(_ context.Context, _ uuid.UUID) (string, error) {
	return s.stripeID, nil
}
func (s *achCustStore) SetStripeCustomerID(_ context.Context, _ uuid.UUID, sc string) error {
	s.stripeID = sc
	return nil
}
func (s *achCustStore) SetDefaultPaymentMethod(_ context.Context, _ uuid.UUID, _, _, _ string, _, _ int, _ *uuid.UUID) error {
	return nil
}

// achSetup implements paymentMethodSetup AND bankAccountSetup.
type achSetup struct{ bankSecret string }

func (s *achSetup) EnsureStripeCustomer(_ context.Context, existingID, _, _ string) (string, error) {
	if existingID != "" {
		return existingID, nil
	}
	return "cus_test", nil
}
func (s *achSetup) CreateSetupIntent(_ context.Context, _ string, _ map[string]string) (string, error) {
	return "seti_card", nil
}
func (s *achSetup) FinalizeSetupIntent(_ context.Context, _ string) (*port.SavedCard, error) {
	return &port.SavedCard{Status: "succeeded"}, nil
}
func (s *achSetup) CreateBankAccountSetupIntent(_ context.Context, _ string, _ map[string]string) (string, error) {
	return s.bankSecret, nil
}

// cardOnlySetup implements paymentMethodSetup but NOT bankAccountSetup.
type cardOnlySetup struct{}

func (cardOnlySetup) EnsureStripeCustomer(_ context.Context, _, _, _ string) (string, error) {
	return "cus_x", nil
}
func (cardOnlySetup) CreateSetupIntent(_ context.Context, _ string, _ map[string]string) (string, error) {
	return "seti", nil
}
func (cardOnlySetup) FinalizeSetupIntent(_ context.Context, _ string) (*port.SavedCard, error) {
	return nil, nil
}

func portalBankCtx(customerID uuid.UUID) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("portal_customer_id", customerID)
	c.Request = httptest.NewRequest(http.MethodPost, "/portal/api/payment-method/bank-setup-intent", nil)
	return c, w
}

func TestStartBankAccountSetup_ReturnsClientSecret(t *testing.T) {
	cid := uuid.New()
	name := "Acme"
	h := &PortalAPIHandler{}
	h.SetPaymentMethodSetup(
		&achCustStore{cust: &domain.Customer{ID: cid, Email: "a@b.co", Name: &name}},
		&achSetup{bankSecret: "seti_bank_secret"},
		"pk_test_x",
	)

	c, w := portalBankCtx(cid)
	h.StartBankAccountSetup(c)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "seti_bank_secret") || !strings.Contains(body, "pk_test_x") {
		t.Errorf("missing client_secret/publishable_key: %s", body)
	}
}

// A non-Stripe (card-only) setup reports ACH unavailable rather than erroring.
func TestStartBankAccountSetup_UnavailableWithoutBankSupport(t *testing.T) {
	cid := uuid.New()
	h := &PortalAPIHandler{}
	h.SetPaymentMethodSetup(
		&achCustStore{cust: &domain.Customer{ID: cid, Email: "a@b.co"}},
		cardOnlySetup{},
		"pk",
	)

	c, w := portalBankCtx(cid)
	h.StartBankAccountSetup(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when bank setup unsupported, got %d (%s)", w.Code, w.Body.String())
	}
}
