package service

import "testing"

// TestDeferredClosing derives closing from the period movement:
// opening + added - released.
func TestDeferredClosing(t *testing.T) {
	cases := []struct {
		opening, added, released, want int64
	}{
		{0, 12000, 2000, 10000},  // fresh period: deferred 12k, recognized 2k
		{10000, 0, 10000, 0},     // fully released
		{5000, 3000, 1000, 7000}, // net growth
		{0, 0, 0, 0},             // no activity
	}
	for _, c := range cases {
		if got := deferredClosing(c.opening, c.added, c.released); got != c.want {
			t.Errorf("deferredClosing(%d,%d,%d) = %d, want %d", c.opening, c.added, c.released, got, c.want)
		}
	}
}
