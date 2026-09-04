// Package porttest provides one Unimplemented* type per interface in
// internal/core/port. Every method panics with the interface and method name,
// so a test double embeds the Unimplemented type and overrides only the
// methods the code under test actually calls: a call the test did not
// anticipate fails loudly instead of returning a silent nil.
//
//	type invoiceRepo struct {
//		porttest.UnimplementedInvoiceRepository
//		byID map[uuid.UUID]*domain.Invoice
//	}
//
//	func (r invoiceRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
//		return r.byID[id], nil
//	}
//
// The types are generated from the port package; regenerate after changing
// an interface with `go generate ./internal/core/port/porttest`.
//
//go:generate go run ./gen
package porttest
