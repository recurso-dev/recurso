package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/recurso-dev/recurso/internal/adapter/alerting"
)

// alertCapture is an httptest server that records every alert POSTed to it,
// exercising the real WebhookAlerter end-to-end.
type alertCapture struct {
	mu     sync.Mutex
	alerts []map[string]string
	srv    *httptest.Server
}

func newAlertCapture(t *testing.T) *alertCapture {
	t.Helper()
	ac := &alertCapture{}
	ac.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var m map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Errorf("bad alert payload: %v (raw: %s)", err, raw)
		}
		ac.mu.Lock()
		ac.alerts = append(ac.alerts, m)
		ac.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ac.srv.Close)
	return ac
}

func (ac *alertCapture) received() []map[string]string {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	out := make([]map[string]string, len(ac.alerts))
	copy(out, ac.alerts)
	return out
}

// flakyComponent is a component whose health the test controls.
type flakyComponent struct {
	mu   sync.Mutex
	down bool
}

func (f *flakyComponent) set(down bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.down = down
}

func (f *flakyComponent) check(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.down {
		return errors.New("connection refused")
	}
	return nil
}

func TestHealthAlertTransitionsFireOnce(t *testing.T) {
	ac := newAlertCapture(t)
	alerter := alerting.NewWebhookAlerter(ac.srv.URL, alerting.FormatJSON)

	pg := &flakyComponent{}
	s := NewHealthAlertScheduler(alerter, []ComponentCheck{
		{Name: "postgres", Severity: alerting.SeverityCritical, Check: pg.check},
	}, time.Minute)

	ctx := context.Background()

	// Healthy steady state: no alerts.
	s.evaluate(ctx)
	s.evaluate(ctx)
	if n := len(ac.received()); n != 0 {
		t.Fatalf("healthy steady state fired %d alerts, want 0", n)
	}

	// up -> down: exactly one "degraded" alert.
	pg.set(true)
	s.evaluate(ctx)
	got := ac.received()
	if len(got) != 1 {
		t.Fatalf("up->down fired %d alerts, want 1", len(got))
	}
	if got[0]["severity"] != "critical" {
		t.Errorf("severity = %q, want critical", got[0]["severity"])
	}
	if got[0]["title"] != "postgres degraded" {
		t.Errorf("title = %q, want \"postgres degraded\"", got[0]["title"])
	}

	// Down steady state: no repeat-fire on subsequent ticks.
	s.evaluate(ctx)
	s.evaluate(ctx)
	s.evaluate(ctx)
	if n := len(ac.received()); n != 1 {
		t.Fatalf("down steady state repeat-fired: %d alerts, want 1", n)
	}

	// down -> up: exactly one "recovered" alert, severity info.
	pg.set(false)
	s.evaluate(ctx)
	got = ac.received()
	if len(got) != 2 {
		t.Fatalf("down->up fired %d total alerts, want 2", len(got))
	}
	if got[1]["title"] != "postgres recovered" {
		t.Errorf("title = %q, want \"postgres recovered\"", got[1]["title"])
	}
	if got[1]["severity"] != "info" {
		t.Errorf("recovery severity = %q, want info", got[1]["severity"])
	}

	// Recovered steady state: silent again.
	s.evaluate(ctx)
	if n := len(ac.received()); n != 2 {
		t.Fatalf("recovered steady state fired extra alerts: %d, want 2", n)
	}
}

func TestHealthAlertDownAtFirstEvaluationFires(t *testing.T) {
	// A dependency already down at boot (e.g. TigerBeetle never connected)
	// fires one degraded alert on the first evaluation, then stays silent.
	ac := newAlertCapture(t)
	alerter := alerting.NewWebhookAlerter(ac.srv.URL, alerting.FormatJSON)

	tb := &flakyComponent{down: true}
	s := NewHealthAlertScheduler(alerter, []ComponentCheck{
		{Name: "tigerbeetle", Severity: alerting.SeverityWarning, Check: tb.check},
	}, time.Minute)

	ctx := context.Background()
	s.evaluate(ctx)
	s.evaluate(ctx)

	got := ac.received()
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 alert, got %d", len(got))
	}
	if got[0]["title"] != "tigerbeetle degraded" || got[0]["severity"] != "warning" {
		t.Errorf("unexpected alert: %v", got[0])
	}
}

func TestHealthAlertPerComponentState(t *testing.T) {
	// One component going down must not mask or reset another's state.
	ac := newAlertCapture(t)
	alerter := alerting.NewWebhookAlerter(ac.srv.URL, alerting.FormatJSON)

	pg := &flakyComponent{}
	redis := &flakyComponent{}
	s := NewHealthAlertScheduler(alerter, []ComponentCheck{
		{Name: "postgres", Severity: alerting.SeverityCritical, Check: pg.check},
		{Name: "redis", Severity: alerting.SeverityWarning, Check: redis.check},
	}, time.Minute)

	ctx := context.Background()
	redis.set(true)
	s.evaluate(ctx) // redis degraded (warning)
	pg.set(true)
	s.evaluate(ctx) // postgres degraded (critical); redis silent (steady down)

	got := ac.received()
	if len(got) != 2 {
		t.Fatalf("expected 2 alerts, got %d: %v", len(got), got)
	}
	if got[0]["title"] != "redis degraded" || got[0]["severity"] != "warning" {
		t.Errorf("first alert = %v", got[0])
	}
	if got[1]["title"] != "postgres degraded" || got[1]["severity"] != "critical" {
		t.Errorf("second alert = %v", got[1])
	}
}

func TestHealthAlertNoopAlerterWhenUnconfigured(t *testing.T) {
	// With the no-op alerter (ALERT_WEBHOOK_URL unset) evaluation still
	// works and never panics or blocks.
	pg := &flakyComponent{down: true}
	s := NewHealthAlertScheduler(alerting.NoopAlerter{}, []ComponentCheck{
		{Name: "postgres", Severity: alerting.SeverityCritical, Check: pg.check},
	}, 0) // also exercises the 60s default interval
	if s.interval != 60*time.Second {
		t.Errorf("default interval = %v, want 60s", s.interval)
	}
	s.evaluate(context.Background())
	s.evaluate(context.Background())
}

func TestHealthAlertStartStop(t *testing.T) {
	ac := newAlertCapture(t)
	alerter := alerting.NewWebhookAlerter(ac.srv.URL, alerting.FormatJSON)

	pg := &flakyComponent{down: true}
	s := NewHealthAlertScheduler(alerter, []ComponentCheck{
		{Name: "postgres", Severity: alerting.SeverityCritical, Check: pg.check},
	}, 10*time.Millisecond)

	s.Start()
	// Let several ticks elapse while the component stays down.
	deadline := time.After(2 * time.Second)
	for len(ac.received()) == 0 {
		select {
		case <-deadline:
			t.Fatal("no alert fired within 2s of Start")
		case <-time.After(5 * time.Millisecond):
		}
	}
	time.Sleep(100 * time.Millisecond) // many more ticks, still down

	s.Stop()
	s.Stop() // sync.Once: second Stop must not panic

	if n := len(ac.received()); n != 1 {
		t.Fatalf("ticker repeat-fired on steady state: %d alerts, want 1", n)
	}
}
