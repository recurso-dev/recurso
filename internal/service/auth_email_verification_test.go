package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/recurso-dev/recurso/internal/core/domain"
)

// --- email-verification fakes ----------------------------------------------

type fakeVerifyRepo struct {
	byHash map[string]*domain.EmailVerificationToken
	byID   map[uuid.UUID]*domain.EmailVerificationToken
}

func newFakeVerifyRepo() *fakeVerifyRepo {
	return &fakeVerifyRepo{
		byHash: map[string]*domain.EmailVerificationToken{},
		byID:   map[uuid.UUID]*domain.EmailVerificationToken{},
	}
}
func (r *fakeVerifyRepo) Create(_ context.Context, t *domain.EmailVerificationToken) error {
	cp := *t
	r.byHash[t.TokenHash] = &cp
	r.byID[t.ID] = &cp
	return nil
}
func (r *fakeVerifyRepo) GetByTokenHash(_ context.Context, h string) (*domain.EmailVerificationToken, error) {
	t, ok := r.byHash[h]
	if !ok {
		return nil, domain.ErrInvalidVerificationToken
	}
	cp := *t
	return &cp, nil
}
func (r *fakeVerifyRepo) MarkUsed(_ context.Context, id uuid.UUID) (bool, error) {
	t, ok := r.byID[id]
	if !ok || t.UsedAt != nil {
		return false, nil
	}
	now := time.Now().UTC()
	t.UsedAt = &now
	r.byHash[t.TokenHash].UsedAt = &now
	return true, nil
}

type fakeVerifyMailer struct {
	sends   int
	lastTo  string
	lastURL string
}

func (m *fakeVerifyMailer) SendVerification(_ context.Context, to, verifyURL string) error {
	m.sends++
	m.lastTo = to
	m.lastURL = verifyURL
	return nil
}

// newVerifyAuth builds an AuthService with email verification wired, returning
// the collaborators the tests inspect.
func newVerifyAuth() (*AuthService, *fakeUserRepo, *fakeVerifyRepo, *fakeVerifyMailer) {
	ur := newFakeUserRepo()
	sr := newFakeSessionRepo()
	svc := NewAuthService(ur, sr, newFakeTenants(), time.Hour)
	verify := newFakeVerifyRepo()
	mailer := &fakeVerifyMailer{}
	// appBaseURL is normally set by ConfigurePasswordReset; set it so the verify
	// link is well-formed for tokenFromURL.
	svc.ConfigurePasswordReset(newFakeResetRepo(), &fakeMailer{}, "https://dash.recurso.test/")
	svc.ConfigureEmailVerification(verify, mailer)
	return svc, ur, verify, mailer
}

// --- tests -----------------------------------------------------------------

func TestRegister_IssuesVerificationEmailAndUserStartsUnverified(t *testing.T) {
	svc, ur, verify, mailer := newVerifyAuth()

	res, err := svc.Register(context.Background(), "Acme", "Alice", "alice@acme.com", "supersecret", "", "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if res.User.IsEmailVerified() {
		t.Fatal("new user should start unverified")
	}
	if mailer.sends != 1 {
		t.Fatalf("expected 1 verification email at signup, got %d", mailer.sends)
	}
	if mailer.lastTo != "alice@acme.com" {
		t.Fatalf("verification sent to %q", mailer.lastTo)
	}
	if len(verify.byID) != 1 {
		t.Fatalf("expected 1 verification token persisted, got %d", len(verify.byID))
	}
	// The persisted stored value is the token HASH, never the raw token.
	tok := tokenFromURL(t, mailer.lastURL)
	if _, ok := verify.byHash[tok]; ok {
		t.Fatal("raw token must never be stored; only its hash")
	}

	got, err := ur.GetByEmail(context.Background(), "alice@acme.com")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.IsEmailVerified() {
		t.Fatal("persisted user should be unverified before verifying")
	}
}

func TestVerifyEmail_HappyPath_MarksVerifiedAndIsSingleUse(t *testing.T) {
	svc, ur, _, mailer := newVerifyAuth()
	if _, err := svc.Register(context.Background(), "Acme", "Alice", "alice@acme.com", "supersecret", "", ""); err != nil {
		t.Fatalf("register: %v", err)
	}
	raw := tokenFromURL(t, mailer.lastURL)

	if err := svc.VerifyEmail(context.Background(), raw); err != nil {
		t.Fatalf("verify: %v", err)
	}
	got, _ := ur.GetByEmail(context.Background(), "alice@acme.com")
	if !got.IsEmailVerified() {
		t.Fatal("user should be verified after VerifyEmail")
	}

	// Second use of the same token must fail (single-use gate).
	if err := svc.VerifyEmail(context.Background(), raw); !errors.Is(err, domain.ErrInvalidVerificationToken) {
		t.Fatalf("replayed token: want ErrInvalidVerificationToken, got %v", err)
	}
}

func TestVerifyEmail_BadOrEmptyToken(t *testing.T) {
	svc, _, _, _ := newVerifyAuth()
	for _, raw := range []string{"", "not-a-real-token"} {
		if err := svc.VerifyEmail(context.Background(), raw); !errors.Is(err, domain.ErrInvalidVerificationToken) {
			t.Fatalf("token %q: want ErrInvalidVerificationToken, got %v", raw, err)
		}
	}
}

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	svc, _, verify, mailer := newVerifyAuth()
	if _, err := svc.Register(context.Background(), "Acme", "Alice", "alice@acme.com", "supersecret", "", ""); err != nil {
		t.Fatalf("register: %v", err)
	}
	raw := tokenFromURL(t, mailer.lastURL)

	// Force the stored token to be expired.
	past := time.Now().UTC().Add(-time.Hour)
	for _, tk := range verify.byHash {
		tk.ExpiresAt = past
	}
	for _, tk := range verify.byID {
		tk.ExpiresAt = past
	}

	if err := svc.VerifyEmail(context.Background(), raw); !errors.Is(err, domain.ErrInvalidVerificationToken) {
		t.Fatalf("expired token: want ErrInvalidVerificationToken, got %v", err)
	}
}

func TestRequestEmailVerification_NoopWhenAlreadyVerified(t *testing.T) {
	svc, ur, _, mailer := newVerifyAuth()
	if _, err := svc.Register(context.Background(), "Acme", "Alice", "alice@acme.com", "supersecret", "", ""); err != nil {
		t.Fatalf("register: %v", err)
	}
	sendsAfterSignup := mailer.sends

	user, _ := ur.GetByEmail(context.Background(), "alice@acme.com")
	if err := ur.MarkEmailVerified(context.Background(), user.ID); err != nil {
		t.Fatalf("mark verified: %v", err)
	}
	user, _ = ur.GetByEmail(context.Background(), "alice@acme.com")

	if err := svc.RequestEmailVerification(context.Background(), user); err != nil {
		t.Fatalf("request verification (already verified): %v", err)
	}
	if mailer.sends != sendsAfterSignup {
		t.Fatalf("already-verified resend should not send an email; sends went %d -> %d", sendsAfterSignup, mailer.sends)
	}
}

func TestRequestEmailVerification_NilUser(t *testing.T) {
	svc, _, _, _ := newVerifyAuth()
	if err := svc.RequestEmailVerification(context.Background(), nil); !errors.Is(err, domain.ErrUserRequired) {
		t.Fatalf("nil user: want ErrUserRequired, got %v", err)
	}
}
