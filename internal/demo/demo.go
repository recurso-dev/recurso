// Package demo implements the public-sandbox guarantee from
// docs/spec_demo_mode.md: with DEMO_MODE=true the instance is safe to
// expose to strangers — every outward-reaching adapter is forced to its
// mock/console implementation AT THE CONSTRUCTION SITE (a stray env var
// cannot re-enable egress), destructive settings are guarded, the data set
// auto-seeds on boot, and a reset worker restores it on an interval.
// The demo can never email a human, charge a card, submit an IRN, deliver
// a webhook, or reach a SaaS.
package demo

import (
	"os"
	"strings"
	"time"
)

// EnvVar selects demo mode.
const EnvVar = "DEMO_MODE"

// ResetIntervalEnvVar overrides the reset cadence (Go duration; default 1h).
const ResetIntervalEnvVar = "DEMO_RESET_INTERVAL"

// DefaultResetInterval is the wipe-and-reseed cadence when unset (D3).
const DefaultResetInterval = time.Hour

// Enabled reports whether the instance runs as a public demo sandbox
// (DEMO_MODE=true, case-insensitive).
func Enabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(EnvVar)), "true")
}

// ResetInterval returns the configured reset cadence, defaulting to
// DefaultResetInterval on absence or an unparseable value.
func ResetInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv(ResetIntervalEnvVar))
	if raw == "" {
		return DefaultResetInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < time.Minute {
		return DefaultResetInterval
	}
	return d
}
