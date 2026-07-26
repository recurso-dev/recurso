package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// recordingGSTRSource records the entityID it was handed so we can prove the
// service threads the per-entity scope through to the read layer.
type recordingGSTRSource struct {
	gotEntity   *uuid.UUID
	invoices    []domain.GSTR1Invoice
	creditNotes []domain.GSTR1CreditNote
}

func (s *recordingGSTRSource) GetGSTR1Invoices(_ context.Context, _ uuid.UUID, entityID *uuid.UUID, _, _ time.Time) ([]domain.GSTR1Invoice, error) {
	s.gotEntity = entityID
	return s.invoices, nil
}
func (s *recordingGSTRSource) GetGSTR1CreditNotes(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _, _ time.Time) ([]domain.GSTR1CreditNote, error) {
	return s.creditNotes, nil
}

// GetGSTR1 / GetGSTR3B pass the requested entity scope through to the source; a
// nil entity (single-entity / consolidated) is passed as nil unchanged.
func TestGSTR_PassesEntityScope(t *testing.T) {
	entityID := uuid.New()

	src := &recordingGSTRSource{}
	svc := NewGSTRService(src)

	if _, err := svc.GetGSTR1(context.Background(), uuid.New(), &entityID, 3, 2026); err != nil {
		t.Fatalf("GetGSTR1: %v", err)
	}
	if src.gotEntity == nil || *src.gotEntity != entityID {
		t.Errorf("GSTR1 entity scope = %v, want %v", src.gotEntity, entityID)
	}

	src.gotEntity = &entityID // reset to a non-nil so we can prove nil is passed
	if _, err := svc.GetGSTR3B(context.Background(), uuid.New(), nil, 3, 2026); err != nil {
		t.Fatalf("GetGSTR3B: %v", err)
	}
	if src.gotEntity != nil {
		t.Errorf("GSTR3B nil scope should pass nil, got %v", src.gotEntity)
	}
}
