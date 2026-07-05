package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// ImportFile is the JSON input format for cmd/import. Plans are matched by
// code, customers by email, subscriptions by external_id — re-running an
// import skips anything that already exists.
type ImportFile struct {
	Plans         []PlanInput         `json:"plans"`
	Customers     []CustomerInput     `json:"customers"`
	Subscriptions []SubscriptionInput `json:"subscriptions"`
}

type PlanInput struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Amount        int64  `json:"amount"` // minor units
	Currency      string `json:"currency"`
	IntervalUnit  string `json:"interval_unit"` // day|week|month|year
	IntervalCount int    `json:"interval_count"`
}

type CustomerInput struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Phone         string `json:"phone"`
	Country       string `json:"country"`
	State         string `json:"state"`
	City          string `json:"city"`
	Line1         string `json:"line1"`
	Zip           string `json:"zip"`
	TaxID         string `json:"tax_id"`
	GSTIN         string `json:"gstin"`
	PlaceOfSupply string `json:"place_of_supply"`
}

type SubscriptionInput struct {
	ExternalID         string `json:"external_id"` // id in the source system; idempotency key
	CustomerEmail      string `json:"customer_email"`
	PlanCode           string `json:"plan_code"`
	Status             string `json:"status"` // active|trialing|paused|canceled|past_due|unpaid
	CurrentPeriodStart string `json:"current_period_start"`
	CurrentPeriodEnd   string `json:"current_period_end"`
	PaymentTerms       string `json:"payment_terms"`
}

var validStatuses = map[string]bool{
	string(domain.SubscriptionStatusActive):   true,
	string(domain.SubscriptionStatusTrialing): true,
	string(domain.SubscriptionStatusPaused):   true,
	string(domain.SubscriptionStatusCanceled): true,
	string(domain.SubscriptionStatusPastDue):  true,
	string(domain.SubscriptionStatusUnpaid):   true,
}

var validIntervals = map[string]bool{"day": true, "week": true, "month": true, "year": true}

// Validate checks the whole file and returns every problem found, so a user
// fixes their export in one pass instead of one error at a time.
func (f *ImportFile) Validate() []error {
	var errs []error
	planCodes := map[string]bool{}
	for i, p := range f.Plans {
		at := fmt.Sprintf("plans[%d] (%s)", i, p.Code)
		if p.Code == "" {
			errs = append(errs, fmt.Errorf("%s: code is required", at))
		}
		if planCodes[p.Code] {
			errs = append(errs, fmt.Errorf("%s: duplicate plan code", at))
		}
		planCodes[p.Code] = true
		if p.Name == "" {
			errs = append(errs, fmt.Errorf("%s: name is required", at))
		}
		if p.Amount < 0 {
			errs = append(errs, fmt.Errorf("%s: amount must be >= 0 (minor units)", at))
		}
		if len(p.Currency) != 3 {
			errs = append(errs, fmt.Errorf("%s: currency must be a 3-letter ISO code", at))
		}
		if !validIntervals[p.IntervalUnit] {
			errs = append(errs, fmt.Errorf("%s: interval_unit must be day, week, month, or year", at))
		}
		if p.IntervalCount < 1 {
			errs = append(errs, fmt.Errorf("%s: interval_count must be >= 1", at))
		}
	}

	emails := map[string]bool{}
	for i, c := range f.Customers {
		at := fmt.Sprintf("customers[%d] (%s)", i, c.Email)
		if c.Email == "" || !strings.Contains(c.Email, "@") {
			errs = append(errs, fmt.Errorf("%s: a valid email is required", at))
		}
		key := strings.ToLower(c.Email)
		if emails[key] {
			errs = append(errs, fmt.Errorf("%s: duplicate customer email", at))
		}
		emails[key] = true
		if c.Name == "" {
			errs = append(errs, fmt.Errorf("%s: name is required", at))
		}
		if c.Country != "" && len(c.Country) != 2 {
			errs = append(errs, fmt.Errorf("%s: country must be a 2-letter ISO code", at))
		}
	}

	extIDs := map[string]bool{}
	for i, s := range f.Subscriptions {
		at := fmt.Sprintf("subscriptions[%d] (%s)", i, s.ExternalID)
		if s.ExternalID == "" {
			errs = append(errs, fmt.Errorf("%s: external_id is required (idempotency key)", at))
		}
		if extIDs[s.ExternalID] {
			errs = append(errs, fmt.Errorf("%s: duplicate external_id", at))
		}
		extIDs[s.ExternalID] = true
		if !emails[strings.ToLower(s.CustomerEmail)] {
			errs = append(errs, fmt.Errorf("%s: customer_email %q not present in customers list", at, s.CustomerEmail))
		}
		if s.PlanCode != "" && len(f.Plans) > 0 && !planCodes[s.PlanCode] {
			errs = append(errs, fmt.Errorf("%s: plan_code %q not present in plans list (it must already exist in Recurso)", at, s.PlanCode))
		}
		if s.PlanCode == "" {
			errs = append(errs, fmt.Errorf("%s: plan_code is required", at))
		}
		if !validStatuses[s.Status] {
			errs = append(errs, fmt.Errorf("%s: status %q invalid (active, trialing, paused, canceled, past_due, unpaid)", at, s.Status))
		}
		start, err1 := time.Parse(time.RFC3339, s.CurrentPeriodStart)
		if err1 != nil {
			errs = append(errs, fmt.Errorf("%s: current_period_start must be RFC3339 (e.g. 2026-06-15T00:00:00Z)", at))
		}
		end, err2 := time.Parse(time.RFC3339, s.CurrentPeriodEnd)
		if err2 != nil {
			errs = append(errs, fmt.Errorf("%s: current_period_end must be RFC3339", at))
		}
		if err1 == nil && err2 == nil && !end.After(start) {
			errs = append(errs, fmt.Errorf("%s: current_period_end must be after current_period_start", at))
		}
	}
	return errs
}

