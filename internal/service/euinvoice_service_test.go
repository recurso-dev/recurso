package service

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/einvoice_eu"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

type fakeEUConfig struct{ cfg *domain.TenantEUConfig }

func (f *fakeEUConfig) GetByTenantEntity(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (*domain.TenantEUConfig, error) {
	return f.cfg, nil
}

type fakeEUStore struct{ saved []*domain.EUInvoice }

func (f *fakeEUStore) Upsert(_ context.Context, e *domain.EUInvoice) error {
	f.saved = append(f.saved, e)
	return nil
}

func euCustomer() *domain.Customer {
	name := "Beta Sàrl"
	vat := "FR12345678901"
	return &domain.Customer{
		ID:             uuid.New(),
		Name:           &name,
		Email:          "billing@beta.fr",
		TaxID:          &vat,
		BillingAddress: domain.BillingAddress{Line1: "1 Rue", City: "Paris", Zip: "75001", Country: "France"},
	}
}

func enabledEUConfig(tenantID uuid.UUID) *domain.TenantEUConfig {
	return &domain.TenantEUConfig{
		TenantID: tenantID, Enabled: true, LegalName: "Acme GmbH",
		VATNumber: "DE123456789", CountryCode: "DE", Street: "Hauptstr. 1", City: "Berlin", PostalZone: "10115",
	}
}

// TestEUEInvoice_OptInGate: without an enabled config, generation is a silent
// no-op — nothing is generated or stored.
func TestEUEInvoice_OptInGate(t *testing.T) {
	inv := sampleEUInvoice()
	inv.TenantID = uuid.New()

	for _, tc := range []struct {
		name string
		cfg  *domain.TenantEUConfig
	}{
		{"no config", nil},
		{"config disabled", &domain.TenantEUConfig{TenantID: inv.TenantID, Enabled: false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeEUStore{}
			svc := NewEUEInvoiceService(&fakeEUConfig{cfg: tc.cfg}, store, einvoice_eu.NewMockTransport())
			rec, err := svc.GenerateForInvoice(context.Background(), inv, euCustomer())
			if err != nil || rec != nil {
				t.Fatalf("want (nil,nil) for opted-out tenant, got (%v,%v)", rec, err)
			}
			if len(store.saved) != 0 {
				t.Fatalf("nothing should be persisted when opted out, got %d", len(store.saved))
			}
		})
	}
}

// TestEUEInvoice_GeneratesTransmitsPersists: an opted-in tenant produces a UBL
// document, transmits it (mock), and stores a 'sent' record with the document.
func TestEUEInvoice_GeneratesTransmitsPersists(t *testing.T) {
	inv := sampleEUInvoice()
	inv.TenantID = uuid.New()
	store := &fakeEUStore{}
	svc := NewEUEInvoiceService(&fakeEUConfig{cfg: enabledEUConfig(inv.TenantID)}, store, einvoice_eu.NewMockTransport())

	rec, err := svc.GenerateForInvoice(context.Background(), inv, euCustomer())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if rec.Status != domain.EUInvoiceStatusSent {
		t.Fatalf("status = %q, want sent", rec.Status)
	}
	if rec.MessageID == "" || rec.Document == "" || rec.Syntax != domain.EUInvoiceSyntaxUBL {
		t.Fatalf("record incomplete: %+v", rec)
	}
	if len(store.saved) != 1 || store.saved[0].InvoiceID != inv.ID {
		t.Fatalf("want one persisted record for the invoice, got %d", len(store.saved))
	}
	// The stored document is the UBL — buyer country was normalized France -> FR.
	if !strings.Contains(rec.Document, "<cbc:IdentificationCode>FR</cbc:IdentificationCode>") {
		t.Errorf("document missing normalized buyer country FR")
	}
}

// TestBuildEUBuyer_CountryNormalization proves name→ISO2 mapping and passthrough.
func TestBuildEUBuyer_CountryNormalization(t *testing.T) {
	for in, want := range map[string]string{"France": "FR", "germany": "DE", "NL": "NL", "es": "ES"} {
		c := &domain.Customer{BillingAddress: domain.BillingAddress{Country: in}}
		if got := buildEUBuyer(c).CountryCode; got != want {
			t.Errorf("country %q -> %q, want %q", in, got, want)
		}
	}
	// VAT + name come from the customer.
	b := buildEUBuyer(euCustomer())
	if b.VATID != "FR12345678901" || b.Name != "Beta Sàrl" {
		t.Errorf("buyer party wrong: %+v", b)
	}
}
