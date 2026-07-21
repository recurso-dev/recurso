package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// --- Minimal mocks (embed the port interface, override what we exercise) ---

type portalTestCustomerRepo struct {
	port.CustomerRepository
	gotCustomerID uuid.UUID
	calls         int
}

func (m *portalTestCustomerRepo) UpdatePaymentMethod(ctx context.Context, customerID uuid.UUID, brand, last4 string, expMonth, expYear int) error {
	m.calls++
	m.gotCustomerID = customerID
	return nil
}

type portalTestInvoiceRepo struct {
	port.InvoiceRepository
	invoices map[uuid.UUID]*domain.Invoice
}

func (m *portalTestInvoiceRepo) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if inv, ok := m.invoices[id]; ok {
		return inv, nil
	}
	return nil, domain.ErrDisputeNotFound
}

type portalTestDisputeRepo struct {
	port.DisputeRepository
	created []*domain.InvoiceDispute
}

func (m *portalTestDisputeRepo) GetOpenByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*domain.InvoiceDispute, error) {
	return nil, nil
}
func (m *portalTestDisputeRepo) Create(ctx context.Context, d *domain.InvoiceDispute) error {
	m.created = append(m.created, d)
	return nil
}

func newTestContext(method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

// --- Mandate re-auth (ENG-5 Phase 3a) mocks ---

type mandateTestCustomerStore struct {
	cust *domain.Customer
}

func (m *mandateTestCustomerStore) GetByIDPublic(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return m.cust, nil
}
func (m *mandateTestCustomerStore) GetStripeCustomerID(ctx context.Context, id uuid.UUID) (string, error) {
	return "", nil
}
func (m *mandateTestCustomerStore) SetStripeCustomerID(ctx context.Context, id uuid.UUID, s string) error {
	return nil
}
func (m *mandateTestCustomerStore) SetDefaultPaymentMethod(ctx context.Context, id uuid.UUID, pm, brand, last4 string, em, ey int, gatewayConnectionID *uuid.UUID) error {
	return nil
}

type mandateTestInvoices struct{ list []*domain.Invoice }

func (m *mandateTestInvoices) GetByCustomerID(ctx context.Context, id uuid.UUID) ([]*domain.Invoice, error) {
	return m.list, nil
}

type mandateTestSvc struct {
	gotCtx   context.Context
	gotInput service.CreateMandateInput
}

func (m *mandateTestSvc) CreateMandate(ctx context.Context, in service.CreateMandateInput) (*service.CreateMandateOutput, error) {
	m.gotCtx = ctx
	m.gotInput = in
	return &service.CreateMandateOutput{
		Mandate: &domain.Mandate{ID: uuid.New(), Status: domain.MandateStatusCreated},
		AuthURL: "https://rzp.io/authorize/test",
	}, nil
}

func TestPortalMandateReauth_UnavailableWithoutRazorpay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPortalAPIHandler(nil) // no SetMandateReauth wiring

	c, w := newTestContext(http.MethodPost, "/portal/api/payment-method/mandate", "")
	c.Set("portal_customer_id", uuid.New())

	h.StartMandateReauth(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s, want 503", w.Code, w.Body.String())
	}
}

