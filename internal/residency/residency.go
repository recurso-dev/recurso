// Package residency implements the self-hosted data-residency guarantee from
// docs/spec_india_decisive.md: with RESIDENCY_MODE=self_hosted, financial data
// leaves the deployment only through channels the operator explicitly
// configures for their own statutory/collection needs (payment gateways, the
// GSP for e-invoicing, their SMTP relay, their webhook endpoints). Every
// optional third-party SaaS egress — telemetry, accounting-SaaS sync, the
// external sales-tax API — is hard-disabled at its construction site, so a
// stray env var or a connection row left in the database cannot re-enable it.
// The operator-facing statement lives in docs/india-data-residency.md.
package residency

import (
	"os"
	"strings"
)

// EnvVar is the environment variable that selects the residency mode.
const EnvVar = "RESIDENCY_MODE"

// ModeSelfHosted is the value that activates the guarantee.
const ModeSelfHosted = "self_hosted"

// SelfHosted reports whether the deployment runs under the self-hosted
// residency guarantee (RESIDENCY_MODE=self_hosted, case-insensitive).
func SelfHosted() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(EnvVar)), ModeSelfHosted)
}
