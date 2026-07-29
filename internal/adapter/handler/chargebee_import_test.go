package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

func newChargebeeImportHandler(existingEmail string) *ChargebeeImportHandler {
	cust := &fakeImportCustomers{}
	if existingEmail != "" {
		cust.existing = []*domain.Customer{{Email: existingEmail}}
	}
	svc := service.NewChargebeeImportService(cust, &fakeImportCatalog{}, &fakeRefRepo{})
	return NewChargebeeImportHandler(svc)
}

const chargebeeBody = `{
  "customers":[
    {"id":"cb_new","email":"new@acme.com"},
    {"id":"cb_dupe","email":"existing@acme.com"}
  ],
  "plans":[{"id":"pro","name":"Pro","price":4900,"period":1,"period_unit":"month","currency_code":"usd","status":"active"}],
  "subscriptions":[{"id":"sub_1","customer_id":"cb_new","plan_id":"pro","status":"active"}]
}`

func TestChargebeeImportPreview_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newChargebeeImportHandler("existing@acme.com")

	c, w := jsonCtx(http.MethodPost, "/v1/import/chargebee/preview", chargebeeBody)
	c.Set("tenant_id", uuid.New())
	h.Preview(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", w.Code, w.Body.String())
	}
	var plan struct {
		Summary map[string]int `json:"summary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if plan.Summary["customer.create"] != 1 || plan.Summary["customer.link_existing"] != 1 {
		t.Errorf("customer summary wrong: %v", plan.Summary)
	}
	if plan.Summary["plan.create"] != 1 || plan.Summary["subscription.create"] != 1 {
		t.Errorf("plan/subscription summary wrong: %v", plan.Summary)
	}
}

func TestChargebeeImportPreview_MalformedJsonIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newChargebeeImportHandler("")

	c, w := jsonCtx(http.MethodPost, "/v1/import/chargebee/preview", `{bad`)
	c.Set("tenant_id", uuid.New())
	h.Preview(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
