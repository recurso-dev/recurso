package httperr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRespondCanonicalShape(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Respond(c, http.StatusNotFound, CodeNotFound, "invoice not found")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Fatalf("code = %q, want %q", body.Error.Code, "not_found")
	}
	if body.Error.Message != "invoice not found" {
		t.Fatalf("message = %q, want %q", body.Error.Message, "invoice not found")
	}
}

func TestAbortStopsChain(t *testing.T) {
	w := httptest.NewRecorder()
	r := gin.New()
	handlerRan := false
	r.GET("/x", func(c *gin.Context) {
		Abort(c, http.StatusUnauthorized, CodeUnauthorized, "nope")
	}, func(c *gin.Context) {
		handlerRan = true
	})
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if handlerRan {
		t.Fatal("Abort must stop the handler chain")
	}
}

func TestCodeForStatus(t *testing.T) {
	cases := map[int]string{
		http.StatusBadRequest:          CodeValidationFailed,
		http.StatusUnprocessableEntity: CodeValidationFailed,
		http.StatusUnauthorized:        CodeUnauthorized,
		http.StatusForbidden:           CodeForbidden,
		http.StatusNotFound:            CodeNotFound,
		http.StatusConflict:            CodeConflict,
		http.StatusTooManyRequests:     CodeRateLimited,
		http.StatusInternalServerError: CodeInternalError,
		http.StatusServiceUnavailable:  CodeInternalError,
	}
	for status, want := range cases {
		if got := CodeForStatus(status); got != want {
			t.Errorf("CodeForStatus(%d) = %q, want %q", status, got, want)
		}
	}
}
