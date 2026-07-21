package handler

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func ctxWithQuery(rawQuery string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/x?"+rawQuery, nil)
	return c
}

func TestParseLiabilityPeriod(t *testing.T) {
	curYear := time.Now().UTC().Year()

	t.Run("default is current calendar year", func(t *testing.T) {
		from, to, err := parseLiabilityPeriod(ctxWithQuery(""))
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if from != time.Date(curYear, 1, 1, 0, 0, 0, 0, time.UTC) ||
			to != time.Date(curYear+1, 1, 1, 0, 0, 0, 0, time.UTC) {
			t.Errorf("default window = %v..%v", from, to)
		}
	})

	t.Run("year selects that calendar year", func(t *testing.T) {
		from, to, err := parseLiabilityPeriod(ctxWithQuery("year=2024"))
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if from.Year() != 2024 || to.Year() != 2025 {
			t.Errorf("year=2024 -> %v..%v", from, to)
		}
	})

	t.Run("from+to explicit window (to exclusive)", func(t *testing.T) {
		from, to, err := parseLiabilityPeriod(ctxWithQuery("from=2026-04-01&to=2026-07-01"))
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if from != time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC) ||
			to != time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) {
			t.Errorf("window = %v..%v", from, to)
		}
	})

	for _, tc := range []struct {
		name, q string
	}{
		{"from without to", "from=2026-04-01"},
		{"to without from", "to=2026-07-01"},
		{"bad from", "from=nope&to=2026-07-01"},
		{"to not after from", "from=2026-07-01&to=2026-04-01"},
		{"invalid year", "year=1999"},
	} {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			if _, _, err := parseLiabilityPeriod(ctxWithQuery(tc.q)); err == nil {
				t.Errorf("want error for %q", tc.q)
			}
		})
	}
}