func TestPortalMandateReauth_InjectsTenantAndSizesCapFromInvoices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	customerID := uuid.New()

	store := &mandateTestCustomerStore{cust: &domain.Customer{ID: customerID, TenantID: tenantID}}
	invoices := &mandateTestInvoices{list: []*domain.Invoice{
		{Total: 50000, Status: domain.InvoiceStatusVoid}, // void: must be ignored
		{Total: 12000, Status: domain.InvoiceStatusPaid}, // largest countable
		{Total: 9000, Status: domain.InvoiceStatusPastDue},
	}}
	svc := &mandateTestSvc{}

	h := NewPortalAPIHandler(nil)
	h.SetMandateReauth(store, svc, invoices)

	c, w := newTestContext(http.MethodPost, "/portal/api/payment-method/mandate", "")
	c.Set("portal_customer_id", customerID)

	h.StartMandateReauth(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", w.Code, w.Body.String())
	}
	if svc.gotInput.TenantID != tenantID {
		t.Errorf("input tenant = %v, want %v", svc.gotInput.TenantID, tenantID)
	}
	if svc.gotInput.CustomerID != customerID {
		t.Errorf("input customer = %v, want %v", svc.gotInput.CustomerID, customerID)
	}
	// 2x the largest non-void invoice, not the void 50000 one.
	if svc.gotInput.MaxAmount != 24000 {
		t.Errorf("max amount = %d, want 24000 (2x largest non-void invoice)", svc.gotInput.MaxAmount)
	}
	// MandateService reads through the tenant-scoped repo — the handler must
	// inject the customer's tenant into the context.
	if got, _ := svc.gotCtx.Value(domain.TenantIDKey).(uuid.UUID); got != tenantID {
		t.Errorf("ctx tenant = %v, want %v", got, tenantID)
	}
	var resp struct {
		Data struct {
			AuthURL string `json:"auth_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Data.AuthURL == "" {
		t.Errorf("response missing auth_url: %s", w.Body.String())
	}
}

func TestPortalMandateReauth_NoBillingHistoryConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &mandateTestCustomerStore{cust: &domain.Customer{ID: uuid.New(), TenantID: uuid.New()}}
	h := NewPortalAPIHandler(nil)
	h.SetMandateReauth(store, &mandateTestSvc{}, &mandateTestInvoices{})

	c, w := newTestContext(http.MethodPost, "/portal/api/payment-method/mandate", "")
	c.Set("portal_customer_id", uuid.New())

	h.StartMandateReauth(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s, want 409", w.Code, w.Body.String())
	}
}

func TestPortalUpdatePaymentMethod_UsesSessionCustomerNotBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	custRepo := &portalTestCustomerRepo{}
	svc := service.NewPortalService(custRepo, nil, nil, nil, nil, nil, nil, "")
	h := NewPortalAPIHandler(svc)

	sessionCustomer := uuid.New()
	otherCustomer := uuid.New()

	// The body tries to smuggle a different customer_id; it must be ignored.
	body := `{"card_brand":"visa","card_last4":"4242","card_exp_month":12,"card_exp_year":2030,"customer_id":"` + otherCustomer.String() + `"}`
	c, w := newTestContext(http.MethodPut, "/portal/api/payment-method", body)
	c.Set("portal_customer_id", sessionCustomer)

	h.UpdatePaymentMethod(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", w.Code, w.Body.String())
	}
	if custRepo.calls != 1 {
		t.Fatalf("repo calls = %d, want 1", custRepo.calls)
	}
	if custRepo.gotCustomerID != sessionCustomer {
		t.Errorf("updated customer = %v, want session customer %v (body customer_id must be ignored)", custRepo.gotCustomerID, sessionCustomer)
	}
}

func TestPortalRaiseDispute_RejectsForeignInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	owner := uuid.New()
	attacker := uuid.New()
	invID := uuid.New()

	invRepo := &portalTestInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{
		invID: {ID: invID, TenantID: uuid.New(), CustomerID: owner},
	}}
	dispRepo := &portalTestDisputeRepo{}
	svc := service.NewPortalService(nil, invRepo, nil, nil, dispRepo, nil, nil, "")
	h := NewPortalAPIHandler(svc)

	c, w := newTestContext(http.MethodPost, "/portal/api/invoices/"+invID.String()+"/dispute", `{"reason":"give me a refund"}`)
	c.Params = gin.Params{{Key: "id", Value: invID.String()}}
	c.Set("portal_customer_id", attacker)

	h.RaiseDispute(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for foreign invoice", w.Code)
	}
	if len(dispRepo.created) != 0 {
		t.Fatalf("dispute created for foreign invoice: %d", len(dispRepo.created))
	}
}

func TestPortalRaiseDispute_SucceedsForOwnedInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	owner := uuid.New()
	invID := uuid.New()

	invRepo := &portalTestInvoiceRepo{invoices: map[uuid.UUID]*domain.Invoice{
		invID: {ID: invID, TenantID: uuid.New(), CustomerID: owner},
	}}
	dispRepo := &portalTestDisputeRepo{}
	svc := service.NewPortalService(nil, invRepo, nil, nil, dispRepo, nil, nil, "")
	h := NewPortalAPIHandler(svc)

	c, w := newTestContext(http.MethodPost, "/portal/api/invoices/"+invID.String()+"/dispute", `{"reason":"double charged"}`)
	c.Params = gin.Params{{Key: "id", Value: invID.String()}}
	c.Set("portal_customer_id", owner)

	h.RaiseDispute(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", w.Code, w.Body.String())
	}
	if len(dispRepo.created) != 1 {
		t.Fatalf("dispute count = %d, want 1", len(dispRepo.created))
	}
	var resp struct {
		Data domain.InvoiceDispute `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad response json: %v", err)
	}
	if resp.Data.CustomerID != owner || resp.Data.Reason != "double charged" {
		t.Errorf("unexpected dispute in response: %+v", resp.Data)
	}
}
