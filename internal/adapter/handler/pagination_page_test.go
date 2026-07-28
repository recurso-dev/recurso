package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParsePageLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		query          string
		wantLimit, off int
	}{
		{"", defaultPageLimit, 0},                       // no params → default, page 1
		{"?limit=20", 20, 0},                            // explicit limit
		{"?limit=20&page=3", 20, 40},                    // page 3 of 20 → offset 40
		{"?page=2", defaultPageLimit, defaultPageLimit}, // default limit, page 2
		{"?limit=99999", maxPageLimit, 0},               // over cap → capped, not downgraded
		{"?limit=0", defaultPageLimit, 0},               // invalid → default
		{"?limit=-5", defaultPageLimit, 0},              // negative → default
		{"?page=0", defaultPageLimit, 0},                // page 0 → page 1
		{"?page=-1", defaultPageLimit, 0},               // negative page → page 1
		{"?limit=abc", defaultPageLimit, 0},             // non-numeric → default
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/x"+c.query, nil)
		gotLimit, gotOff := parsePageLimit(ctx)
		if gotLimit != c.wantLimit || gotOff != c.off {
			t.Errorf("%q: got limit=%d offset=%d, want %d/%d", c.query, gotLimit, gotOff, c.wantLimit, c.off)
		}
	}
}
