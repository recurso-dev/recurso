// Package einvoice_eu holds EU e-invoice transport adapters (Track C). The mock
// stands in for a real Peppol Access Point / national platform until a provider
// is wired, so the document layer and service can be built and tested end to end
// without external credentials.
package einvoice_eu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/recurso-dev/recurso/internal/core/domain"
	"github.com/recurso-dev/recurso/internal/core/port"
)

// MockTransport accepts any well-formed document and reports it "sent",
// returning a deterministic message id derived from the document bytes (so a
// retransmission of the same document is recognizable in tests).
type MockTransport struct{}

func NewMockTransport() *MockTransport { return &MockTransport{} }

var _ port.EUInvoiceTransport = (*MockTransport)(nil)

func (m *MockTransport) Transmit(_ context.Context, _ domain.EUInvoiceSyntax, _ string, document []byte) (*domain.EUInvoiceTransmission, error) {
	sum := sha256.Sum256(document)
	return &domain.EUInvoiceTransmission{
		MessageID: "mock-" + hex.EncodeToString(sum[:8]),
		Status:    domain.EUInvoiceStatusSent,
	}, nil
}
