package handler

import "testing"

// TestIsGatewayPaymentID guards the ENG-188 fix: only real payment ids (pay_/
// pi_/ch_) may be written to invoices.gateway_payment_id — an order_* or empty
// value would poison it and break refunds.
func TestIsGatewayPaymentID(t *testing.T) {
	valid := []string{"pay_ABC123", "pi_123", "ch_456"}
	for _, id := range valid {
		if !isGatewayPaymentID(id) {
			t.Errorf("isGatewayPaymentID(%q) = false, want true", id)
		}
	}
	invalid := []string{"order_XYZ", "", "abc", "receipt_1"}
	for _, id := range invalid {
		if isGatewayPaymentID(id) {
			t.Errorf("isGatewayPaymentID(%q) = true, want false", id)
		}
	}
}
