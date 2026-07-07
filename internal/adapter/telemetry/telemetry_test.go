package telemetry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// memStore is an in-memory Store fake with the same flip-once semantics as
// the Postgres implementation.
type memStore struct {
	mu    sync.Mutex
	id    uuid.UUID
	flags map[string]bool
}

func newMemStore() *memStore {
	return &memStore{id: uuid.New(), flags: map[string]bool{}}
}

func (m *memStore) EnsureInstance(context.Context) (uuid.UUID, map[string]bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]bool, len(m.flags))
	for k, v := range m.flags {
		out[k] = v
	}
	return m.id, out, nil
}

func (m *memStore) MarkMilestone(_ context.Context, name string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flags[name] {
		return false, nil
	}
	m.flags[name] = true
	return true, nil
}

// eventServer records every event POSTed to it and signals arrivals.
type eventServer struct {
	mu     sync.Mutex
	events []map[string]any
	got    chan map[string]any
	srv    *httptest.Server
}

func newEventServer(t *testing.T) *eventServer {
	t.Helper()
	es := &eventServer{got: make(chan map[string]any, 64)}
	es.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			return
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Errorf("payload is not a JSON object: %v (raw: %s)", err, raw)
			return
		}
		es.mu.Lock()
		es.events = append(es.events, m)
		es.mu.Unlock()
		es.got <- m
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(es.srv.Close)
	return es
}

func (es *eventServer) wait(t *testing.T, event string) map[string]any {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case m := <-es.got:
			if m["event"] == event {
				return m
			}
		case <-deadline:
			t.Fatalf("timed out waiting for event %q (received: %v)", event, es.received())
		}
	}
}

func (es *eventServer) received() []map[string]any {
	es.mu.Lock()
	defer es.mu.Unlock()
	out := make([]map[string]any, len(es.events))
	copy(out, es.events)
	return out
}

func (es *eventServer) countByName(name string) int {
	n := 0
	for _, e := range es.received() {
		if e["event"] == name {
			n++
		}
	}
	return n
}

func newTestClient(t *testing.T, es *eventServer, store Store, counts CountsFunc) *Client {
	t.Helper()
	c := New(Config{
		Enabled:           true,
		Endpoint:          es.srv.URL,
		Version:           "test-1.0.0",
		Deployment:        "binary",
		Store:             store,
		Counts:            counts,
		HeartbeatInterval: time.Hour, // only the initial heartbeat fires in tests
	})
	if c == nil {
		t.Fatal("New returned nil for an enabled config")
	}
	t.Cleanup(c.Stop)
	return c
}

// TestDisabledByDefault: without TELEMETRY_OPTIN=true there is no client, no
// network call, and no panic from any hook.
func TestDisabledByDefault(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	defer srv.Close()

	// Endpoint configured but opt-in absent: still fully off.
	t.Setenv("TELEMETRY_ENDPOINT", srv.URL)
	t.Setenv("TELEMETRY_OPTIN", "")

	c := NewFromEnv(nil, "test") // nil DB: must not be touched when disabled
	if c != nil {
		t.Fatal("NewFromEnv should return nil when TELEMETRY_OPTIN is not \"true\"")
	}

	// Every hook must be a no-op on the nil client.
	c.Start(context.Background())
	c.MilestoneFirstPlan()
	c.MilestoneFirstCustomer()
	c.MilestoneFirstInvoice()
	c.MilestoneFirstPayment()
	c.Stop()

	if calls != 0 {
		t.Fatalf("expected zero telemetry calls when disabled, got %d", calls)
	}

	if got := New(Config{Enabled: false}); got != nil {
		t.Fatal("New should return nil when Enabled is false")
	}
}

// TestNewFromEnvRejectsNonTrue: only the exact string "true" opts in.
func TestNewFromEnvRejectsNonTrue(t *testing.T) {
	for _, v := range []string{"1", "TRUE", "yes", "on", "false"} {
		t.Setenv("TELEMETRY_OPTIN", v)
		if c := NewFromEnv(nil, "test"); c != nil {
			t.Errorf("TELEMETRY_OPTIN=%q should not enable telemetry", v)
		}
	}
}

