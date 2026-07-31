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
	f.existing = append(f.existing, c)
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

type fakeImportSubs struct{ created []*domain.Subscription }

func (f *fakeImportSubs) Create(_ context.Context, sub *domain.Subscription) error {
	f.created = append(f.created, sub)
	return nil
}

type fakeRefRepo struct {
	refs []*domain.ImportExternalRef
	// conflictOn makes Create return ErrDuplicateImportRef for these external ids
	// WITHOUT them appearing in List* — simulating a concurrent commit that
	// claimed the ref after loadState snapshotted but before this commit's create.
	conflictOn map[string]bool
}

func newFakeRefRepo() *fakeRefRepo { return &fakeRefRepo{} }
func (f *fakeRefRepo) Create(_ context.Context, ref *domain.ImportExternalRef) error {
	if f.conflictOn[ref.ExternalID] {
		return domain.ErrDuplicateImportRef
	}
	for _, r := range f.refs {
		if r.ExternalID == ref.ExternalID {
			return domain.ErrDuplicateImportRef
		}
	}
	cp := *ref
	f.refs = append(f.refs, &cp)
	return nil
}
func (f *fakeRefRepo) ListExternalIDs(_ context.Context, _ uuid.UUID, _ string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, r := range f.refs {
		out[r.ExternalID] = true
	}
	return out, nil
}
func (f *fakeRefRepo) ListRefs(_ context.Context, _ uuid.UUID, _ string) ([]*domain.ImportExternalRef, error) {
	return f.refs, nil
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
		Subscriptions: []stripeimport.Subscription{
			{ID: "sub_a", Customer: "cus_a", Status: "active", CurrentPeriodStart: 1_700_000_000, CurrentPeriodEnd: 1_702_600_000,
				Items: stripeimport.SubscriptionItems{Data: []stripeimport.SubscriptionItem{{Price: stripeimport.Price{ID: "price_m"}}}}},
		},
	}
}

func newImportSvc() (*StripeImportService, *fakeImportCustomers, *fakeImportCatalog, *fakeImportSubs, *fakeRefRepo) {
	cust := &fakeImportCustomers{}
	cat := &fakeImportCatalog{}
	subs := &fakeImportSubs{}
	refs := newFakeRefRepo()
	return NewStripeImportService(cust, cat, subs, refs), cust, cat, subs, refs
}

// --- tests -----------------------------------------------------------------

func TestCommit_CreatesCustomersPlansAndSubscriptions(t *testing.T) {
	svc, cust, cat, subs, refs := newImportSvc()
	res, err := svc.Commit(context.Background(), uuid.New(), importFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Created["customer"] != 2 || res.Created["plan"] != 1 || res.Created["subscription"] != 1 {
		t.Fatalf("created counts wrong: %v (failures: %+v)", res.Created, res.Failures)
	}
	if len(cust.created) != 2 || len(cat.created) != 1 || len(subs.created) != 1 {
		t.Errorf("create calls wrong: customers=%d plans=%d subs=%d", len(cust.created), len(cat.created), len(subs.created))
	}
	// One ref per created object (2 customers + 1 plan + 1 subscription).
	if len(refs.refs) != 4 {
		t.Errorf("want 4 idempotency refs, got %d", len(refs.refs))
	}
	sub := subs.created[0]
	if sub.CustomerID == uuid.Nil || sub.PlanID == uuid.Nil {
		t.Error("subscription customer/plan not resolved")
	}
	if sub.StripeSubscriptionID != "sub_a" {
		t.Errorf("stripe sub id not carried: %q", sub.StripeSubscriptionID)
	}
	if sub.Status != domain.SubscriptionStatusActive {
		t.Errorf("status = %s, want active", sub.Status)
	}
	// Period + anchor come straight from Stripe (no re-billing of the cycle).
	if sub.CurrentPeriodEnd.Unix() != 1_702_600_000 || sub.BillingAnchor.Unix() != 1_702_600_000 {
		t.Errorf("period/anchor not taken from Stripe: end=%d anchor=%d", sub.CurrentPeriodEnd.Unix(), sub.BillingAnchor.Unix())
	}
}

func TestCommit_SubscriptionLinksToExistingCustomer(t *testing.T) {
	svc, cust, _, subs, _ := newImportSvc()
	existingID := uuid.New()
	cust.existing = []*domain.Customer{{ID: existingID, Email: "a@acme.com"}}

	res, err := svc.Commit(context.Background(), uuid.New(), importFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if len(subs.created) != 1 {
		t.Fatalf("want 1 subscription, got %d (failures %+v)", len(subs.created), res.Failures)
	}
	if subs.created[0].CustomerID != existingID {
		t.Errorf("subscription should point at the existing (linked) customer id")
	}
}

func TestCommit_IsIdempotentOnReRun(t *testing.T) {
	svc, _, _, subs, _ := newImportSvc()
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
	if len(subs.created) != 1 {
		t.Errorf("subscription created twice across runs: %d", len(subs.created))
	}
	for _, it := range res2.Plan.Items {
		if it.Action != stripeimport.ActionSkipImported {
			t.Errorf("%s: want skip_already_imported on re-run, got %s", it.StripeID, it.Action)
		}
	}
}

func TestCommit_PerObjectFailureIsReportedNotFatal(t *testing.T) {
	svc, cust, _, subs, refs := newImportSvc()
	cust.failEmail = "a@acme.com"

	res, err := svc.Commit(context.Background(), uuid.New(), importFixture())
	if err != nil {
		t.Fatalf("commit should not hard-fail on a per-object error: %v", err)
	}
	// cus_a fails; its subscription can no longer resolve its customer → also fails.
	if res.Created["customer"] != 1 || res.Created["plan"] != 1 {
		t.Errorf("survivors not created: %v", res.Created)
	}
	if res.Created["subscription"] != 0 || len(subs.created) != 0 {
		t.Errorf("subscription on the failed customer must not import: %v", res.Created)
	}
	for _, r := range refs.refs {
		if r.ExternalID == "cus_a" {
			t.Error("failed create must not record an idempotency ref")
		}
	}
}

func TestPreview_NoSideEffects(t *testing.T) {
	svc, cust, cat, subs, refs := newImportSvc()
	plan, err := svc.Preview(context.Background(), uuid.New(), importFixture())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if plan.Summary["customer.create"] != 2 || plan.Summary["plan.create"] != 1 || plan.Summary["subscription.create"] != 1 {
		t.Errorf("preview summary wrong: %v", plan.Summary)
	}
	if len(cust.created) != 0 || len(cat.created) != 0 || len(subs.created) != 0 || len(refs.refs) != 0 {
		t.Error("preview must not create anything")
	}
}
