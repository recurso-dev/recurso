package accounting

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// rateLimitSleep waits between rate-limited attempts; a package variable so
// tests can record the waits instead of actually sleeping. Returns the
// context error if the wait is interrupted.
var rateLimitSleep = func(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

const (
	// rateLimitMaxRetries bounds how often a single call re-attempts after
	// 429 before surfacing the error (the sync log records it and the next
	// sweep re-pushes the entity).
	rateLimitMaxRetries = 2
	// rateLimitMaxWait caps a single Retry-After wait: Xero's per-minute
	// window never legitimately asks for more, and a misbehaving header
	// must not stall the sweep (which runs under a 15-minute budget).
	rateLimitMaxWait = 65 * time.Second
	// rateLimitDefaultWait is used when a 429 carries no usable Retry-After.
	rateLimitDefaultWait = 5 * time.Second
)

// doWithRateLimitRetry executes req, honoring provider rate limits: on 429
// it waits for the Retry-After the provider specifies (Xero: 60 calls/min
// per tenant) and retries a bounded number of times. Sweeps push hundreds
// of records serially, so honoring Retry-After self-paces the whole sweep
// to the provider's limit instead of burning the tail of a run into error
// rows (observed live: a forced Xero re-push 429'd every record past ~60).
//
// The request body (when present) must have been created by
// http.NewRequest* from an in-memory reader so GetBody can replay it.
func doWithRateLimitRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		attemptReq := req
		if attempt > 0 {
			// The previous attempt consumed the body; clone and replay it.
			attemptReq = req.Clone(ctx)
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				attemptReq.Body = body
			}
		}

		resp, err := http.DefaultClient.Do(attemptReq)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests || attempt >= rateLimitMaxRetries {
			return resp, nil
		}
		_ = resp.Body.Close()

		wait := rateLimitDefaultWait
		if s := resp.Header.Get("Retry-After"); s != "" {
			if n, convErr := strconv.Atoi(s); convErr == nil && n > 0 {
				wait = time.Duration(n) * time.Second
				if wait > rateLimitMaxWait {
					wait = rateLimitMaxWait
				}
			}
		}
		if err := rateLimitSleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}
