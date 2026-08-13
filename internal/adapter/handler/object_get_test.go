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
	"github.com/recurso-dev/recurso/internal/core/port"
	"github.com/recurso-dev/recurso/internal/service"
)

// --- GET /v1/invoices/:id -------------------------------------------------

// oneInvoiceRepo mirrors the REAL repository's contract: GetByID enforces the
// tenant from the context key (a foreign invoice is invisible), which is the
// property the handler's flat-404 relies on.
type oneInvoiceRepo struct {
	port.InvoiceRepository
	inv *domain.Invoice
}

func (r *oneInvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	tenantID, _ := ctx.Value(domain.TenantIDKey).(uuid.UUID)
	if r.inv != nil && r.inv.ID == id && r.inv.TenantID == tenantID {
		return r.inv, nil
	}
	return nil, nil
}

func newInvoiceGetHandler(inv *domain.Invoice) *SubscriptionHandler {
	svc := service.NewSubscriptionService(nil, &oneInvoiceRepo{inv: inv}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	return NewSubscriptionHandler(svc)
}

func doGetInvoice(h *SubscriptionHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/invoices/"+id, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetInvoice(c)
	return w
}

func TestGetInvoiceReturnsOwnedRow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	inv := &domain.Invoice{ID: uuid.New(), TenantID: tenantID, InvoiceNumber: "INV-1", Total: 1000, Currency: "USD"}
	if got := doGetInvoice(newInvoiceGetHandler(inv), tenantID, inv.ID.String()).Code; got != http.StatusOK {
		t.Fatalf("owned invoice: got %d want 200", got)
	}
}

func TestGetInvoiceCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inv := &domain.Invoice{ID: uuid.New(), TenantID: uuid.New()}
	if got := doGetInvoice(newInvoiceGetHandler(inv), uuid.New(), inv.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant read: got %d want 404 (flat — never confirm existence)", got)
	}
}

func TestGetInvoiceBadIdIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if got := doGetInvoice(newInvoiceGetHandler(nil), uuid.New(), "nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad id: got %d want 400", got)
	}
}

// --- GET /v1/invoices/:id/journal-entries ---------------------------------

func doGetInvoiceJournal(h *SubscriptionHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/invoices/"+id+"/journal-entries", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetInvoiceJournalEntries(c)
	return w
}

func TestGetInvoiceJournalOwnedReturnsEntriesArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	inv := &domain.Invoice{ID: uuid.New(), TenantID: tenantID}
	w := doGetInvoiceJournal(newInvoiceGetHandler(inv), tenantID, inv.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("owned invoice journal: got %d want 200", w.Code)
	}
	// An existing invoice always returns an entries array (empty here — no ledger
	// wired in this harness), never 404. This is the draft-invoice case: exists,
	// no postings yet.
	if !strings.Contains(w.Body.String(), `"entries"`) {
		t.Fatalf("expected an entries array, got %s", w.Body.String())
	}
}

func TestGetInvoiceJournalCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inv := &domain.Invoice{ID: uuid.New(), TenantID: uuid.New()}
	if got := doGetInvoiceJournal(newInvoiceGetHandler(inv), uuid.New(), inv.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant journal: got %d want 404 (flat — never confirm existence)", got)
	}
}

func TestGetInvoiceJournalBadIdIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if got := doGetInvoiceJournal(newInvoiceGetHandler(nil), uuid.New(), "nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad id: got %d want 400", got)
	}
}

// --- GET /v1/credit-notes/:id ----------------------------------------------

type oneCreditNoteRepo struct {
	service.CreditNoteRepository
	cn *domain.CreditNote
}

func (r *oneCreditNoteRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*domain.CreditNote, error) {
	if r.cn != nil && r.cn.ID == id && r.cn.TenantID == tenantID {
		return r.cn, nil
	}
	return nil, nil
}

func newCreditNoteGetHandler(cn *domain.CreditNote) *CreditNoteHandler {
	svc := service.NewCreditNoteService(&oneCreditNoteRepo{cn: cn}, nil, nil, nil)
	return NewCreditNoteHandler(svc)
}

func doGetCreditNote(h *CreditNoteHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/credit-notes/"+id, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetCreditNote(c)
	return w
}

