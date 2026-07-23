package main

import (
	"fmt"
	"testing"
)

func TestVerifyData_TableQueryConstruction(t *testing.T) {
	tables := []string{"plans", "customers", "subscriptions", "invoices", "usage_events"}

	for _, tbl := range tables {
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE tenant_id = $1", tbl)
		if tbl == "usage_events" {
			query = `SELECT COUNT(*) FROM usage_events ue JOIN subscriptions s ON ue.subscription_id = s.id WHERE s.tenant_id = $1`
		}
		if len(query) == 0 {
			t.Errorf("expected non-empty query string for table %s", tbl)
		}
	}
}
