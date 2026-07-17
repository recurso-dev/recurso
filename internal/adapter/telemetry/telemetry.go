// Package telemetry implements opt-in, anonymous instance telemetry.
//
// It exists to answer one question the project cannot otherwise measure:
// how many self-hosted instances reach their first real invoice.
//
// Privacy rules (see docs/telemetry.md for the full contract):
//   - Strictly opt-in: nothing is sent and nothing is written unless
//     TELEMETRY_OPTIN=true. NewFromEnv returns nil otherwise, and every
//     method is safe to call on a nil *Client.
//   - Anonymous: the instance is identified by a random UUID generated once
//     and stored in the telemetry_instance table. It is not derived from
//     hostnames, MACs, license keys, or anything else identifying.
//   - Coarse: events carry milestones and bucketed counts ("1-9", "10-99",
//     "100+") — never PII, never amounts, never names/emails/keys.
//   - Fire-and-forget: one POST per event, 5-second timeout, no retries,
//     failures logged at debug level only. Telemetry can never affect a
//     request path.
package telemetry

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/recurso-dev/recurso/internal/residency"
)

// DefaultEndpoint receives events unless TELEMETRY_ENDPOINT overrides it.
// (The collector infrastructure behind this hostname is stood up separately;
// until it exists, enabled instances POST into the void and log at debug.)
const DefaultEndpoint = "https://telemetry.recurso.dev/v1/events"

// Milestone names, as sent in the "event" field and persisted as flags on
// the telemetry_instance row.
const (
	milestoneFirstPlan     = "first_plan"
	milestoneFirstCustomer = "first_customer"
	milestoneFirstInvoice  = "first_invoice"
	milestoneFirstPayment  = "first_payment"
)

const sendTimeout = 5 * time.Second

// CountsFunc returns coarse instance totals for the daily heartbeat. The
// exact numbers never leave the process — they are bucketed before sending.
type CountsFunc func(ctx context.Context) (tenants, subscriptions int64, err error)

// Config configures a telemetry Client. Used directly in tests; production
// wiring goes through NewFromEnv.
type Config struct {
	Enabled           bool
	Endpoint          string        // default: DefaultEndpoint
	Version           string        // build version stamped into every event
	Deployment        string        // "docker" | "binary"; auto-detected when empty
	Store             Store         // instance ID + milestone flag persistence
	Counts            CountsFunc    // heartbeat counts; nil sends heartbeats without counts
	HeartbeatInterval time.Duration // default: 24h
	Logger            *slog.Logger  // default: slog.Default()
}

// Client sends anonymous telemetry events. A nil *Client is valid and does
// nothing, so callers can hook milestones unconditionally.
type Client struct {
	endpoint  string
	version   string
	deploy    string
	store     Store
	counts    CountsFunc
	heartbeat time.Duration
	http      *http.Client
	logger    *slog.Logger

	mu         sync.Mutex
	instanceID uuid.UUID
	ready      bool
	fired      map[string]bool

	stopOnce sync.Once
	stop     chan struct{}
	wg       sync.WaitGroup
}

// NewFromEnv builds a Client from TELEMETRY_OPTIN / TELEMETRY_ENDPOINT.
// It returns nil — meaning telemetry is fully disabled, no network calls,
// no rows written — unless TELEMETRY_OPTIN is exactly "true". Under
// RESIDENCY_MODE=self_hosted, telemetry stays disabled even when opted in:
// the residency guarantee outranks the opt-in.
func NewFromEnv(database *sql.DB, version string) *Client {
	if os.Getenv("TELEMETRY_OPTIN") != "true" || residency.SelfHosted() {
		return nil
	}
	return New(Config{
		Enabled:  true,
		Endpoint: os.Getenv("TELEMETRY_ENDPOINT"),
		Version:  version,
		Store:    NewPostgresStore(database),
		Counts:   postgresCounts(database),
	})
}

