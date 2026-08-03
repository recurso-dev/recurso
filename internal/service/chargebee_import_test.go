package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/importer/chargebee"
)

func chargebeeFixture() *chargebee.Export {
	return &chargebee.Export{
		Customers: []chargebee.Customer{
			{ID: "cb_a", Email: "a@acme.com", FirstName: "A"},
			{ID: "cb_b", Email: "b@acme.com", FirstName: "B"},
		},
		Plans: []chargebee.Plan{
			{ID: "pro", Name: "Pro", Price: 4900, Period: 1, PeriodUnit: "month", CurrencyCode: "usd", Status: "active"},
		},
		Subscriptions: []chargebee.Subscription{
			{ID: "sub_a", CustomerID: "cb_a", PlanID: "pro", Status: "active", CurrentTermStart: 1_700_000_000, CurrentTermEnd: 1_702_600_000},
		},
	}
}

func newChargebeeSvc() (*ChargebeeImportService, *fakeImportCustomers, *fakeImportCatalog, *fakeImportSubs, *fakeRefRepo) {
	cust := &fakeImportCustomers{}
	cat := &fakeImportCatalog{}
	subs := &fakeImportSubs{}
	refs := newFakeRefRepo()
	return NewChargebeeImportService(cust, cat, subs, refs), cust, cat, subs, refs
}

