package service

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

type fakeDemoUsers struct {
	port.UserRepository
	user *domain.User
}

func (f *fakeDemoUsers) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if f.user != nil && f.user.Email == email {
		return f.user, nil
	}
	return nil, nil
}

func TestDemoServiceSeedArgs(t *testing.T) {
	tenantID := uuid.New()
	users := &fakeDemoUsers{user: &domain.User{ID: uuid.New(), TenantID: tenantID, Email: DemoUserEmail}}
	svc := NewDemoService(nil, users, nil, "/bin/demo-seed")

	var got [][]string
	svc.runSeed = func(ctx context.Context, args ...string) error {
		got = append(got, args)
		return nil
	}

	// Bootstrapped instance: EnsureBootstrapped only re-seeds (idempotent path).
	id, err := svc.EnsureBootstrapped(context.Background())
	if err != nil || id != tenantID {
		t.Fatalf("EnsureBootstrapped = %v/%v", id, err)
	}
	if len(got) != 1 || got[0][0] != "--account="+tenantID.String() || len(got[0]) != 1 {
		t.Fatalf("boot seed args = %v, want [--account=<tenant>] without --reset", got)
	}

	// Reset passes --reset so the seeder purges first.
	if err := svc.Reset(context.Background()); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	last := got[len(got)-1]
	if len(last) != 2 || last[1] != "--reset" {
		t.Fatalf("reset args = %v, want --account + --reset", last)
	}
}

func TestDemoServiceMissingUserAndBinary(t *testing.T) {
	svc := NewDemoService(nil, &fakeDemoUsers{}, nil, "")
	if _, err := svc.DemoUser(context.Background()); err == nil || !strings.Contains(err.Error(), "not bootstrapped") {
		t.Fatalf("DemoUser err = %v, want not-bootstrapped", err)
	}
	// Reset with no user errors cleanly (worker logs + retries next tick).
	if err := svc.Reset(context.Background()); err == nil {
		t.Fatal("Reset without a demo user must error")
	}
}
