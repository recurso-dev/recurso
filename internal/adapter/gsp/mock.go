package gsp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/swapnull-in/recur-so/internal/core/domain"
	"github.com/swapnull-in/recur-so/internal/core/port"
)

type MockGSPAdapter struct{}

func NewMockGSPAdapter() *MockGSPAdapter {
	return &MockGSPAdapter{}
}

func (m *MockGSPAdapter) GenerateIRN(ctx context.Context, invoice *domain.Invoice) (*port.EInvoiceResponse, error) {
	// Simulate API latency
	time.Sleep(100 * time.Millisecond)

	// Simulate success for India B2B
	// In a real adapter, we'd validate here, but the service should decide when to call.
	
	// Generate dummy 64-char hex IRN
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	irn := hex.EncodeToString(bytes)

	// Generate dummy Signed QR Code (JWT-like)
	signedQR := fmt.Sprintf("eyJhbGciOiJIUzI1NiJ9.irn:%s.invoice:%s", irn, invoice.InvoiceNumber)

	return &port.EInvoiceResponse{
		IRN:          irn,
		SignedQRCode: signedQR,
		Status:       "GENERATED",
		AckNo:        fmt.Sprintf("ACK%d", time.Now().UnixNano()),
	}, nil
}

func (m *MockGSPAdapter) GenerateIRNFull(ctx context.Context, req *port.EInvoiceRequest) (*port.EInvoiceResponse, error) {
	return m.GenerateIRN(ctx, req.Invoice)
}

func (m *MockGSPAdapter) CancelIRN(ctx context.Context, irn string, reason string) error {
	return nil
}

func (m *MockGSPAdapter) GetIRNByDocDetails(ctx context.Context, docType, docNum, docDate string) (*port.EInvoiceResponse, error) {
	return &port.EInvoiceResponse{
		Status:       "GENERATED",
		IRN:          "mock-irn-for-" + docNum,
		AckNo:        "MOCK-ACK",
		AckDate:      time.Now().Format("02/01/2006 15:04:05"),
		SignedQRCode: "mock-qr-code",
	}, nil
}
