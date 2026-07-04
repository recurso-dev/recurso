package accounting

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/swapnull-in/recur-so/internal/core/domain"
)

// TallyAdapter exports data as JSONL files for Tally ERP import.
// Tally uses XML/CSV import natively; this JSONL intermediate format
// can be converted by a separate tool or script.
type TallyAdapter struct {
	exportDir string
	mu        sync.Mutex
}

func NewTallyAdapter(exportDir string) *TallyAdapter {
	if exportDir == "" {
		exportDir = "/tmp/tally-exports"
	}
	_ = os.MkdirAll(exportDir, 0755)
	return &TallyAdapter{exportDir: exportDir}
}

// tallyRecord wraps an entity with metadata for JSONL export
type tallyRecord struct {
	Type      string      `json:"type"`
	ID        string      `json:"id"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

func (a *TallyAdapter) SyncCustomer(ctx context.Context, customer *domain.Customer) error {
	name := ""
	if customer.Name != nil {
		name = *customer.Name
	}

	record := tallyRecord{
		Type:      "customer",
		ID:        customer.ID.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"name":    name,
			"email":   customer.Email,
			"phone":   customer.Phone,
			"address": fmt.Sprintf("%s, %s, %s %s, %s", customer.BillingAddress.Line1, customer.BillingAddress.City, customer.BillingAddress.State, customer.BillingAddress.Zip, customer.BillingAddress.Country),
			"gstin":   ptrStr(customer.GSTIN),
			"tax_id":  ptrStr(customer.TaxID),
		},
	}

	return a.appendRecord(record)
}

func (a *TallyAdapter) SyncInvoice(ctx context.Context, invoice *domain.Invoice) error {
	record := tallyRecord{
		Type:      "invoice",
		ID:        invoice.ID.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"invoice_number": invoice.InvoiceNumber,
			"customer_id":    invoice.CustomerID.String(),
			"currency":       invoice.Currency,
			"subtotal":       float64(invoice.Subtotal) / 100,
			"tax_amount":     float64(invoice.TaxAmount) / 100,
			"igst":           float64(invoice.IGSTAmount) / 100,
			"cgst":           float64(invoice.CGSTAmount) / 100,
			"sgst":           float64(invoice.SGSTAmount) / 100,
			"total":          float64(invoice.Total) / 100,
			"status":         string(invoice.Status),
			"due_date":       invoice.DueDate.Format("2006-01-02"),
			"hsn_code":       invoice.HSNCode,
		},
	}

	return a.appendRecord(record)
}

func (a *TallyAdapter) SyncProduct(ctx context.Context, plan *domain.Plan) error {
	record := tallyRecord{
		Type:      "product",
		ID:        plan.ID.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"name":           plan.Name,
			"code":           plan.Code,
			"interval_unit":  string(plan.IntervalUnit),
			"interval_count": plan.IntervalCount,
			"active":         plan.Active,
		},
	}

	return a.appendRecord(record)
}

func (a *TallyAdapter) appendRecord(record tallyRecord) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	filename := fmt.Sprintf("tally-export-%s.jsonl", time.Now().Format("2006-01-02"))
	path := filepath.Join(a.exportDir, filename)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open tally export file: %w", err)
	}
	defer func() { _ = f.Close() }()

	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal tally record: %w", err)
	}

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("failed to write tally record: %w", err)
	}

	slog.Debug("Tally export record written", "type", record.Type, "id", record.ID)
	return nil
}

func ptrStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
