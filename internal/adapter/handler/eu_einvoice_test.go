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

type fakeEUConfigStore struct {
	cfg      *domain.TenantEUConfig
	upserted *domain.TenantEUConfig
}

func (f *fakeEUConfigStore) GetByTenantID(_ context.Context, _ uuid.UUID) (*domain.TenantEUConfig, error) {
	return f.cfg, nil
}
func (f *fakeEUConfigStore) Upsert(_ context.Context, c *domain.TenantEUConfig) error {
	f.upserted = c
	return nil
}

func euCtx(method, body string) (*gin.Context, *fakeEUConfigStore, func() int, func() []byte) {
	gin.SetMode(gin.TestMode)
	store := &fakeEUConfigStore{}
	c, w := newTestContext(method, "/settings/eu-einvoice", body)
	c.Set("tenant_id", uuid.New())
	return c, store, func() int { return w.Code }, func() []byte { return w.Body.Bytes() }
}

// TestUpdateEUConfig_EnableRequiresSellerIdentity: opting in without a complete
// seller identity is a 400, so a tenant can't enable a config that would fail on
// the first invoice.
func TestUpdateEUConfig_EnableRequiresSellerIdentity(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"enabled no legal name", `{"enabled":true,"vat_number":"DE123","country_code":"DE"}`},
		{"enabled no vat", `{"enabled":true,"legal_name":"Acme","country_code":"DE"}`},
		{"enabled no country", `{"enabled":true,"legal_name":"Acme","vat_number":"DE123"}`},
		{"bad country length", `{"enabled":false,"country_code":"DEU"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, store, code, _ := euCtx(http.MethodPut, tc.body)
			NewEUConfigHandler(store).UpdateEUConfig(c)
			if code() != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", code())
			}
			if store.upserted != nil {
				t.Fatal("nothing should be persisted on a validation failure")
			}
		})
	}
}

// TestUpdateEUConfig_Valid normalizes and persists a complete config.
func TestUpdateEUConfig_Valid(t *testing.T) {
	c, store, code, _ := euCtx(http.MethodPut,
		`{"enabled":true,"legal_name":" Acme GmbH ","vat_number":"de 123 456 789","country_code":"de","city":"Berlin"}`)
	NewEUConfigHandler(store).UpdateEUConfig(c)
	if code() != http.StatusOK {
		t.Fatalf("status = %d, want 200", code())
	}
	if store.upserted == nil {
		t.Fatal("config was not persisted")
	}
	got := store.upserted
	if !got.Enabled || got.LegalName != "Acme GmbH" || got.VATNumber != "DE123456789" || got.CountryCode != "DE" {
		t.Fatalf("normalization wrong: %+v", got)
	}
}

// TestGetEUConfig_DefaultWhenUnset returns an empty disabled config when none.
func TestGetEUConfig_DefaultWhenUnset(t *testing.T) {
	c, store, code, body := euCtx(http.MethodGet, "")
	NewEUConfigHandler(store).GetEUConfig(c)
	if code() != http.StatusOK {
		t.Fatalf("status = %d, want 200", code())
	}
	var resp struct {
		Data EUConfigDTO `json:"data"`
	}
	if err := json.Unmarshal(body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.Enabled {
		t.Fatal("default config should be disabled")
	}
}
