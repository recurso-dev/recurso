package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// The previously-unbounded list endpoints are clamped def=max=1000: absent or
// junk input becomes 1000 (a pure DoS-bound), an explicit smaller limit is
// honored, and past-the-cap requests are clamped, with offset for paging.
func TestParseLimitOffset(t *testing.T) {
	cases := []struct {
		query      string
		wantLimit  int
		wantOffset int
	}{
		{"", 1000, 0},
		{"?limit=50", 50, 0},
		{"?limit=50&offset=100", 50, 100},
		{"?limit=999999", 1000, 0},
		{"?limit=-5&offset=-3", 1000, 0},
		{"?limit=junk&offset=junk", 1000, 0},
	}
	for _, tc := range cases {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/x"+tc.query, nil)
		limit, offset := parseLimitOffset(c, 1000, 1000)
		if limit != tc.wantLimit || offset != tc.wantOffset {
			t.Errorf("%q -> %d/%d, want %d/%d", tc.query, limit, offset, tc.wantLimit, tc.wantOffset)
		}
	}
}

// With def < max, an over-cap request is clamped to max rather than reset to
// def: a caller asking for 300 of a 250-cap list gets 250, not 50.
func TestClampLimitOffset_OverCapClampsToMax(t *testing.T) {
	cases := []struct{ limit, offset, wantLimit, wantOffset int }{
		{300, 0, 250, 0},
		{0, 10, 50, 10},
		{-1, -1, 50, 0},
		{250, 5, 250, 5},
		{7, 0, 7, 0},
	}
	for _, tc := range cases {
		limit, offset := clampLimitOffset(tc.limit, tc.offset, 50, 250)
		if limit != tc.wantLimit || offset != tc.wantOffset {
			t.Errorf("clampLimitOffset(%d,%d,50,250) = %d/%d, want %d/%d", tc.limit, tc.offset, limit, offset, tc.wantLimit, tc.wantOffset)
		}
	}
}
