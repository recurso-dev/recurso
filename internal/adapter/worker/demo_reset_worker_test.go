package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type spyResetter struct {
	calls int
	err   error
}

func (s *spyResetter) Reset(ctx context.Context) error {
	s.calls++
	return s.err
}

func TestDemoResetWorkerRunOnce(t *testing.T) {
	spy := &spyResetter{}
	w := NewDemoResetWorker(spy, time.Hour)
	w.RunOnce(context.Background())
	if spy.calls != 1 {
		t.Fatalf("resets = %d, want 1", spy.calls)
	}

	// Failures never panic or stop the worker; the next tick retries.
	spy.err = errors.New("db busy")
	w.RunOnce(context.Background())
	if spy.calls != 2 {
		t.Fatalf("resets = %d, want 2 (failure swallowed)", spy.calls)
	}
}

func TestDemoResetWorkerStopIdempotent(t *testing.T) {
	w := NewDemoResetWorker(&spyResetter{}, time.Hour)
	w.Start()
	w.Stop()
	w.Stop() // double-stop must not panic (stopOnce)
}
