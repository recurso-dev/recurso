package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// --- in-memory fakes -------------------------------------------------------

type fakeUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{users: map[uuid.UUID]*domain.User{}} }

func (r *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	for _, e := range r.users {
		if strings.EqualFold(e.Email, u.Email) {
			return domain.ErrDuplicateEmail
		}
	}
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if strings.EqualFold(u.Email, email) {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *fakeUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	for _, u := range r.users {
		if strings.EqualFold(u.Email, email) {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok || u.TenantID != tenantID {
		return nil, domain.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *fakeUserRepo) GetByIDGlobal(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *fakeUserRepo) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.PasswordHash = hash
	return nil
}

func (r *fakeUserRepo) SetMFASecret(_ context.Context, tenantID, id uuid.UUID, secret string) error {
	u, ok := r.users[id]
	if !ok || u.TenantID != tenantID {
		return domain.ErrUserNotFound
	}
	u.MFASecret = secret
	return nil
}

func (r *fakeUserRepo) SetMFAEnabled(_ context.Context, tenantID, id uuid.UUID, enabled bool) error {
	u, ok := r.users[id]
	if !ok || u.TenantID != tenantID {
		return domain.ErrUserNotFound
	}
	u.MFAEnabled = enabled
	return nil
}

func (r *fakeUserRepo) SetMFALastTimestep(_ context.Context, tenantID, id uuid.UUID, timestep int64) error {
	u, ok := r.users[id]
	if !ok || u.TenantID != tenantID {
		return domain.ErrUserNotFound
	}
	if timestep > u.MFALastTimestep { // monotonic, mirrors the real repo guard
		u.MFALastTimestep = timestep
	}
	return nil
}

func (r *fakeUserRepo) RegisterFailedLogin(_ context.Context, id uuid.UUID, lockThreshold int, lockFor time.Duration) error {
	u, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.FailedLoginAttempts++
	if u.FailedLoginAttempts >= lockThreshold {
		t := time.Now().Add(lockFor)
		u.LockedUntil = &t
	}
	return nil
}

func (r *fakeUserRepo) ClearFailedLogins(_ context.Context, id uuid.UUID) error {
	if u, ok := r.users[id]; ok {
		u.FailedLoginAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

func (r *fakeUserRepo) ClearMFA(_ context.Context, tenantID, id uuid.UUID) error {
	u, ok := r.users[id]
	if !ok || u.TenantID != tenantID {
		return domain.ErrUserNotFound
	}
	u.MFAEnabled = false
	u.MFASecret = ""
	return nil
}

func (r *fakeUserRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*domain.User, error) {
	var out []*domain.User
	for _, u := range r.users {
		if u.TenantID == tenantID {
			cp := *u
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *fakeUserRepo) UpdateRole(_ context.Context, tenantID, id uuid.UUID, role domain.Role) error {
	u, ok := r.users[id]
	if !ok || u.TenantID != tenantID {
		return domain.ErrUserNotFound
	}
	u.Role = role
	return nil
}

func (r *fakeUserRepo) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	u, ok := r.users[id]
	if !ok || u.TenantID != tenantID {
		return domain.ErrUserNotFound
	}
	delete(r.users, id)
	return nil
}

func (r *fakeUserRepo) CountOwners(_ context.Context, tenantID uuid.UUID) (int, error) {
	n := 0
	for _, u := range r.users {
		if u.TenantID == tenantID && u.Role == domain.RoleOwner {
			n++
		}
	}
	return n, nil
}

type fakeSessionRepo struct {
	sessions map[string]*domain.Session
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{sessions: map[string]*domain.Session{}}
}

func (r *fakeSessionRepo) Create(_ context.Context, s *domain.Session) error {
	cp := *s
	r.sessions[s.TokenHash] = &cp
	return nil
}
func (r *fakeSessionRepo) GetByTokenHash(_ context.Context, h string) (*domain.Session, error) {
	s, ok := r.sessions[h]
	if !ok {
		return nil, domain.ErrSessionNotFound
	}
	cp := *s
	return &cp, nil
}
func (r *fakeSessionRepo) DeleteByTokenHash(_ context.Context, h string) error {
	delete(r.sessions, h)
	return nil
}
func (r *fakeSessionRepo) DeleteByUser(_ context.Context, userID uuid.UUID) error {
	for h, s := range r.sessions {
		if s.UserID == userID {
			delete(r.sessions, h)
		}
	}
	return nil
}
func (r *fakeSessionRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Session, error) {
	for _, s := range r.sessions {
		if s.ID == id {
			cp := *s
			return &cp, nil
		}
	}
	return nil, domain.ErrSessionNotFound
}
func (r *fakeSessionRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*domain.Session, error) {
	var out []*domain.Session
	for _, s := range r.sessions {
		if s.UserID == userID {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *fakeSessionRepo) DeleteByID(_ context.Context, id uuid.UUID) error {
	for h, s := range r.sessions {
		if s.ID == id {
			delete(r.sessions, h)
		}
	}
	return nil
}
func (r *fakeSessionRepo) DeleteByUserExcept(_ context.Context, userID uuid.UUID, exceptHash string) error {
	for h, s := range r.sessions {
		if s.UserID == userID && h != exceptHash {
			delete(r.sessions, h)
		}
	}
	return nil
}

type fakeTenants struct {
	tenants map[uuid.UUID]*domain.Tenant
}

func newFakeTenants() *fakeTenants { return &fakeTenants{tenants: map[uuid.UUID]*domain.Tenant{}} }

func (f *fakeTenants) Register(_ context.Context, name, email string) (*domain.Tenant, *domain.APIKey, error) {
	t := &domain.Tenant{ID: uuid.New(), Name: name, Email: email, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.tenants[t.ID] = t
	k := &domain.APIKey{ID: uuid.New(), TenantID: t.ID, KeyValue: "sk_live_" + uuid.NewString(), Type: "secret", IsActive: true}
	return t, k, nil
}

func (f *fakeTenants) GetAccount(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	t, ok := f.tenants[id]
	if !ok {
		return nil, errors.New("tenant not found")
	}
	return t, nil
}

func newTestAuth() (*AuthService, *fakeUserRepo, *fakeSessionRepo) {
	ur := newFakeUserRepo()
	sr := newFakeSessionRepo()
	svc := NewAuthService(ur, sr, newFakeTenants(), time.Hour)
	return svc, ur, sr
}

// --- tests -----------------------------------------------------------------

func TestRegister_CreatesTenantOwnerAndSession(t *testing.T) {
	svc, ur, _ := newTestAuth()
	res, err := svc.Register(context.Background(), "Acme", "Alice", "Alice@Example.com", "supersecret", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if res.User.Role != domain.RoleOwner {
		t.Fatalf("role = %q, want owner", res.User.Role)
	}
	if res.User.Email != "alice@example.com" {
		t.Fatalf("email not normalized: %q", res.User.Email)
	}
	if res.APIKey == nil || res.APIKey.KeyValue == "" {
		t.Fatal("expected an API key in register result")
	}
	if len(ur.users) != 1 {
		t.Fatalf("user count = %d, want 1", len(ur.users))
	}
	// Session token resolves to the owner.
	u, err := svc.ResolveSession(context.Background(), res.SessionToken)
	if err != nil {
		t.Fatalf("resolve session: %v", err)
	}
	if u.ID != res.User.ID {
		t.Fatalf("session resolved to wrong user")
	}
}

func TestRegister_DuplicateEmailRejected(t *testing.T) {
	svc, _, _ := newTestAuth()
	if _, err := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", ""); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := svc.Register(context.Background(), "Other", "Bob", "A@B.com", "supersecret", "")
	if !errors.Is(err, domain.ErrDuplicateEmail) {
		t.Fatalf("err = %v, want ErrDuplicateEmail", err)
	}
}

func TestRegister_WeakPasswordRejected(t *testing.T) {
	svc, _, _ := newTestAuth()
	_, err := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "short", "")
	if !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("err = %v, want ErrWeakPassword", err)
	}
}

func TestLogin_SuccessAndGenericErrors(t *testing.T) {
	svc, _, _ := newTestAuth()
	if _, err := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", ""); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Success.
	res, err := svc.Login(context.Background(), "A@B.com", "supersecret", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.SessionToken == "" {
		t.Fatal("expected a session token")
	}

	// Wrong password → generic invalid credentials.
	if _, err := svc.Login(context.Background(), "a@b.com", "wrongpass", ""); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("wrong password err = %v, want ErrInvalidCredentials", err)
	}
	// Unknown email → SAME generic error (no enumeration).
	if _, err := svc.Login(context.Background(), "nobody@nowhere.com", "supersecret", ""); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("unknown email err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogout_InvalidatesSession(t *testing.T) {
	svc, _, _ := newTestAuth()
	res, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")

	if err := svc.Logout(context.Background(), res.SessionToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.ResolveSession(context.Background(), res.SessionToken); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("resolve after logout err = %v, want ErrSessionNotFound", err)
	}
}

func TestResolveSession_ExpiredRejected(t *testing.T) {
	ur := newFakeUserRepo()
	sr := newFakeSessionRepo()
	svc := NewAuthService(ur, sr, newFakeTenants(), time.Hour)
	res, _ := svc.Register(context.Background(), "Acme", "Alice", "a@b.com", "supersecret", "")

	// Force the stored session to be expired.
	for h, s := range sr.sessions {
		s.ExpiresAt = time.Now().Add(-time.Minute)
		sr.sessions[h] = s
	}
	if _, err := svc.ResolveSession(context.Background(), res.SessionToken); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("expired session err = %v, want ErrSessionNotFound", err)
	}
}

func TestTeam_CreateAndCrossTenantIsolation(t *testing.T) {
	svc, _, _ := newTestAuth()
	a, _ := svc.Register(context.Background(), "TenantA", "Alice", "alice@a.com", "supersecret", "")
	b, _ := svc.Register(context.Background(), "TenantB", "Bob", "bob@b.com", "supersecret", "")

	// Alice adds a member to tenant A.
	m, err := svc.CreateUser(context.Background(), a.Tenant.ID, "member@a.com", "Mel", domain.RoleMember, "supersecret")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Bob (tenant B) cannot modify tenant A's member → treated as not found.
	if _, err := svc.UpdateUserRole(context.Background(), b.Tenant.ID, m.ID, domain.RoleAdmin); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("cross-tenant update err = %v, want ErrUserNotFound", err)
	}
	if err := svc.DeleteUser(context.Background(), b.Tenant.ID, b.User.ID, m.ID); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("cross-tenant delete err = %v, want ErrUserNotFound", err)
	}
}

func TestTeam_LastOwnerProtected(t *testing.T) {
	svc, _, _ := newTestAuth()
	a, _ := svc.Register(context.Background(), "Acme", "Alice", "alice@a.com", "supersecret", "")

	// Cannot demote the last owner.
	if _, err := svc.UpdateUserRole(context.Background(), a.Tenant.ID, a.User.ID, domain.RoleMember); !errors.Is(err, domain.ErrLastOwner) {
		t.Fatalf("demote last owner err = %v, want ErrLastOwner", err)
	}
	// Cannot delete the last owner.
	if err := svc.DeleteUser(context.Background(), a.Tenant.ID, uuid.Nil, a.User.ID); !errors.Is(err, domain.ErrLastOwner) {
		t.Fatalf("delete last owner err = %v, want ErrLastOwner", err)
	}

	// Add a second owner, then the original can be demoted.
	o2, err := svc.CreateUser(context.Background(), a.Tenant.ID, "owner2@a.com", "Ozzy", domain.RoleOwner, "supersecret")
	if err != nil {
		t.Fatalf("create 2nd owner: %v", err)
	}
	if _, err := svc.UpdateUserRole(context.Background(), a.Tenant.ID, a.User.ID, domain.RoleMember); err != nil {
		t.Fatalf("demote with 2 owners: %v", err)
	}
	_ = o2
}

func TestTeam_SelfLockoutRejected(t *testing.T) {
	svc, _, _ := newTestAuth()
	a, _ := svc.Register(context.Background(), "Acme", "Alice", "alice@a.com", "supersecret", "")
	// Add a 2nd owner so the last-owner rule is not what blocks the delete.
	svc.CreateUser(context.Background(), a.Tenant.ID, "owner2@a.com", "Ozzy", domain.RoleOwner, "supersecret") //nolint:errcheck

	if err := svc.DeleteUser(context.Background(), a.Tenant.ID, a.User.ID, a.User.ID); !errors.Is(err, domain.ErrSelfLockout) {
		t.Fatalf("self-delete err = %v, want ErrSelfLockout", err)
	}
}
