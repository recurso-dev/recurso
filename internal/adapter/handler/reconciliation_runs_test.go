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
// structurally, so it can be wired via SetRunStore from this package.
type stubRunStore struct{ list []domain.ReconciliationRun }

func (s *stubRunStore) Create(context.Context, *domain.ReconciliationRun) error { return nil }
func (s *stubRunStore) ListByTenant(context.Context, uuid.UUID, int) ([]domain.ReconciliationRun, error) {
	return s.list, nil
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
