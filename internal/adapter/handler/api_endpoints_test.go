package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/handler"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.RespondSuccess(c, gin.H{"status": "ok"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	dataObj, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object in response")
	}

	if dataObj["status"] != "ok" {
		t.Errorf("expected status ok, got %v", dataObj["status"])
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handler.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "invalid payload")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	errObj, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error object in response")
	}

	if errObj["code"] != "INVALID_INPUT" {
		t.Errorf("expected code INVALID_INPUT, got %v", errObj["code"])
	}
}

func TestRespondList(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	items := []string{"item1", "item2"}
	handler.RespondList(c, items, 2, 1, 10)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	metaObj, ok := resp["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected meta object in list response")
	}

	if metaObj["total"] != float64(2) {
		t.Errorf("expected total 2, got %v", metaObj["total"])
	}
}
