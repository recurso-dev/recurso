package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type fakeCustomerLister struct{ customers []*domain.Customer }

func (f *fakeCustomerLister) ListCustomers(_ context.Context, _ uuid.UUID, _ domain.CustomerFilter) ([]*domain.Customer, error) {
	return f.customers, nil
}

type fakePlanLister struct{ plans []*domain.Plan }

func (f *fakePlanLister) ListPlans(_ context.Context, _ uuid.UUID, _ domain.PlanFilter) ([]*domain.Plan, error) {
	return f.plans, nil
}

// previewPlan is the decoded shape of the preview response (subset).
type previewPlan struct {
	Items []struct {
		Kind     string `json:"kind"`
		StripeID string `json:"stripe_id"`
		Action   string `json:"action"`
	} `json:"items"`
	Summary  map[string]int `json:"summary"`
	Warnings []string       `json:"warnings"`
}

func newStripeImportHandler(existingEmail string) *StripeImportHandler {
	cl := &fakeCustomerLister{}
	if existingEmail != "" {
		cl.customers = []*domain.Customer{{Email: existingEmail}}
	}
	return NewStripeImportHandler(cl, &fakePlanLister{})
}

func withTenant(c *gin.Context) {
	c.Set("tenant_id", uuid.New())
}

func TestStripeImportPreview_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newStripeImportHandler("existing@acme.com")

	body := `{
      "customers":[
        {"id":"cus_new","email":"new@acme.com"},
        {"id":"cus_dupe","email":"existing@acme.com"}
      ],
      "products":[{"id":"prod_1","name":"Pro"}],
      "prices":[{"id":"price_m","product":"prod_1","unit_amount":4900,"currency":"usd","recurring":{"interval":"month"}}],
      "subscriptions":[{"id":"sub_1","customer":"cus_new","status":"active","items":{"data":[{"price":{"id":"price_m"}}]}}]
    }`
	c, w := jsonCtx(http.MethodPost, "/v1/import/stripe/preview", body)
	withTenant(c)
	h.Preview(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", w.Code, w.Body.String())
	}
	var plan previewPlan
	if err := json.Unmarshal(w.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// The customer already present in Recurso links; the new one creates.
	if plan.Summary["customer.create"] != 1 || plan.Summary["customer.link_existing"] != 1 {
		t.Errorf("customer summary wrong: %v", plan.Summary)
	}
	if plan.Summary["plan.create"] != 1 || plan.Summary["subscription.create"] != 1 {
		t.Errorf("plan/subscription summary wrong: %v", plan.Summary)
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
