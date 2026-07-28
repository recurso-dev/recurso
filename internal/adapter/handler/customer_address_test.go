package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// recordingCustomerRepo records the customer passed to Create so the test can
// assert what the handler mapped from the request. Only Create is exercised.
type recordingCustomerRepo struct {
	port.CustomerRepository
	created *domain.Customer
}

func (r *recordingCustomerRepo) Create(_ context.Context, cust *domain.Customer) error {
	r.created = cust
	return nil
}

func TestCreateCustomerAcceptsNestedBillingAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &recordingCustomerRepo{}
	h := NewCustomerHandler(service.NewCustomerService(repo), nil)

	body := []byte(`{
		"email": "nested@example.com",
		"name": "Nested Addr",
		"billing_address": {"country": "US", "state": "CA", "postal_code": "90210", "line1": "1 Main St", "city": "LA"}
	}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/customers", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", uuid.New())

	h.CreateCustomer(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d want 201; body %s", w.Code, w.Body.String())
	}
	got := repo.created
	if got == nil {
		t.Fatal("customer was not created")
	}
	// The nested address must have been folded into the flat fields — the
	// silent-drop bug this closes.
	a := got.BillingAddress
	if a.Country != "US" || a.State != "CA" || a.Zip != "90210" || a.Line1 != "1 Main St" || a.City != "LA" {
		t.Fatalf("nested address dropped: %+v", a)
	}
}

func TestCreateCustomerFlatWinsOverNested(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &recordingCustomerRepo{}
	h := NewCustomerHandler(service.NewCustomerService(repo), nil)

	body := []byte(`{
		"email": "flat@example.com",
		"name": "Flat Wins",
		"country": "DE",
		"billing_address": {"country": "US"}
	}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/customers", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", uuid.New())

	h.CreateCustomer(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d want 201; body %s", w.Code, w.Body.String())
	}
	if repo.created.BillingAddress.Country != "DE" {
		t.Fatalf("flat country should win: got %q want DE", repo.created.BillingAddress.Country)
	}
}
