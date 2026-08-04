package fx

import (
	"context"
	"sync"
	"testing"
)

// TestStaticRatesProvider_ConcurrentAccess proves the shared static provider —
// the one wired into every reporting service as s.fxFallback and hit from many
// concurrent report requests — is race-free under simultaneous reads and admin
// writes. Run under `-race` this is the regression lock for #21: the per-report
// fxNormalizer needs no lock precisely because the shared providers it wraps
// carry the mutex. A missing lock here would fail the race detector.
func TestStaticRatesProvider_ConcurrentAccess(t *testing.T) {
	p := NewStaticRatesProvider()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = p.GetRate(ctx, "USD", "INR")
			_, _ = p.GetRate(ctx, "EUR", "USD")
			_ = p.RateMetadata()
		}()
		go func(n int) {
			defer wg.Done()
			// Concurrent admin writes to the same map the readers touch.
			p.SetRate("USD", "ZZZ", float64(n)+1.5)
		}(i)
	}
	wg.Wait()
}
