package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

type stubBrandingStore struct {
	b        *domain.TenantInvoiceBranding
	upserted *domain.TenantInvoiceBranding
}

func (s *stubBrandingStore) GetByTenantID(ctx context.Context, t uuid.UUID) (*domain.TenantInvoiceBranding, error) {
	return s.b, nil
}
func (s *stubBrandingStore) Upsert(ctx context.Context, b *domain.TenantInvoiceBranding) error {
	s.upserted = b
	return nil
}

func pngDataURL(nBytes int) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(make([]byte, nBytes))
}

func TestInvoiceBranding_GetEmptyDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewInvoiceBrandingHandler(&stubBrandingStore{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", uuid.New())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/settings/invoice-branding", nil)

	h.GetBranding(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"data"`) {
		t.Errorf("missing data envelope: %s", w.Body.String())
	}
}

func TestInvoiceBranding_PutRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &stubBrandingStore{}
	h := NewInvoiceBrandingHandler(store)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", uuid.New())
	body := fmt.Sprintf(`{"company_name":" Acme Labs ","logo_data_url":%q,"signatory_name":"J. Doe","bank_details":"HDFC 000","terms":"Net 30"}`, pngDataURL(64))
	c.Request = httptest.NewRequest(http.MethodPut, "/v1/settings/invoice-branding", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateBranding(c)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	if store.upserted == nil {
		t.Fatal("nothing upserted")
	}
	if store.upserted.CompanyName != "Acme Labs" {
		t.Errorf("company name not trimmed: %q", store.upserted.CompanyName)
	}
	if store.upserted.LogoDataURL == "" {
		t.Error("logo not persisted")
	}
}

// The image validation is the security boundary that makes the stored value
// safe to render as template.URL — these rejects are load-bearing.
func TestInvoiceBranding_RejectsUnsafeImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bad := []struct {
		name, logo string
	}{
		{"svg (script-capable)", "data:image/svg+xml;base64,PHN2Zz48L3N2Zz4="},
		{"attribute breakout", `data:image/png;base64,abc" onerror="alert(1)`},
		{"http url", "https://example.com/logo.png"},
		{"oversized", pngDataURL(301 * 1024)},
		{"not base64", "data:image/png;base64,%%%%"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			store := &stubBrandingStore{}
			h := NewInvoiceBrandingHandler(store)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("tenant_id", uuid.New())
			body := fmt.Sprintf(`{"logo_data_url":%q}`, tc.logo)
			c.Request = httptest.NewRequest(http.MethodPut, "/v1/settings/invoice-branding", strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			h.UpdateBranding(c)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("%s accepted: code=%d body=%s", tc.name, w.Code, w.Body.String())
			}
			if store.upserted != nil {
				t.Fatalf("%s reached storage", tc.name)
			}
		})
	}
}
