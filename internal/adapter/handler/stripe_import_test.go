package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// Minimal fakes implementing the service's dependency interfaces, so the handler
// test drives a real *service.StripeImportService end-to-end over HTTP.

type fakeImportCustomers struct{ existing []*domain.Customer }

func (f *fakeImportCustomers) ListCustomers(_ context.Context, _ uuid.UUID, _ domain.CustomerFilter) ([]*domain.Customer, error) {
	return f.existing, nil
}
func (f *fakeImportCustomers) CreateCustomer(_ context.Context, in service.CreateCustomerInput) (*domain.Customer, error) {
	c := &domain.Customer{ID: uuid.New(), Email: in.Email}
	f.existing = append(f.existing, c)
	return c, nil
}

type fakeImportCatalog struct{ existing []*domain.Plan }

func (f *fakeImportCatalog) ListPlans(_ context.Context, _ uuid.UUID, _ domain.PlanFilter) ([]*domain.Plan, error) {
	return f.existing, nil
}
func (f *fakeImportCatalog) CreatePlan(_ context.Context, in service.CreatePlanInput) (*domain.Plan, error) {
	p := &domain.Plan{ID: uuid.New(), Code: in.Code}
	f.existing = append(f.existing, p)
	return p, nil
}

type fakeRefRepo struct{ ids map[string]bool }

func (f *fakeRefRepo) Create(_ context.Context, ref *domain.ImportExternalRef) error {
	if f.ids == nil {
		f.ids = map[string]bool{}
	}
	f.ids[ref.ExternalID] = true
	return nil
}
func (f *fakeRefRepo) ListExternalIDs(_ context.Context, _ uuid.UUID, _ string) (map[string]bool, error) {
	return f.ids, nil
}

func newStripeImportHandler(existingEmail string) *StripeImportHandler {
	cust := &fakeImportCustomers{}
	if existingEmail != "" {
		cust.existing = []*domain.Customer{{Email: existingEmail}}
	}
	svc := service.NewStripeImportService(cust, &fakeImportCatalog{}, &fakeRefRepo{ids: map[string]bool{}})
	return NewStripeImportHandler(svc)
}

func withTenant(c *gin.Context) { c.Set("tenant_id", uuid.New()) }

const sampleImportBody = `{
  "customers":[
    {"id":"cus_new","email":"new@acme.com"},
    {"id":"cus_dupe","email":"existing@acme.com"}
  ],
  "products":[{"id":"prod_1","name":"Pro"}],
  "prices":[{"id":"price_m","product":"prod_1","unit_amount":4900,"currency":"usd","recurring":{"interval":"month"}}]
}`

type previewPlan struct {
	Summary map[string]int `json:"summary"`
}

func TestStripeImportPreview_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newStripeImportHandler("existing@acme.com")

	c, w := jsonCtx(http.MethodPost, "/v1/import/stripe/preview", sampleImportBody)
	withTenant(c)
	h.Preview(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", w.Code, w.Body.String())
	}
	var plan previewPlan
	if err := json.Unmarshal(w.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if plan.Summary["customer.create"] != 1 || plan.Summary["customer.link_existing"] != 1 {
		t.Errorf("customer summary wrong: %v", plan.Summary)
	}
	if plan.Summary["plan.create"] != 1 {
		t.Errorf("plan summary wrong: %v", plan.Summary)
	}
}

func TestStripeImportCommit_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newStripeImportHandler("")

	c, w := jsonCtx(http.MethodPost, "/v1/import/stripe/commit", sampleImportBody)
	withTenant(c)
	h.Commit(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", w.Code, w.Body.String())
	}
	var res struct {
		Created map[string]int `json:"created"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.Created["customer"] != 2 || res.Created["plan"] != 1 {
		t.Errorf("commit created wrong: %v", res.Created)
	}
}

func TestStripeImportPreview_MalformedJsonIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newStripeImportHandler("")

	c, w := jsonCtx(http.MethodPost, "/v1/import/stripe/preview", `{not json`)
	withTenant(c)
	h.Preview(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestStripeImportPreview_EmptyBodyIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newStripeImportHandler("")

	c, w := jsonCtx(http.MethodPost, "/v1/import/stripe/preview", ``)
	withTenant(c)
	h.Preview(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
