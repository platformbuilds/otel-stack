package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// ---- helpers ----

type upstreams struct {
	prom *httptest.Server
	logs *httptest.Server
	ch   *httptest.Server
}

func startUpstreams(t *testing.T) *upstreams {
	t.Helper()

	// Prometheus fake: handle /-/healthy and /api/v1/query_range
	prom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/-/healthy":
			w.WriteHeader(200)
			w.Write([]byte("OK"))
			return
		case strings.HasPrefix(r.URL.Path, "/api/v1/query_range"):
			if r.Method != http.MethodGet {
				http.Error(w, "method", 405)
				return
			}
			q := r.URL.Query()
			if q.Get("query") == "" || q.Get("start") == "" || q.Get("end") == "" || q.Get("step") == "" {
				http.Error(w, "params", 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
			return
		default:
			http.NotFound(w, r)
		}
	}))

	// VictoriaLogs fake: handle /select/logsql/query (form-urlencoded)
	logs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/select/logsql/query" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			http.Error(w, "content-type", 400)
			return
		}
		if !strings.Contains(string(b), "query=") {
			http.Error(w, "body", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"hits": 1}`))
	}))

	// ClickHouse fake: return JSONEachRow for all POSTs
	ch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We loosely detect the handler by looking at SQL pattern the handlers send,
		// but we can simply return a valid JSONEachRow for each case.
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		// Very small bodies that match each handler's expectations:
		// /api/traces/list expects trace_roots rows
		// /api/traces/:id expects otel_traces rows
		// /api/traces/:id/flame expects span rows with start_ns/end_ns
		// Instead of trying to detect, just send something valid for all readers.
		body := "" +
			`{"TraceId":"t1","StartTs":"2025-01-01 10:00:00","DurationMs":100.0,"RootService":"api","RootOperation":"GET /x","Status":"OK","SpanCount":5,"TopService":"api","TopServiceMs":90.0}` + "\n" +
			`{"TraceId":"T","SpanId":"A","ParentSpanId":"","SpanName":"root","SpanKind":"SERVER","ServiceName":"api","start_ns":0,"end_ns":1000000,"SpanAttributes":{},"StatusCode":"OK","StatusMessage":""}` + "\n" +
			`{"SpanId":"A","ParentSpanId":"","SpanName":"root","ServiceName":"api","start_ns":0,"end_ns":1000000}` + "\n"
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))

	return &upstreams{prom: prom, logs: logs, ch: ch}
}

func stopUpstreams(u *upstreams) {
	u.prom.Close()
	u.logs.Close()
	u.ch.Close()
}

func withEnv(t *testing.T, key, val string) {
	t.Helper()
	prev := os.Getenv(key)
	_ = os.Setenv(key, val)
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, prev)
		}
	})
}

// ---- tests ----

func TestRoutes_Healthz_Readyz(t *testing.T) {
	u := startUpstreams(t)
	defer stopUpstreams(u)
	withEnv(t, "PROM_URL", u.prom.URL) // readyz will probe this

	h := New()
	ts := httptest.NewServer(h)
	defer ts.Close()

	// /healthz
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("healthz status=%d", resp.StatusCode)
	}

	// /readyz (calls /-/healthy on prom)
	resp2, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Fatalf("readyz status=%d", resp2.StatusCode)
	}
}

func TestMetricsQuery_RouteAndMethod(t *testing.T) {
	u := startUpstreams(t)
	defer stopUpstreams(u)
	withEnv(t, "PROM_URL", u.prom.URL)

	ts := httptest.NewServer(New())
	defer ts.Close()

	// POST is required
	body := map[string]any{"query": "up", "start": float64(1700000000), "end": float64(1700000600), "step": float64(60)}
	bs, _ := json.Marshal(body)
	res, err := http.Post(ts.URL+"/api/metrics/query", "application/json", bytes.NewReader(bs))
	if err != nil {
		t.Fatalf("POST metrics: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("metrics status=%d", res.StatusCode)
	}

	// GET should not exist (404)
	res2, _ := http.Get(ts.URL + "/api/metrics/query")
	if res2.StatusCode == 200 {
		t.Fatalf("GET metrics unexpectedly 200")
	}
}

func TestLogsSearch_RouteAndMethod(t *testing.T) {
	u := startUpstreams(t)
	defer stopUpstreams(u)
	withEnv(t, "VLOGS_URL", u.logs.URL)

	ts := httptest.NewServer(New())
	defer ts.Close()

	reqBody := `{"query":"error _time:5m"}`
	res, err := http.Post(ts.URL+"/api/logs/search", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST logs: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("logs status=%d", res.StatusCode)
	}

	// Wrong method
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/logs/search", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode == 200 {
		t.Fatalf("GET logs unexpectedly 200")
	}
}

func TestTracesEndpoints_SyntaxAndHappyPaths(t *testing.T) {
	u := startUpstreams(t)
	defer stopUpstreams(u)

	withEnv(t, "CH_HTTP_URL", u.ch.URL)
	withEnv(t, "CH_DATABASE", "default")

	ts := httptest.NewServer(New())
	defer ts.Close()

	// POST /api/traces/list
	post := map[string]any{
		"from": time.Now().Add(-time.Hour).Unix(),
		"to":   time.Now().Unix(),
		"filters": map[string]any{
			"status": []string{"OK"},
		},
		"sort": map[string]any{"by": "duration", "order": "DESC"},
		"page": map[string]any{"size": 10},
	}
	bs, _ := json.Marshal(post)
	res, err := http.Post(ts.URL+"/api/traces/list", "application/json", bytes.NewReader(bs))
	if err != nil {
		t.Fatalf("POST traces list: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("traces list status=%d", res.StatusCode)
	}

	// GET /api/traces/:id
	resp, err := http.Get(ts.URL + "/api/traces/T")
	if err != nil {
		t.Fatalf("GET trace: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("trace status=%d", resp.StatusCode)
	}

	// GET /api/traces/:id/flame
	resp2, err := http.Get(ts.URL + "/api/traces/T/flame?groupBy=service_operation&mode=self")
	if err != nil {
		t.Fatalf("GET flame: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Fatalf("flame status=%d", resp2.StatusCode)
	}

	// Suggestions
	resp3, _ := http.Get(ts.URL + "/api/traces/suggest/services?q=api")
	if resp3.StatusCode != 200 {
		t.Fatalf("suggest services status=%d", resp3.StatusCode)
	}
	resp4, _ := http.Get(ts.URL + "/api/traces/suggest/operations?q=GET")
	if resp4.StatusCode != 200 {
		t.Fatalf("suggest operations status=%d", resp4.StatusCode)
	}
	resp5, _ := http.Get(ts.URL + "/api/traces/suggest/attributes?key=http.method&q=GET")
	if resp5.StatusCode != 200 {
		t.Fatalf("suggest attributes status=%d", resp5.StatusCode)
	}
}

func TestUnknownRoutes_Return404(t *testing.T) {
	ts := httptest.NewServer(New())
	defer ts.Close()

	resp, _ := http.Get(ts.URL + "/does/not/exist")
	if resp.StatusCode != 404 {
		t.Fatalf("unknown route should be 404, got %d", resp.StatusCode)
	}
}
