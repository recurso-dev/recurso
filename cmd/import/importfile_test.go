package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func validFile() *ImportFile {
	return &ImportFile{
		Plans: []PlanInput{
			{Code: "PRO-USD", Name: "Pro", Amount: 2900, Currency: "USD", IntervalUnit: "month", IntervalCount: 1},
		},
		Customers: []CustomerInput{
			{Email: "jane@example.com", Name: "Jane User", Country: "US"},
		},
		Subscriptions: []SubscriptionInput{
			{
				ExternalID:         "sub_ext_1",
				CustomerEmail:      "jane@example.com",
				PlanCode:           "PRO-USD",
				Status:             "active",
				CurrentPeriodStart: "2026-06-15T00:00:00Z",
				CurrentPeriodEnd:   "2026-07-15T00:00:00Z",
			},
		},
	}
}

func TestValidate_ValidFile(t *testing.T) {
	if errs := validFile().Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidate_CatchesProblems(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ImportFile)
		wantSub string
	}{
		{"missing plan code", func(f *ImportFile) { f.Plans[0].Code = "" }, "code is required"},
		{"bad currency", func(f *ImportFile) { f.Plans[0].Currency = "USDT" }, "3-letter"},
		{"bad interval", func(f *ImportFile) { f.Plans[0].IntervalUnit = "fortnight" }, "interval_unit"},
		{"negative amount", func(f *ImportFile) { f.Plans[0].Amount = -1 }, "amount"},
		{"bad email", func(f *ImportFile) { f.Customers[0].Email = "nope" }, "valid email"},
		{"missing customer name", func(f *ImportFile) { f.Customers[0].Name = "" }, "name is required"},
		{"missing external id", func(f *ImportFile) { f.Subscriptions[0].ExternalID = "" }, "external_id"},
		{"unknown customer", func(f *ImportFile) { f.Subscriptions[0].CustomerEmail = "ghost@example.com" }, "not present in customers"},
		{"unknown plan", func(f *ImportFile) { f.Subscriptions[0].PlanCode = "NOPE" }, "not present in plans"},
		{"bad status", func(f *ImportFile) { f.Subscriptions[0].Status = "zombie" }, "status"},
		{"bad period start", func(f *ImportFile) { f.Subscriptions[0].CurrentPeriodStart = "June 15" }, "RFC3339"},
		{"end before start", func(f *ImportFile) { f.Subscriptions[0].CurrentPeriodEnd = "2026-05-01T00:00:00Z" }, "must be after"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := validFile()
			tc.mutate(f)
			errs := f.Validate()
			if len(errs) == 0 {
				t.Fatal("expected a validation error, got none")
			}
			found := false
			for _, e := range errs {
				if strings.Contains(e.Error(), tc.wantSub) {
					found = true
				}
			}
			if !found {
				t.Errorf("no error containing %q in %v", tc.wantSub, errs)
			}
		})
	}
}

func TestValidate_DuplicatesRejected(t *testing.T) {
	f := validFile()
	f.Customers = append(f.Customers, CustomerInput{Email: "JANE@example.com", Name: "Dup"})
	f.Subscriptions = append(f.Subscriptions, f.Subscriptions[0])
	errs := f.Validate()
	var dupEmail, dupExt bool
	for _, e := range errs {
		if strings.Contains(e.Error(), "duplicate customer email") {
			dupEmail = true
		}
		if strings.Contains(e.Error(), "duplicate external_id") {
			dupExt = true
		}
	}
	if !dupEmail || !dupExt {
		t.Errorf("expected duplicate email and external_id errors, got: %v", errs)
	}
}

func TestValidate_SubscriptionPlanMayExistInDB(t *testing.T) {
	// With no plans in the file, plan_code can't be checked locally — it is
	// resolved against the database at import time, not a validation error.
	f := validFile()
	f.Plans = nil
	if errs := f.Validate(); len(errs) != 0 {
		t.Fatalf("expected no errors when plans list is empty, got: %v", errs)
	}
}

func TestParseCustomersCSV(t *testing.T) {
	csvData := `email,name,country,state,unknown_col
jane@example.com,Jane User,US,CA,ignored
ram@example.in,Ram Gupta,IN,MH,x`
	customers, err := ParseCustomersCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(customers) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(customers))
	}
	if customers[0].Email != "jane@example.com" || customers[0].Country != "US" {
		t.Errorf("unexpected first customer: %+v", customers[0])
	}
	if customers[1].State != "MH" {
		t.Errorf("State = %q, want MH", customers[1].State)
	}
}

