package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeWaitlist struct {
	added []string
}

func (f *fakeWaitlist) Add(ctx context.Context, email, name, company, source string) (bool, error) {
	f.added = append(f.added, email)
	return true, nil
}

func TestWaitlistJoin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeWaitlist{}
	h := NewWaitlistHandler(store)

	// Valid signup stores the email.
	c, w := newTestContext(http.MethodPost, "/waitlist", `{"email":"founder@example.com","source":"website-cta"}`)
	h.Join(c)
	if w.Code != http.StatusOK || len(store.added) != 1 {
		t.Fatalf("valid signup: code=%d stored=%v", w.Code, store.added)
	}

	// Honeypot hit reports success but stores nothing.
	c, w = newTestContext(http.MethodPost, "/waitlist", `{"email":"bot@example.com","website":"http://spam"}`)
	h.Join(c)
	if w.Code != http.StatusOK || len(store.added) != 1 {
		t.Fatalf("honeypot: code=%d stored=%v (must not store)", w.Code, store.added)
	}

	// Invalid email rejected.
	c, w = newTestContext(http.MethodPost, "/waitlist", `{"email":"not-an-email"}`)
	h.Join(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid email: code=%d, want 400", w.Code)
	}
}