// New builds a Client from cfg. Returns nil when cfg.Enabled is false.
func New(cfg Config) *Client {
	if !cfg.Enabled {
		return nil
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 24 * time.Hour
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Deployment == "" {
		cfg.Deployment = detectDeployment()
	}
	return &Client{
		endpoint:  cfg.Endpoint,
		version:   cfg.Version,
		deploy:    cfg.Deployment,
		store:     cfg.Store,
		counts:    cfg.Counts,
		heartbeat: cfg.HeartbeatInterval,
		http:      &http.Client{Timeout: sendTimeout},
		logger:    cfg.Logger.With("component", "telemetry"),
		fired:     map[string]bool{},
		stop:      make(chan struct{}),
	}
}

// Start loads (or creates) the anonymous instance row, sends instance_started
// plus an initial heartbeat, and begins the daily heartbeat loop. Safe on nil.
func (c *Client) Start(ctx context.Context) {
	if c == nil {
		return
	}
	instanceID, fired, err := c.store.EnsureInstance(ctx)
	if err != nil {
		// Without an instance ID nothing can be sent; stay dormant.
		c.logger.Debug("telemetry disabled: could not load instance identity", "error", err)
		return
	}
	c.mu.Lock()
	c.instanceID = instanceID
	for name, done := range fired {
		if done {
			c.fired[name] = true
		}
	}
	c.ready = true
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.send("instance_started", map[string]any{
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
			"deployment": c.deploy,
		})
		c.sendHeartbeat()

		ticker := time.NewTicker(c.heartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.sendHeartbeat()
			case <-c.stop:
				return
			}
		}
	}()
}

// Stop ends the heartbeat loop. Safe on nil and safe to call more than once.
func (c *Client) Stop() {
	if c == nil {
		return
	}
	c.stopOnce.Do(func() { close(c.stop) })
	c.wg.Wait()
}

// MilestoneFirstPlan records that this instance created its first plan.
func (c *Client) MilestoneFirstPlan() { c.milestone(milestoneFirstPlan) }

// MilestoneFirstCustomer records that this instance created its first customer.
func (c *Client) MilestoneFirstCustomer() { c.milestone(milestoneFirstCustomer) }

// MilestoneFirstInvoice records that this instance generated its first invoice —
// the activation metric.
func (c *Client) MilestoneFirstInvoice() { c.milestone(milestoneFirstInvoice) }

// MilestoneFirstPayment records that this instance collected its first payment.
func (c *Client) MilestoneFirstPayment() { c.milestone(milestoneFirstPayment) }

// milestone fires each milestone at most once per instance (flag persisted in
// the telemetry_instance row, cached in memory) and never blocks the caller.
func (c *Client) milestone(name string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if !c.ready || c.fired[name] {
		c.mu.Unlock()
		return
	}
	c.fired[name] = true // check-once: later calls return above
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		flipped, err := c.store.MarkMilestone(context.Background(), name)
		if err != nil {
			c.logger.Debug("telemetry milestone not persisted", "milestone", name, "error", err)
			return
		}
		if !flipped {
			return // already recorded by an earlier run or another replica
		}
		c.send("milestone_"+name, nil)
	}()
}

// sendHeartbeat sends the daily heartbeat with bucketed counts.
func (c *Client) sendHeartbeat() {
	props := map[string]any{
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"deployment": c.deploy,
	}
	if c.counts != nil {
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		tenants, subs, err := c.counts(ctx)
		cancel()
		if err != nil {
			c.logger.Debug("telemetry heartbeat counts unavailable", "error", err)
		} else {
			props["tenants"] = bucket(tenants)
			props["subscriptions"] = bucket(subs)
		}
	}
	c.send("heartbeat", props)
}

// send POSTs one event: a single attempt with a 5s timeout, no retries, and
// failures logged at debug only.
func (c *Client) send(event string, props map[string]any) {
	payload := map[string]any{
		"event":       event,
		"instance_id": c.instanceID.String(),
		"version":     c.version,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range props {
		payload[k] = v
	}

	body, err := json.Marshal(payload)
	if err != nil {
		c.logger.Debug("telemetry event not sent", "event", event, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		c.logger.Debug("telemetry event not sent", "event", event, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		c.logger.Debug("telemetry event not sent", "event", event, "error", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.Debug("telemetry event rejected", "event", event, "status", resp.StatusCode)
	}
}

// bucket collapses an exact count into a coarse range so real numbers never
// leave the instance.
func bucket(n int64) string {
	switch {
	case n <= 0:
		return "0"
	case n < 10:
		return "1-9"
	case n < 100:
		return "10-99"
	default:
		return "100+"
	}
}

// detectDeployment returns a coarse hint of how the binary runs: "docker"
// when a container marker file exists, otherwise "binary".
func detectDeployment() string {
	for _, marker := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(marker); err == nil {
			return "docker"
		}
	}
	return "binary"
}
