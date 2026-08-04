// Package safego runs detached (fire-and-forget) goroutines with panic
// recovery. An unrecovered panic in ANY goroutine terminates the whole Go
// process — gin's Recovery middleware only covers the request goroutine, so a
// nil-deref or template panic inside a detached notifier/sync goroutine would
// take down the entire API and every in-flight request with it. Route every
// such goroutine through Go so a panic is logged and contained instead.
package safego

import (
	"log/slog"
	"runtime/debug"
)

// Go runs fn in a new goroutine, recovering any panic and logging it with the
// given name and stack instead of letting it crash the process. Use it for
// every detached goroutine whose panic no request-scoped recover would catch.
func Go(name string, fn func()) {
	go Run(name, fn)
}

// Run invokes fn synchronously with the same panic recovery as Go. It is the
// building block Go uses, exposed for callers that already hold a goroutine
// (e.g. a scheduler's per-tick body) and only need the recover wrapper.
func Run(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("recovered panic in detached goroutine — contained, not crashed",
				"goroutine", name, "panic", r, "stack", string(debug.Stack()))
		}
	}()
	fn()
}
