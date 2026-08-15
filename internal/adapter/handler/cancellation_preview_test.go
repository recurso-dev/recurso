package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// newCancelPreviewHandler wires a subscription service with just the reads the
// preview needs (subscription + plan). No rev-rec is wired, so the forfeit
// figure is 0 here — the forfeit arithmetic is covered by a service-level test.
func newCancelPreviewHandler(sub *domain.Subscription, plan *domain.Plan) *CancellationHandler {
	svc := service.NewSubscriptionService(
		&oneSubscriptionRepo{sub: sub}, nil, &planByIDRepo{plan: plan},
		nil, nil, nil, nil, nil, nil, nil, nil, nil)
	return NewCancellationHandler(svc, nil, nil)
}

func doCancelPreview(h *CancellationHandler, tenantID uuid.UUID, id, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/subscriptions/"+id+"/cancel-preview"+query, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.PreviewCancel(c)
	return w
}

func TestCancelPreviewImmediate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: plan.ID, Status: domain.SubscriptionStatusActive}
	w := doCancelPreview(newCancelPreviewHandler(sub, plan), tenantID, sub.ID.String(), "?immediately=true")
	if w.Code != http.StatusOK {
		t.Fatalf("immediate preview: got %d want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{`"immediately":true`, `"resulting_status":"canceled"`, `"cancel_at_period_end":false`, `"avoided_future_recurring":200000`, `"flat_fee_refund":0`, `"deferred_revenue_forfeited":0`} {
		if !strings.Contains(body, want) {
			t.Fatalf("immediate preview missing %s: %s", want, body)
		}
	}
}

func TestCancelPreviewAtPeriodEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: plan.ID, Status: domain.SubscriptionStatusActive}
	body := doCancelPreview(newCancelPreviewHandler(sub, plan), tenantID, sub.ID.String(), "").Body.String()
	// Default (no ?immediately) previews an at-period-end cancel: status unchanged,
	// cancel_at_period_end true, and no revenue forfeited now.
	for _, want := range []string{`"immediately":false`, `"cancel_at_period_end":true`, `"resulting_status":"active"`, `"deferred_revenue_forfeited":0`} {
		if !strings.Contains(body, want) {
			t.Fatalf("period-end preview missing %s: %s", want, body)
		}
	}
}

func TestCancelPreviewAlreadyCanceledIsNoOp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: plan.ID, Status: domain.SubscriptionStatusCanceled}
	body := doCancelPreview(newCancelPreviewHandler(sub, plan), tenantID, sub.ID.String(), "?immediately=true").Body.String()
	// Re-canceling forfeits/avoids nothing.
	for _, want := range []string{`"resulting_status":"canceled"`, `"deferred_revenue_forfeited":0`, `"avoided_future_recurring":0`} {
		if !strings.Contains(body, want) {
			t.Fatalf("already-canceled preview missing %s: %s", want, body)
		}
	}
}

func TestCancelPreviewIsIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: plan.ID, Status: domain.SubscriptionStatusActive}
	h := newCancelPreviewHandler(sub, plan)
	// Repeated previews report the same money fields (pure read, no mutation).
	// effective_date is a wall-clock timestamp and is deliberately excluded.
	moneyFields := func(body string) map[string]any {
		var env struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &env); err != nil {
			t.Fatalf("decode preview: %v (%s)", err, body)
		}
		out := map[string]any{}
		for _, k := range []string{"deferred_revenue_forfeited", "recognized_as_breakage", "avoided_future_recurring", "flat_fee_refund", "resulting_status", "cancel_at_period_end"} {
			out[k] = env.Data[k]
		}
		return out
	}
	a := moneyFields(doCancelPreview(h, tenantID, sub.ID.String(), "?immediately=true").Body.String())
	b := moneyFields(doCancelPreview(h, tenantID, sub.ID.String(), "?immediately=true").Body.String())
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("preview not idempotent: %v vs %v", a, b)
	}
}

func TestCancelPreviewCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: uuid.New(), PlanID: plan.ID, Status: domain.SubscriptionStatusActive}
	if got := doCancelPreview(newCancelPreviewHandler(sub, plan), uuid.New(), sub.ID.String(), "?immediately=true").Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant preview: got %d want 404 (flat)", got)
	}
}

func TestCancelPreviewBadIdIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if got := doCancelPreview(newCancelPreviewHandler(nil, nil), uuid.New(), "nope", "").Code; got != http.StatusBadRequest {
		t.Fatalf("bad id: got %d want 400", got)
	}
}