// ParseCustomersCSV reads customers from CSV with a header row. Recognized
// columns: email, name, phone, country, state, city, line1, zip, tax_id,
// gstin, place_of_supply. Unknown columns are ignored.
func ParseCustomersCSV(r io.Reader) ([]CustomerInput, error) {
	rows, header, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	var out []CustomerInput
	for _, row := range rows {
		get := func(col string) string { return csvField(header, row, col) }
		out = append(out, CustomerInput{
			Email:         get("email"),
			Name:          get("name"),
			Phone:         get("phone"),
			Country:       get("country"),
			State:         get("state"),
			City:          get("city"),
			Line1:         get("line1"),
			Zip:           get("zip"),
			TaxID:         get("tax_id"),
			GSTIN:         get("gstin"),
			PlaceOfSupply: get("place_of_supply"),
		})
	}
	return out, nil
}

// ParseSubscriptionsCSV reads subscriptions from CSV with a header row.
// Recognized columns: external_id, customer_email, plan_code, status,
// current_period_start, current_period_end, payment_terms.
func ParseSubscriptionsCSV(r io.Reader) ([]SubscriptionInput, error) {
	rows, header, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	var out []SubscriptionInput
	for _, row := range rows {
		get := func(col string) string { return csvField(header, row, col) }
		out = append(out, SubscriptionInput{
			ExternalID:         get("external_id"),
			CustomerEmail:      get("customer_email"),
			PlanCode:           get("plan_code"),
			Status:             get("status"),
			CurrentPeriodStart: get("current_period_start"),
			CurrentPeriodEnd:   get("current_period_end"),
			PaymentTerms:       get("payment_terms"),
		})
	}
	return out, nil
}

// ParsePlansCSV reads plans from CSV with a header row. Recognized columns:
// code, name, amount, currency, interval_unit, interval_count.
func ParsePlansCSV(r io.Reader) ([]PlanInput, error) {
	rows, header, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	var out []PlanInput
	for i, row := range rows {
		get := func(col string) string { return csvField(header, row, col) }
		amount, err := strconv.ParseInt(strings.TrimSpace(get("amount")), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("plans csv row %d: amount %q is not an integer (minor units)", i+2, get("amount"))
		}
		count := 1
		if v := strings.TrimSpace(get("interval_count")); v != "" {
			count, err = strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("plans csv row %d: interval_count %q is not an integer", i+2, v)
			}
		}
		out = append(out, PlanInput{
			Code:          get("code"),
			Name:          get("name"),
			Amount:        amount,
			Currency:      get("currency"),
			IntervalUnit:  get("interval_unit"),
			IntervalCount: count,
		})
	}
	return out, nil
}

// ParseJSON reads a full ImportFile from JSON.
func ParseJSON(r io.Reader) (*ImportFile, error) {
	var f ImportFile
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("invalid import JSON: %w", err)
	}
	return &f, nil
}

func readCSV(r io.Reader) (rows [][]string, header map[string]int, err error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	all, err := cr.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid CSV: %w", err)
	}
	if len(all) < 1 {
		return nil, nil, fmt.Errorf("CSV has no header row")
	}
	header = map[string]int{}
	for i, col := range all[0] {
		header[strings.ToLower(strings.TrimSpace(col))] = i
	}
	return all[1:], header, nil
}

func csvField(header map[string]int, row []string, col string) string {
	idx, ok := header[col]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
