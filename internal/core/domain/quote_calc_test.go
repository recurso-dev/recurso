package domain

import "testing"

func TestQuoteCalculateTotals(t *testing.T) {
	t.Run("itemized lines derive amount from quantity*unit_price", func(t *testing.T) {
		q := &Quote{LineItems: []LineItem{
			{Quantity: 2, UnitPrice: 1500}, // 3000
			{Quantity: 3, UnitPrice: 1000}, // 3000
		}, TaxAmount: 500, DiscountAmount: 100}
		q.CalculateTotals()
		if q.LineItems[0].Amount != 3000 || q.Subtotal != 6000 {
			t.Fatalf("itemized: line0=%d subtotal=%d, want 3000/6000", q.LineItems[0].Amount, q.Subtotal)
		}
		if q.Total != 6400 { // 6000 + 500 - 100
			t.Fatalf("total=%d, want 6400", q.Total)
		}
	})

	t.Run("lump-sum line honors direct amount", func(t *testing.T) {
		// A line sent with only `amount` (quantity/unit_price zero) must not be
		// silently zeroed — the API-created-quote-total-stays-0 bug (S4).
		q := &Quote{LineItems: []LineItem{{Description: "Consulting", Amount: 50000}}}
		q.CalculateTotals()
		if q.Subtotal != 50000 || q.Total != 50000 {
			t.Fatalf("lump-sum: subtotal=%d total=%d, want 50000/50000", q.Subtotal, q.Total)
		}
	})

	t.Run("mixed itemized and lump-sum", func(t *testing.T) {
		q := &Quote{LineItems: []LineItem{
			{Quantity: 2, UnitPrice: 1000},           // 2000 itemized
			{Description: "Setup fee", Amount: 5000}, // lump sum
		}}
		q.CalculateTotals()
		if q.Subtotal != 7000 {
			t.Fatalf("mixed subtotal=%d, want 7000", q.Subtotal)
		}
	})
}