func TestChargebeeCommit_CreatesCustomersPlansAndSubscriptions(t *testing.T) {
	svc, cust, cat, subs, refs := newChargebeeSvc()
	res, err := svc.Commit(context.Background(), uuid.New(), chargebeeFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Created["customer"] != 2 || res.Created["plan"] != 1 || res.Created["subscription"] != 1 {
		t.Fatalf("created counts wrong: %v (failures %+v)", res.Created, res.Failures)
	}
	if len(cust.created) != 2 || len(cat.created) != 1 || len(subs.created) != 1 {
		t.Errorf("create calls wrong: customers=%d plans=%d subs=%d", len(cust.created), len(cat.created), len(subs.created))
	}
	if len(refs.refs) != 4 {
		t.Errorf("want 4 idempotency refs, got %d", len(refs.refs))
	}
	sub := subs.created[0]
	if sub.CustomerID == uuid.Nil || sub.PlanID == uuid.Nil {
		t.Error("subscription customer/plan not resolved")
	}
	if sub.Status != domain.SubscriptionStatusActive {
		t.Errorf("status = %s, want active", sub.Status)
	}
	if sub.CurrentPeriodEnd.Unix() != 1_702_600_000 || sub.BillingAnchor.Unix() != 1_702_600_000 {
		t.Errorf("period/anchor not from Chargebee term end")
	}
	// Plan maps money 1:1, deterministic code.
	if cat.created[0].Amount != 4900 || cat.created[0].Code != "chargebee_pro" {
		t.Errorf("plan mapping wrong: %+v", cat.created[0])
	}
}

// Regression: a concurrent (or retried) commit must not double-insert a
// subscription. Claim-first records the idempotency ref before the create, so a
// ref-claim conflict at commit time skips the create instead of inserting a
// second subscription that would double-bill at renewal.
func TestChargebeeCommit_ClaimFirstSkipsDuplicateSubscription(t *testing.T) {
	svc, _, _, subs, refs := newChargebeeSvc()
	// loadState sees no subscription ref (so BuildPlan decides to create sub_a),
	// but the ref-claim conflicts — as if another commit claimed it in between.
	refs.conflictOn = map[string]bool{"sub_a": true}

	res, err := svc.Commit(context.Background(), uuid.New(), chargebeeFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if len(subs.created) != 0 {
		t.Errorf("claim-first should skip the create on a duplicate ref; got %d subscriptions created (double-insert)", len(subs.created))
	}
	if res.Created["subscription"] != 0 {
		t.Errorf("subscription must not be counted as created; got %d", res.Created["subscription"])
	}
	// Customers and plans still import normally — only the conflicting sub is skipped.
	if res.Created["customer"] != 2 || res.Created["plan"] != 1 {
		t.Errorf("customers/plans should still import: %v", res.Created)
	}
}

func TestChargebeeCommit_LinksExistingCustomer(t *testing.T) {
	svc, cust, _, subs, _ := newChargebeeSvc()
	existingID := uuid.New()
	cust.existing = []*domain.Customer{{ID: existingID, Email: "a@acme.com"}}

	res, err := svc.Commit(context.Background(), uuid.New(), chargebeeFixture())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Created["customer"] != 1 { // only cb_b created; cb_a linked
		t.Errorf("want 1 customer created, got %d", res.Created["customer"])
	}
	if len(subs.created) != 1 || subs.created[0].CustomerID != existingID {
		t.Errorf("subscription should resolve to the existing linked customer")
	}
}

func TestChargebeeCommit_IsIdempotent(t *testing.T) {
	svc, _, _, subs, _ := newChargebeeSvc()
	tenant := uuid.New()
	exp := chargebeeFixture()
	if _, err := svc.Commit(context.Background(), tenant, exp); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	res2, err := svc.Commit(context.Background(), tenant, exp)
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}
	if len(res2.Created) != 0 {
		t.Errorf("re-run should create nothing, got %v", res2.Created)
	}
	if len(subs.created) != 1 {
		t.Errorf("subscription created twice across runs: %d", len(subs.created))
	}
}

func TestChargebeePreview_NoSideEffects(t *testing.T) {
	svc, cust, cat, subs, refs := newChargebeeSvc()
	plan, err := svc.Preview(context.Background(), uuid.New(), chargebeeFixture())
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

// ---- Compare gate (mirrors the Stripe compare tests) ----

func chargebeeCompareFixture(t *testing.T, tenantID uuid.UUID) (*ChargebeeImportService, *chargebee.Export, *fakeSubReader) {
	t.Helper()
	exp := chargebeeFixture()

	custA := &domain.Customer{ID: uuid.New(), Email: "a@acme.com"}
	custB := &domain.Customer{ID: uuid.New(), Email: "b@acme.com"}
	plan := &domain.Plan{
		ID: uuid.New(), Code: "chargebee_pro", IntervalUnit: domain.IntervalMonth, IntervalCount: 1,
		Prices: []domain.Price{{Currency: "USD", Amount: 4900}},
	}
	sub := exp.Subscriptions[0]
	rec := &domain.Subscription{
		ID: uuid.New(), TenantID: tenantID, CustomerID: custA.ID, PlanID: plan.ID,
		Status:           "active",
		CurrentPeriodEnd: time.Unix(sub.CurrentTermEnd, 0).UTC(),
	}

	refs := newFakeRefRepo()
	refs.refs = []*domain.ImportExternalRef{
		{Kind: domain.ImportKindCustomer, ExternalID: "cb_a", RecursoID: custA.ID},
		{Kind: domain.ImportKindCustomer, ExternalID: "cb_b", RecursoID: custB.ID},
		{Kind: domain.ImportKindPlan, ExternalID: "pro", RecursoID: plan.ID},
		{Kind: domain.ImportKindSubscription, ExternalID: sub.ID, RecursoID: rec.ID},
	}
	reader := &fakeSubReader{subs: map[uuid.UUID]*domain.Subscription{rec.ID: rec}}
	svc := NewChargebeeImportService(
		&fakeImportCustomers{existing: []*domain.Customer{custA, custB}},
		&fakeImportCatalog{existing: []*domain.Plan{plan}},
		&fakeImportSubs{}, refs,
	)
	svc.SetSubscriptionReader(reader)
	return svc, exp, reader
}

func TestChargebeeCompare_Ready(t *testing.T) {
	tenantID := uuid.New()
	svc, exp, _ := chargebeeCompareFixture(t, tenantID)
	rep, err := svc.Compare(context.Background(), tenantID, exp)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Ready {
		t.Fatalf("expected ready, issues: %+v", rep.Issues)
	}
	if rep.Customers.Matched != 2 || rep.Plans.Matched != 1 || rep.Subscriptions.Matched != 1 {
		t.Fatalf("counts: %+v %+v %+v", rep.Customers, rep.Plans, rep.Subscriptions)
	}
}

func TestChargebeeCompare_FlagsDriftAndNonRenewing(t *testing.T) {
	tenantID := uuid.New()
	svc, exp, reader := chargebeeCompareFixture(t, tenantID)
	// Chargebee says non_renewing; the imported sub forgot CancelAtPeriodEnd,
	// and its period end drifted 3 days.
	exp.Subscriptions[0].Status = "non_renewing"
	for _, rec := range reader.subs {
		rec.CurrentPeriodEnd = rec.CurrentPeriodEnd.Add(72 * time.Hour)
	}

	rep, err := svc.Compare(context.Background(), tenantID, exp)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Ready {
		t.Fatal("expected NOT ready")
	}
	var cancel, drift bool
	for _, is := range rep.Issues {
		if is.Field == "cancel_at_period_end" && is.Source == "true" {
			cancel = true
		}
		if is.Field == "current_period_end" {
			drift = true
		}
	}
	if !cancel || !drift {
		t.Fatalf("missing issues (cancel=%t drift=%t): %+v", cancel, drift, rep.Issues)
	}
}