// TestInstanceStartedShape: the boot event carries exactly the documented
// anonymous fields.
func TestInstanceStartedShape(t *testing.T) {
	es := newEventServer(t)
	store := newMemStore()
	c := newTestClient(t, es, store, nil)

	c.Start(context.Background())
	got := es.wait(t, "instance_started")

	want := map[string]string{
		"event":      "instance_started",
		"version":    "test-1.0.0",
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"deployment": "binary",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("instance_started[%q] = %v, want %v", k, got[k], v)
		}
	}
	id, ok := got["instance_id"].(string)
	if !ok {
		t.Fatalf("instance_id missing or not a string: %v", got["instance_id"])
	}
	if parsed, err := uuid.Parse(id); err != nil || parsed != store.id {
		t.Errorf("instance_id = %q, want stored random UUID %q", id, store.id)
	}
	ts, ok := got["timestamp"].(string)
	if !ok {
		t.Fatalf("timestamp missing or not a string: %v", got["timestamp"])
	}
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("timestamp %q is not RFC3339: %v", ts, err)
	}
	// The payload must contain nothing beyond the documented keys.
	allowed := map[string]bool{
		"event": true, "instance_id": true, "version": true, "timestamp": true,
		"os": true, "arch": true, "deployment": true,
	}
	for k := range got {
		if !allowed[k] {
			t.Errorf("instance_started contains undocumented field %q", k)
		}
	}
}

// TestMilestoneFiresOnce: concurrent hooks, repeated hooks, and a restarted
// client all produce exactly one milestone event.
func TestMilestoneFiresOnce(t *testing.T) {
	es := newEventServer(t)
	store := newMemStore()
	c := newTestClient(t, es, store, nil)
	c.Start(context.Background())
	es.wait(t, "instance_started")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.MilestoneFirstInvoice()
		}()
	}
	wg.Wait()

	got := es.wait(t, "milestone_first_invoice")
	if got["instance_id"] != store.id.String() {
		t.Errorf("milestone instance_id = %v, want %v", got["instance_id"], store.id)
	}
	c.Stop() // waits for in-flight sends
	if n := es.countByName("milestone_first_invoice"); n != 1 {
		t.Fatalf("milestone_first_invoice sent %d times, want exactly 1", n)
	}

	// A new client over the same store (simulated restart) must not re-fire.
	c2 := newTestClient(t, es, store, nil)
	c2.Start(context.Background())
	es.wait(t, "instance_started")
	c2.MilestoneFirstInvoice()
	c2.Stop()
	if n := es.countByName("milestone_first_invoice"); n != 1 {
		t.Fatalf("milestone re-fired after restart: sent %d times, want 1", n)
	}
}

// TestHeartbeatBucketing: heartbeats carry bucketed ranges, never the exact
// counts.
func TestHeartbeatBucketing(t *testing.T) {
	es := newEventServer(t)
	counts := func(context.Context) (int64, int64, error) { return 5, 250, nil }
	c := newTestClient(t, es, newMemStore(), counts)

	c.Start(context.Background())
	got := es.wait(t, "heartbeat")

	if got["tenants"] != "1-9" {
		t.Errorf("heartbeat tenants = %v, want \"1-9\"", got["tenants"])
	}
	if got["subscriptions"] != "100+" {
		t.Errorf("heartbeat subscriptions = %v, want \"100+\"", got["subscriptions"])
	}
	raw, _ := json.Marshal(got)
	for _, field := range []string{"tenants", "subscriptions"} {
		if v, ok := got[field].(string); !ok || v == "5" || v == "250" {
			t.Errorf("heartbeat %s leaked an exact count: %v (payload %s)", field, got[field], raw)
		}
	}
}

func TestBucket(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0"}, {-3, "0"},
		{1, "1-9"}, {9, "1-9"},
		{10, "10-99"}, {99, "10-99"},
		{100, "100+"}, {1_000_000, "100+"},
	}
	for _, tc := range cases {
		if got := bucket(tc.n); got != tc.want {
			t.Errorf("bucket(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

// TestUnreachableEndpointNeverBlocksOrPanics: a dead endpoint is a debug log,
// nothing more.
func TestUnreachableEndpointNeverBlocksOrPanics(t *testing.T) {
	c := New(Config{
		Enabled:           true,
		Endpoint:          "http://127.0.0.1:1", // nothing listens here
		Version:           "test",
		Deployment:        "binary",
		Store:             newMemStore(),
		HeartbeatInterval: time.Hour,
	})
	c.Start(context.Background())
	c.MilestoneFirstPlan()
	c.MilestoneFirstPayment()
	c.Stop() // must return promptly even though every send failed
}
