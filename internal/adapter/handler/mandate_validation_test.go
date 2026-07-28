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

// phonelessCustomerRepo returns a customer with no phone, so the UPI guards
// fire. Embedded interface: only GetByID may be called.
type phonelessCustomerRepo struct {
	port.CustomerRepository
	c *domain.Customer
}

func (r *phonelessCustomerRepo) GetByID(context.Context, uuid.UUID) (*domain.Customer, error) {
	return r.c, nil
}

// A UPI (INR) mandate without a phone/VPA is the CALLER's mistake — it must
// surface as a 400 with the guard's message, not a generic 500 (live smoke
// finding: the phone guard produced "internal error").
func TestCreateMandateGuardFailuresAre400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	custID := uuid.New()
	svc := service.NewMandateService(nil, nil, &phonelessCustomerRepo{c: &domain.Customer{ID: custID, TenantID: tenantID}}, nil)
	h := NewMandateHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := []byte(`{"customer_id":"` + custID.String() + `","currency":"INR","max_amount":5000,"frequency":"monthly"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/mandates", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", tenantID)

	h.CreateMandate(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("guard failure status = %d, want 400; body: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("phone")) {
		t.Fatalf("400 body should carry the guard message, got: %s", w.Body.String())
	}
}
