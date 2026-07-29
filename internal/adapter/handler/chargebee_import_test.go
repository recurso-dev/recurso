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
	svc := service.NewChargebeeImportService(cust, &fakeImportCatalog{}, &fakeImportSubs{}, &fakeRefRepo{})
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

func TestChargebeeImportCommit_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newChargebeeImportHandler("existing@acme.com")

	c, w := jsonCtx(http.MethodPost, "/v1/import/chargebee/commit", chargebeeBody)
	c.Set("tenant_id", uuid.New())
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
	// cb_new created; cb_dupe links; pro plan created; sub_1 created.
	if res.Created["customer"] != 1 || res.Created["plan"] != 1 || res.Created["subscription"] != 1 {
		t.Errorf("commit created wrong: %v", res.Created)
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