func TestGetCreditNoteReturnsOwnedRow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	cn := &domain.CreditNote{ID: uuid.New(), TenantID: tenantID, Amount: 500, Currency: "USD"}
	if got := doGetCreditNote(newCreditNoteGetHandler(cn), tenantID, cn.ID.String()).Code; got != http.StatusOK {
		t.Fatalf("owned credit note: got %d want 200", got)
	}
}

func TestGetCreditNoteCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cn := &domain.CreditNote{ID: uuid.New(), TenantID: uuid.New()}
	if got := doGetCreditNote(newCreditNoteGetHandler(cn), uuid.New(), cn.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant read: got %d want 404", got)
	}
}

// --- GET /v1/invoices?customer_id= ------------------------------------------

// customerScopedInvoiceRepo proves the handler routes a customer_id query to
// the tenant-scoped customer list, not the whole-tenant list.
type customerScopedInvoiceRepo struct {
	port.InvoiceRepository
	tenantID   uuid.UUID
	customerID uuid.UUID
	rows       []*domain.Invoice
}

func (r *customerScopedInvoiceRepo) ListByCustomerPaginated(_ context.Context, tenantID, customerID uuid.UUID, _, _ int) ([]*domain.Invoice, error) {
	if tenantID == r.tenantID && customerID == r.customerID {
		return r.rows, nil
	}
	return nil, nil
}

func (r *customerScopedInvoiceRepo) CountByCustomer(_ context.Context, tenantID, customerID uuid.UUID) (int, error) {
	if tenantID == r.tenantID && customerID == r.customerID {
		return len(r.rows), nil
	}
	return 0, nil
}

func doListInvoices(h *SubscriptionHandler, tenantID uuid.UUID, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/invoices"+query, nil)
	c.Set("tenant_id", tenantID)
	h.ListInvoices(c)
	return w
}

func TestListInvoicesCustomerFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID, customerID := uuid.New(), uuid.New()
	repo := &customerScopedInvoiceRepo{
		tenantID:   tenantID,
		customerID: customerID,
		rows:       []*domain.Invoice{{ID: uuid.New(), TenantID: tenantID, CustomerID: customerID}},
	}
	h := NewSubscriptionHandler(service.NewSubscriptionService(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	if got := doListInvoices(h, tenantID, "?customer_id="+customerID.String()).Code; got != http.StatusOK {
		t.Fatalf("customer-scoped list: got %d want 200", got)
	}
	if got := doListInvoices(h, tenantID, "?customer_id=nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad customer_id: got %d want 400", got)
	}
}

// --- GET /v1/invoices?subscription_id= ---------------------------------------

type subscriptionScopedInvoiceRepo struct {
	port.InvoiceRepository
	tenantID       uuid.UUID
	subscriptionID uuid.UUID
	rows           []*domain.Invoice
}

func (r *subscriptionScopedInvoiceRepo) ListBySubscriptionPaginated(_ context.Context, tenantID, subscriptionID uuid.UUID, _, _ int) ([]*domain.Invoice, error) {
	if tenantID == r.tenantID && subscriptionID == r.subscriptionID {
		return r.rows, nil
	}
	return nil, nil
}

func (r *subscriptionScopedInvoiceRepo) CountBySubscription(_ context.Context, tenantID, subscriptionID uuid.UUID) (int, error) {
	if tenantID == r.tenantID && subscriptionID == r.subscriptionID {
		return len(r.rows), nil
	}
	return 0, nil
}

func TestListInvoicesSubscriptionFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID, subID := uuid.New(), uuid.New()
	repo := &subscriptionScopedInvoiceRepo{
		tenantID:       tenantID,
		subscriptionID: subID,
		rows:           []*domain.Invoice{{ID: uuid.New(), TenantID: tenantID}},
	}
	h := NewSubscriptionHandler(service.NewSubscriptionService(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	if got := doListInvoices(h, tenantID, "?subscription_id="+subID.String()).Code; got != http.StatusOK {
		t.Fatalf("subscription-scoped list: got %d want 200", got)
	}
	if got := doListInvoices(h, tenantID, "?subscription_id=nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad subscription_id: got %d want 400", got)
	}
}
