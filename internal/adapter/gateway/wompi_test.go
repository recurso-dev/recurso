package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWompiGateway_BaseURL(t *testing.T) {
	gwTest := NewWompiGateway("pub_test_123", "prv_test_456", "events_sec", "integ_sec")
	if gwTest.BaseURL() != wompiSandboxURL {
		t.Errorf("expected sandbox URL, got %s", gwTest.BaseURL())
	}

	gwProd := NewWompiGateway("pub_prod_123", "prv_prod_456", "events_sec", "integ_sec")
	if gwProd.BaseURL() != wompiProductionURL {
		t.Errorf("expected prod URL, got %s", gwProd.BaseURL())
	}
}

func TestWompiGateway_ComputeIntegritySignature(t *testing.T) {
	gw := NewWompiGateway("pub_test_x", "prv_test_x", "events_x", "my_integrity_secret")
	ref := "ORDER-1234"
	amount := int64(5000000)
	currency := "COP"

	expectedRaw := "ORDER-12345000000COPmy_integrity_secret"
	expectedHash := sha256.Sum256([]byte(expectedRaw))
	expected := hex.EncodeToString(expectedHash[:])

	actual := gw.ComputeIntegritySignature(ref, amount, currency)
	if actual != expected {
		t.Errorf("expected signature %s, got %s", expected, actual)
	}
}

func TestWompiGateway_VerifyWebhookChecksum(t *testing.T) {
	gw := NewWompiGateway("pub_test_x", "prv_test_x", "secret123", "integ")

	rawJSON := `{
		"event": "transaction.updated",
		"data": {
			"transaction": {
				"id": "tx_999",
				"status": "APPROVED",
				"amount_in_cents": 50000
			}
		},
		"timestamp": 1700000000,
		"signature": {
			"properties": ["transaction.id", "transaction.status", "transaction.amount_in_cents"],
			"checksum": "PLACEHOLDER"
		}
	}`

	var parsed map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		t.Fatal(err)
	}

	// Correct hash: "tx_999" + "APPROVED" + "50000" + "1700000000" + "secret123"
	concat := "tx_999APPROVED500001700000000secret123"
	sum := sha256.Sum256([]byte(concat))
	validChecksum := hex.EncodeToString(sum[:])

	// Update parsed checksum
	parsed["signature"].(map[string]any)["checksum"] = validChecksum

	if !gw.VerifyWebhookChecksum(parsed, validChecksum) {
		t.Errorf("expected valid checksum to pass")
	}

	if gw.VerifyWebhookChecksum(parsed, "bad_checksum") {
		t.Errorf("expected bad checksum to fail")
	}
}

func TestWompiGateway_CreateOrder(t *testing.T) {
	gw := NewWompiGateway("pub_test_x", "prv_test_x", "events_x", "my_integrity_secret")
	order, err := gw.CreateOrder(context.Background(), 25000, "COP", "INV-001", "inv_id_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Gateway != "wompi" {
		t.Errorf("expected gateway wompi, got %s", order.Gateway)
	}
	if order.Currency != "COP" {
		t.Errorf("expected currency COP, got %s", order.Currency)
	}

	_, errUSD := gw.CreateOrder(context.Background(), 25000, "USD", "INV-001", "inv_id_1")
	if errUSD == nil {
		t.Errorf("expected error for non-COP currency")
	}
}

func TestWompiGateway_GetAcceptanceToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-merchant-public-key") != "pub_test_abc" {
			t.Errorf("missing or invalid merchant key")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"presigned_acceptance": map[string]any{
					"acceptance_token": "token_jwt_xyz",
				},
			},
		})
	}))
	defer server.Close()

	gw := NewWompiGateway("pub_test_abc", "prv_test_abc", "events_x", "integ_x")
	gw.baseURL = server.URL

	token, err := gw.GetAcceptanceToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "token_jwt_xyz" {
		t.Errorf("expected token_jwt_xyz, got %s", token)
	}

	// Second call should return cached token without server hit
	server.Close()
	cachedToken, err := gw.GetAcceptanceToken(context.Background())
	if err != nil || cachedToken != "token_jwt_xyz" {
		t.Errorf("expected cached token")
	}
}
