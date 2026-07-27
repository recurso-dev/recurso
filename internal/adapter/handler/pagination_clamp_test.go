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
