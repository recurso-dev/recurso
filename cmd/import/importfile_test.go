package main

import (
	"strings"
	"testing"
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
