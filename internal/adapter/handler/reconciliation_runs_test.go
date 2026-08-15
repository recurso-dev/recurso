package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/service"
)

// stubRunStore satisfies the service's (unexported) run-store interface
// structurally, so it can be wired via SetRunStore from this package. detail
// mirrors the real repo's tenant scoping: GetByID returns it only when the id
// matches (foreign/unknown ids resolve to nil → 404 at the handler).
type stubRunStore struct {
	list   []domain.ReconciliationRun
	detail *domain.ReconciliationRunDetail
}

func (s *stubRunStore) Create(context.Context, *domain.ReconciliationRun, []domain.ReconciliationRunDiscrepancy) error {
	return nil
}
func (s *stubRunStore) ListByTenant(context.Context, uuid.UUID, int) ([]domain.ReconciliationRun, error) {
	return s.list, nil
}
func (s *stubRunStore) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.ReconciliationRunDetail, error) {
	if s.detail != nil && s.detail.ID == id && s.detail.TenantID == tenantID {
		return s.detail, nil
	}
	return nil, nil
}

func TestListReconciliationRunsReturnsHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewReconciliationService(nil, nil)
	svc.SetRunStore(&stubRunStore{list: []domain.ReconciliationRun{
		{ID: uuid.New(), InvoicesChecked: 9, TotalDiscrepancies: 0},
	}})
	h := NewReconciliationHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/finance/reconciliation/runs", nil)
	c.Set("tenant_id", uuid.New())
	h.ListReconciliationRuns(c)

	if w.Code != http.StatusOK {
		t.Fatalf("run history: got %d want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "total_discrepancies") {
		t.Fatalf("expected a run summary in the body, got %s", w.Body.String())
	}
}

func doGetRun(h *ReconciliationHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/finance/reconciliation/runs/"+id, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetReconciliationRun(c)
	return w
}

func newRunGetHandler(detail *domain.ReconciliationRunDetail) *ReconciliationHandler {
	svc := service.NewReconciliationService(nil, nil)
	svc.SetRunStore(&stubRunStore{detail: detail})
	return NewReconciliationHandler(svc)
}

func TestGetReconciliationRunReturnsDetailWithDiscrepancies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	invID := uuid.New()
	detail := &domain.ReconciliationRunDetail{
		ReconciliationRun: domain.ReconciliationRun{ID: uuid.New(), TenantID: tenantID, InvoicesChecked: 9, TotalDiscrepancies: 1},
		Discrepancies: []domain.ReconciliationRunDiscrepancy{
			{Type: "invoice_amount_mismatch", InvoiceID: &invID, ExpectedAmount: 10000, FoundAmount: 9000},
		},
	}
	w := doGetRun(newRunGetHandler(detail), tenantID, detail.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("owned run: got %d want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "invoice_amount_mismatch") || !strings.Contains(body, `"discrepancies"`) {
		t.Fatalf("expected the discrepancy detail in the body, got %s", body)
	}
}

func TestGetReconciliationRunCleanRunHasEmptyDiscrepancies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	detail := &domain.ReconciliationRunDetail{
		ReconciliationRun: domain.ReconciliationRun{ID: uuid.New(), TenantID: tenantID, TotalDiscrepancies: 0},
		Discrepancies:     []domain.ReconciliationRunDiscrepancy{},
	}
	w := doGetRun(newRunGetHandler(detail), tenantID, detail.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("clean run: got %d want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"discrepancies":[]`) {
		t.Fatalf("clean run should carry an empty discrepancy array, got %s", w.Body.String())
	}
}

func TestGetReconciliationRunCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	detail := &domain.ReconciliationRunDetail{
		ReconciliationRun: domain.ReconciliationRun{ID: uuid.New(), TenantID: uuid.New()},
	}
	if got := doGetRun(newRunGetHandler(detail), uuid.New(), detail.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant run: got %d want 404 (flat)", got)
	}
}

func TestGetReconciliationRunBadIdIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if got := doGetRun(newRunGetHandler(nil), uuid.New(), "nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad id: got %d want 400", got)
	}
}

func TestListReconciliationRunsUnauthorizedWithoutTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewReconciliationHandler(service.NewReconciliationService(nil, nil))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/finance/reconciliation/runs", nil)
	h.ListReconciliationRuns(c) // no tenant_id set
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing tenant: got %d want 401", w.Code)
	}
}