func TestParseSubscriptionsCSV(t *testing.T) {
	csvData := `external_id,customer_email,plan_code,status,current_period_start,current_period_end
sub_1,jane@example.com,PRO-USD,active,2026-06-15T00:00:00Z,2026-07-15T00:00:00Z`
	subs, err := ParseSubscriptionsCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 || subs[0].ExternalID != "sub_1" || subs[0].Status != "active" {
		t.Errorf("unexpected subs: %+v", subs)
	}
}

func TestParsePlansCSV_BadAmount(t *testing.T) {
	csvData := "code,name,amount,currency,interval_unit\nPRO,Pro,twenty,USD,month"
	if _, err := ParsePlansCSV(strings.NewReader(csvData)); err == nil {
		t.Fatal("expected error for non-integer amount")
	}
}

func TestParseJSON_UnknownFieldRejected(t *testing.T) {
	if _, err := ParseJSON(strings.NewReader(`{"customerz": []}`)); err == nil {
		t.Fatal("expected error for unknown top-level field")
	}
}

// fakeStore is an in-memory store for exercising -update and -cancel-missing
// without a database. It records every write so tests can assert exactly what
// was (and was not) touched.
type fakeStore struct {
	customers     map[string]uuid.UUID    // lower(email) -> id
	plans         map[string]existingPlan // code -> plan
	subscriptions map[string]uuid.UUID    // external_id -> id
	list          []existingSubscription  // returned by ListSubscriptions

	updatedCustomers []uuid.UUID
	updatedPlans     []uuid.UUID
	updatedSubs      []uuid.UUID
	canceledSubs     []uuid.UUID
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		customers:     map[string]uuid.UUID{},
		plans:         map[string]existingPlan{},
		subscriptions: map[string]uuid.UUID{},
	}
}

func (f *fakeStore) CustomerIDByEmail(_ context.Context, _ uuid.UUID, email string) (uuid.UUID, bool, error) {
	id, ok := f.customers[strings.ToLower(email)]
	return id, ok, nil
}

func (f *fakeStore) UpdateCustomer(_ context.Context, _, id uuid.UUID, _ CustomerInput) error {
	f.updatedCustomers = append(f.updatedCustomers, id)
	return nil
}

func (f *fakeStore) PlanByCode(_ context.Context, _ uuid.UUID, code string) (*existingPlan, error) {
	p, ok := f.plans[code]
	if !ok {
		return nil, nil
	}
	return &p, nil
}

func (f *fakeStore) UpdatePlan(_ context.Context, _ uuid.UUID, plan existingPlan, _ PlanInput) error {
	f.updatedPlans = append(f.updatedPlans, plan.ID)
	return nil
}

func (f *fakeStore) SubscriptionIDByExternalID(_ context.Context, _ uuid.UUID, externalID string) (uuid.UUID, bool, error) {
	id, ok := f.subscriptions[externalID]
	return id, ok, nil
}

func (f *fakeStore) UpdateSubscription(_ context.Context, _, id uuid.UUID, _ SubscriptionInput) error {
	f.updatedSubs = append(f.updatedSubs, id)
	return nil
}

func (f *fakeStore) ListSubscriptions(_ context.Context, _ uuid.UUID) ([]existingSubscription, error) {
	return f.list, nil
}

func (f *fakeStore) CancelSubscription(_ context.Context, _, id uuid.UUID, _ string) error {
	f.canceledSubs = append(f.canceledSubs, id)
	return nil
}

func TestUpdateMode_UpdatesExistingEntities(t *testing.T) {
	fs := newFakeStore()
	custID, planID, subID := uuid.New(), uuid.New(), uuid.New()
	fs.customers["jane@example.com"] = custID
	fs.plans["PRO-USD"] = existingPlan{ID: planID, Name: "Old Pro"}
	fs.subscriptions["sub_ext_1"] = subID

	file := validFile()
	im := &importer{store: fs, update: true}

	ctx := context.Background()
	im.importPlans(ctx, file.Plans)
	im.importCustomers(ctx, file.Customers)
	im.importSubscriptions(ctx, file.Subscriptions)

	if im.updated != 3 {
		t.Fatalf("updated = %d, want 3", im.updated)
	}
	if im.created != 0 || im.skipped != 0 || im.failed != 0 {
		t.Fatalf("created/skipped/failed = %d/%d/%d, want 0/0/0", im.created, im.skipped, im.failed)
	}
	if len(fs.updatedCustomers) != 1 || len(fs.updatedPlans) != 1 || len(fs.updatedSubs) != 1 {
		t.Fatalf("writes = cust:%d plan:%d sub:%d, want 1/1/1",
			len(fs.updatedCustomers), len(fs.updatedPlans), len(fs.updatedSubs))
	}
}

