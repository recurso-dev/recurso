package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// countingGSTConfigs counts lookups so the tests can assert the N+1 is gone.
type countingGSTConfigs struct {
	cfg   *domain.TenantGSTConfig
	calls int
}

func (c *countingGSTConfigs) GetByTenantID(context.Context, uuid.UUID) (*domain.TenantGSTConfig, error) {
	c.calls++
	return c.cfg, nil
}

// Per-line tax resolution must not re-read the GST config and primary entity
// for every line of the same invoice (#186): one resolve per tenant per TTL.
func TestSellerJurisdiction_CachedAcrossLines(t *testing.T) {
	gst := &countingGSTConfigs{} // no GST registration → falls through to entity
	entityCalls := 0
	r := NewTaxResolver(gst, "IN", "TN").WithPrimaryEntityCountry(func(context.Context, uuid.UUID) string {
		entityCalls++
		return "US"
	})

	tenant := uuid.New()
	for range 5 {
		if c, _, _ := r.sellerJurisdiction(context.Background(), tenant); c != "US" {
			t.Fatalf("seller country = %s, want US", c)
		}
	}
	if gst.calls != 1 || entityCalls != 1 {
		t.Errorf("5 lines cost %d GST-config + %d entity reads, want 1 + 1", gst.calls, entityCalls)
	}

	// A different tenant is its own cache entry.
	if c, _, _ := r.sellerJurisdiction(context.Background(), uuid.New()); c != "US" {
		t.Fatalf("second tenant seller country = %s, want US", c)
	}
	if gst.calls != 2 {
		t.Errorf("second tenant should miss the cache, GST-config reads = %d, want 2", gst.calls)
	}
}

// A jurisdiction write invalidates the cache, so the change lands on the very
// next invoice — not after the TTL.
func TestSellerJurisdiction_InvalidateReflectsWrites(t *testing.T) {
	gst := &countingGSTConfigs{}
	r := NewTaxResolver(gst, "IN", "TN")

	tenant := uuid.New()
	if c, _, _ := r.sellerJurisdiction(context.Background(), tenant); c != "IN" {
		t.Fatalf("pre-registration seller country = %s, want IN (env default)", c)
	}

	// The tenant registers for GST; without invalidation the cached env default
	// would keep serving for the TTL.
	gst.cfg = &domain.TenantGSTConfig{GSTIN: "33AAAAA0000A1Z5", StateCode: "33"}
	if c, _, _ := r.sellerJurisdiction(context.Background(), tenant); c != "IN" {
		t.Fatalf("cached read should still serve, got %s", c)
	}
	if gst.calls != 1 {
		t.Fatalf("cached read hit the repo (%d calls)", gst.calls)
	}

	r.InvalidateSellerJurisdiction(tenant)
	c, state, cfg := r.sellerJurisdiction(context.Background(), tenant)
	if c != "IN" || state != "33" || cfg == nil {
		t.Errorf("post-invalidation = %s/%s cfg=%v, want IN/33 with config", c, state, cfg)
	}
	if gst.calls != 2 {
		t.Errorf("invalidation should force a re-read, GST-config reads = %d, want 2", gst.calls)
	}
}
