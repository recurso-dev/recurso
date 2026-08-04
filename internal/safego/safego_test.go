package safego

import (
	"sync"
	"testing"
)

// TestRun_RecoversPanic proves the whole point: a panicking fn does NOT
// propagate (which would crash the process) — Run returns normally.
func TestRun_RecoversPanic(t *testing.T) {
	// If the panic escaped, the test binary would crash rather than fail.
	Run("test-panic", func() { panic("boom") })
	Run("test-nil-deref", func() {
		var p *int
		_ = *p // nil pointer dereference — panics
	})
}

// TestRun_RunsFn confirms the happy path still executes the body.
func TestRun_RunsFn(t *testing.T) {
	ran := false
	Run("test-ok", func() { ran = true })
	if !ran {
		t.Fatal("fn was not run")
	}
}

// TestGo_RunsAndContainsPanic runs both a panicking and a normal detached
// goroutine; the process surviving to the WaitGroup is the assertion.
func TestGo_RunsAndContainsPanic(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)
	Go("test-go-panic", func() { defer wg.Done(); panic("detached boom") })
	done := false
	Go("test-go-ok", func() { defer wg.Done(); done = true })
	wg.Wait()
	if !done {
		t.Fatal("normal detached goroutine did not run")
	}
}
