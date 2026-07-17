package telemetry

import (
	"testing"

	"github.com/recurso-dev/recurso/internal/residency"
)

// The residency guarantee outranks the telemetry opt-in: even with
// TELEMETRY_OPTIN=true, RESIDENCY_MODE=self_hosted must yield a nil client
// (no network calls, no rows written).
func TestNewFromEnv_ResidencyOverridesOptIn(t *testing.T) {
	t.Setenv("TELEMETRY_OPTIN", "true")
	t.Setenv(residency.EnvVar, residency.ModeSelfHosted)
	if c := NewFromEnv(nil, "test"); c != nil {
		t.Fatal("NewFromEnv must return nil under RESIDENCY_MODE=self_hosted even when opted in")
	}
}
