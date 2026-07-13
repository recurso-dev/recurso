package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type stubConnRepo struct {
	port.AccountingConnectionRepository
	conns []*domain.AccountingConnection
}

func (s *stubConnRepo) GetActiveConnections(_ context.Context) ([]*domain.AccountingConnection, error) {
	return s.conns, nil
}

type stubSyncer struct {
	called  []uuid.UUID
	failFor uuid.UUID
}

func (s *stubSyncer) SyncAllForTenant(_ context.Context, tenantID uuid.UUID, _ bool) error {
	s.called = append(s.called, tenantID)
	if tenantID == s.failFor {
		return errors.New("boom")
	}
	return nil
}

// TestAccountingSyncWorker_RunSync_DedupsPerTenant guards the loop's control
// flow: multiple active connections for the same tenant trigger exactly one sync
// (per-distinct-tenant dedup), and one tenant's failure doesn't abort the rest.
func TestAccountingSyncWorker_RunSync_DedupsPerTenant(t *testing.T) {
	tA, tB := uuid.New(), uuid.New()
	conns := []*domain.AccountingConnection{
		{TenantID: tA}, // two active connections for tenant A
		{TenantID: tA},
		{TenantID: tB},
	}
	syncer := &stubSyncer{failFor: tB} // tenant B errors

	w := &AccountingSyncWorker{connRepo: &stubConnRepo{conns: conns}, syncer: syncer}
	w.RunSync(context.Background())

	if len(syncer.called) != 2 {
		t.Fatalf("synced %d times, want 2 (one per distinct tenant despite 3 connections)", len(syncer.called))
	}
	countA, countB := 0, 0
	for _, id := range syncer.called {
		switch id {
		case tA:
			countA++
		case tB:
			countB++
		}
	}
	if countA != 1 || countB != 1 {
		t.Errorf("per-tenant sync counts: A=%d B=%d, want 1 and 1 (B's error must not abort A)", countA, countB)
	}
}
