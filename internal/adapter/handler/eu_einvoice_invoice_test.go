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

type fakeEUInvoiceReader struct {
	rec *domain.EUInvoice
}

func (f *fakeEUInvoiceReader) GetByInvoiceID(context.Context, uuid.UUID) (*domain.EUInvoice, error) {
	return f.rec, nil
}

type fakeInvoiceOwner struct{ inv *domain.Invoice }

func (f *fakeInvoiceOwner) GetByID(context.Context, uuid.UUID) (*domain.Invoice, error) {
	return f.inv, nil
}

type fakeEUCustomer struct{ c *domain.Customer }

func (f *fakeEUCustomer) GetByID(context.Context, uuid.UUID) (*domain.Customer, error) {
	return f.c, nil
}

type fakeEUGenerator struct {
	rec    *domain.EUInvoice
	err    error
	called bool
}

func (f *fakeEUGenerator) GenerateForInvoice(context.Context, *domain.Invoice, *domain.Customer) (*domain.EUInvoice, error) {
	f.called = true
	return f.rec, f.err
}

// TestGetEUEInvoice_TenantScoped: an invoice belonging to another tenant is a 404,
// even though the eu_einvoices read itself is not tenant-scoped.
func TestGetEUEInvoice_TenantScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenant := uuid.New()
	invID := uuid.New()
	// Invoice owned by a DIFFERENT tenant.
	owner := &fakeInvoiceOwner{inv: &domain.Invoice{ID: invID, TenantID: uuid.New()}}
	h := NewEUEInvoiceHandler(&fakeEUInvoiceReader{}, owner, &fakeEUCustomer{}, &fakeEUGenerator{})

	c, w := newTestContext(http.MethodGet, "/invoices/"+invID.String()+"/eu-einvoice", "")
	c.Set("tenant_id", tenant)
	c.Params = gin.Params{{Key: "id", Value: invID.String()}}
	h.GetEUEInvoice(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for a cross-tenant invoice", w.Code)
	}
}

// TestGetEUEInvoice_ReturnsRecord returns the stored EU e-invoice for an owned invoice.
func TestGetEUEInvoice_ReturnsRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenant := uuid.New()
	invID := uuid.New()
	owner := &fakeInvoiceOwner{inv: &domain.Invoice{ID: invID, TenantID: tenant}}
	rec := &domain.EUInvoice{InvoiceID: invID, TenantID: tenant, Status: domain.EUInvoiceStatusSent, MessageID: "msg-1"}
	h := NewEUEInvoiceHandler(&fakeEUInvoiceReader{rec: rec}, owner, &fakeEUCustomer{}, &fakeEUGenerator{})

	c, w := newTestContext(http.MethodGet, "/invoices/"+invID.String()+"/eu-einvoice", "")
	c.Set("tenant_id", tenant)
	c.Params = gin.Params{{Key: "id", Value: invID.String()}}
	h.GetEUEInvoice(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data *domain.EUInvoice `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data == nil || resp.Data.MessageID != "msg-1" {
		t.Fatalf("expected the stored record, got %+v", resp.Data)
	}
}

// TestRetryEUEInvoice_RegeneratesForOwnedInvoice: retry loads the invoice+customer
// and re-runs generation; the returned record reflects the new outcome.
func TestRetryEUEInvoice_RegeneratesForOwnedInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenant := uuid.New()
	invID := uuid.New()
	custID := uuid.New()
	owner := &fakeInvoiceOwner{inv: &domain.Invoice{ID: invID, TenantID: tenant, CustomerID: custID}}
	gen := &fakeEUGenerator{rec: &domain.EUInvoice{InvoiceID: invID, Status: domain.EUInvoiceStatusSent}}
	h := NewEUEInvoiceHandler(&fakeEUInvoiceReader{}, owner, &fakeEUCustomer{c: &domain.Customer{ID: custID}}, gen)

	c, w := newTestContext(http.MethodPost, "/invoices/"+invID.String()+"/eu-einvoice/retry", "")
	c.Set("tenant_id", tenant)
	c.Params = gin.Params{{Key: "id", Value: invID.String()}}
	h.RetryEUEInvoice(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !gen.called {
		t.Fatal("GenerateForInvoice was not called")
	}
}

// TestRetryEUEInvoice_NotOptedIn: a nil record with no error (tenant not opted in)
// returns a clear message and data:null, not an error.
func TestRetryEUEInvoice_NotOptedIn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenant := uuid.New()
	invID := uuid.New()
	owner := &fakeInvoiceOwner{inv: &domain.Invoice{ID: invID, TenantID: tenant}}
	h := NewEUEInvoiceHandler(&fakeEUInvoiceReader{}, owner, &fakeEUCustomer{c: &domain.Customer{}}, &fakeEUGenerator{})

	c, w := newTestContext(http.MethodPost, "/invoices/"+invID.String()+"/eu-einvoice/retry", "")
	c.Set("tenant_id", tenant)
	c.Params = gin.Params{{Key: "id", Value: invID.String()}}
	h.RetryEUEInvoice(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data    *domain.EUInvoice `json:"data"`
		Message string            `json:"message"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data != nil || resp.Message == "" {
		t.Fatalf("expected data:null + a message, got data=%+v msg=%q", resp.Data, resp.Message)
	}
}
