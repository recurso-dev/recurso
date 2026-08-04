package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// revokeSessionRepo records the id passed to Delete so the test can assert the
// server-side row is actually revoked on logout.
type revokeSessionRepo struct {
	port.PortalSessionRepository
	deletedID uuid.UUID
	deleted   bool
}

func (r *revokeSessionRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.deletedID = id
	r.deleted = true
	return nil
}

// TestRevokeSession_DeletesServerSideRow is the regression lock for the portal
// logout bug: clearing the cookie left the session row valid for its full TTL,
// so a token captured by the X-Portal-Session header path survived logout.
// RevokeSession must delete the row by id.
func TestRevokeSession_DeletesServerSideRow(t *testing.T) {
	repo := &revokeSessionRepo{}
	svc := NewPortalService(nil, nil, nil, repo, nil, nil, nil, "")

	sess := &domain.PortalSession{ID: uuid.New(), CustomerID: uuid.New()}
	if err := svc.RevokeSession(context.Background(), sess.ID); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	if !repo.deleted {
		t.Fatal("RevokeSession did not call the session repo's Delete")
	}
	if repo.deletedID != sess.ID {
		t.Errorf("deleted id = %v, want %v", repo.deletedID, sess.ID)
	}
}
