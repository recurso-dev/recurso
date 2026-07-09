package service

import (
	"context"
	"math"
	"testing"

	"github.com/google/uuid"
)

// mockNexusProvider implements NexusProvider for resolver tests. Keys are
// upper-case state codes (the resolver passes the buyer state upper-cased).
type mockNexusProvider struct {
	states map[string]bool
}

func (m *mockNexusProvider) NexusFor(_ context.Context, _ uuid.UUID, state string) (bool, bool, error) {
	return len(m.states) > 0, m.states[state], nil
}

// A declared-nexus state is taxed exactly as it would be without gating.
func TestNexusGating_DeclaredState_ChargesNormally(t *testing.T) {
	r := NewTaxResolver(&mockGSTConfigProvider{}, "US", "CA").
		WithSalesTaxProvider(&mockSalesTaxProvider{rate: 0.0865}).
		WithNexusRepo(&mockNexusProvider{states: map[string]bool{"CA": true}})

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), usCustomer(), "USD", 9900, "")

	want := int64(math.Round(9900 * 0.0865))
	if res.Total != want {
		t.Errorf("Total = %d, want %d (declared-nexus state must tax normally)", res.Total, want)
	}
	if res.TaxType == "no_nexus" {
		t.Errorf("TaxType = %q, a declared-nexus state must not be gated", res.TaxType)
	}
}

// A buyer in a state the tenant has NOT declared → 0% with a no_nexus note.
func TestNexusGating_UndeclaredState_ZeroNoNexus(t *testing.T) {
	r := NewTaxResolver(&mockGSTConfigProvider{}, "US", "CA").
		WithSalesTaxProvider(&mockSalesTaxProvider{rate: 0.0865}).
		WithNexusRepo(&mockNexusProvider{states: map[string]bool{"TX": true}}) // buyer is CA

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), usCustomer(), "USD", 9900, "")

	if res.Total != 0 {
		t.Errorf("Total = %d, want 0 (no nexus in buyer state)", res.Total)
	}
	if res.TaxType != "no_nexus" {
		t.Errorf("TaxType = %q, want 'no_nexus'", res.TaxType)
	}
}

// A tenant that has declared ZERO nexus states is never gated — today's behaviour.
func TestNexusGating_ZeroDeclared_NotGated(t *testing.T) {
	r := NewTaxResolver(&mockGSTConfigProvider{}, "US", "CA").
		WithSalesTaxProvider(&mockSalesTaxProvider{rate: 0.0865}).
		WithNexusRepo(&mockNexusProvider{states: map[string]bool{}})

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), usCustomer(), "USD", 9900, "")

	want := int64(math.Round(9900 * 0.0865))
	if res.Total != want {
		t.Errorf("Total = %d, want %d (a tenant with no declared nexus must not be gated)", res.Total, want)
	}
	if res.TaxType == "no_nexus" {
		t.Errorf("TaxType = %q, a zero-declared tenant must not be gated", res.TaxType)
	}
}

// A nil nexus repo (feature off) preserves today's behaviour exactly.
func TestNexusGating_NilRepo_NotGated(t *testing.T) {
	r := NewTaxResolver(&mockGSTConfigProvider{}, "US", "CA").
		WithSalesTaxProvider(&mockSalesTaxProvider{rate: 0.0865})

	res := r.ResolveInvoiceTax(context.Background(), uuid.New(), usCustomer(), "USD", 9900, "")

	want := int64(math.Round(9900 * 0.0865))
	if res.Total != want {
		t.Errorf("Total = %d, want %d (nil nexus repo must not gate)", res.Total, want)
	}
}
