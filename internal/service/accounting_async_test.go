package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// gatedConnRepo blocks ListByTenant (the first repo call SyncAllForTenant
// makes) until released, so tests can hold a background sync "running".
type gatedConnRepo struct {
	acctSyncConnRepo
	entered chan struct{}
	release chan struct{}
}

func (g *gatedConnRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.AccountingConnection, error) {
	g.entered <- struct{}{}
	<-g.release
	return nil, nil
}

func TestTriggerSyncAsyncSingleFlightPerTenant(t *testing.T) {
	repo := &gatedConnRepo{entered: make(chan struct{}, 8), release: make(chan struct{})}
	svc := newAcctSyncService(nil, &acctSyncCustomerRepo{}, &acctSyncInvoiceRepo{}, &acctSyncPlanRepo{})
	svc.SetConnectionRepo(repo)

	tenant := uuid.New()
	if !svc.TriggerSyncAsync(tenant) {
		t.Fatal("first trigger should start")
	}
	// Wait until the background sweep is actually inside SyncAllForTenant.
	select {
	case <-repo.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("background sync never started")
	}

	// Same tenant while running: refused.
	if svc.TriggerSyncAsync(tenant) {
		t.Fatal("second trigger for the same tenant should be refused while one runs")
	}
	// A different tenant is independent.
	other := uuid.New()
	if !svc.TriggerSyncAsync(other) {
		t.Fatal("other tenant should not be blocked by this tenant's sync")
	}
	select {
	case <-repo.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("other tenant's sync never started")
	}

	// Release both sweeps; the slot must free up again.
	close(repo.release)
	deadline := time.Now().Add(2 * time.Second)
	for !svc.TriggerSyncAsync(tenant) {
		if time.Now().After(deadline) {
			t.Fatal("slot never freed after sync finished")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
