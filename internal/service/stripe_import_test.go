package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	stripeimport "github.com/recurso-dev/recurso/internal/importer/stripe"
)

// --- fakes -----------------------------------------------------------------

type fakeImportCustomers struct {
	existing  []*domain.Customer
	created   []CreateCustomerInput
	failEmail string // CreateCustomer fails for this email
}

func (f *fakeImportCustomers) ListCustomers(_ context.Context, _ uuid.UUID, _ domain.CustomerFilter) ([]*domain.Customer, error) {
	return f.existing, nil
}
func (f *fakeImportCustomers) CreateCustomer(_ context.Context, in CreateCustomerInput) (*domain.Customer, error) {
	if f.failEmail != "" && in.Email == f.failEmail {
		return nil, errors.New("create customer failed")
	}
	f.created = append(f.created, in)
	c := &domain.Customer{ID: uuid.New(), Email: in.Email}
	f.existing = append(f.existing, c) // mirror a real DB: it's now present
	return c, nil
}

type fakeImportCatalog struct {
	existing []*domain.Plan
	created  []CreatePlanInput
}

func (f *fakeImportCatalog) ListPlans(_ context.Context, _ uuid.UUID, _ domain.PlanFilter) ([]*domain.Plan, error) {
	return f.existing, nil
}
func (f *fakeImportCatalog) CreatePlan(_ context.Context, in CreatePlanInput) (*domain.Plan, error) {
	f.created = append(f.created, in)
	p := &domain.Plan{ID: uuid.New(), Code: in.Code}
	f.existing = append(f.existing, p)
	return p, nil
}

type fakeRefRepo struct{ ids map[string]bool }

func newFakeRefRepo() *fakeRefRepo { return &fakeRefRepo{ids: map[string]bool{}} }
func (f *fakeRefRepo) Create(_ context.Context, ref *domain.ImportExternalRef) error {
	if f.ids[ref.ExternalID] {
		return domain.ErrDuplicateImportRef
	}
	f.ids[ref.ExternalID] = true
	return nil
}
func (f *fakeRefRepo) ListExternalIDs(_ context.Context, _ uuid.UUID, _ string) (map[string]bool, error) {
	out := map[string]bool{}
	for k := range f.ids {
		out[k] = true
	}
	return out, nil
}

func importFixture() *stripeimport.Export {
	return &stripeimport.Export{
		Customers: []stripeimport.Customer{
			{ID: "cus_a", Email: "a@acme.com", Name: "A"},
			{ID: "cus_b", Email: "b@acme.com", Name: "B"},
		},
		Products: []stripeimport.Product{{ID: "prod_1", Name: "Pro"}},
		Prices: []stripeimport.Price{
			{ID: "price_m", Product: "prod_1", UnitAmount: 4900, Currency: "usd", Recurring: &stripeimport.Recurring{Interval: "month"}},
		},
	}
}

func newImportSvc() (*StripeImportService, *fakeImportCustomers, *fakeImportCatalog, *fakeRefRepo) {
	cust := &fakeImportCustomers{}
	cat := &fakeImportCatalog{}
	refs := newFakeRefRepo()
	return NewStripeImportService(cust, cat, refs), cust, cat, refs
}

// --- tests -----------------------------------------------------------------

func TestCommit_CreatesCustomersAndPlansAndRecordsRefs(t *testing.T) {
	svc, cust, cat, refs := newImportSvc()
	res, err := svc.Commit(context.Background(), uuid.New(), importFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Created["customer"] != 2 {
		t.Errorf("want 2 customers created, got %d", res.Created["customer"])
	}
	if res.Created["plan"] != 1 {
		t.Errorf("want 1 plan created, got %d", res.Created["plan"])
	}
	if len(res.Failures) != 0 {
		t.Errorf("unexpected failures: %+v", res.Failures)
	}
	if len(cust.created) != 2 || len(cat.created) != 1 {
		t.Errorf("create calls wrong: customers=%d plans=%d", len(cust.created), len(cat.created))
	}
	// One ref per created object (2 customers + 1 plan).
	if len(refs.ids) != 3 {
		t.Errorf("want 3 idempotency refs, got %d", len(refs.ids))
	}
	// Plan maps money 1:1 and code is deterministic.
	if cat.created[0].Amount != 4900 || cat.created[0].Code != "stripe_price_m" {
		t.Errorf("plan mapping wrong: %+v", cat.created[0])
	}
}

func TestCommit_IsIdempotentOnReRun(t *testing.T) {
	svc, _, _, _ := newImportSvc()
	tenant := uuid.New()
	exp := importFixture()

	if _, err := svc.Commit(context.Background(), tenant, exp); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	res2, err := svc.Commit(context.Background(), tenant, exp)
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}
	if len(res2.Created) != 0 {
		t.Fatalf("re-run should create nothing, got %v", res2.Created)
	}
	// Every item should show as already-imported in the recomputed plan.
	for _, it := range res2.Plan.Items {
		if it.Action != stripeimport.ActionSkipImported {
			t.Errorf("%s: want skip_already_imported on re-run, got %s", it.StripeID, it.Action)
		}
	}
}

func TestCommit_LinksExistingCustomerByEmailInsteadOfCreating(t *testing.T) {
	svc, cust, _, _ := newImportSvc()
	cust.existing = []*domain.Customer{{ID: uuid.New(), Email: "a@acme.com"}}

	res, err := svc.Commit(context.Background(), uuid.New(), importFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	// a@acme.com already exists → linked, not created; only b@acme.com is created.
	if res.Created["customer"] != 1 {
		t.Errorf("want 1 customer created (b only), got %d", res.Created["customer"])
	}
	for _, in := range cust.created {
		if in.Email == "a@acme.com" {
			t.Error("existing customer must not be re-created")
		}
	}
}

func TestCommit_PerObjectFailureIsReportedNotFatal(t *testing.T) {
	svc, cust, _, refs := newImportSvc()
	cust.failEmail = "a@acme.com"

	res, err := svc.Commit(context.Background(), uuid.New(), importFixture())
	if err != nil {
		t.Fatalf("commit should not hard-fail on a per-object error: %v", err)
	}
	if len(res.Failures) != 1 || res.Failures[0].StripeID != "cus_a" {
		t.Errorf("want 1 failure for cus_a, got %+v", res.Failures)
	}
	// The other customer and the plan still import.
	if res.Created["customer"] != 1 || res.Created["plan"] != 1 {
		t.Errorf("survivors not created: %v", res.Created)
	}
	// No ref recorded for the failed customer.
	if refs.ids["cus_a"] {
		t.Error("failed create must not record an idempotency ref")
	}
}

func TestPreview_NoSideEffects(t *testing.T) {
	svc, cust, cat, refs := newImportSvc()
	plan, err := svc.Preview(context.Background(), uuid.New(), importFixture())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if plan.Summary["customer.create"] != 2 || plan.Summary["plan.create"] != 1 {
		t.Errorf("preview summary wrong: %v", plan.Summary)
	}
	if len(cust.created) != 0 || len(cat.created) != 0 || len(refs.ids) != 0 {
		t.Error("preview must not create anything")
	}
}
