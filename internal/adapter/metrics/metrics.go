// Package metrics is a small, dependency-free Prometheus metrics collector for
// the API: HTTP request counts + latency histograms (labelled by method, route,
// status) plus Go runtime gauges, emitted in the Prometheus text exposition
// format. It intentionally avoids pulling in client_golang — the surface here is
// just what /metrics needs.
package metrics

import (
	"fmt"
	"io"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// defaultBuckets are standard Prometheus latency buckets (seconds).
var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type hist struct {
	raw   []uint64 // per-bucket (non-cumulative) hit counts, len == len(buckets)+1 (last = +Inf); cumulative computed at write time
	sum   float64
	count uint64
}

// HTTPMetrics accumulates request counters and latency histograms keyed by
// (method, route, status). Route is the gin route template (bounded cardinality),
// never the raw path.
type HTTPMetrics struct {
	mu        sync.Mutex
	reqCount  map[string]uint64 // key: method\x1froute\x1fstatus
	durations map[string]*hist  // key: method\x1froute
	buckets   []float64
	start     time.Time
}

func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		reqCount:  map[string]uint64{},
		durations: map[string]*hist{},
		buckets:   defaultBuckets,
		start:     time.Now(),
	}
}

const sep = "\x1f"

// Observe records one completed request.
func (m *HTTPMetrics) Observe(method, route string, status int, seconds float64) {
	if route == "" {
		route = "<unmatched>"
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.reqCount[method+sep+route+sep+strconv.Itoa(status)]++

	dkey := method + sep + route
	h := m.durations[dkey]
	if h == nil {
		h = &hist{raw: make([]uint64, len(m.buckets)+1)}
		m.durations[dkey] = h
	}
	h.sum += seconds
	h.count++
	// Find the first bucket whose upper bound >= seconds; the last slot is +Inf.
	idx := sort.SearchFloat64s(m.buckets, seconds)
	h.raw[idx]++
}

// WriteProm writes all metrics in the Prometheus text exposition format.
func (m *HTTPMetrics) WriteProm(w io.Writer) {
	m.mu.Lock()
	// Snapshot under lock, format outside to keep the lock short.
	type reqLine struct {
		key string
		v   uint64
	}
	reqs := make([]reqLine, 0, len(m.reqCount))
	for k, v := range m.reqCount {
		reqs = append(reqs, reqLine{k, v})
	}
	type durLine struct {
		key string
		cum []uint64
		sum float64
		cnt uint64
	}
	durs := make([]durLine, 0, len(m.durations))
	for k, h := range m.durations {
		cum := make([]uint64, len(h.raw))
		var running uint64
		for i, c := range h.raw {
			running += c
			cum[i] = running
		}
		durs = append(durs, durLine{k, cum, h.sum, h.count})
	}
	buckets := m.buckets
	uptime := time.Since(m.start).Seconds()
	m.mu.Unlock()

	sort.Slice(reqs, func(i, j int) bool { return reqs[i].key < reqs[j].key })
	sort.Slice(durs, func(i, j int) bool { return durs[i].key < durs[j].key })

	var b strings.Builder

	b.WriteString("# HELP http_requests_total Total HTTP requests processed.\n")
	b.WriteString("# TYPE http_requests_total counter\n")
	for _, r := range reqs {
		p := strings.Split(r.key, sep) // method, route, status
		fmt.Fprintf(&b, "http_requests_total{method=%q,route=%q,status=%q} %d\n",
			p[0], p[1], p[2], r.v)
	}

	b.WriteString("# HELP http_request_duration_seconds HTTP request latency.\n")
	b.WriteString("# TYPE http_request_duration_seconds histogram\n")
	for _, d := range durs {
		p := strings.Split(d.key, sep) // method, route
		labels := fmt.Sprintf("method=%q,route=%q", p[0], p[1])
		for i, ub := range buckets {
			fmt.Fprintf(&b, "http_request_duration_seconds_bucket{%s,le=%q} %d\n", labels, ftoa(ub), d.cum[i])
		}
		fmt.Fprintf(&b, "http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", labels, d.cnt)
		fmt.Fprintf(&b, "http_request_duration_seconds_sum{%s} %s\n", labels, ftoa(d.sum))
		fmt.Fprintf(&b, "http_request_duration_seconds_count{%s} %d\n", labels, d.cnt)
	}

	// Runtime + process gauges (read at scrape time).
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	writeGauge(&b, "go_goroutines", "Number of goroutines.", float64(runtime.NumGoroutine()))
	writeGauge(&b, "go_memstats_alloc_bytes", "Heap bytes allocated and in use.", float64(ms.Alloc))
	writeGauge(&b, "go_memstats_sys_bytes", "Total bytes obtained from the OS.", float64(ms.Sys))
	writeGauge(&b, "go_gc_cycles_total", "Completed GC cycles.", float64(ms.NumGC))
	writeGauge(&b, "process_uptime_seconds", "Seconds since the process started.", uptime)

	_, _ = io.WriteString(w, b.String())
}

func writeGauge(b *strings.Builder, name, help string, v float64) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s gauge\n%s %s\n", name, help, name, name, ftoa(v))
}

// ftoa formats a float without a trailing ".000000" where it's an integer, and
// avoids scientific notation for the values we emit.
func ftoa(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", f), "0"), ".")
}
