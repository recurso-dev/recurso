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

// --- GET /v1/subscriptions/:id/history ------------------------------------

type oneSubscriptionRepo struct {
	port.SubscriptionRepository
	sub *domain.Subscription
}

func (r *oneSubscriptionRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Subscription, error) {
	if r.sub != nil && r.sub.ID == id {
		return r.sub, nil
	}
	return nil, nil
}

func doGetSubHistory(sub *domain.Subscription, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	svc := service.NewSubscriptionService(&oneSubscriptionRepo{sub: sub}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := NewSubscriptionHandler(svc)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/subscriptions/"+id+"/history", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetSubscriptionHistory(c)
	return w
}

func TestGetSubscriptionHistoryOwnedReturnsHistoryArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID}
	// No history reader wired → an owned subscription yields an empty (non-nil)
	// timeline, 200 — never 404.
	w := doGetSubHistory(sub, tenantID, sub.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("owned subscription history: got %d want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"history"`) {
		t.Fatalf("expected a history array, got %s", w.Body.String())
	}
}

func TestGetSubscriptionHistoryCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sub := &domain.Subscription{ID: uuid.New(), TenantID: uuid.New()}
	if got := doGetSubHistory(sub, uuid.New(), sub.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant subscription history: got %d want 404", got)
	}
}

// --- GET /v1/invoices/:id/status-history ----------------------------------

func doGetInvoiceStatusHistory(h *SubscriptionHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/invoices/"+id+"/status-history", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetInvoiceStatusHistory(c)
	return w
}

func TestGetInvoiceStatusHistoryOwnedReturnsHistoryArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	inv := &domain.Invoice{ID: uuid.New(), TenantID: tenantID}
	// No history reader wired → an owned invoice yields an empty (non-nil)
	// timeline, 200 — never 404.
	w := doGetInvoiceStatusHistory(newInvoiceGetHandler(inv), tenantID, inv.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("owned invoice status-history: got %d want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"history"`) {
		t.Fatalf("expected a history array, got %s", w.Body.String())
	}
}

func TestGetInvoiceStatusHistoryCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inv := &domain.Invoice{ID: uuid.New(), TenantID: uuid.New()}
	if got := doGetInvoiceStatusHistory(newInvoiceGetHandler(inv), uuid.New(), inv.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant status-history: got %d want 404", got)
	}
}

func TestGetInvoiceJournalBadIdIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if got := doGetInvoiceJournal(newInvoiceGetHandler(nil), uuid.New(), "nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad id: got %d want 400", got)
	}
}

// --- GET /v1/invoices/:id/payment-attempts --------------------------------

func doGetInvoiceAttempts(h *SubscriptionHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/invoices/"+id+"/payment-attempts", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetInvoicePaymentAttempts(c)
	return w
}

func TestGetInvoiceAttemptsOwnedReturnsArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	inv := &domain.Invoice{ID: uuid.New(), TenantID: tenantID}
	w := doGetInvoiceAttempts(newInvoiceGetHandler(inv), tenantID, inv.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("owned invoice attempts: got %d want 200", w.Code)
	}
	// Existing invoice → an attempts array (empty here — no lister wired), not 404.
	if !strings.Contains(w.Body.String(), `"attempts"`) {
		t.Fatalf("expected an attempts array, got %s", w.Body.String())
	}
}

func TestGetInvoiceAttemptsCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inv := &domain.Invoice{ID: uuid.New(), TenantID: uuid.New()}
	if got := doGetInvoiceAttempts(newInvoiceGetHandler(inv), uuid.New(), inv.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant attempts: got %d want 404 (flat)", got)
	}
}

func TestGetInvoiceAttemptsBadIdIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if got := doGetInvoiceAttempts(newInvoiceGetHandler(nil), uuid.New(), "nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad id: got %d want 400", got)
	}
}

// --- GET /v1/payment-attempts (payments log) -------------------------------

type mockAttemptLister struct {
	items  []domain.PaymentAttemptListItem
	total  int
	detail *domain.PaymentAttemptDetail
}

