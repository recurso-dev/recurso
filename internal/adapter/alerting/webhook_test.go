package alerting

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// captureServer records every JSON body POSTed to it.
type captureServer struct {
	mu     sync.Mutex
	bodies []map[string]string
	srv    *httptest.Server
}

func newCaptureServer(t *testing.T, status int) *captureServer {
	t.Helper()
	cs := &captureServer{}
	cs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		var m map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Errorf("payload is not a flat JSON object: %v (raw: %s)", err, raw)
		}
		cs.mu.Lock()
		cs.bodies = append(cs.bodies, m)
		cs.mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(cs.srv.Close)
	return cs
}

func (cs *captureServer) received() []map[string]string {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	out := make([]map[string]string, len(cs.bodies))
	copy(out, cs.bodies)
	return out
}

func TestWebhookAlerterJSONFormat(t *testing.T) {
	cs := newCaptureServer(t, http.StatusOK)
	a := NewWebhookAlerter(cs.srv.URL, FormatJSON)

	if err := a.Send(context.Background(), SeverityCritical, "postgres degraded", "connection refused"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	got := cs.received()
	if len(got) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(got))
	}
	p := got[0]
	if p["severity"] != "critical" {
		t.Errorf("severity = %q, want critical", p["severity"])
	}
	if p["title"] != "postgres degraded" {
		t.Errorf("title = %q", p["title"])
	}
	if p["body"] != "connection refused" {
		t.Errorf("body = %q", p["body"])
	}
	if p["source"] != "recurso" {
		t.Errorf("source = %q, want recurso", p["source"])
	}
	if _, err := time.Parse(time.RFC3339, p["timestamp"]); err != nil {
		t.Errorf("timestamp %q is not RFC3339: %v", p["timestamp"], err)
	}
}

func TestWebhookAlerterSlackFormat(t *testing.T) {
	cs := newCaptureServer(t, http.StatusOK)
	a := NewWebhookAlerter(cs.srv.URL, FormatSlack)

	if err := a.Send(context.Background(), SeverityWarning, "redis degraded", "dial tcp: refused"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	got := cs.received()
	if len(got) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(got))
	}
	p := got[0]
	if len(p) != 1 {
		t.Errorf("slack payload must contain only \"text\", got %v", p)
	}
	text, ok := p["text"]
	if !ok {
		t.Fatalf("slack payload missing \"text\": %v", p)
	}
	for _, want := range []string{"[WARNING]", "redis degraded", "dial tcp: refused"} {
		if !strings.Contains(text, want) {
			t.Errorf("slack text %q missing %q", text, want)
		}
	}
}

func TestWebhookAlerterUnknownFormatFallsBackToJSON(t *testing.T) {
	cs := newCaptureServer(t, http.StatusOK)
	a := NewWebhookAlerter(cs.srv.URL, "carrier-pigeon")

	if err := a.Send(context.Background(), SeverityInfo, "t", "b"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	got := cs.received()
	if len(got) != 1 || got[0]["source"] != "recurso" {
		t.Fatalf("expected JSON-format payload, got %v", got)
	}
}

func TestWebhookAlerterNon2xxReturnsError(t *testing.T) {
	cs := newCaptureServer(t, http.StatusInternalServerError)
	a := NewWebhookAlerter(cs.srv.URL, FormatJSON)

	if err := a.Send(context.Background(), SeverityCritical, "t", "b"); err == nil {
		t.Fatal("expected error on 500 response, got nil")
	}
	// Exactly one POST — no retries, ever.
	if n := len(cs.received()); n != 1 {
		t.Fatalf("expected exactly 1 delivery attempt, got %d", n)
	}
}

func TestWebhookAlerterUnreachableReturnsError(t *testing.T) {
	a := NewWebhookAlerter("http://127.0.0.1:1/nope", FormatJSON)
	if err := a.Send(context.Background(), SeverityCritical, "t", "b"); err == nil {
		t.Fatal("expected error for unreachable webhook, got nil")
	}
}

func TestNewFromEnvUnsetIsNoop(t *testing.T) {
	t.Setenv("ALERT_WEBHOOK_URL", "")
	t.Setenv("ALERT_WEBHOOK_FORMAT", "")

	a := NewFromEnv()
	if _, ok := a.(NoopAlerter); !ok {
		t.Fatalf("expected NoopAlerter when ALERT_WEBHOOK_URL is unset, got %T", a)
	}
	if err := a.Send(context.Background(), SeverityCritical, "t", "b"); err != nil {
		t.Fatalf("noop Send must never error: %v", err)
	}
}

func TestNewFromEnvSet(t *testing.T) {
	cs := newCaptureServer(t, http.StatusOK)
	t.Setenv("ALERT_WEBHOOK_URL", cs.srv.URL)
	t.Setenv("ALERT_WEBHOOK_FORMAT", "slack")

	a := NewFromEnv()
	wa, ok := a.(*WebhookAlerter)
	if !ok {
		t.Fatalf("expected *WebhookAlerter, got %T", a)
	}
	if wa.format != FormatSlack {
		t.Errorf("format = %q, want slack", wa.format)
	}
	if err := a.Send(context.Background(), SeverityInfo, "t", "b"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if n := len(cs.received()); n != 1 {
		t.Fatalf("expected 1 delivery, got %d", n)
	}
}
