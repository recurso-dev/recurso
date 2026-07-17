package residency_test

// The residency guarantee's choke-point tests live next to each guarded
// component (telemetry, accounting). This package-level test file documents
// where the guarantee is enforced so a reviewer can audit the full egress
// surface from one place:
//
//   - internal/adapter/telemetry.NewFromEnv     → nil under self_hosted
//     (tested in telemetry/residency_test.go)
//   - service.AccountingService.getAdapterForConnection → refuses
//     QuickBooks/Xero under self_hosted (tested in
//     service/accounting_residency_test.go)
//   - cmd/api/main.go TaxJar wiring             → provider skipped
//   - cmd/api/main.go OAuth configs             → blanked, connect flow off
//
// Channels intentionally NOT guarded (operator-configured, functionally
// required, disclosed in docs/india-data-residency.md): payment gateways
// (Stripe/Razorpay), the GSP for e-invoicing, SMTP, outbound webhooks to the
// operator's own endpoints.
