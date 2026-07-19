package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// DemoService owns the public-sandbox lifecycle (docs/spec_demo_mode.md):
// bootstrapping the demo tenant/user/API key on first boot, locating the
// demo user for /auth/demo sessions, and the wipe-and-reseed used by the
// reset worker. Only constructed when demo.Enabled().

// DemoUserEmail is the seeded dashboard identity visitors are logged in as.
const DemoUserEmail = "demo@demo.recurso.dev"

// DemoAPIKey is the stable test-mode key shown to curl users (grandfathered
// format, same key `make demo` prints).
const DemoAPIKey = "sk_test_12345"

// demoKeyCreator is the tenant-repo slice that stores API keys.
type demoKeyCreator interface {
	CreateAPIKey(ctx context.Context, key *domain.APIKey) error
}

// demoSeedRunner executes the seed binary; swapped in tests.
type demoSeedRunner func(ctx context.Context, args ...string) error

type DemoService struct {
	auth    *AuthService
	users   port.UserRepository
	keys    demoKeyCreator
	seedBin string // path to the demo_seed binary ("" = seeding skipped with a warning)
	runSeed demoSeedRunner
}

func NewDemoService(auth *AuthService, users port.UserRepository, keys demoKeyCreator, seedBin string) *DemoService {
	s := &DemoService{auth: auth, users: users, keys: keys, seedBin: seedBin}
	s.runSeed = func(ctx context.Context, args ...string) error {
		runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(runCtx, s.seedBin, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			detail := string(out)
			if len(detail) > 500 {
				detail = detail[:500] + "..."
			}
			return fmt.Errorf("demo seed: %w: %s", err, detail)
		}
		return nil
	}
	return s
}

// DemoUser resolves the seeded demo identity.
func (s *DemoService) DemoUser(ctx context.Context) (*domain.User, error) {
	user, err := s.users.GetByEmail(ctx, DemoUserEmail)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("demo user not bootstrapped yet")
	}
	return user, nil
}

// EnsureBootstrapped makes the sandbox ready on boot: demo tenant + owner
// user (random unusable password — entry is via /auth/demo), the stable
// test API key, and the rich seed data. Idempotent: an already-bootstrapped
// instance only re-checks.
func (s *DemoService) EnsureBootstrapped(ctx context.Context) (uuid.UUID, error) {
	user, err := s.users.GetByEmail(ctx, DemoUserEmail)
	if err != nil {
		return uuid.Nil, err
	}
	if user == nil {
		// Random password: nobody logs in with credentials; /auth/demo opens
		// sessions directly, and the guard blocks password resets.
		raw := make([]byte, 24)
		if _, err := rand.Read(raw); err != nil {
			return uuid.Nil, err
		}
		if _, err := s.auth.Register(ctx, "Demo Co", "Demo User", DemoUserEmail, hex.EncodeToString(raw), "demo-bootstrap"); err != nil {
			return uuid.Nil, fmt.Errorf("demo bootstrap register: %w", err)
		}
		user, err = s.users.GetByEmail(ctx, DemoUserEmail)
		if err != nil || user == nil {
			return uuid.Nil, fmt.Errorf("demo user missing after register: %w", err)
		}
		// Stable curl key; failure is non-fatal (dashboard still works).
		if err := s.keys.CreateAPIKey(ctx, &domain.APIKey{
			ID:        uuid.New(),
			TenantID:  user.TenantID,
			KeyValue:  DemoAPIKey,
			Type:      "secret",
			IsActive:  true,
			Livemode:  false,
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			slog.Warn("demo bootstrap: API key creation failed", "error", err)
		}
	}

	s.seed(ctx, user.TenantID, false)
	return user.TenantID, nil
}

// Reset wipes the demo tenant's seeded data and reseeds it pristine.
func (s *DemoService) Reset(ctx context.Context) error {
	user, err := s.DemoUser(ctx)
	if err != nil {
		return err
	}
	s.seed(ctx, user.TenantID, true)
	return nil
}

// seed runs the demo_seed binary (idempotent without reset: it refuses to
// double-seed). Missing binary logs loudly instead of failing boot.
func (s *DemoService) seed(ctx context.Context, tenantID uuid.UUID, reset bool) {
	if s.seedBin == "" {
		slog.Warn("DEMO_MODE: demo_seed binary not configured (DEMO_SEED_BIN); dashboard will be empty",
			"hint", "go build -o demo-seed ./cmd/demo_seed && DEMO_SEED_BIN=./demo-seed")
		return
	}
	args := []string{"--account=" + tenantID.String()}
	if reset {
		args = append(args, "--reset")
	}
	if err := s.runSeed(ctx, args...); err != nil {
		// "already present" from the double-seed guard is the idempotent
		// happy path on warm boots.
		slog.Info("demo seed run finished with message", "reset", reset, "detail", err.Error())
		return
	}
	slog.Info("demo data seeded", "tenant_id", tenantID, "reset", reset)
}