func TestUpdateMode_DryRunDoesNotWrite(t *testing.T) {
	fs := newFakeStore()
	fs.customers["jane@example.com"] = uuid.New()
	fs.plans["PRO-USD"] = existingPlan{ID: uuid.New(), Name: "Old Pro"}
	fs.subscriptions["sub_ext_1"] = uuid.New()

	file := validFile()
	im := &importer{store: fs, update: true, dryRun: true}

	ctx := context.Background()
	im.importPlans(ctx, file.Plans)
	im.importCustomers(ctx, file.Customers)
	im.importSubscriptions(ctx, file.Subscriptions)

	if im.updated != 3 {
		t.Fatalf("updated (would-update) = %d, want 3", im.updated)
	}
	if len(fs.updatedCustomers)+len(fs.updatedPlans)+len(fs.updatedSubs) != 0 {
		t.Fatalf("dry-run wrote %d records, want 0",
			len(fs.updatedCustomers)+len(fs.updatedPlans)+len(fs.updatedSubs))
	}
}

func TestUpdateMode_OffSkipsExisting(t *testing.T) {
	fs := newFakeStore()
	fs.plans["PRO-USD"] = existingPlan{ID: uuid.New(), Name: "Old Pro"}

	im := &importer{store: fs} // update flag off
	im.importPlans(context.Background(), validFile().Plans)

	if im.skipped != 1 || im.updated != 0 {
		t.Fatalf("skipped/updated = %d/%d, want 1/0", im.skipped, im.updated)
	}
	if len(fs.updatedPlans) != 0 {
		t.Fatalf("expected no writes with -update off, got %d", len(fs.updatedPlans))
	}
}

func TestCancelMissing_CancelsImportOriginOnly(t *testing.T) {
	fs := newFakeStore()
	importOrigin := existingSubscription{ID: uuid.New(), ReferenceID: "sub_gone", Status: "active"}
	dashboardOrigin := existingSubscription{ID: uuid.New(), ReferenceID: "", Status: "active"}
	keep := existingSubscription{ID: uuid.New(), ReferenceID: "sub_ext_1", Status: "active"}
	fs.list = []existingSubscription{importOrigin, dashboardOrigin, keep}

	// The file lists sub_ext_1 (kept); sub_gone is absent; dashboard sub has no external_id.
	im := &importer{store: fs, cancelMissing: true}
	im.cancelMissingSubscriptions(context.Background(), validFile().Subscriptions)

	if im.canceled != 1 {
		t.Fatalf("canceled = %d, want 1", im.canceled)
	}
	if len(fs.canceledSubs) != 1 || fs.canceledSubs[0] != importOrigin.ID {
		t.Fatalf("expected only the import-origin sub canceled, got %v", fs.canceledSubs)
	}
}

func TestCancelMissing_DryRunReportsWithoutWriting(t *testing.T) {
	fs := newFakeStore()
	fs.list = []existingSubscription{
		{ID: uuid.New(), ReferenceID: "sub_gone", Status: "active"},
		{ID: uuid.New(), ReferenceID: "", Status: "active"}, // dashboard-origin
	}

	im := &importer{store: fs, cancelMissing: true, dryRun: true}
	im.cancelMissingSubscriptions(context.Background(), validFile().Subscriptions)

	if im.canceled != 1 {
		t.Fatalf("would-cancel = %d, want 1", im.canceled)
	}
	if len(fs.canceledSubs) != 0 {
		t.Fatalf("dry-run canceled %d subs, want 0", len(fs.canceledSubs))
	}
}

func TestCancelMissing_RefusesWhenFileHasNoSubscriptions(t *testing.T) {
	fs := newFakeStore()
	fs.list = []existingSubscription{
		{ID: uuid.New(), ReferenceID: "sub_gone", Status: "active"},
	}

	im := &importer{store: fs, cancelMissing: true}
	im.cancelMissingSubscriptions(context.Background(), nil)

	if im.canceled != 0 || len(fs.canceledSubs) != 0 {
		t.Fatalf("expected refusal to cancel with empty file, canceled=%d writes=%d", im.canceled, len(fs.canceledSubs))
	}
}
