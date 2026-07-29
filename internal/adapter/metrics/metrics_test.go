package metrics

import (
	"regexp"
	"strings"
	"testing"
)

func render(m *HTTPMetrics) string {
	var b strings.Builder
	m.WriteProm(&b)
	return b.String()
}

func TestHTTPMetrics_CounterAndFormat(t *testing.T) {
	m := NewHTTPMetrics()
	m.Observe("GET", "/v1/customers", 200, 0.01)
	m.Observe("GET", "/v1/customers", 200, 0.02)
	m.Observe("POST", "/v1/customers", 500, 0.3)

	out := render(m)

	if !strings.Contains(out, `http_requests_total{method="GET",route="/v1/customers",status="200"} 2`) {
		t.Errorf("GET counter missing/wrong:\n%s", out)
	}
	if !strings.Contains(out, `http_requests_total{method="POST",route="/v1/customers",status="500"} 1`) {
		t.Errorf("POST counter missing/wrong:\n%s", out)
	}
	// Required TYPE/HELP headers.
	for _, want := range []string{
		"# TYPE http_requests_total counter",
		"# TYPE http_request_duration_seconds histogram",
		"# TYPE go_goroutines gauge",
		"process_uptime_seconds",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output", want)
		}
	}
}

func TestHTTPMetrics_HistogramBucketsAreCumulativeAndConsistent(t *testing.T) {
	m := NewHTTPMetrics()
	// Latencies: 0.01 (→ le=0.01), 0.2 (→ le=0.25), 3 (→ le=5)
	m.Observe("GET", "/x", 200, 0.01)
	m.Observe("GET", "/x", 200, 0.2)
	m.Observe("GET", "/x", 200, 3)

	out := render(m)

	// +Inf bucket must equal the total count (3).
	reInf := regexp.MustCompile(`http_request_duration_seconds_bucket\{method="GET",route="/x",le="\+Inf"\} (\d+)`)
	if mch := reInf.FindStringSubmatch(out); mch == nil || mch[1] != "3" {
		t.Fatalf("+Inf bucket should be 3, got %v\n%s", mch, out)
	}
	// _count must be 3 and _sum must be 3.21.
	if !strings.Contains(out, `http_request_duration_seconds_count{method="GET",route="/x"} 3`) {
		t.Errorf("count wrong:\n%s", out)
	}
	if !strings.Contains(out, `http_request_duration_seconds_sum{method="GET",route="/x"} 3.21`) {
		t.Errorf("sum wrong:\n%s", out)
	}

	// Buckets must be monotonically non-decreasing.
	reBucket := regexp.MustCompile(`le="([0-9.]+)"\} (\d+)`)
	var prev = -1
	for _, mch := range reBucket.FindAllStringSubmatch(out, -1) {
		var v int
		_, _ = fmtSscan(mch[2], &v)
		if v < prev {
			t.Errorf("buckets not monotonic: %d after %d\n%s", v, prev, out)
		}
		prev = v
	}
	// le=0.005 should be 0 (nothing that fast); le=0.25 should be 2 (0.01 + 0.2).
	if !strings.Contains(out, `le="0.005"} 0`) {
		t.Errorf("le=0.005 should be 0:\n%s", out)
	}
	if !strings.Contains(out, `le="0.25"} 2`) {
		t.Errorf("le=0.25 should be 2:\n%s", out)
	}
}

func TestHTTPMetrics_UnmatchedRouteLabel(t *testing.T) {
	m := NewHTTPMetrics()
	m.Observe("GET", "", 404, 0.001) // gin FullPath is "" for unmatched routes
	out := render(m)
	if !strings.Contains(out, `route="<unmatched>"`) {
		t.Errorf("empty route should render as <unmatched>:\n%s", out)
	}
}

// fmtSscan is a tiny wrapper so the test doesn't import fmt just for Sscan.
func fmtSscan(s string, v *int) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	*v = n
	return 1, nil
}
