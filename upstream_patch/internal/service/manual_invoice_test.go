package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

func manualBase() CreateManualInvoiceInput {
	return CreateManualInvoiceInput{
		TenantID:    uuid.New(),
		CustomerID:  uuid.New(),
		Currency:    "usd",
		InvoiceDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Lines: []ManualInvoiceLine{
			{Description: "Platform fee", Amount: 123456},
			{Description: "Connection fee", Amount: 50000},
		},
	}
}

var manualNow = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

func TestBuildManualInvoice_HappyPath(t *testing.T) {
	in := manualBase()
	inv, err := buildManualInvoice(in, manualNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Currency != "USD" {
		t.Errorf("currency = %q, want USD (normalised)", inv.Currency)
	}
	if inv.Total != 173456 || inv.Subtotal != 173456 || inv.AmountDue != 173456 || inv.TaxAmount != 0 {
		t.Errorf("amounts = sub %d tax %d total %d due %d, want 173456/0/173456/173456",
			inv.Subtotal, inv.TaxAmount, inv.Total, inv.AmountDue)
	}
	if inv.Status != domain.InvoiceStatusOpen {
		t.Errorf("status = %q, want open", inv.Status)
	}
	if inv.BillingReason != domain.BillingReasonManual || inv.SubscriptionID != nil {
		t.Errorf("billing reason %q / subscription %v, want manual / nil", inv.BillingReason, inv.SubscriptionID)
	}
	if !inv.CreatedAt.Equal(in.InvoiceDate) {
		t.Errorf("invoice date = %v, want %v", inv.CreatedAt, in.InvoiceDate)
	}
	// Default terms are net30 from the invoice date, not from "now".
	if want := in.InvoiceDate.AddDate(0, 0, 30); !inv.DueDate.Equal(want) {
		t.Errorf("due date = %v, want %v", inv.DueDate, want)
	}
	if inv.DunningManagedBy != manualDunningOwner {
		t.Errorf("dunning owner = %q, want %q (scheduler must not collect externally imported invoices)", inv.DunningManagedBy, manualDunningOwner)
	}
	if len(inv.LineItems) != 2 {
		t.Fatalf("lines = %d, want 2", len(inv.LineItems))
	}
	var sum int64
	for _, l := range inv.LineItems {
		sum += l.Amount
		if l.InvoiceID != inv.ID {
			t.Errorf("line %q not linked to the invoice", l.Description)
		}
		if l.CGSTAmount != 0 || l.SGSTAmount != 0 || l.IGSTAmount != 0 || l.TaxRate != 0 {
			t.Errorf("line %q carries tax; manual invoices are tax-free", l.Description)
		}
		if l.TaxableAmount != l.Amount {
			t.Errorf("line %q taxable %d != amount %d", l.Description, l.TaxableAmount, l.Amount)
		}
	}
	if sum != inv.Total {
		t.Errorf("Σ lines %d != total %d", sum, inv.Total)
	}
	for i := 1; i < len(inv.LineItems); i++ {
		if !inv.LineItems[i].CreatedAt.After(inv.LineItems[i-1].CreatedAt) {
			t.Errorf("line %d is not stamped after line %d; display order would be random", i+1, i)
		}
	}
}

func TestBuildManualInvoice_WaiverLineReducesTotal(t *testing.T) {
	in := manualBase()
	in.Lines = []ManualInvoiceLine{
		{Description: "Data feed", Amount: 1500_00},
		{Description: "Data feed waiver", Amount: -500_00},
	}
	inv, err := buildManualInvoice(in, manualNow)
	if err != nil {
		t.Fatalf("a negative line must be accepted: %v", err)
	}
	if inv.Total != 1000_00 || inv.Status != domain.InvoiceStatusOpen {
		t.Errorf("total %d status %q, want 100000 / open", inv.Total, inv.Status)
	}
	if inv.LineItems[1].Amount != -500_00 {
		t.Errorf("waiver line amount = %d, want -50000 kept as a visible negative line", inv.LineItems[1].Amount)
	}
}

func TestBuildManualInvoice_FullyWaivedIsPaidZero(t *testing.T) {
	in := manualBase()
	in.Lines = []ManualInvoiceLine{
		{Description: "Utility fee", Amount: 4002_18},
		{Description: "Utility fee waiver", Amount: -4002_18},
	}
	inv, err := buildManualInvoice(in, manualNow)
	if err != nil {
		t.Fatalf("a net-zero invoice must be accepted: %v", err)
	}
	if inv.Total != 0 || inv.Status != domain.InvoiceStatusPaid || inv.PaidAt == nil {
		t.Errorf("total %d status %q paidAt %v, want 0 / paid / set", inv.Total, inv.Status, inv.PaidAt)
	}
}

func TestBuildManualInvoice_NegativeNetRejected(t *testing.T) {
	in := manualBase()
	in.Lines = []ManualInvoiceLine{
		{Description: "fee", Amount: 100},
		{Description: "waiver", Amount: -101},
	}
	_, err := buildManualInvoice(in, manualNow)
	if !errors.Is(err, ErrInvalidManualInvoice) || !strings.Contains(err.Error(), "credit note") {
		t.Errorf("err = %v, want ErrInvalidManualInvoice pointing at credit notes", err)
	}
}

func TestBuildManualInvoice_DueDateRules(t *testing.T) {
	in := manualBase()
	explicit := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	in.DueDate = &explicit
	in.PaymentTerms = "net60"
	inv, err := buildManualInvoice(in, manualNow)
	if err != nil || !inv.DueDate.Equal(explicit) {
		t.Fatalf("explicit due date must win over terms: due=%v err=%v", inv.DueDate, err)
	}

	before := in.InvoiceDate.AddDate(0, 0, -1)
	in.DueDate = &before
	if _, err := buildManualInvoice(in, manualNow); !errors.Is(err, ErrInvalidManualInvoice) {
		t.Errorf("due date before invoice date: err = %v, want ErrInvalidManualInvoice", err)
	}

	in.DueDate = nil
	in.PaymentTerms = "net0"
	inv, err = buildManualInvoice(in, manualNow)
	if err != nil || !inv.DueDate.Equal(in.InvoiceDate) {
		t.Errorf("net0 due = %v err=%v, want invoice date", inv.DueDate, err)
	}
}

func TestBuildManualInvoice_ZeroInvoiceDateDefaultsToNow(t *testing.T) {
	in := manualBase()
	in.InvoiceDate = time.Time{}
	inv, err := buildManualInvoice(in, manualNow)
	if err != nil || !inv.CreatedAt.Equal(manualNow) {
		t.Errorf("created = %v err=%v, want now", inv.CreatedAt, err)
	}
}

func TestBuildManualInvoice_QuantityAndUnitAmount(t *testing.T) {
	in := manualBase()
	in.Lines = []ManualInvoiceLine{
		{Description: "seats", Quantity: 3, UnitAmount: 1000, Amount: 3000},
		{Description: "derived unit", Quantity: 4, Amount: 2000},
		{Description: "plain"}, // zero-amount line is allowed
	}
	inv, err := buildManualInvoice(in, manualNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l := inv.LineItems[0]; l.Quantity != 3 || l.UnitAmount != 1000 || l.Amount != 3000 {
		t.Errorf("line 1 = qty %d unit %d amt %d", l.Quantity, l.UnitAmount, l.Amount)
	}
	if l := inv.LineItems[1]; l.Quantity != 4 || l.UnitAmount != 500 || l.Amount != 2000 {
		t.Errorf("line 2 = qty %d unit %d amt %d, want unit derived as 500", l.Quantity, l.UnitAmount, l.Amount)
	}
	if l := inv.LineItems[2]; l.Quantity != 1 {
		t.Errorf("line 3 quantity = %d, want default 1", l.Quantity)
	}
}

func TestBuildManualInvoice_Rejections(t *testing.T) {
	long := strings.Repeat("x", maxManualDescriptionLen+1)
	manyLines := make([]ManualInvoiceLine, maxManualInvoiceLines+1)
	for i := range manyLines {
		manyLines[i] = ManualInvoiceLine{Description: "l", Amount: 1}
	}
	cases := []struct {
		name   string
		mutate func(*CreateManualInvoiceInput)
		want   string
	}{
		{"bad currency length", func(i *CreateManualInvoiceInput) { i.Currency = "US" }, "currency"},
		{"non-letter currency", func(i *CreateManualInvoiceInput) { i.Currency = "U5D" }, "currency"},
		{"empty currency", func(i *CreateManualInvoiceInput) { i.Currency = "" }, "currency"},
		{"no lines", func(i *CreateManualInvoiceInput) { i.Lines = nil }, "at least one line"},
		{"too many lines", func(i *CreateManualInvoiceInput) { i.Lines = manyLines }, "too many lines"},
		{"blank description", func(i *CreateManualInvoiceInput) { i.Lines[0].Description = "   " }, "line 1: description"},
		{"long description", func(i *CreateManualInvoiceInput) { i.Lines[1].Description = long }, "line 2: description"},
		{"amount out of range", func(i *CreateManualInvoiceInput) { i.Lines[0].Amount = maxManualLineAmount + 1 }, "out of range"},
		{"negative quantity", func(i *CreateManualInvoiceInput) { i.Lines[0].Quantity = -1 }, "quantity"},
		{"qty x unit mismatch", func(i *CreateManualInvoiceInput) {
			i.Lines[0] = ManualInvoiceLine{Description: "d", Quantity: 2, UnitAmount: 10, Amount: 25}
		}, "does not equal amount"},
		{"uneven division", func(i *CreateManualInvoiceInput) {
			i.Lines[0] = ManualInvoiceLine{Description: "d", Quantity: 3, Amount: 10}
		}, "divide evenly"},
		{"long external ref", func(i *CreateManualInvoiceInput) { i.ExternalRef = strings.Repeat("r", maxManualExternalRefLen+1) }, "external_ref"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := manualBase()
			tc.mutate(&in)
			_, err := buildManualInvoice(in, manualNow)
			if !errors.Is(err, ErrInvalidManualInvoice) {
				t.Fatalf("err = %v, want ErrInvalidManualInvoice", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %q, want it to mention %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSameManualInvoice(t *testing.T) {
	base := manualBase()
	want, err := buildManualInvoice(base, manualNow)
	if err != nil {
		t.Fatal(err)
	}
	// A stored copy: same content, lines in a different order, other IDs.
	stored := *want
	stored.ID = uuid.New()
	stored.InvoiceNumber = "UKCO-000001"
	stored.LineItems = []domain.InvoiceItem{want.LineItems[1], want.LineItems[0]}
	if err := sameManualInvoice(&stored, want); err != nil {
		t.Errorf("identical content (reordered lines) must match, got %v", err)
	}

	cases := map[string]func(*domain.Invoice){
		"other customer": func(i *domain.Invoice) { i.CustomerID = uuid.New() },
		"other entity":   func(i *domain.Invoice) { e := uuid.New(); i.EntityID = &e },
		"other currency": func(i *domain.Invoice) { i.Currency = "EUR" },
		"other total":    func(i *domain.Invoice) { i.Total++ },
		"fewer lines":    func(i *domain.Invoice) { i.LineItems = i.LineItems[:1] },
		"other amount": func(i *domain.Invoice) {
			i.LineItems = []domain.InvoiceItem{{Amount: 1}, {Amount: want.Total - 1}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := stored
			s.LineItems = append([]domain.InvoiceItem(nil), stored.LineItems...)
			mutate(&s)
			if err := sameManualInvoice(&s, want); !errors.Is(err, ErrManualInvoiceRefConflict) {
				t.Errorf("err = %v, want ErrManualInvoiceRefConflict", err)
			}
		})
	}
}
