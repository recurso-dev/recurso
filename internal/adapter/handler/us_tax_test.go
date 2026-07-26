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

type stubUSTaxStore struct {
	cfg      *domain.TenantUSTaxConfig
	upserted *domain.TenantUSTaxConfig
}

func (s *stubUSTaxStore) GetByTenantID(ctx context.Context, t uuid.UUID) (*domain.TenantUSTaxConfig, error) {
	return s.cfg, nil
}
func (s *stubUSTaxStore) Upsert(ctx context.Context, c *domain.TenantUSTaxConfig) error {
	s.upserted = c
	return nil
}

func TestUSTaxConfig_GetEmptyDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewUSTaxConfigHandler(&stubUSTaxStore{cfg: nil})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", uuid.New())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/settings/tax/us", nil)

	h.GetUSTaxConfig(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"data"`) {
		t.Errorf("missing data envelope: %s", w.Body.String())
	}
}

func TestUSTaxConfig_PutRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &stubUSTaxStore{}
	h := NewUSTaxConfigHandler(store)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	tid := uuid.New()
	c.Set("tenant_id", tid)
	c.Request = httptest.NewRequest(http.MethodPut, "/v1/settings/tax/us",
		strings.NewReader(`{"legal_name":"Acme Inc","ein":" 12-3456789 ","address":"1 Market St"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateUSTaxConfig(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if store.upserted == nil {
		t.Fatal("nothing upserted")
	}
	if store.upserted.TenantID != tid {
		t.Errorf("tenant = %v, want %v", store.upserted.TenantID, tid)
	}
	if store.upserted.LegalName != "Acme Inc" || store.upserted.EIN != "12-3456789" || store.upserted.Address != "1 Market St" {
		t.Errorf("upserted = %+v (EIN should be trimmed)", store.upserted)
	}
}

// The US W-9 identity overrides the seller block of a US (non-GST) invoice, and
// leaves a GST invoice untouched.
func TestApplyUSSellerIdentity(t *testing.T) {
	h := &InvoicePDFHandler{}
	h.SetUSTaxIdentity(&stubUSTaxStore{cfg: &domain.TenantUSTaxConfig{
		LegalName: "Acme Inc", EIN: "12-3456789", Address: "1 Market St, SF",
	}})

	us := service.PDFInvoiceData{ShowGST: false, SellerName: "EnvCo", SellerTaxLabel: "Tax ID", SellerTaxID: "old"}
	h.applyUSSellerIdentity(context.Background(), uuid.New(), &us)
	if us.SellerName != "Acme Inc" || us.SellerTaxLabel != "EIN" || us.SellerTaxID != "12-3456789" || us.SellerAddress != "1 Market St, SF" {
		t.Errorf("US override wrong: %+v", us)
	}

	gst := service.PDFInvoiceData{ShowGST: true, SellerName: "EnvCo", SellerTaxLabel: "GSTIN", SellerTaxID: "29ABCDE1234F1Z5"}
	h.applyUSSellerIdentity(context.Background(), uuid.New(), &gst)
	if gst.SellerName != "EnvCo" || gst.SellerTaxID != "29ABCDE1234F1Z5" {
		t.Errorf("GST invoice must be untouched: %+v", gst)
	}

	// Nil resolver is a no-op.
	nilH := &InvoicePDFHandler{}
	d := service.PDFInvoiceData{ShowGST: false, SellerName: "EnvCo"}
	nilH.applyUSSellerIdentity(context.Background(), uuid.New(), &d)
	if d.SellerName != "EnvCo" {
		t.Errorf("nil resolver should be a no-op: %+v", d)
	}
}
