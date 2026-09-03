package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/httperr"
)

func init() { gin.SetMode(gin.TestMode) }

func decodeErrorEnvelope(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v (body %q)", err, w.Body.String())
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected {error:{code,message}} envelope, got %q", w.Body.String())
	}
	return errObj
}

// The canonical error envelope is the one shape every client depends on.
func TestRespondError_CanonicalEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	respondError(c, http.StatusBadRequest, codeValidationFailed, "invalid payload")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	errObj := decodeErrorEnvelope(t, w)
	if errObj["code"] != httperr.CodeValidationFailed || errObj["message"] != "invalid payload" {
		t.Fatalf("envelope = %v", errObj)
	}
}

// respondErrorStatus derives the machine code from the status; the webhook
// dedup paths rely on 503 mapping to a stable code.
func TestRespondErrorStatus_DerivesCode(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	respondErrorStatus(c, http.StatusServiceUnavailable, "dedup store unavailable; retry")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	errObj := decodeErrorEnvelope(t, w)
	if errObj["code"] != httperr.CodeForStatus(http.StatusServiceUnavailable) {
		t.Fatalf("code = %v, want %v", errObj["code"], httperr.CodeForStatus(http.StatusServiceUnavailable))
	}
}

// respondInternalError must never echo the underlying error to the client.
func TestRespondInternalError_DoesNotLeak(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/x", nil)

	respondInternalError(c, errors.New("pq: relation \"secret_table\" does not exist"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	errObj := decodeErrorEnvelope(t, w)
	if errObj["message"] != "internal error" {
		t.Fatalf("leaked driver detail: %v", errObj["message"])
	}
}
