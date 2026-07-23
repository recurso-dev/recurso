package demo_test

import (
	"testing"
	"time"

	"github.com/recurso-dev/recurso/internal/demo"
)

func TestDemo_Enabled(t *testing.T) {
	t.Setenv(demo.EnvVar, "true")
	if !demo.Enabled() {
		t.Error("expected Enabled() to return true when DEMO_MODE=true")
	}

	t.Setenv(demo.EnvVar, "false")
	if demo.Enabled() {
		t.Error("expected Enabled() to return false when DEMO_MODE=false")
	}

	t.Setenv(demo.EnvVar, "")
	if demo.Enabled() {
		t.Error("expected Enabled() to return false when DEMO_MODE is unset")
	}
}

func TestDemo_ResetInterval(t *testing.T) {
	t.Setenv(demo.ResetIntervalEnvVar, "")
	if got := demo.ResetInterval(); got != demo.DefaultResetInterval {
		t.Errorf("expected default %v, got %v", demo.DefaultResetInterval, got)
	}

	t.Setenv(demo.ResetIntervalEnvVar, "30m")
	if got := demo.ResetInterval(); got != 30*time.Minute {
		t.Errorf("expected 30m, got %v", got)
	}

	// Invalid or too small durations revert to default
	t.Setenv(demo.ResetIntervalEnvVar, "invalid")
	if got := demo.ResetInterval(); got != demo.DefaultResetInterval {
		t.Errorf("expected default %v for invalid, got %v", demo.DefaultResetInterval, got)
	}
}
