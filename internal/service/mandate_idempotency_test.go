package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestMandateDebitIdempotencyKey proves the ENG-190 property: the key is stable
// across retries of the SAME cycle (LastDebitAt unchanged) and changes only when
// a cycle is successfully debited (LastDebitAt advances) — so a re-attempt of a
// cycle that already charged the gateway carries the same key and dedupes.
func TestMandateDebitIdempotencyKey(t *testing.T) {
	id := uuid.New()
	t0 := time.Unix(1_700_000_000, 0)
	m := &domain.Mandate{ID: id, LastDebitAt: &t0}

	// Two attempts of the same cycle -> identical key.
	k1 := mandateDebitIdempotencyKey(m)
	k2 := mandateDebitIdempotencyKey(m)
	if k1 != k2 {
		t.Fatalf("same-cycle key changed: %q vs %q", k1, k2)
	}

	// After a successful debit (LastDebitAt advances), the next cycle keys differ.
	t1 := t0.Add(30 * 24 * time.Hour)
	m.LastDebitAt = &t1
	if next := mandateDebitIdempotencyKey(m); next == k1 {
		t.Fatalf("key did not change after a successful debit: %q", next)
	}

	// First-ever debit (no LastDebitAt) is keyed on 0.
	first := &domain.Mandate{ID: id}
	if got, want := mandateDebitIdempotencyKey(first), "md-"+id.String()+"-0"; got != want {
		t.Fatalf("first-debit key = %q, want %q", got, want)
	}
}
