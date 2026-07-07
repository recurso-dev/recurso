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
	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
	"github.com/swapnull-in/recur-so/internal/service"
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