func (m *mockAttemptLister) ListByInvoice(_ context.Context, _, _ uuid.UUID) ([]*domain.PaymentAttempt, error) {
	return nil, nil
}
func (m *mockAttemptLister) List(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]domain.PaymentAttemptListItem, int, error) {
	return m.items, m.total, nil
}

// GetByID mirrors the real repo's tenant scoping (WHERE tenant_id=$1 AND id=$2):
// a foreign tenant, or an unknown id, resolves to nothing.
func (m *mockAttemptLister) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.PaymentAttemptDetail, error) {
	if m.detail != nil && m.detail.ID == id && m.detail.TenantID == tenantID {
		return m.detail, nil
	}
	return nil, nil
}

func TestListPaymentAttemptsReturnsPagedLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewSubscriptionService(nil, &oneInvoiceRepo{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	svc.SetPaymentAttemptLister(&mockAttemptLister{
		items: []domain.PaymentAttemptListItem{{
			PaymentAttempt: domain.PaymentAttempt{ID: uuid.New(), Status: domain.PaymentAttemptFailed, Amount: 5000},
			InvoiceNumber:  "INV-9",
		}},
		total: 1,
	})
	h := NewSubscriptionHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/payment-attempts?status=failed", nil)
	c.Set("tenant_id", uuid.New())
	h.ListPaymentAttempts(c)

	if w.Code != http.StatusOK {
		t.Fatalf("payments log: got %d want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "INV-9") || !strings.Contains(body, `"total":1`) {
		t.Fatalf("expected the attempt + a pagination total, got %s", body)
	}
}

// --- GET /v1/subscriptions/:id/financial-summary ---------------------------

type planByIDRepo struct {
	port.PlanRepository
	plan *domain.Plan
}

func (r *planByIDRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Plan, error) {
	if r.plan != nil && r.plan.ID == id {
		return r.plan, nil
	}
	return nil, nil
}

type subSummaryInvoiceRepo struct {
	port.InvoiceRepository
	rows []domain.CustomerFinancialSummaryCurrency
}

func (r *subSummaryInvoiceRepo) GetSubscriptionFinancialSummary(_ context.Context, _, _ uuid.UUID) ([]domain.CustomerFinancialSummaryCurrency, error) {
	return r.rows, nil
}

func newSubFinSummaryHandler(sub *domain.Subscription, plan *domain.Plan) *SubscriptionHandler {
	svc := service.NewSubscriptionService(
		&oneSubscriptionRepo{sub: sub}, &subSummaryInvoiceRepo{}, &planByIDRepo{plan: plan},
		nil, nil, nil, nil, nil, nil, nil, nil, nil)
	return NewSubscriptionHandler(svc)
}

func planWith(amount int64, unit domain.IntervalUnit, count int, currency string) *domain.Plan {
	return &domain.Plan{
		ID: uuid.New(), IntervalUnit: unit, IntervalCount: count,
		Prices: []domain.Price{{Currency: currency, Amount: amount}},
	}
}

func doGetSubFinSummary(h *SubscriptionHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/subscriptions/"+id+"/financial-summary", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetSubscriptionFinancialSummary(c)
	return w
}

func TestSubFinancialSummaryMonthlyActiveMRR(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: plan.ID, Status: domain.SubscriptionStatusActive}
	w := doGetSubFinSummary(newSubFinSummaryHandler(sub, plan), tenantID, sub.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("owned summary: got %d want 200", w.Code)
	}
	body := w.Body.String()
	// Monthly plan → MRR == list price; the subscription will renew → next invoice.
	for _, want := range []string{`"mrr":200000`, `"recurring_amount":200000`, `"currency":"USD"`, `"next_invoice_date"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("summary missing %s: %s", want, body)
		}
	}
}

func TestSubFinancialSummaryAnnualNormalizesMRR(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	// 1,200,000/yr normalizes to 100,000/mo — MRR must not equal the list price.
	plan := planWith(1200000, domain.IntervalYear, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: plan.ID, Status: domain.SubscriptionStatusActive}
	body := doGetSubFinSummary(newSubFinSummaryHandler(sub, plan), tenantID, sub.ID.String()).Body.String()
	if !strings.Contains(body, `"mrr":100000`) || !strings.Contains(body, `"recurring_amount":1200000`) {
		t.Fatalf("annual MRR not normalized to monthly: %s", body)
	}
}

func TestSubFinancialSummaryCanceledHasZeroMRRNoNextInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: plan.ID, Status: domain.SubscriptionStatusCanceled}
	body := doGetSubFinSummary(newSubFinSummaryHandler(sub, plan), tenantID, sub.ID.String()).Body.String()
	// Non-active → MRR 0 (by the same rule GetMRR uses) and no forecast next invoice.
	if !strings.Contains(body, `"mrr":0`) {
		t.Fatalf("canceled subscription must have MRR 0: %s", body)
	}
	if strings.Contains(body, `"next_invoice_date"`) {
		t.Fatalf("canceled subscription must not forecast a next invoice: %s", body)
	}
}

func TestSubFinancialSummaryPausedHasZeroMRR(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: plan.ID, Status: domain.SubscriptionStatusPaused}
	body := doGetSubFinSummary(newSubFinSummaryHandler(sub, plan), tenantID, sub.ID.String()).Body.String()
	if !strings.Contains(body, `"mrr":0`) || strings.Contains(body, `"next_invoice_date"`) {
		t.Fatalf("paused subscription: want MRR 0 and no next invoice, got %s", body)
	}
}

func TestSubFinancialSummaryCancelAtPeriodEndNoNextInvoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	// Active but set to cancel at period end → still MRR>0 today, but no renewal.
	sub := &domain.Subscription{ID: uuid.New(), TenantID: tenantID, PlanID: plan.ID, Status: domain.SubscriptionStatusActive, CancelAtPeriodEnd: true}
	body := doGetSubFinSummary(newSubFinSummaryHandler(sub, plan), tenantID, sub.ID.String()).Body.String()
	if !strings.Contains(body, `"mrr":200000`) {
		t.Fatalf("active sub keeps MRR even when cancel-at-period-end: %s", body)
	}
	if strings.Contains(body, `"next_invoice_date"`) {
		t.Fatalf("cancel-at-period-end must not forecast a next invoice: %s", body)
	}
}

func TestSubFinancialSummaryCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	plan := planWith(200000, domain.IntervalMonth, 1, "USD")
	sub := &domain.Subscription{ID: uuid.New(), TenantID: uuid.New(), PlanID: plan.ID, Status: domain.SubscriptionStatusActive}
	if got := doGetSubFinSummary(newSubFinSummaryHandler(sub, plan), uuid.New(), sub.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant summary: got %d want 404 (flat)", got)
	}
}

func TestSubFinancialSummaryBadIdIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if got := doGetSubFinSummary(newSubFinSummaryHandler(nil, nil), uuid.New(), "nope").Code; got != http.StatusBadRequest {
		t.Fatalf("bad id: got %d want 400", got)
	}
}

// --- GET /v1/payment-attempts/:id (payment object) -------------------------

func newPaymentGetHandler(detail *domain.PaymentAttemptDetail) *SubscriptionHandler {
	svc := service.NewSubscriptionService(nil, &oneInvoiceRepo{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	svc.SetPaymentAttemptLister(&mockAttemptLister{detail: detail})
	return NewSubscriptionHandler(svc)
}

func doGetPayment(h *SubscriptionHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/payment-attempts/"+id, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetPaymentAttempt(c)
	return w
}

func sampleFailedPayment(tenantID uuid.UUID) *domain.PaymentAttemptDetail {
	return &domain.PaymentAttemptDetail{
		PaymentAttempt: domain.PaymentAttempt{
			ID: uuid.New(), TenantID: tenantID, InvoiceID: uuid.New(),
			Gateway: "stripe", Method: "card", Status: domain.PaymentAttemptFailed,
			FailureCode: "card_declined", Amount: 5000,
		},
		InvoiceNumber: "INV-9", Currency: "USD", CustomerID: uuid.New(),
	}
}

func TestGetPaymentReturnsOwnedRowWithContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	pa := sampleFailedPayment(tenantID)
	w := doGetPayment(newPaymentGetHandler(pa), tenantID, pa.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("owned payment: got %d want 200", w.Code)
	}
	body := w.Body.String()
	// The response resolves the invoice-level context (customer, invoice, currency)
	// and preserves the failure state — the point of the addressable object.
	for _, want := range []string{`"customer_id"`, `"invoice_number":"INV-9"`, `"currency":"USD"`, `"failure_code":"card_declined"`, `"status":"failed"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("payment body missing %s: %s", want, body)
		}
	}
}

func TestGetPaymentCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pa := sampleFailedPayment(uuid.New())
	if got := doGetPayment(newPaymentGetHandler(pa), uuid.New(), pa.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant payment: got %d want 404 (flat — never confirm existence)", got)
	}
}

func TestGetPaymentUnknownIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	if got := doGetPayment(newPaymentGetHandler(sampleFailedPayment(tenantID)), tenantID, uuid.New().String()).Code; got != http.StatusNotFound {
		t.Fatalf("unknown payment id: got %d want 404", got)
	}
}

func TestGetPaymentBadIdIs400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if got := doGetPayment(newPaymentGetHandler(nil), uuid.New(), "nope").Code; got != http.StatusBadRequest {
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

// --- GET /v1/disputes/:id ---------------------------------------------------

type oneDisputeRepo struct {
	port.DisputeRepository
	d *domain.InvoiceDispute
}

func (r *oneDisputeRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.InvoiceDispute, error) {
	if r.d != nil && r.d.ID == id {
		return r.d, nil
	}
	return nil, nil
}

func doGetDispute(d *domain.InvoiceDispute, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	h := NewDisputeHandler(service.NewDisputeService(&oneDisputeRepo{d: d}))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/disputes/"+id, nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetDispute(c)
	return w
}

func TestGetDisputeReturnsOwnedRow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	d := &domain.InvoiceDispute{ID: uuid.New(), TenantID: tenantID, InvoiceID: uuid.New(), Status: domain.DisputeStatusOpen}
	if got := doGetDispute(d, tenantID, d.ID.String()).Code; got != http.StatusOK {
		t.Fatalf("owned dispute: got %d want 200", got)
	}
}

func TestGetDisputeCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	d := &domain.InvoiceDispute{ID: uuid.New(), TenantID: uuid.New()}
	if got := doGetDispute(d, uuid.New(), d.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant dispute: got %d want 404", got)
	}
}

func doGetCreditNoteJournal(h *CreditNoteHandler, tenantID uuid.UUID, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/credit-notes/"+id+"/journal-entries", nil)
	c.Params = gin.Params{{Key: "id", Value: id}}
	c.Set("tenant_id", tenantID)
	h.GetCreditNoteJournalEntries(c)
	return w
}

func TestGetCreditNoteJournalOwnedReturnsEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantID := uuid.New()
	cn := &domain.CreditNote{ID: uuid.New(), TenantID: tenantID, Amount: 500, Currency: "USD"}
	// No ledger wired → an owned note yields an empty (non-nil) journal, 200.
	w := doGetCreditNoteJournal(newCreditNoteGetHandler(cn), tenantID, cn.ID.String())
	if w.Code != http.StatusOK {
		t.Fatalf("owned credit-note journal: got %d want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "credit_note_id") || !strings.Contains(w.Body.String(), "entries") {
		t.Fatalf("expected the journal envelope, got %s", w.Body.String())
	}
}

func TestGetCreditNoteJournalCrossTenantIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cn := &domain.CreditNote{ID: uuid.New(), TenantID: uuid.New()}
	if got := doGetCreditNoteJournal(newCreditNoteGetHandler(cn), uuid.New(), cn.ID.String()).Code; got != http.StatusNotFound {
		t.Fatalf("cross-tenant journal: got %d want 404", got)
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
