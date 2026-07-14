package service

import (
	"context"
	"strings"
	"testing"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestInviteUser_CreatesUnusableAccountThenAcceptEnablesLogin proves the whole
// invite flow: the admin never sets a password, the invited account can't be
// logged into until the invitee opens the link and sets their own password, and
// then login works.
func TestInviteUser_CreatesUnusableAccountThenAcceptEnablesLogin(t *testing.T) {
	svc, reset, _, _, mailer, _ := newPhase2Auth()
	ctx := context.Background()

	reg, err := svc.Register(ctx, "Acme", "Owner", "owner@acme.com", "ownerpassword", "")
	if err != nil {
		t.Fatalf("register owner: %v", err)
	}

	invited, err := svc.InviteUser(ctx, reg.User.TenantID, reg.User.Role, "bob@acme.com", "Bob", domain.RoleMember)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if invited.Email != "bob@acme.com" || invited.Role != domain.RoleMember || invited.TenantID != reg.User.TenantID {
		t.Fatalf("invited user = %+v, want bob@acme.com / member / same tenant", invited)
	}

	// One single-use token, and an invite email linking to the accept page.
	if !strings.HasPrefix(mailer.lastInviteURL, "https://dash.recurso.test/accept-invite?token=") {
		t.Fatalf("invite url = %q, want accept-invite link", mailer.lastInviteURL)
	}
	if len(reset.byHash) != 1 {
		t.Fatalf("invite tokens created = %d, want 1", len(reset.byHash))
	}

	// The account must NOT be usable before acceptance: the admin-side password
	// is random and unknown, so no guessable password logs in.
	if _, err := svc.Login(ctx, "bob@acme.com", "password", ""); err == nil {
		t.Fatal("invited account logged in before accepting — must be unusable")
	}

	// Accept: the invitee opens the link and sets their own password.
	token := tokenFromURL(t, mailer.lastInviteURL)
	if err := svc.ResetPassword(ctx, token, "bobs-own-password"); err != nil {
		t.Fatalf("accept invite (set password): %v", err)
	}

	// Now login with the invitee's chosen password succeeds.
	res, err := svc.Login(ctx, "bob@acme.com", "bobs-own-password", "")
	if err != nil {
		t.Fatalf("login after accepting invite: %v", err)
	}
	if res.User.Email != "bob@acme.com" {
		t.Fatalf("logged in as %q, want bob@acme.com", res.User.Email)
	}

	// The invite token is single-use — it can't be replayed to reset again.
	if err := svc.ResetPassword(ctx, token, "another-password"); err == nil {
		t.Fatal("invite token was reusable — must be single-use")
	}
}

// TestInviteUser_RejectsOwnerFromNonOwner keeps CreateUser's privilege guard:
// an admin cannot invite an owner.
func TestInviteUser_RejectsOwnerFromNonOwner(t *testing.T) {
	svc, _, _, _, _, _ := newPhase2Auth()
	ctx := context.Background()

	reg, err := svc.Register(ctx, "Acme", "Owner", "owner@acme.com", "ownerpassword", "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := svc.InviteUser(ctx, reg.User.TenantID, domain.RoleAdmin, "evil@acme.com", "Evil", domain.RoleOwner); err != domain.ErrOwnerRoleRequired {
		t.Fatalf("admin inviting an owner: err = %v, want ErrOwnerRoleRequired", err)
	}
}
