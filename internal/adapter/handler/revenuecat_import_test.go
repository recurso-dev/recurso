package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/service"
)

func newRevenueCatImportHandler() *RevenueCatImportHandler {
	svc := service.NewRevenueCatImportService(&fakeImportCustomers{}, &fakeImportCatalog{}, &fakeImportSubs{}, &fakeRefRepo{})
	return NewRevenueCatImportHandler(svc)
}

const revenuecatBody = `{
  "products":[{"id":"monthly","title":"Pro","price":999,"currency":"usd","period_unit":"month","period_count":1}],
  "subscribers":[
    {"app_user_id":"user_a","email":"a@acme.com","subscriptions":[{"product_id":"monthly","store":"app_store","is_active":true}]},
    {"app_user_id":"user_noemail","subscriptions":[{"product_id":"monthly","is_active":true}]}
  ]
}`

func TestRevenueCatImportPreview_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newRevenueCatImportHandler()

	c, w := jsonCtx(http.MethodPost, "/v1/import/revenuecat/preview", revenuecatBody)
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
	if plan.Summary["plan.create"] != 1 || plan.Summary["customer.create"] != 1 || plan.Summary["customer.conflict"] != 1 {
		t.Errorf("summary wrong (expect the no-email subscriber as a conflict): %v", plan.Summary)
	}
}

func TestRevenueCatImportCommit_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newRevenueCatImportHandler()

	c, w := jsonCtx(http.MethodPost, "/v1/import/revenuecat/commit", revenuecatBody)
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
	if res.Created["plan"] != 1 || res.Created["customer"] != 1 || res.Created["subscription"] != 1 {
		t.Errorf("commit created wrong: %v", res.Created)
	}
}

func TestRevenueCatImportPreview_MalformedJsonIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newRevenueCatImportHandler()

	c, w := jsonCtx(http.MethodPost, "/v1/import/revenuecat/preview", `{bad`)
	c.Set("tenant_id", uuid.New())
	h.Preview(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
