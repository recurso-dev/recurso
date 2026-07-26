package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

type stubInFlight struct {
	inFlight bool
	called   bool
}

func (s *stubInFlight) HasInFlightForInvoice(_ context.Context, _ uuid.UUID) (bool, error) {
	s.called = true
	return s.inFlight, nil
}

// An invoice with a settling payment (e.g. an ACH debit still processing) is
// skipped by dunning — it must NOT be re-charged or emailed "payment failed".
// The scheduler here has nil notification/invoice deps, so reaching them would
// panic; a clean return proves the in-flight guard short-circuited first.
func TestDunning_SkipsSettlingInvoice(t *testing.T) {
	stub := &stubInFlight{inFlight: true}
	s := &DunningScheduler{attempts: stub, config: DefaultDunningConfig()}

	s.processInvoice(context.Background(), domain.OverdueInvoice{
		ID:            uuid.New(),
		InvoiceNumber: "INV-ACH-1",
		DueDate:       time.Now().Add(-48 * time.Hour), // overdue, but settling
	})

	if !stub.called {
		t.Fatal("expected the in-flight check to run before any dunning action")
	}
	// Reaching here (no panic from the nil deps) is the assertion: the guard
	// returned before touching notificationSvc / invoiceRepo.
}
