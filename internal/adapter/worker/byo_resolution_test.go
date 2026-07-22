package worker

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// --- shared fakes ---

type fakeTenantLister struct{ ids []uuid.UUID }

func (f fakeTenantLister) ListTenants(_ context.Context) ([]*domain.Tenant, error) {
	out := make([]*domain.Tenant, len(f.ids))
	for i, id := range f.ids {
		out[i] = &domain.Tenant{ID: id}
	}
	return out, nil
}

type fakeCustomerSource struct {
	byTenant map[uuid.UUID][]*domain.Customer
}

func (f fakeCustomerSource) List(_ context.Context, tenantID uuid.UUID, _ domain.CustomerFilter) ([]*domain.Customer, error) {
	return f.byTenant[tenantID], nil
}

type fakeSubCounter struct{}

func (fakeSubCounter) CountActiveByCustomer(_ context.Context, _ uuid.UUID) (map[uuid.UUID]int, error) {
	return map[uuid.UUID]int{}, nil
}

// recordingCRM records which tenants' contacts it upserted (via a shared sink).
type recordingCRM struct {
	name string
	hits *[]string
}

func (r *recordingCRM) UpsertContact(_ context.Context, email string, _ map[string]string) (string, error) {
	*r.hits = append(*r.hits, r.name+":"+email)
	return "id", nil
}

func custWithEmail(email string) *domain.Customer {
	return &domain.Customer{ID: uuid.New(), Email: email}
}

func TestCRMWorker_PerTenantAndFallback(t *testing.T) {
	tA, tB, tC := uuid.New(), uuid.New(), uuid.New()
	var hits []string
	customers := fakeCustomerSource{byTenant: map[uuid.UUID][]*domain.Customer{
		tA: {custWithEmail("a@x.com")},
		tB: {custWithEmail("b@x.com")},
		tC: {custWithEmail("c@x.com")},
	}}
	env := &recordingCRM{name: "env", hits: &hits}
	w := NewCRMSyncWorker(fakeTenantLister{ids: []uuid.UUID{tA, tB, tC}}, customers, fakeSubCounter{}, env)
	// tA has its own CRM; tB has none (→ env); tC returns nil AND we'll make env
	// nil for it via a separate run. Here env is set, so tB/tC use env.
	w.SetPerTenantCRM(func(_ context.Context, tid uuid.UUID) CRMContactUpserter {
		if tid == tA {
			return &recordingCRM{name: "byo", hits: &hits}
		}
		return nil
	})
	if _, err := w.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, h := range hits {
		got[h] = true
	}
	if !got["byo:a@x.com"] {
		t.Fatal("tenant A should use its own CRM client")
	}
	if !got["env:b@x.com"] || !got["env:c@x.com"] {
		t.Fatal("tenants B and C should fall back to the env client")
	}
}

func TestCRMWorker_SkipsWhenNoClient(t *testing.T) {
	tA := uuid.New()
	var hits []string
	customers := fakeCustomerSource{byTenant: map[uuid.UUID][]*domain.Customer{tA: {custWithEmail("a@x.com")}}}
	// No env client (nil) and resolver returns nil -> tenant skipped, no panic.
	w := NewCRMSyncWorker(fakeTenantLister{ids: []uuid.UUID{tA}}, customers, fakeSubCounter{}, nil)
	w.SetPerTenantCRM(func(_ context.Context, _ uuid.UUID) CRMContactUpserter { return nil })
	n, err := w.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 || len(hits) != 0 {
		t.Fatalf("expected no syncs, got n=%d hits=%v", n, hits)
	}
}

// --- export worker ---

type fakeLedger struct{ rows []domain.GeneralLedgerRow }

func (f fakeLedger) GeneralLedger(_ context.Context, _ uuid.UUID, _ *uuid.UUID) ([]domain.GeneralLedgerRow, error) {
	return f.rows, nil
}

type recordingUploader struct {
	name string
	hits *[]string
}

func (u *recordingUploader) PutObject(_ context.Context, key string, _ []byte, _ string) error {
	*u.hits = append(*u.hits, u.name+":"+key)
	return nil
}

func TestExportWorker_PerTenantAndSkip(t *testing.T) {
	tA, tB := uuid.New(), uuid.New()
	var hits []string
	// one GL row so the export isn't skipped as empty
	rows := []domain.GeneralLedgerRow{{TransactionID: uuid.New(), ReferenceID: uuid.New()}}
	// env s3 is nil (no operator bucket); only tA brings its own storage.
	w := NewExportWorker(fakeTenantLister{ids: []uuid.UUID{tA, tB}}, fakeLedger{rows: rows}, nil, "gl/")
	w.SetPerTenantStorage(func(_ context.Context, tid uuid.UUID) ExportUploader {
		if tid == tA {
			return &recordingUploader{name: "byo", hits: &hits}
		}
		return nil // tB has no storage and env is nil -> skipped
	})
	n, err := w.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 tenant exported (only A), got %d", n)
	}
	if len(hits) != 1 || hits[0][:4] != "byo:" {
		t.Fatalf("expected A's own uploader used once, got %v", hits)
	}
}
