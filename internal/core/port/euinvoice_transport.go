package port

import (
	"context"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// EUInvoiceTransport delivers a structured EU e-invoice document to its
// recipient — over the Peppol network (via an Access Point) or a national
// platform. It is the EU analogue of GSPAdapter (India IRN). Increment 1 ships a
// mock; a real provider plugs in behind this interface without touching the
// document layer.
type EUInvoiceTransport interface {
	// Transmit hands a generated document (the syntax's serialized bytes, e.g.
	// UBL 2.1 XML) to the transport for delivery to the recipient's endpoint,
	// returning the transport's message id and status. It must not mutate the
	// document.
	Transmit(ctx context.Context, syntax domain.EUInvoiceSyntax, recipientVATID string, document []byte) (*domain.EUInvoiceTransmission, error)
}
