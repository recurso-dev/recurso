package service

import (
	"testing"

	"github.com/recurso-dev/recurso/internal/core/domain"
)

// TestAccountingPolicy_ModelVersion: the policy maps to the accounting-model
// version stamped on the schedules it produces (ADR-008) — accrual is V2, cash
// (and the zero value) is V1.
func TestAccountingPolicy_ModelVersion(t *testing.T) {
	if got := (AccountingPolicy{RevenueRecognition: RecognitionAccrual}).ModelVersion(); got != domain.AccountingModelV2 {
		t.Errorf("accrual ModelVersion = %d, want %d (V2)", got, domain.AccountingModelV2)
	}
	if got := (AccountingPolicy{RevenueRecognition: RecognitionCash}).ModelVersion(); got != domain.AccountingModelV1 {
		t.Errorf("cash ModelVersion = %d, want %d (V1)", got, domain.AccountingModelV1)
	}
	// Zero value (no recognition method set) is the conservative cash default.
	if got := (AccountingPolicy{}).ModelVersion(); got != domain.AccountingModelV1 {
		t.Errorf("zero-value ModelVersion = %d, want %d (V1)", got, domain.AccountingModelV1)
	}
}

// TestScheduleModelVersion: a schedule built while the invoice is unpaid is the
// accrual model (built at issuance, V2); one built once the invoice is paid is
// the cash model (built at payment, V1).
func TestScheduleModelVersion(t *testing.T) {
	paid := &domain.Invoice{Status: domain.InvoiceStatusPaid}
	if got := scheduleModelVersion(paid); got != domain.AccountingModelV1 {
		t.Errorf("paid invoice → version %d, want %d (V1 cash)", got, domain.AccountingModelV1)
	}
	open := &domain.Invoice{Status: domain.InvoiceStatusOpen}
	if got := scheduleModelVersion(open); got != domain.AccountingModelV2 {
		t.Errorf("open invoice → version %d, want %d (V2 accrual)", got, domain.AccountingModelV2)
	}
}
