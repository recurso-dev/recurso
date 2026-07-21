package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/adapter/email"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

const alertYear = 2026

type fakeNexusStatus struct{ states []domain.NexusStateStatus }

func (f *fakeNexusStatus) Status(_ context.Context, _ uuid.UUID, _ int) ([]domain.NexusStateStatus, error) {
	return f.states, nil
}

type fakeAlertStore struct{ claimed map[string]bool }

func (f *fakeAlertStore) ClaimNexusAlert(_ context.Context, _ uuid.UUID, state string, year int, level string, _ int) (bool, error) {
	if f.claimed == nil {
		f.claimed = map[string]bool{}
	}
	key := state + "|" + level
	if f.claimed[key] {
		return false, nil
	}
	f.claimed[key] = true
	return true, nil
}

type fakeUserLister struct{ users []*domain.User }

func (f *fakeUserLister) ListByTenant(_ context.Context, _ uuid.UUID) ([]*domain.User, error) {
	return f.users, nil
}

type fakeNexusNotifier struct{ sent []email.NexusAlertData }

func (f *fakeNexusNotifier) SendNexusThresholdAlert(_ context.Context, level string, data email.NexusAlertData) error {
	data.Level = level
	f.sent = append(f.sent, data)
	return nil
}

func owner() *domain.User {
	return &domain.User{Email: "founder@acme.com", Name: "Founder", Role: domain.RoleOwner}
}

func ptrTime(t time.Time) *time.Time { return &t }

func newAlertSvc(states []domain.NexusStateStatus, store *fakeAlertStore, notifier *fakeNexusNotifier) *NexusAlertService {
	return NewNexusAlertService(
		&fakeNexusStatus{states: states},
		store,
		&fakeUserLister{users: []*domain.User{owner()}},
		notifier,
		"https://app.recurso.dev",
	)
}

// An approaching (≥80%, no nexus yet) and a this-year crossing each fire exactly
// one alert of the right level; declared, old-year, and below-band states don't.
func TestNexusAlert_LevelSelection(t *testing.T) {
	thisYear := ptrTime(time.Date(alertYear, 3, 1, 0, 0, 0, 0, time.UTC))
	lastYear := ptrTime(time.Date(alertYear-1, 3, 1, 0, 0, 0, 0, time.UTC))
	states := []domain.NexusStateStatus{
		{StateCode: "CA", NexusType: "", ProximityPct: 82, TaxableSales: 8_200_000, TxnCount: 140},     // approaching
		{StateCode: "TX", NexusType: domain.NexusEconomic, EstablishedAt: thisYear, ProximityPct: 130}, // crossed this year
		{StateCode: "NY", NexusType: domain.NexusPhysical, EstablishedAt: thisYear, ProximityPct: 150}, // declared → skip
		{StateCode: "WA", NexusType: domain.NexusEconomic, EstablishedAt: lastYear, ProximityPct: 200}, // crossed last year → skip
		{StateCode: "CO", NexusType: "", ProximityPct: 50},                                             // below band → skip
	}
	store := &fakeAlertStore{}
	notifier := &fakeNexusNotifier{}
	svc := newAlertSvc(states, store, notifier)

	if err := svc.EvaluateAndAlert(context.Background(), uuid.New(), alertYear); err != nil {
		t.Fatalf("EvaluateAndAlert: %v", err)
	}

	got := map[string]string{}
	for _, d := range notifier.sent {
		got[d.State] = d.Level
	}
	if len(got) != 2 {
		t.Fatalf("want 2 alerts, got %d: %+v", len(got), got)
	}
	if got["CA"] != string(domain.NexusAlertApproaching) {
		t.Errorf("CA level = %q, want approaching", got["CA"])
	}
	if got["TX"] != string(domain.NexusAlertCrossed) {
		t.Errorf("TX level = %q, want crossed", got["TX"])
	}
	if _, ok := got["NY"]; ok {
		t.Error("NY (declared physical) should not be alerted")
	}
	if _, ok := got["WA"]; ok {
		t.Error("WA (economic established last year) should not be alerted")
	}
}

// A second run with the same state at the same level sends nothing (dedup via the
// claim store) — so a threshold that stays crossed isn't re-alerted daily.
func TestNexusAlert_DedupAcrossRuns(t *testing.T) {
	states := []domain.NexusStateStatus{
		{StateCode: "CA", NexusType: "", ProximityPct: 90, TaxableSales: 9_000_000, TxnCount: 150},
	}
	store := &fakeAlertStore{}
	notifier := &fakeNexusNotifier{}
	svc := newAlertSvc(states, store, notifier)
	ctx, tenant := context.Background(), uuid.New()

	_ = svc.EvaluateAndAlert(ctx, tenant, alertYear)
	_ = svc.EvaluateAndAlert(ctx, tenant, alertYear) // same day again

	if len(notifier.sent) != 1 {
		t.Fatalf("want exactly 1 alert across two runs, got %d", len(notifier.sent))
	}
}

// The alert carries formatted, recipient-addressed content.
func TestNexusAlert_PayloadShape(t *testing.T) {
	sales := int64(100_000) // 200 transactions
	txns := 200
	states := []domain.NexusStateStatus{{
		StateCode: "CA", NexusType: "", ProximityPct: 85, TaxableSales: 9_240_055, TxnCount: 141,
		Threshold: &domain.NexusThreshold{SalesThreshold: &sales, TxnThreshold: &txns, Combinator: "or"},
	}}
	notifier := &fakeNexusNotifier{}
	svc := newAlertSvc(states, &fakeAlertStore{}, notifier)

	_ = svc.EvaluateAndAlert(context.Background(), uuid.New(), alertYear)

	if len(notifier.sent) != 1 {
		t.Fatalf("want 1 alert, got %d", len(notifier.sent))
	}
	d := notifier.sent[0]
	if d.RecipientEmail != "founder@acme.com" {
		t.Errorf("recipient = %q, want the owner", d.RecipientEmail)
	}
	if d.TaxableSales != "$92,400.55" {
		t.Errorf("taxable sales = %q, want $92,400.55", d.TaxableSales)
	}
	if d.ThresholdText != "$1,000.00 or 200 transactions" {
		t.Errorf("threshold text = %q", d.ThresholdText)
	}
	if d.SettingsURL != "https://app.recurso.dev/settings/tax-nexus" {
		t.Errorf("settings url = %q", d.SettingsURL)
	}
}

func TestFormatUSDCents(t *testing.T) {
	cases := map[int64]string{
		0:         "$0.00",
		5:         "$0.05",
		99:        "$0.99",
		100:       "$1.00",
		123456:    "$1,234.56",
		100000000: "$1,000,000.00",
		-4200:     "-$42.00",
	}
	for in, want := range cases {
		if got := formatUSDCents(in); got != want {
			t.Errorf("formatUSDCents(%d) = %q, want %q", in, got, want)
		}
	}
}
